package indigo

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

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

func (s *Schema) String() string {
	x := strings.Builder{}
	x.WriteString(s.ID)
	if s.Name != "" {
		x.WriteString("  '" + s.Name + "'")
	}
	x.WriteString("\n")
	for _, e := range s.Elements {
		x.WriteString(e.String())
		x.WriteString("\n")
	}

	return x.String()
}

func (e *DataElement) String() string {
	return fmt.Sprintf("  %s (%s)", e.Name, e.Type)
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
	// Implements the stringer interface
	String() string

	// Zero returns a 'template' of the type to enable
	// use of reflection in Evaluators and elsewhere to convert to/from
	// indigo types and the types native to the Evaluators.
	Zero() interface{}
}

// String defines an Indigo string type.
type String struct{}

// Int defines an Indigo int type. The exact "Int" implementation and size
// depends on the evaluator used.
type Int struct{}

// Float defines an Indigo float type. The implementation of the float (size, precision)
// depends on the evaluator used.
type Float struct{}

// Any defines an Indigo type for an "undefined" or unspecified type.
type Any struct{}

// Bool defines an Indigo type for true/false.
type Bool struct{}

// Duration defines an Indigo type for the time.Duration type.
type Duration struct{}

// Timestamp defines an Indigo type for the time.Time type.
type Timestamp struct{}

// Proto defines an Indigo type for a protobuf type.
type Proto struct {
	Protoname string      // fully qualified name of the protobuf type
	Message   interface{} // an empty protobuf instance of the type
}

// List defines an Indigo type representing a slice of values
type List struct {
	ValueType Type // the type of element stored in the list
}

// Map defines an Indigo type representing a map of keys and values.
type Map struct {
	KeyType   Type // the type of the map key
	ValueType Type // the type of the value stored in the map
}

// Zero Methods
func (String) Zero() interface{}    { return string("") }
func (Int) Zero() interface{}       { return int(0) }
func (Bool) Zero() interface{}      { return bool(false) }
func (Float) Zero() interface{}     { return float64(0.0) }
func (Timestamp) Zero() interface{} { return time.Now() }
func (Duration) Zero() interface{}  { return time.Duration(0) }
func (t Proto) Zero() interface{}   { return t.Message }
func (Any) Zero() interface{}       { return nil }

func (t List) Zero() (retval interface{}) {
	defer func() {
		if r := recover(); r != nil {
			retval = nil
		}
	}()

	if t.ValueType == nil {
		return nil
	}

	if t.ValueType.Zero() == nil {
		return nil
	}

	tt := reflect.TypeOf(t.ValueType.Zero())
	if tt == nil {
		return nil
	}

	rt := reflect.SliceOf(tt)
	if rt == nil {
		return nil
	}

	s := reflect.MakeSlice(rt, 0, 0)
	return s.Interface()
}

func (t Map) Zero() (retval interface{}) {
	// A panic handler here because we're using reflection
	defer func() {
		if r := recover(); r != nil {
			retval = nil
		}
	}()

	if t.ValueType == nil {
		return nil
	}

	if t.ValueType.Zero() == nil {
		return nil
	}

	if t.KeyType == nil {
		return nil
	}

	if t.KeyType.Zero() == nil {
		return nil
	}

	tv := reflect.TypeOf(t.ValueType.Zero())
	if tv == nil {
		return nil
	}

	tk := reflect.TypeOf(t.KeyType.Zero())
	if tk == nil {
		return nil
	}

	tm := reflect.MapOf(tk, tv)
	if tm == nil {
		return nil
	}

	m := reflect.MakeMap(tm)
	return m.Interface()
}

// String Methods
func (Int) String() string       { return "int" }
func (Bool) String() string      { return "bool" }
func (String) String() string    { return "string" }
func (Any) String() string       { return "any" }
func (Duration) String() string  { return "duration" }
func (Timestamp) String() string { return "timestamp" }
func (Float) String() string     { return "float" }
func (t Proto) String() string   { return "proto(" + t.Protoname + ")" }
func (t List) String() string    { return fmt.Sprintf("[]%v", t.ValueType) }
func (t Map) String() string     { return fmt.Sprintf("map[%s]%s", t.KeyType, t.ValueType) }

// Value is the result of evaluation returned in the Result.
// Inspect the Type to determine what it is.
type Value struct {
	Val  interface{} // the value stored
	Type Type        // the Indigo type stored
}

// ParseType parses a string that represents an Indigo type and returns the type.
// The primitive types are their lower-case names (string, int, duration, etc.)
// Maps and lists look like Go maps and slices: map[string]float and []string.
// Proto types look like this: proto(protoname)
func ParseType(t string) (Type, error) {

	if strings.Contains(t, "map") {
		return parseMap(t)
	}

	if strings.Contains(t, "[]") {
		return parseList(t)
	}

	if strings.Contains(t, "proto(") {
		return parseProto(t)
	}

	switch t {
	case "string":
		return String{}, nil
	case "int":
		return Int{}, nil
	case "float":
		return Float{}, nil
	case "bool":
		return Bool{}, nil
	case "duration":
		return Duration{}, nil
	case "timestamp":
		return Timestamp{}, nil
	case "any":
		return Any{}, nil
	default:
		return Any{}, fmt.Errorf("unrecognized type: %s", t)
	}
}

// parseMap parses a string and returns an Indigo map type.
// The string must in the format map[<keytype]<valuetype>.
// Example: map[string]int
func parseMap(t string) (Type, error) {

	var keyTypeName string
	var valueTypeName string

	t = strings.ReplaceAll(t, "[", " ")
	t = strings.ReplaceAll(t, "]", " ")

	n, err := fmt.Sscanf(t, "map %s %s", &keyTypeName, &valueTypeName)
	if err != nil {
		return Any{}, err
	}

	if n < 2 {
		return Any{}, fmt.Errorf("wanted 2 items parsed, got %d", n)
	}

	keyType, err := ParseType(keyTypeName)
	if err != nil {
		return Any{}, err
	}

	valueType, err := ParseType(valueTypeName)
	if err != nil {
		return Any{}, err
	}

	return Map{
		KeyType:   keyType,
		ValueType: valueType,
	}, nil
}

// parseList parses a string and returns an Indigo list type.
// The string must be in the format []<valuetype>
// Example: []string
func parseList(t string) (Type, error) {
	var valueTypeName string
	_, err := fmt.Sscanf(t, "[]%s", &valueTypeName)
	if err != nil {
		return Any{}, err
	}
	valueType, err := ParseType(valueTypeName)
	if err != nil {
		return Any{}, err
	}

	return List{
		ValueType: valueType,
	}, nil
}

// parseProto parses a string and returns a partial Indigo proto type.
// The "Message" field of the proto struct must be suppplied later.
// The string must be in the form proto(<protoname>).
// Example: proto("school.Student")
func parseProto(t string) (Type, error) {
	startParen := strings.Index(t, "(")
	endParen := strings.Index(t, ")")

	if startParen == -1 || endParen == -1 || startParen > endParen || endParen > len(t) || endParen-startParen == 1 {
		return nil, fmt.Errorf("bad proto specification")
	}

	name := t[startParen+1 : endParen]
	return Proto{Protoname: name}, nil
}
