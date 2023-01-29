package cel

// This file contains functions that convert
//   FROM an indigo.Schema
//   TO a CEL schema
//
// The resulting CEL schema is passed to the CEL compiler to validate the rule expression
// and perform type checking on it.

import (
	"fmt"

	"github.com/ezachrisen/indigo"
	celgo "github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	gexpr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// convertIndigoSchemaToDeclarations converts an Indigo Schema to a list of CEL "EnvOption".
// Entries in this list are types that CEL know about (i.e., the schema).
func convertIndigoSchemaToDeclarations(s indigo.Schema) ([]celgo.EnvOption, error) {

	// declarations are what CEL calls types in their schema
	declarations := []*gexpr.Decl{}

	// for protocol buffer types we also have to register the type separately
	// we'll collect them in types
	types := []interface{}{}

	// custom options will be converted directly to EnvOptions
	opts := []celgo.EnvOption{}

	for _, d := range s.Elements {
		typ, err := exprType(d.Type)
		if err != nil {
			return nil, fmt.Errorf("converting element %q in schema %q to CEL type: %v", d.Name, s.Name, err)
		}

		if typ == nil {
			continue
		}
		declarations = append(declarations, decls.NewVar(d.Name, typ))

		if v, ok := d.Type.(indigo.Proto); ok {
			types = append(types, v.Message)
		}
	}

	for _, d := range s.Elements {
		opt, err := convertIndigoToOpt(d)
		if err != nil {
			return nil, fmt.Errorf("convertin element %q in schema %q to CEL option: %v", d.Name, s.Name, err)
		}
		if opt == nil {
			continue
		}
		opts = append(opts, opt)
	}

	opts = append(opts, celgo.Declarations(declarations...))
	opts = append(opts, celgo.Types(types...))
	if len(opts) == 0 {
		return nil, fmt.Errorf("no valid schema")
	}

	return opts, nil
}

// exprType converts from an indigo type to a expr.Type,
// which is used by CEL to represent types in its schema.
func exprType(t indigo.Type) (*gexpr.Type, error) {

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
		key, err := exprType(v.KeyType)
		if err != nil {
			return nil, fmt.Errorf("setting key of %v map: %w", v.KeyType, err)
		}
		val, err := exprType(v.ValueType)
		if err != nil {
			return nil, fmt.Errorf("setting value of %v map: %w", v.ValueType, err)
		}
		return decls.NewMapType(key, val), nil
	case indigo.List:
		val, err := exprType(v.ValueType)
		if err != nil {
			return nil, fmt.Errorf("setting value of %v list: %w", v.ValueType, err)
		}
		return decls.NewListType(val), nil
	case indigo.Proto:
		n, err := v.ProtoFullName()
		if err != nil {
			return nil, err
		}
		return decls.NewObjectType(n), nil
	default:
		return nil, nil
	}
}

func refVal(t indigo.Value) (ref.Val, error) {

	switch v := t.(type) {
	case indigo.String:
		return types.String(v.Val), nil
	case indigo.Int:
		return types.Int(v.Val), nil
	case indigo.Float:
		return types.Double(v.Val), nil
	case indigo.Bool:
		return types.Bool(v.Val), nil
	case indigo.Duration:
		return types.Duration{v.Val}, nil
	case indigo.Timestamp:
		return types.Timestamp{v.Val}, nil
	// case indigo.Map:
	// 	key, err := exprType(v.KeyType)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("setting key of %v map: %w", v.KeyType, err)
	// 	}
	// 	val, err := exprType(v.ValueType)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("setting value of %v map: %w", v.ValueType, err)
	// 	}
	// 	return decls.NewMapType(key, val), nil
	// case indigo.List:
	// 	val, err := exprType(v.ValueType)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("setting value of %v list: %w", v.ValueType, err)
	// 	}
	// 	return decls.NewListType(val), nil
	case indigo.Proto:
		m := v.Val
		tr, err := types.NewRegistry(m)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialize type registry to convert proto to ref.Val: %w", err)
		}
		nv := tr.NativeToValue(m)
		return nv, nil
	default:
		return nil, fmt.Errorf("unsupported Indigo type: %T", t)
	}
}

func isError(val ref.Val) bool {
	switch val.(type) {
	case *types.Err:
		return true
	default:
		return false
	}
}
func convertIndigoToOpt(e indigo.DataElement) (celgo.EnvOption, error) {

	switch v := e.Type.(type) {
	case indigo.BinaryFunction:
		return binaryFunction(e.Name, v)
	default:
		return nil, nil

	}
}

func celType(t indigo.Type) (*celgo.Type, error) {
	x, err := exprType(t)
	if err != nil {
		return nil, err
	}

	y, err := celgo.ExprTypeToType(x)
	if err != nil {
		return nil, err
	}
	return y, nil

}
