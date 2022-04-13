package cel_test

import (
	"context"
	"testing"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"

	"github.com/matryer/is"
)

func TestCustomFunc(t *testing.T) {

	is := is.New(t)

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "flights", Type: indigo.Map{KeyType: indigo.String{}, ValueType: indigo.String{}}},
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
	t.Error("FAILED FUNC")
}
