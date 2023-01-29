package cel

import (
	"fmt"
	"reflect"

	"github.com/ezachrisen/indigo"
	celgo "github.com/google/cel-go/cel"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// binaryFunction creates a CEL declaration for a binary function (a function that takes
// two parameters and returns a value).
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

// binaryWrapper wraps an Indigo binary function in a closure that handles converting
// between Indigo's types and CEL's types, taking care to ensure that the values a
// passed and returned are of the types declared in the Indigo scheema for the function.
func binaryWrapper(name string, f indigo.BinaryFunction) func(lhs, rhs ref.Val) ref.Val {

	return func(lhs, rhs ref.Val) ref.Val {
		// TODO: check input types against schema

		x, err := f.Func(lhs, rhs)
		if err != nil {
			return types.NewErr(err.Error())
		}

		if reflect.TypeOf(f.Return) != reflect.TypeOf(x) {
			return types.NewErr("%w: function %s, wanted %q, got a %q", indigo.ErrUnexpectedReturnType, name, f.Return, x)
		}

		ref, err := refVal(x)
		if err != nil {
			return types.NewErr("function %q return value: %w", name, err)
		}

		return ref
	}

}
