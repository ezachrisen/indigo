package cel

// This file contains functions that convert
//   FROM an indigo.Schema
//   TO a CEL schema
//
// The resulting CEL schema is passed to the CEL compiler to validate the rule expression
// and perform type checking on it.

import (
	"fmt"
	"reflect"

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

func binaryWrapper(name string, f indigo.BinaryFunction) func(lhs, rhs ref.Val) ref.Val {

	return func(lhs, rhs ref.Val) ref.Val {

		// reg, err := types.NewRegistry(&gexpr.ParsedExpr{})
		// if err != nil {
		// 	return types.NewErr("cannot initialize type registry")
		// }

		// wantType, err := celType(f.Return)
		// if err != nil {
		// 	return types.NewErr("the expected return value from %s is not supported by CEL", name)
		// }

		x, err := f.Func(lhs, rhs)
		if err != nil {
			return types.NewErr(err.Error())
		}

		if reflect.TypeOf(f.Return) != reflect.TypeOf(x) {
			return types.NewErr("expeted %s to return a %q, got a %q", name, f.Return, x)
		}

		ref, err := refVal(x)

		return nil
		// val, err := celType(x)
		// if err != nil {
		// 	return types.NewErr("return value (%#v) from %s could not be converted to CEL value: %w", x, name, err)
		// }

		// return
		// return val
	}

}

func refVal(t indigo.Type) (ref.Val, error) {

	switch v := t.(type) {
	case indigo.String:
		return ref.
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

func binaryFunction(name string, v indigo.BinaryFunction) (celgo.EnvOption, error) {
	if v.Func == nil {
		return nil, fmt.Errorf("%q missing function", name)
	}

	lhs, err := celType(v.LHS)
	if err != nil {
		return nil, err
	}

	rhs, err := celType(v.RHS)
	if err != nil {
		return nil, err
	}

	ret, err := celType(v.Return)
	if err != nil {
		return nil, err
	}

	return celgo.Function(name,
		celgo.Overload(fmt.Sprintf("%s_%s_%s", name, v.LHS, v.RHS),
			[]*celgo.Type{lhs, rhs},
			ret,
			celgo.BinaryBinding(binaryWrapper(name, v)))), nil

}
