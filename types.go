package indigo

// Schema defines the keys (variable names) and their data types used in a
// rule expression. The same keys and types must be supplied in the data map
// when rules are evaluated.
type Schema struct {
	ID       string
	Name     string
	Meta     interface{}
	Elements []DataElement
}

// DataElement defines a named variable in a schema
type DataElement struct {
	// Short, user-friendly name of the variable. This is the name
	// that will be used in rules to refer to data passed in.
	//
	// RESERVED NAMES:
	//   selfKey (see const)
	Name string

	// One of the Type interface defined.
	Type Type

	// Optional description of the type.
	Description string
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
