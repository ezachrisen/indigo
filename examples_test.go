package indigo_test

import (
	"fmt"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"github.com/ezachrisen/indigo/schema"
)

// Example showing basic use of the Indigo rules engine
// with the CEL evaluator
func Example() {

	// Step 1: Create a schema
	schema := schema.Schema{
		Elements: []schema.DataElement{
			{Name: "message", Type: schema.String{}},
		},
	}

	// Step 2: Create rules
	rule := indigo.Rule{
		ID:     "hello_check",
		Schema: schema,
		Expr:   `message == "hello world"`,
	}

	// Step 3: Create an Engine with a CEL evaluator
	engine := indigo.NewEngine(cel.NewEvaluator())

	// Step 4: Compile the rule
	err := engine.Compile(&rule)
	if err != nil {
		fmt.Println(err)
		return
	}

	// The data we wish to evaluate the rule on
	data := map[string]interface{}{
		"message": "hello world",
	}

	// Step 5: Evaluate and check the results
	results, _ := engine.Eval(&rule, data)
	fmt.Println(results.Pass)
	// Output: true
}
