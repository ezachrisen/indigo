package cel

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

import (
	"fmt"
	"reflect"

}	"github.com/ezachrisen/indigo"
	celgo "github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types/ref"
	gexpr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

func convertDynamicMessageToProto(r ref.Val, want indigo.Type) (interface{}, error) {

	msg, ok := want.(indigo.Proto)

	if !ok {
		return nil, fmt.Errorf("CEL returned a protobuf, expected %T", want)
	}

	if msg.Message == nil {
		return nil, fmt.Errorf("expected result protocol buffer is nil")
	}

	pb, err := r.ConvertToNative(reflect.TypeOf(msg.Message))
	if err != nil {
		return nil, err
	}

	return pb, nil
}

// convertIndigoSchemaToDeclarations converts an Indigo Schema to a list of CEL "EnvOption".
// Entries in this list are types that CEL know about (i.e., the schema).
func convertIndigoSchemaToDeclarations(s indigo.Schema) ([]celgo.EnvOption, error) {
	declarations := []*gexpr.Decl{}
	types := []interface{}{}

	for _, d := range s.Elements {
		typ, err := convertIndigoToExprType(d.Type)
		//typ, err := convertIndigoToExprType2(d.T)
		if err != nil {
			return nil, err
		}
		declarations = append(declarations, decls.NewVar(d.Name, typ))

		//		types = append(types, d.T)

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

// func kindToExprType(k reflect.Kind, t reflect.Type) (*gexpr.Type, error) {

// 	switch k {
// 	case reflect.Bool:
// 		return decls.Bool, nil
// 	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
// 		return decls.Int, nil
// 	case reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
// 		return decls.Int, nil
// 	case reflect.Float32, reflect.Float64:
// 		return decls.Double, nil
// 	case reflect.String:
// 		return decls.String, nil
// 	// case reflect.TypeOf(time.Time{}).Kind():
// 	// 	fmt.Println("TIme is ", reflect.TypeOf(time.Time{}).Kind())
// 	// 	return decls.Timestamp, nil
// 	// case reflect.TypeOf(time.Duration(0)).Kind():
// 	// 	return decls.Duration, nil
// 	case reflect.Slice:
// 		elem, err := kindToExprType(t.Elem().Kind(), t.Elem())
// 		if err != nil {
// 			return nil, fmt.Errorf("error converting type %v to CEL type", t)
// 		}
// 		return decls.NewListType(elem), nil
// 	case reflect.Map:
// 		elem, err := kindToExprType(t.Elem().Kind(), t.Elem())
// 		if err != nil {
// 			return nil, fmt.Errorf("error converting value in map type %v to CEL type", t)
// 		}

// 		key, err := kindToExprType(t.Key().Kind(), t.Elem())
// 		if err != nil {
// 			return nil, fmt.Errorf("error converting key in map type %v to CEL type", t)
// 		}
// 		return decls.NewMapType(key, elem), nil
// 	case reflect.Ptr:
// 		fmt.Println("POINTER: ", t)
// 		fmt.Println("--ELEM: ", t.Elem())
// 		fmt.Println("--KIND: ", t.Elem().Kind())
// 		k, err := kindToExprType(t.Elem().Kind(), t.Elem())
// 		if err != nil {
// 			return nil, err
// 		}
// 		fmt.Println("-- k:", k)
// 		return decls.NewTypeType(k), nil
// 		//		return k, err
// 		//		return nil, nil
// 	case reflect.Struct:
// 		fmt.Println("Struct: ", t.String())
// 		return decls.NewObjectType(t.String()), nil
// 	default:
// 		return nil, fmt.Errorf("Unknown type: %v, %T (%v)", k, t, t)
// 	}

// }

// // convertIndigoToExprType converts from an indigo type to a expr.Type,
// // which is used by CEL to represent types in its schema.
// func convertIndigoToExprType2(x interface{}) (*gexpr.Type, error) {

// 	t := reflect.TypeOf(x)
// 	return kindToExprType(t.Kind(), t)

// 	// switch t := reflect.TypeOf(x); t.Kind() {
// 	// case reflect.Bool:
// 	// 	return decls.Bool, nil
// 	// case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
// 	// 	return decls.Int, nil
// 	// case reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
// 	// 	return decls.Int, nil
// 	// case reflect.Float32, reflect.Float64:
// 	// 	return decls.Double, nil
// 	// case reflect.String:
// 	// 	return decls.String, nil
// 	// case reflect.Slice:
// 	// 	fmt.Println("Elem: ", t.Elem().Kind())
// 	// 	v := reflect.ValueOf(x)
// 	// 	fmt.Printf("%v, %T\n", v, v)
// 	// 	elem, err := convertIndigoToExprType2(v.Elem().Interface())
// 	// 	if err != nil {
// 	// 		return nil, fmt.Errorf("error converting type %v to CEL type", t)
// 	// 	}
// 	// 	return decls.NewListType(elem), nil
// 	// 	//		return nil, nil
// 	// // 	fmt.Printf("slice: len=%d, %v\n", v.Len(), v.Interface())
// 	// // case reflect.Map:
// 	// // 	fmt.Printf("map: %v\n", v.Interface())
// 	// default:
// 	// 	return nil, fmt.Errorf("Unknown type: %T (%v)", x, x)
// 	// }

// 	// switch v := .(type) {
// 	// case string:
// 	// 	return decls.String, nil
// 	// case int, int32, int64:
// 	// 	return decls.Int, nil
// 	// case float64:
// 	// 	return decls.Double, nil
// 	// case bool:
// 	// 	return decls.Bool, nil
// 	// case time.Duration:
// 	// 	return decls.Duration, nil
// 	// case time.Time:
// 	// 	return decls.Timestamp, nil
// 	// case indigo.Map:
// 	// 	key, err := convertIndigoToExprType(v.KeyType)
// 	// 	if err != nil {
// 	// 		return nil, fmt.Errorf("setting key of %v map: %w", v.KeyType, err)
// 	// 	}
// 	// 	val, err := convertIndigoToExprType(v.ValueType)
// 	// 	if err != nil {
// 	// 		return nil, fmt.Errorf("setting value of %v map: %w", v.ValueType, err)
// 	// 	}
// 	// 	return decls.NewMapType(key, val), nil
// 	// case indigo.List:
// 	// 	val, err := convertIndigoToExprType(v.ValueType)
// 	// 	if err != nil {
// 	// 		return nil, fmt.Errorf("setting value of %v list: %w", v.ValueType, err)
// 	// 	}
// 	// 	return decls.NewListType(val), nil
// 	// case indigo.Proto:
// 	// 	return decls.NewObjectType(v.Protoname), nil
// 	// default:
// 	// 	return nil, fmt.Errorf("unknown indigo type %s", t)
// 	// }

// }

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

// // convertIndigoToExprType converts from an indigo type to a expr.Type,
// // which is used by CEL to represent types in its schema.
// func convertIndigoToExprType2(t interface{}) (*gexpr.Type, error) {

// 	switch v := t.(type) {
// 	case string:
// 		return decls.String, nil
// 	case indigo.Int:
// 		return decls.Int, nil
// 	case indigo.Float:
// 		return decls.Double, nil
// 	case indigo.Bool:
// 		return decls.Bool, nil
// 	case indigo.Duration:
// 		return decls.Duration, nil
// 	case indigo.Timestamp:
// 		return decls.Timestamp, nil
// 	case indigo.Map:
// 		key, err := convertIndigoToExprType(v.KeyType)
// 		if err != nil {
// 			return nil, fmt.Errorf("setting key of %v map: %w", v.KeyType, err)
// 		}
// 		val, err := convertIndigoToExprType(v.ValueType)
// 		if err != nil {
// 			return nil, fmt.Errorf("setting value of %v map: %w", v.ValueType, err)
// 		}
// 		return decls.NewMapType(key, val), nil
// 	case indigo.List:
// 		val, err := convertIndigoToExprType(v.ValueType)
// 		if err != nil {
// 			return nil, fmt.Errorf("setting value of %v list: %w", v.ValueType, err)
// 		}
// 		return decls.NewListType(val), nil
// 	case indigo.Proto:
// 		return decls.NewObjectType(v.Protoname), nil
// 	default:
// 		return nil, fmt.Errorf("unknown indigo type %s", t)
// 	}

// }

// // convertRefValToIndigo converts the result of CEL evaluation (a ref.Val) to an indigo.Value.
// // It checks that the result type (not value) is the type that the rule wanted.
// // If it's not possible to convert the CEL value to the wanted indigo.Type, the result is an
// // empty value and an error
// func convertRefValToIndigo(r ref.Val, want indigo.Type) (retval indigo.Value, reterr error) {
// 	//	fmt.Printf("Val: %v, r.type = %T, r.value() = %T, r.value() = %v, type = %v\n", r, r,
// 	// r.Value(), r.Value(), r.Type())
// 	// A panic handler here because we're using reflection
// 	defer func() {
// 		if r := recover(); r != nil {
// 			retval = indigo.Value{}
// 			reterr = fmt.Errorf("recovered from panic: %v", r)
// 		}
// 	}()

// 	// If the rule didn't specify a desired output type, Indigo will default it to
// 	// a boolean in the Eval step. This condition should never occur.
// 	if want == nil {
// 		return indigo.Value{}, fmt.Errorf("the rule's result type is nil")
// 	}

// 	// Get the 'protoype' zero value wanted by the rule. This is an interface{} containing
// 	// an actual value (int(0), map[string]int{}, etc.)
// 	t := want.Zero()
// 	if t == nil {
// 		return indigo.Value{}, fmt.Errorf("expected a zero indigo.Value, got nil")
// 	}

// 	// Attempt to convert the CEL result value to Indigo's wanted Go type
// 	// The Go type may be a protocol buffer.
// 	s, err := r.ConvertToNative(reflect.TypeOf(t))

// 	if err != nil {
// 		return indigo.Value{}, fmt.Errorf("converting CEL type %T (%v)  to indigo type %T: %w", r, r, want, err)
// 	}

// 	return indigo.Value{
// 		Type: want,
// 		Val:  s,
// 	}, nil

// }

// func convertRefValToIndigo2(r ref.Val) (indigo.Value, error) {
// 	//	fmt.Printf("Val: %v, r.type = %T, r.value() = %T, r.value() = %v, type = %v\n",
// 	// r, r, r.Value(), r.Value(), r.Type())

// 	if r.Value() == nil {
// 		return indigo.Value{}, fmt.Errorf("CEL value is nil")
// 	}

// 	// All types, except for protocol buffers
// 	switch r.Type() {
// 	case types.ListType:
// 		return list(r)
// 	case types.MapType:
// 		return mapType(r)
// 	case types.BoolType, types.IntType, types.DoubleType, types.StringType:
// 		return primitive(r)
// 	case types.TimestampType:
// 		return timestamp(r)
// 	case types.DurationType:
// 		return duration(r)
// 	}

// 	// Protocol buffers
// 	if p, ok := r.Value().(*dynamicpb.Message); ok {
// 		return proto(r, p)
// 	}

// 	return indigo.Value{}, fmt.Errorf("unexpected type %T, %v, (%v)", r, r.Type(), r)
// }

// func primitive(r ref.Val) (indigo.Value, error) {

// 	switch v := r.Value().(type) {
// 	case bool:
// 		return indigo.Value{v, indigo.Bool{}}, nil
// 	case int, int32, int64:
// 		return indigo.Value{v, indigo.Int{}}, nil
// 	case float32, float64:
// 		return indigo.Value{v, indigo.Float{}}, nil
// 	case string:
// 		return indigo.Value{v, indigo.String{}}, nil
// 	default:
// 		return indigo.Value{}, fmt.Errorf("unexected type")
// 	}

// }

// func list(r ref.Val) (indigo.Value, error) {

// 	fmt.Println("Typename: ", r.Type().TypeName())
// 	fmt.Printf("Type of r.Value(): %T\n", r.Value())

// 	switch lv := r.Value().(type) {
// 	case []ref.Val:
// 		var elemTyp ref.Type
// 		var multipleTypes bool
// 		for _, v := range lv {
// 			tmp := v.Type()
// 			if elemTyp == nil {
// 				elemTyp = tmp
// 			} else {
// 				if tmp != elemTyp {
// 					multipleTypes = true
// 				}
// 			}
// 		}
// 		if !multipleTypes {
// 			fmt.Println("Not multiple types")
// 			ind, err := convertRefValToIndigo2(lv[0])
// 			fmt.Println("Element type is ", ind)
// 			if err != nil {
// 				return indigo.Value{}, fmt.Errorf("converting list element type %T to indigo type: %v", lv[0], err)
// 			}
// 			return indigo.Value{lv, ind.Type}, nil
// 		} else {
// 			fmt.Println("Muliple types ")
// 			return indigo.Value{lv, indigo.Any{}}, nil
// 		}
// 	// case
// 	// 	fmt.Printf("Plain list: %v, %T, %v\n", lv, lv, reflect.TypeOf(lv).Elem())
// 	// x := lv[0]
// 	// //		fmt.Printf("x = %T, %v\n", x, x)
// 	// //		fmt.Printf("x conv: %T, %v\n", x.ConvertToType(x.Type()), x.ConvertToType(x.Type()))
// 	// //elementType, err := primitive(x)
// 	// if err != nil {
// 	// 	return indigo.Value{}, fmt.Errorf("unknown cel type: %T", x)
// 	// }
// 	// //
// 	//fmt.Printf("Element type: %v\n", elementType)

// 	case pr.List:
// 		// 	// it's a proto list
// 		it := indigo.Any{}
// 		var multipleTypes bool
// 		fmt.Printf("Proto List: %v, %T\n", lv, lv)
// 		for i := 0; i < lv.Len(); i++ {
// 			v := lv.Get(i)
// 			tmp, err := indigo.PrimitiveGoToIndigo(v.Interface())
// 			fmt.Printf("tmp = %v, %T, it= %v, %T\n", tmp, tmp, it, it)
// 			if err != nil {
// 				return indigo.Value{}, fmt.Errorf("converting list element type %T to indigo primitve type: %v", v, err)
// 			}
// 			if  == nil {

// 			if tmp != it {
// 				multipleTypes = true
// 			}
// 		}

// 		switch multipleTypes {
// 		case false:
// 			fmt.Println("Returning list of ", it)
// 			return indigo.Value{lv, indigo.List{ValueType: it}}, nil
// 		case true:
// 			fmt.Println("Returing multi list")
// 			return indigo.Value{lv, indigo.List{ValueType: indigo.Any{}}}, nil
// 		}

// 	default:
// 		fmt.Printf("unknown type: %T\n", lv)
// 	}

// 	// lv := r.Value().(pr.List)
// 	// fmt.Printf("r.Value() = %T\nr.Value()=%+v\n", r.Value(), r.Value())
// 	// fmt.Printf("lv = %T\n", lv)
// 	// fmt.Printf("newelement type: %T\n", lv.NewElement().Interface())
// 	// fmt.Printf("Value of list = %T\n ", pr.ValueOfList(lv).Interface())

// 	// for i := 0; i < lv.Len(); i++ {
// 	// 	fmt.Printf("i = %d, val = %v, type = %T %s\n", i, lv.Get(i), lv.Get(i), lv.Get(i).Interface())
// 	// }

// 	// m := r.Value().(dynamicpb.Message)
// 	// fmt.Printf("m=%T\nm=%+v", m, m)

// 	// _, ok := want.(indigo.List)
// 	// if !ok {
// 	return indigo.Value{}, fmt.Errorf("type assertion to indigo.List failed")
// 	// }

// 	// t := want.Zero()
// 	// if t == nil {
// 	// 	return indigo.Value{}, fmt.Errorf("expected a zero list, got nil")
// 	// }

// 	// t2, ok := want.Zero().(reflect.Value)
// 	// if !ok {
// 	// 	return indigo.Value{}, fmt.Errorf("wanted reflect.Value, got %T", t)
// 	// }

// 	// s, err := r.ConvertToNative(reflect.TypeOf(t2.Interface()))
// 	// if err != nil {
// 	// 	return indigo.Value{}, fmt.Errorf("converting CEL list to indigo.List: %w", err)
// 	// }
// 	// return indigo.Value{
// 	// 	Type: want,
// 	// 	Val:  s,
// 	// }, nil
// }

// func mapType(r ref.Val) (indigo.Value, error) {
// 	// _, ok := want.(indigo.Map)
// 	// if !ok {
// 	return indigo.Value{}, fmt.Errorf("type assertion to indigo.Map failed")
// 	//	}

// 	// t := want.Zero()
// 	// if t == nil {
// 	// 	return indigo.Value{}, fmt.Errorf("expected a zero map, got nil")
// 	// }

// 	// t2, ok := want.Zero().(reflect.Value)
// 	// if !ok {
// 	// 	return indigo.Value{}, fmt.Errorf("wanted reflect.Value, got %T", t)
// 	// }

// 	// s, err := r.ConvertToNative(reflect.TypeOf(t2.Interface()))
// 	// if err != nil {
// 	// 	return indigo.Value{}, fmt.Errorf("converting CEL map to indigo.Map: %w", err)
// 	// }
// 	// return indigo.Value{
// 	// 	Type: want,
// 	// 	Val:  s,
// 	// }
// 	///	, nil
// }

// func timestamp(r ref.Val) (indigo.Value, error) {
// 	v := r.Value()
// 	if v == nil {
// 		return indigo.Value{}, fmt.Errorf("expected value from CEL, got nil")
// 	}

// 	t, ok := v.(time.Time)
// 	if !ok {
// 		return indigo.Value{}, fmt.Errorf("expected timestamp value from CEL, got %T", v)
// 	}

// 	return indigo.Value{
// 		Type: indigo.Timestamp{},
// 		Val:  t,
// 	}, nil
// }

// func duration(r ref.Val) (indigo.Value, error) {
// 	v := r.Value()
// 	if v == nil {
// 		return indigo.Value{}, fmt.Errorf("expected value from CEL, got nil")
// 	}

// 	d, ok := v.(time.Duration)
// 	if !ok {
// 		return indigo.Value{}, fmt.Errorf("expected duration value from CEL, got %T", v)
// 	}

// 	return indigo.Value{
// 		Type: indigo.Duration{},
// 		Val:  d,
// 	}, nil
// }

// func proto(r ref.Val, p *dynamicpb.Message) (indigo.Value, error) {
// 	// wantType, ok := want.(indigo.Proto)
// 	// if !ok {
// 	return indigo.Value{}, fmt.Errorf("proto")
// 	//	}

// 	// if wantType.Message == nil {
// 	// 	return indigo.Value{}, fmt.Errorf("missing message in indigo.Proto type")
// 	// }

// 	// pb, err := r.ConvertToNative(reflect.TypeOf(wantType.Message))
// 	// if err != nil {
// 	// 	return indigo.Value{}, fmt.Errorf("conversion to %T failed: %w", wantType.Message, err)
// 	// }

// 	// return indigo.Value{
// 	// 	Type: want,
// 	// 	Val:  pb,
// 	// }, nil
// }
