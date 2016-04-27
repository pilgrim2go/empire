package empire

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	shellwords "github.com/mattn/go-shellwords"
	"github.com/remind101/empire/pkg/constraints"
)

// Acceptable exposure protocols. This will be expanded to tcp/ssl/etc down the
// road.
var Protocols = map[string]bool{
	"http":  true,
	"https": true,
	"tcp":   false,
	"ssl":   false,
}

// DefaultQuantities maps a process type to the default number of instances to
// run.
var DefaultQuantities = map[string]int{
	"web": 1,
}

// Command represents the actual shell command that gets executed for a given
// ProcessType.
type Command []string

// ParseCommand parses a string into a Command, taking quotes and other shell
// words into account.
func ParseCommand(command string) (Command, error) {
	return shellwords.Parse(command)
}

// Scan implements the sql.Scanner interface.
func (c *Command) Scan(src interface{}) error {
	bytes, ok := src.([]byte)
	if !ok {
		return error(errors.New("Scan source was not []bytes"))
	}

	var cmd Command
	if err := json.Unmarshal(bytes, &cmd); err != nil {
		return err
	}
	*c = cmd

	return nil
}

// Value implements the driver.Value interface.
func (c Command) Value() (driver.Value, error) {
	raw, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return driver.Value(raw), nil
}

// String returns the string reprsentation of the command.
func (c Command) String() string {
	return strings.Join([]string(c), " ")
}

// Process holds configuration information about a Process.
type Process struct {
	Command  Command              `json:"Command,omitempty"`
	Quantity int                  `json:"Quantity,omitempty"`
	Memory   constraints.Memory   `json:"Memory,omitempty"`
	CPUShare constraints.CPUShare `json:"CPUShare,omitempty"`
	Nproc    constraints.Nproc    `json:"Nproc,omitempty"`
	Expose   *Exposure            `json:"Expose,omitempty"`
}

// Exposure holds exposure settings for the Process.
type Exposure struct {
	Protocol string `json:"Protocol"`
	External bool   `json:"External"`
	Cert     string `json:"Cert,omitempty"` // Only relevant for SSL/HTTPS types.
}

func (e *Exposure) IsValid() error {
	if _, ok := Protocols[e.Protocol]; !ok {
		return fmt.Errorf("unable to expose %v", e.Protocol)
	}

	return nil
}

func (p *Process) IsValid() error {
	if p.Expose != nil {
		return p.Expose.IsValid()
	}

	return nil
}

// Constraints returns a constraints.Constraints from this Process definition.
func (p *Process) Constraints() Constraints {
	return Constraints{
		Memory:   p.Memory,
		CPUShare: p.CPUShare,
		Nproc:    p.Nproc,
	}
}

// SetConstraints sets the memory/cpu/nproc for this Process to the given
// constraints.
func (p *Process) SetConstraints(c Constraints) {
	p.Memory = c.Memory
	p.CPUShare = c.CPUShare
	p.Nproc = c.Nproc
}

// Formation represents a collection of named processes and their configuration.
type Formation map[string]Process

func (f Formation) IsValid() error {
	for name, p := range f {
		if err := p.IsValid(); err != nil {
			return fmt.Errorf("%s process is invalid: %v", name, err)
		}
	}

	return nil
}

// Scan implements the sql.Scanner interface.
func (f *Formation) Scan(src interface{}) error {
	bytes, ok := src.([]byte)
	if !ok {
		return error(errors.New("Scan source was not []bytes"))
	}

	formation := make(Formation)
	if err := json.Unmarshal(bytes, &formation); err != nil {
		return err
	}
	*f = formation

	return nil
}

// Value implements the driver.Value interface.
func (f Formation) Value() (driver.Value, error) {
	if f == nil {
		return nil, nil
	}

	raw, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}

	return driver.Value(raw), nil
}

// Merge merges in the existing quantity and constraints from the old Formation
// into this Formation.
func (f Formation) Merge(other Formation) Formation {
	new := make(Formation)

	for name, p := range f {
		if existing, found := other[name]; found {
			// If the existing Formation already had a process
			// configuration for this process type, copy over the
			// instance count.
			p.Quantity = existing.Quantity
			p.SetConstraints(existing.Constraints())
		} else {
			p.Quantity = DefaultQuantities[name]
			p.SetConstraints(DefaultConstraints)
		}

		new[name] = p
	}

	return new
}
