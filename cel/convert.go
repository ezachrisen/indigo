// This file contains functions that convert to and from CEL's type system and Indigo's.
// CEL uses two type systems.
// The first type system is used to represent data types for the values in the expression.
// This CEL "schema" is used to check the expression during compilation.
// There are 2 functions used to create the CEL "schema":
//  - convertIndigoSchemaToDeclarations
//  - convertIndigoToExprType
//
// The second type system is used to represent values produced by the evaluation.
// There is one funtion used to convert the produced CEL values to an Indigo type:
//  - convertRefValToIndigo
//
package cel

import (
	"fmt"
	"reflect"
	"time"

	"github.com/ezachrisen/indigo"
	celgo "github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	gexpr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/protobuf/types/dynamicpb"
)

// convertIndigoToExprType converts from an indigo type to a expr.Type,
// which is used by CEL to represent types in its schema.
func convertIndigoToExprType(t indigo.Type) (*gexpr.Type, error) {

	switch v := t.(type) {
	case indigo.String:
		return decls.String, nil
	case indigo.Int:
		return decls.Int, nil
	case indigo.Float:
		return decls.Double, nil
	case indigo.Bool:
		return decls.Bool, nil
	case indigo.Duration:
		return decls.Duration, nil
	case indigo.Timestamp:
		return decls.Timestamp, nil
	case indigo.Map:
		key, err := convertIndigoToExprType(v.KeyType)
		if err != nil {
			return nil, fmt.Errorf("setting key of %v map: %w", v.KeyType, err)
		}
		val, err := convertIndigoToExprType(v.ValueType)
		if err != nil {
			return nil, fmt.Errorf("setting value of %v map: %w", v.ValueType, err)
		}
		return decls.NewMapType(key, val), nil
	case indigo.List:
		val, err := convertIndigoToExprType(v.ValueType)
		if err != nil {
			return nil, fmt.Errorf("setting value of %v list: %w", v.ValueType, err)
		}
		return decls.NewListType(val), nil
	case indigo.Proto:
		return decls.NewObjectType(v.Protoname), nil
	default:
		return nil, fmt.Errorf("unknown indigo type %s", t)
	}

}

// convertRefValToIndigo converts the result of CEL evaluation (a ref.Val) to an indigo.Value.
// It checks that the result type (not value) is the type that the rule wanted.
// If it's not possible to convert the CEL value to the wanted indigo.Type, the result is an
// empty value and an error
func convertRefValToIndigo(r ref.Val, want indigo.Type) (retval indigo.Value, reterr error) {
	//	fmt.Printf("Val: %v, r.type = %T, r.value() = %T, r.value() = %v, type = %v\n", r, r,
	// r.Value(), r.Value(), r.Type())
	// A panic handler here because we're using reflection
	defer func() {
		if r := recover(); r != nil {
			retval = indigo.Value{}
			reterr = fmt.Errorf("recovered from panic: %v", r)
		}
	}()

	// If the rule didn't specify a desired output type, Indigo will default it to
	// a boolean in the Eval step. This condition should never occur.
	if want == nil {
		return indigo.Value{}, fmt.Errorf("the rule's result type is nil")
	}

	// Get the 'protoype' zero value wanted by the rule. This is an interface{} containing
	// an actual value (int(0), map[string]int{}, etc.)
	t := want.Zero()
	if t == nil {
		return indigo.Value{}, fmt.Errorf("expected a zero indigo.Value, got nil")
	}

	// Attempt to convert the CEL reesult value to Indigo's wanted Go type
	// The Go type may be a protocol buffer.
	s, err := r.ConvertToNative(reflect.TypeOf(t))

	if err != nil {
		return indigo.Value{}, fmt.Errorf("converting CEL type %T (%v)  to indigo type %T: %w", r, r, want, err)
	}

	return indigo.Value{
		Type: want,
		Val:  s,
	}, nil

}

// convertIndigoSchemaToDeclarations converts an Indigo Schema to a list of CEL "EnvOption".
// Entries in this list are types that CEL know about (i.e., the schema).
func convertIndigoSchemaToDeclarations(s indigo.Schema) ([]celgo.EnvOption, error) {
	declarations := []*gexpr.Decl{}
	types := []interface{}{}

	for _, d := range s.Elements {
		typ, err := convertIndigoToExprType(d.Type)
		if err != nil {
			return nil, err
		}
		declarations = append(declarations, decls.NewVar(d.Name, typ))

		if v, ok := d.Type.(indigo.Proto); ok {
			types = append(types, v.Message)
		}
	}
	opts := []celgo.EnvOption{}
	opts = append(opts, celgo.Declarations(declarations...))
	opts = append(opts, celgo.Types(types...))
	if len(opts) == 0 {
		return nil, fmt.Errorf("no valid schema")
	}

	return opts, nil
}

func convertRefValToIndigo2(r ref.Val) (indigo.Value, error) {
	//	fmt.Printf("Val: %v, r.type = %T, r.value() = %T, r.value() = %v, type = %v\n", r, r, r.Value(), r.Value(), r.Type())

	if r.Value() == nil {
		return indigo.Value{}, fmt.Errorf("CEL value is nil")
	}

	// All types, except for protocol buffers
	switch r.Type() {
	case types.ListType:
		return list(r)
	case types.MapType:
		return mapType(r)
	case types.BoolType, types.IntType, types.DoubleType, types.StringType:
		return primitive(r)
	case types.TimestampType:
		return timestamp(r)
	case types.DurationType:
		return duration(r)
	}

	// Protocol buffers
	if p, ok := r.Value().(*dynamicpb.Message); ok {
		return proto(r, p)
	}

	return indigo.Value{}, fmt.Errorf("unexpected type %T, %v, (%v)", r, r.Type(), r)
}

func primitive(r ref.Val) (indigo.Value, error) {

	switch v := r.Value().(type) {
	case bool:
		return indigo.Value{v, indigo.Bool{}}, nil
	case int, int32, int64:
		return indigo.Value{v, indigo.Int{}}, nil
	case float32, float64:
		return indigo.Value{v, indigo.Float{}}, nil
	case string:
		return indigo.Value{v, indigo.String{}}, nil
	default:
		return indigo.Value{}, fmt.Errorf("unexected type")
	}

}

func list(r ref.Val) (indigo.Value, error) {
	// _, ok := want.(indigo.List)
	// if !ok {
	return indigo.Value{}, fmt.Errorf("type assertion to indigo.List failed")
	// }

	// t := want.Zero()
	// if t == nil {
	// 	return indigo.Value{}, fmt.Errorf("expected a zero list, got nil")
	// }

	// t2, ok := want.Zero().(reflect.Value)
	// if !ok {
	// 	return indigo.Value{}, fmt.Errorf("wanted reflect.Value, got %T", t)
	// }

	// s, err := r.ConvertToNative(reflect.TypeOf(t2.Interface()))
	// if err != nil {
	// 	return indigo.Value{}, fmt.Errorf("converting CEL list to indigo.List: %w", err)
	// }
	// return indigo.Value{
	// 	Type: want,
	// 	Val:  s,
	// }, nil
}

func mapType(r ref.Val) (indigo.Value, error) {
	// _, ok := want.(indigo.Map)
	// if !ok {
	return indigo.Value{}, fmt.Errorf("type assertion to indigo.Map failed")
	//	}

	// t := want.Zero()
	// if t == nil {
	// 	return indigo.Value{}, fmt.Errorf("expected a zero map, got nil")
	// }

	// t2, ok := want.Zero().(reflect.Value)
	// if !ok {
	// 	return indigo.Value{}, fmt.Errorf("wanted reflect.Value, got %T", t)
	// }

	// s, err := r.ConvertToNative(reflect.TypeOf(t2.Interface()))
	// if err != nil {
	// 	return indigo.Value{}, fmt.Errorf("converting CEL map to indigo.Map: %w", err)
	// }
	// return indigo.Value{
	// 	Type: want,
	// 	Val:  s,
	// }
	///	, nil
}

func timestamp(r ref.Val) (indigo.Value, error) {
	v := r.Value()
	if v == nil {
		return indigo.Value{}, fmt.Errorf("expected value from CEL, got nil")
	}

	t, ok := v.(time.Time)
	if !ok {
		return indigo.Value{}, fmt.Errorf("expected timestamp value from CEL, got %T", v)
	}

	return indigo.Value{
		Type: indigo.Timestamp{},
		Val:  t,
	}, nil
}

func duration(r ref.Val) (indigo.Value, error) {
	v := r.Value()
	if v == nil {
		return indigo.Value{}, fmt.Errorf("expected value from CEL, got nil")
	}

	d, ok := v.(time.Duration)
	if !ok {
		return indigo.Value{}, fmt.Errorf("expected duration value from CEL, got %T", v)
	}

	return indigo.Value{
		Type: indigo.Duration{},
		Val:  d,
	}, nil
}

func proto(r ref.Val, p *dynamicpb.Message) (indigo.Value, error) {
	// wantType, ok := want.(indigo.Proto)
	// if !ok {
	return indigo.Value{}, fmt.Errorf("proto")
	//	}

	// if wantType.Message == nil {
	// 	return indigo.Value{}, fmt.Errorf("missing message in indigo.Proto type")
	// }

	// pb, err := r.ConvertToNative(reflect.TypeOf(wantType.Message))
	// if err != nil {
	// 	return indigo.Value{}, fmt.Errorf("conversion to %T failed: %w", wantType.Message, err)
	// }

	// return indigo.Value{
	// 	Type: want,
	// 	Val:  pb,
	// }, nil
}
