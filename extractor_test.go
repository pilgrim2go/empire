package empire

import (
	"archive/tar"
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/pkg/httpmock"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/procfile"
	"github.com/stretchr/testify/assert"
)

func TestCMDExtractor(t *testing.T) {
	api := httpmock.NewServeReplay(t).Add(httpmock.PathHandler(t,
		"GET /images/remind101:acme-inc/json",
		200, `{ "Config": { "Cmd": ["/go/bin/app","server"] } }`,
	))

	c, s := newTestDockerClient(t, api)
	defer s.Close()

	e := CMDExtractor{
		client: c,
	}

	procfile, err := e.Extract(nil, image.Image{
		Tag:        "acme-inc",
		Repository: "remind101",
	}, nil)
	assert.NoError(t, err)

	expected := []byte(`web:
  command:
  - /go/bin/app
  - server
`)

	assert.Equal(t, expected, procfile)
}

func TestProcfileExtractor(t *testing.T) {
	api := httpmock.NewServeReplay(t).Add(httpmock.PathHandler(t,
		"POST /containers/create",
		200, `{ "ID": "abc" }`,
	)).Add(httpmock.PathHandler(t,
		"GET /containers/abc/json",
		200, `{}`,
	)).Add(httpmock.PathHandler(t,
		"POST /containers/abc/copy",
		200, tarProcfile(t),
	)).Add(httpmock.PathHandler(t,
		"DELETE /containers/abc",
		200, `{}`,
	))

	c, s := newTestDockerClient(t, api)
	defer s.Close()

	e := FileExtractor{
		client: c,
	}

	procfile, err := e.Extract(nil, image.Image{
		Tag:        "acme-inc",
		Repository: "remind101",
	}, nil)
	assert.NoError(t, err)
	expected := []byte(`web: rails server`)
	assert.Equal(t, expected, procfile)

}

func TestProcfileFallbackExtractor(t *testing.T) {
	api := httpmock.NewServeReplay(t).Add(httpmock.PathHandler(t,
		"POST /containers/create",
		200, `{ "ID": "abc" }`,
	)).Add(httpmock.PathHandler(t,
		"GET /containers/abc/json",
		200, `{}`,
	)).Add(httpmock.PathHandler(t,
		"POST /containers/abc/copy",
		404, ``,
	)).Add(httpmock.PathHandler(t,
		"DELETE /containers/abc",
		200, `{}`,
	)).Add(httpmock.PathHandler(t,
		"GET /images/remind101:acme-inc/json",
		200, `{ "Config": { "Cmd": ["/go/bin/app","server"] } }`,
	))

	c, s := newTestDockerClient(t, api)
	defer s.Close()

	e := MultiExtractor(
		NewFileExtractor(c),
		NewCMDExtractor(c),
	)

	procfile, err := e.Extract(nil, image.Image{
		Tag:        "acme-inc",
		Repository: "remind101",
	}, nil)
	assert.NoError(t, err)

	expected := []byte(`web:
  command:
  - /go/bin/app
  - server
`)

	assert.Equal(t, expected, procfile)
}

func TestFormationFromProcfile(t *testing.T) {
	tests := []struct {
		app      *App
		procfile procfile.Procfile

		formation Formation
		err       error
	}{
		// Standard Procfile with an app with no domains or cert.
		{
			&App{},
			procfile.StandardProcfile{
				"web":    "./bin/web",
				"worker": "./bin/worker",
			},
			Formation{
				"web": Process{
					Command: Command{"./bin/web"},
					Expose: &Exposure{
						External: false,
						Protocol: "http",
					},
				},
				"worker": Process{
					Command: Command{"./bin/worker"},
				},
			},
			nil,
		},

		// Standard Procfile with an app with a domain and cert.
		{
			&App{
				Exposure: ExposePublic,
				Cert:     "cert",
			},
			procfile.StandardProcfile{
				"web":    "./bin/web",
				"worker": "./bin/worker",
			},
			Formation{
				"web": Process{
					Command: Command{"./bin/web"},
					Expose: &Exposure{
						External: true,
						Protocol: "https",
						Cert:     "cert",
					},
				},
				"worker": Process{
					Command: Command{"./bin/worker"},
				},
			},
			nil,
		},

		// Extended Procfile with basic settings.
		{
			&App{
				Cert: "cert",
			},
			procfile.ExtendedProcfile{
				"web": procfile.Process{
					Command: "./bin/web",
					Expose: &procfile.Exposure{
						External: true,
						Protocol: "ssl",
					},
				},
				"worker": procfile.Process{
					Command: []string{"./bin/worker"},
				},
			},
			Formation{
				"web": Process{
					Command: Command{"./bin/web"},
					Expose: &Exposure{
						External: true,
						Protocol: "ssl",
						Cert:     "cert",
					},
				},
				"worker": Process{
					Command: Command{"./bin/worker"},
				},
			},
			nil,
		},
	}

	for _, tt := range tests {
		formation, err := formationFromProcfile(tt.app, tt.procfile)
		assert.Equal(t, tt.err, err)
		assert.Equal(t, tt.formation, formation)
	}
}

// newTestDockerClient returns a docker.Client configured to talk to the given http.Handler
func newTestDockerClient(t *testing.T, fakeDockerAPI http.Handler) (*docker.Client, *httptest.Server) {
	s := httptest.NewServer(fakeDockerAPI)

	c, err := docker.NewClient(s.URL)
	if err != nil {
		t.Fatal(err)
	}

	return c, s
}

func tarProcfile(t *testing.T) string {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	var files = []struct {
		Name, Body string
	}{
		{"Procfile", "web: rails server"},
	}

	for _, file := range files {
		hdr := &tar.Header{
			Name: file.Name,
			Size: int64(len(file.Body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(file.Body)); err != nil {
			t.Fatal(err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	return buf.String()
}
