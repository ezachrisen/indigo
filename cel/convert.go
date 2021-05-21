package cel

import (
	"fmt"
	"reflect"

	"github.com/ezachrisen/indigo"
	"github.com/google/cel-go/common/types/ref"
)

// convertRefValToIndigo converts the result of CEL evaluation (a ref.Val) to an indigo.Value
func convertRefValToIndigo(r ref.Val, want indigo.Type) (retval indigo.Value, reterr error) {
	//	fmt.Printf("Val: %v, r.type = %T, r.value() = %T, r.value() = %v, type = %v\n", r, r, r.Value(), r.Value(), r.Type())
	// A panic handler here because we're using reflection
	defer func() {
		if r := recover(); r != nil {
			retval = indigo.Value{}
			reterr = fmt.Errorf("recovered from panic: %v", r)
		}
	}()

	if want == nil {
		return indigo.Value{}, fmt.Errorf("the rule's result type is nil")
	}

	t := want.Zero()
	if t == nil {
		return indigo.Value{}, fmt.Errorf("expected a zero indigo.Value, got nil")
	}

	s, err := r.ConvertToNative(reflect.TypeOf(t))

	if err != nil {
		return indigo.Value{}, fmt.Errorf("converting CEL type %T (%v)  to indigo type %T: %w", r, r, want, err)
	}

	return indigo.Value{
		Type: want,
		Val:  s,
	}, nil

}

// // convertRefValToIndigo converts the result of CEL evaluation to an indigo.Value
// // CEL returns a ref.Val
// func convertRefValToIndigo2(r ref.Val, want indigo.Type) (indigo.Value, error) {
// 	//	fmt.Printf("Val: %v, r.type = %T, r.value() = %T, r.value() = %v, type = %v\n", r, r, r.Value(), r.Value(), r.Type())

// 	if r.Value() == nil {
// 		return indigo.Value{}, fmt.Errorf("CEL value is nil")
// 	}

// 	// All types, except for protocol buffers
// 	switch r.Type() {
// 	case types.ListType:
// 		return list(r, want)
// 	case types.MapType:
// 		return mapType(r, want)
// 	case types.BoolType, types.IntType, types.DoubleType:
// 		return primitive(r)
// 	case types.TimestampType:
// 		return timestamp(r)
// 	case types.DurationType:
// 		return duration(r)
// 	}

// 	// Protocol buffers
// 	if p, ok := r.Value().(*dynamicpb.Message); ok {
// 		return proto(r, p, want)
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

// func list(r ref.Val, want indigo.Type) (indigo.Value, error) {
// 	_, ok := want.(indigo.List)
// 	if !ok {
// 		return indigo.Value{}, fmt.Errorf("type assertion to indigo.List failed, got type %T", want)
// 	}

// 	t := want.Zero()
// 	if t == nil {
// 		return indigo.Value{}, fmt.Errorf("expected a zero list, got nil")
// 	}

// 	t2, ok := want.Zero().(reflect.Value)
// 	if !ok {
// 		return indigo.Value{}, fmt.Errorf("wanted reflect.Value, got %T", t)
// 	}

// 	s, err := r.ConvertToNative(reflect.TypeOf(t2.Interface()))
// 	if err != nil {
// 		return indigo.Value{}, fmt.Errorf("converting CEL list to indigo.List: %w", err)
// 	}
// 	return indigo.Value{
// 		Type: want,
// 		Val:  s,
// 	}, nil
// }

// func mapType(r ref.Val, want indigo.Type) (indigo.Value, error) {
// 	_, ok := want.(indigo.Map)
// 	if !ok {
// 		return indigo.Value{}, fmt.Errorf("type assertion to indigo.Map failed, got type %T", want)
// 	}

// 	t := want.Zero()
// 	if t == nil {
// 		return indigo.Value{}, fmt.Errorf("expected a zero map, got nil")
// 	}

// 	t2, ok := want.Zero().(reflect.Value)
// 	if !ok {
// 		return indigo.Value{}, fmt.Errorf("wanted reflect.Value, got %T", t)
// 	}

// 	s, err := r.ConvertToNative(reflect.TypeOf(t2.Interface()))
// 	if err != nil {
// 		return indigo.Value{}, fmt.Errorf("converting CEL map to indigo.Map: %w", err)
// 	}
// 	return indigo.Value{
// 		Type: want,
// 		Val:  s,
// 	}, nil
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

// func proto(r ref.Val, p *dynamicpb.Message, want indigo.Type) (indigo.Value, error) {
// 	wantType, ok := want.(indigo.Proto)
// 	if !ok {
// 		return indigo.Value{}, fmt.Errorf("wanted  %T, got %v", want, p.ProtoReflect().Descriptor().FullName())
// 	}

// 	if wantType.Message == nil {
// 		return indigo.Value{}, fmt.Errorf("missing message in indigo.Proto type")
// 	}

// 	pb, err := r.ConvertToNative(reflect.TypeOf(wantType.Message))
// 	if err != nil {
// 		return indigo.Value{}, fmt.Errorf("conversion to %T failed: %w", wantType.Message, err)
// 	}

// 	return indigo.Value{
// 		Type: want,
// 		Val:  pb,
// 	}, nil
// }
