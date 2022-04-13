package cel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"

	"github.com/matryer/is"
)

func TestCustomFunc(t *testing.T) {

	is := is.New(t)

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "flights", Type: indigo.Map{KeyType: indigo.String{}, ValueType: indigo.String{}}},
			{Name: "contains",
				Type: indigo.Func{
					Func:            mapContainsKeyValue,
					Name:            "contains",
					DefinedOn:       indigo.Map{KeyType: indigo.String{}, ValueType: indigo.String{}},
					Args:            []indigo.Type{indigo.String{}, indigo.String{}},
					ReturnValueType: indigo.Bool{},
				},
			},
		},
	}

	rule := indigo.Rule{
		Schema:     schema,
		ResultType: indigo.Bool{},
		Expr:       `flights.contains("UA1500", "On Time")`,
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	is.NoErr(err)

	data := map[string]interface{}{
		"flights": map[string]string{"UA1500": "On Time", "DL232": "Delayed", "AA1622": "Delayed"},
	}

	results, err := engine.Eval(context.Background(), &rule, data)
	is.NoErr(err)
	is.Equal(results.Pass, true)
	fmt.Printf("Result: %v\n", results.Value)
}

// mapContainsKeyValue implements the custom function:
//   `map.contains(key, value) bool`.
//func mapContainsKeyValue(args ...ref.Val) ref.Val {
func mapContainsKeyValue(inargs1 interface{}) interface{} {
	// // Check the argument input count.
	// fmt.Printf("inargs = %+v, type = %T\n", inargs1, inargs1)
	// for _, z := range inargs1 {
	// 	fmt.Printf("%v %T\n", z, z)
	// }
	// if len(inargs1) != 1 {
	// 	return types.NewErr("expected 1 arg")
	// }
	// inargs, ok := inargs1[0].([]ref.Val)
	// if !ok {
	// 	return types.NewErr("can't do type conversion")
	// }
	// if len(inargs) != 3 {
	// 	return types.NewErr(fmt.Sprintf("no such overload; expected 3 args, got %d", len(inargs)))
	// }

	args, ok := inargs1.([]ref.Val)
	if !ok {
		return types.NewErr(fmt.Sprintf("expected an array of ref.Val, got %T", inargs1))
	}

	// var args []ref.Val
	// for _, a := range inargs {
	// 	v, ok := a.(ref.Val)
	// 	if !ok {
	// 		return types.NewErr("received function argument to mapContainsKeyValue that could not be converted to ref.Val")
	// 	}
	// 	args = append(args, v)
	// }

	obj := args[0]
	m, isMap := obj.(traits.Mapper)
	// Ensure the argument is a CEL map type, otherwise error.
	// The type-checking is a best effort check to ensure that the types provided
	// to functions match the ones specified; however, it is always possible that
	// the implementation does not match the declaration. Always check arguments
	// types whenever there is a possibility that your function will deal with
	// dynamic content.
	if !isMap {
		// The helper ValOrErr ensures that errors on input are propagated.
		return types.ValOrErr(obj, "no such overload")
	}

	// CEL has many interfaces for dealing with different type abstractions.
	// The traits.Mapper interface unifies field presence testing on proto
	// messages and maps.
	key := args[1]
	v, found := m.Find(key)
	// If not found and the value was non-nil, the value is an error per the
	// `Find` contract. Propagate it accordingly.
	if !found {
		if v != nil {
			return types.ValOrErr(v, "unsupported key type")
		}
		// Return CEL False if the key was not found.
		return types.False
	}
	// Otherwise whether the value at the key equals the value provided.
	return v.Equal(args[2])
}
