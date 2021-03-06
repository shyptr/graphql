package ast

import (
	"github.com/shyptr/graphql/errors"
	"github.com/shyptr/graphql/kinds"
)

// A GraphQL query can be parameterized with variables, maximizing query reuse,
// and avoiding costly string building in clients at runtime.
//
// If not defined as constant (for example, in DefaultValue), a Variable can be supplied for an input value.
//
// Variables must be defined at the top of an operation and are in scope throughout the execution of that operation.
//
// In this example, we want to fetch a profile picture size based on the size of a particular device:
//
// query getZuckProfile($devicePicSize: Int) {
//   user(id: 4) {
//     id
//     name
//     profilePic(size: $devicePicSize)
//   }
// }
// Values for those variables are provided to a GraphQL service along with a request so they may be substituted during execution.
// If providing JSON for the variables’ values, we could run this query and request profilePic of size 60 width:
//
// {
//   "devicePicSize": 60
// }
// Variable use within Fragments
// Query variables can be used within fragments. Query variables have global scope with a given operation,
// so a variable used within a fragment must be declared in any top‐level operation that transitively consumes that fragment.
// If a variable is referenced in a fragment and is included by an operation that does not define that variable,
// the operation cannot be executed.
type Variable struct {
	Kind string          `json:"kind"`
	Name *Name           `json:"name"`
	Loc  errors.Location `json:"loc"`
}

func (v *Variable) GetKind() string {
	return kinds.Variable
}

func (v *Variable) Location() errors.Location {
	return v.Loc
}

func (v *Variable) GetValue() interface{} { return v.Name }

type VariableDefinition struct {
	Kind         string          `json:"kind"`
	Var          *Variable       `json:"variable"`
	Type         Type            `json:"type"`
	DefaultValue Value           `json:"defaultValue"`
	Directives   []*Directive    `json:"directives"`
	Loc          errors.Location `json:"loc"`
}

func (v *VariableDefinition) GetKind() string {
	return kinds.VariableDefinition
}

func (v *VariableDefinition) Location() errors.Location {
	return v.Loc
}
