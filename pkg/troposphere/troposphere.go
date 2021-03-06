// Package troposphere is a Go version of the Python package and provides Go
// primitives for building CloudFormation templates.
package troposphere

// Template represents a CloudFormation template that can be built.
type Template struct {
	Conditions map[string]interface{}
	Outputs    map[string]Output
	Parameters map[string]Parameter
	Resources  map[string]Resource
}

// NewTemplate returns an initialized Template.
func NewTemplate() *Template {
	return &Template{
		Conditions: make(map[string]interface{}),
		Outputs:    make(map[string]Output),
		Parameters: make(map[string]Parameter),
		Resources:  make(map[string]Resource),
	}
}

// Parameter represents a CloudFormation parameter.
type Parameter struct {
	Type        interface{} `json:"Type,omitempty"`
	Description interface{} `json:"Description,omitempty"`
	Default     interface{} `json:"Default,omitempty"`
}

// Output represents an CloudFormation output.
type Output struct {
	Value interface{} `json:"Value,omitempty"`
}

// Resource represents a CloudFormation Resource.
type Resource struct {
	Condition  interface{} `json:"Condition,omitempty"`
	Properties interface{} `json:"Properties,omitempty"`
	Type       interface{} `json:"Type,omitempty"`
	Version    interface{} `json:"Version,omitempty"`
}
