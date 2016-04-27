package procfile

import (
	"fmt"
	"io"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Procfile is a Go representation of process configuration.
type Procfile interface {
}

// ExtendedProcfile represents the extended Procfile format.
type ExtendedProcfile map[string]Process

type Process struct {
	Command interface{} `yaml:"command"`
	Expose  *Exposure   `yaml:"expose,omitempty"`
}

// alias so we can get the standard unmarshaller.
type process Process

var errCommand = &yaml.TypeError{
	Errors: []string{fmt.Sprintf("command should be provided as a string or []string")},
}

// UnmarshalYAML implements the yaml.Unmarshaller interface.
func (p *Process) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var proc process
	err := unmarshal(&proc)
	if err != nil {
		return err
	}
	*p = Process(proc)

	p.Command, err = coerceCommand(p.Command)
	return err
}

// coerceCommand coerces the command to a string or []string, or returns an
// error if it's invalid.
func coerceCommand(v interface{}) (interface{}, error) {
	switch v := v.(type) {
	case string:
		return v, nil
	case []interface{}:
		var a []string
		for _, s := range v {
			s, ok := s.(string)
			if !ok {
				return nil, errCommand
			}
			a = append(a, s)
		}
		return a, nil
	default:
		return nil, errCommand
	}
}

type Exposure struct {
	Protocol string `yaml:"protocol"`
	External bool   `yaml:"external"`
}

// StandardProcfile represents a standard Procfile.
type StandardProcfile map[string]string

// Marshal marshals the Procfile to yaml format.
func Marshal(p Procfile) ([]byte, error) {
	return yaml.Marshal(p)
}

// ParseError is returned when there are errors parsing the yaml document.
type ParseError struct {
	YamlErrors *yaml.TypeError
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("error parsing Procfile: %s", e.YamlErrors.Error())
}

// Parse parses the Procfile by reading from r.
func Parse(r io.Reader) (Procfile, error) {
	raw, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return ParseProcfile(raw)
}

// ParseProcfile takes a byte slice representing a YAML Procfile and parses it
// into a Procfile.
func ParseProcfile(b []byte) (p Procfile, err error) {
	p, err = parseStandardProcfile(b)
	if err != nil {
		p, err = parseExtendedProcfile(b)
	}

	if err, ok := err.(*yaml.TypeError); ok {
		return p, &ParseError{YamlErrors: err}
	}

	return p, err
}

func parseExtendedProcfile(b []byte) (Procfile, error) {
	y := make(ExtendedProcfile)
	err := yaml.Unmarshal(b, &y)
	return y, err
}

func parseStandardProcfile(b []byte) (Procfile, error) {
	y := make(StandardProcfile)
	err := yaml.Unmarshal(b, &y)
	return y, err
}
