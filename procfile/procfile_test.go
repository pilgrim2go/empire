package procfile

import (
	"io"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/stretchr/testify/assert"
)

var parseTests = []struct {
	in  io.Reader
	out Procfile
	err error
}{
	// Simple standard Procfile.
	{
		strings.NewReader(`---
web: ./bin/web`),
		StandardProcfile{
			"web": "./bin/web",
		},
		nil,
	},

	// Extended Procfile with health checks and http exposure.
	{
		strings.NewReader(`---
web:
  command: ./bin/web`),
		ExtendedProcfile{
			"web": Process{
				Command: "./bin/web",
			},
		},
		nil,
	},

	// Extended Procfile with health checks and http exposure.
	{
		strings.NewReader(`---
web:
  command:
    - nginx
    - -g
    - daemon off;
  expose:
    protocol: tcp
    external: true`),
		ExtendedProcfile{
			"web": Process{
				Command: []string{
					"nginx",
					"-g",
					"daemon off;",
				},
				Expose: &Exposure{
					External: true,
					Protocol: "tcp",
				},
			},
		},
		nil,
	},

	// Extended Procfile with malformed command
	{
		strings.NewReader(`---
web:
  command:
    - nginx: g
worker:
  command:
    nginx: g`),
		ExtendedProcfile{},
		&ParseError{
			YamlErrors: &yaml.TypeError{
				Errors: []string{
					"command should be provided as a string or []string",
					"command should be provided as a string or []string",
				},
			},
		},
	},
}

func TestParse(t *testing.T) {
	for _, tt := range parseTests {
		p, err := Parse(tt.in)
		assert.Equal(t, tt.err, err)
		assert.Equal(t, tt.out, p)
	}
}
