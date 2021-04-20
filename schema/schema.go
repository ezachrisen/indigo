// Package schema defines the data types used by Indigo.
package schema

// Schema defines the keys (variable names) and their data types used in a
// rule expression. The same keys and types must be supplied in the data map
// when rules are evaluated.
type Schema struct {
	// Identifier for the schema. Useful for the hosting application; not used by Indigo internally.
	ID string `json:"id,omitempty"`
	// User-friendly name for the schema
	Name string `json:"name,omitempty"`
	// A user-friendly description of the schema
	Description string `json:"description,omitempty"`
	// User-defined value
	Meta interface{} `json:"-"`
	// List of data elements supported by this schema
	Elements []DataElement `json:"elements,omitempty"`
}

// DataElement defines a named variable in a schema
type DataElement struct {
	// Short, user-friendly name of the variable. This is the name
	// that will be used in rules to refer to data passed in.
	//
	// RESERVED NAMES:
	//   selfKey (see const)
	Name string `json:"name"`

	// One of the Type interface defined.
	Type Type `json:"type"`

	// Optional description of the type.
	Description string `json:"description"`
}

// Type defines a type in the Indigo type system.
// These types are used to define schemas, define required
// evaluation results, and to interpret evaluation results.
// Not all implementations of Evaluator support all types.
type Type interface {
	String() string
}

type String struct{}
type Int struct{}
type Float struct{}
type Any struct{}
type Bool struct{}
type Duration struct{}
type Timestamp struct{}
type Proto struct {
	Protoname string
	Message   interface{}
}
type List struct {
	ValueType Type
}

type Map struct {
	KeyType   Type
	ValueType Type
}

func (t Int) String() string       { return "int" }
func (t Bool) String() string      { return "bool" }
func (t String) String() string    { return "string" }
func (t List) String() string      { return "list" }
func (t Map) String() string       { return "map" }
func (t Any) String() string       { return "any" }
func (t Duration) String() string  { return "duration" }
func (t Timestamp) String() string { return "timestamp" }
func (t Float) String() string     { return "float" }
func (t Proto) String() string     { return "proto " + t.Protoname }

// The value returned in the Result.
// Inspect the Typ to determine what it is.
type Value struct {
	Val interface{}
	Typ Type
}
