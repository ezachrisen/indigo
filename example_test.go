package indigo_test

import (
	"context"
	"fmt"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
)

// Example showing basic use of the Indigo rules engine
// with the CEL evaluator
func Example() {

	// Step 1: Create a schema
	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "message", Type: indigo.String{}},
		},
	}

	// Step 2: Create rules
	rule := indigo.Rule{
		ID:         "hello_check",
		Schema:     schema,
		Expr:       `message == "hello world"`,
		ResultType: indigo.Bool{},
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
	data := map[string]any{
		"message": "hello world",
	}

	// Step 5: Evaluate and check the results
	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(results.ExpressionPass)
	}
	// Output: true
}

// Demonstrate parsing indigo types represented as strings
func ExampleParseType() {

	// Parse a string to obtain the Indigo type.
	raw, err := indigo.ParseType("map[int]float")
	if err != nil {
		fmt.Println(err)
	}

	// Check that we actually got a Map type
	t, ok := raw.(indigo.Map)
	if !ok {
		fmt.Println("Incorrect type!")
	}

	fmt.Println(t.KeyType, t.ValueType)
	// Output: int float

}

// Demonstrate applying a function to a rule
func ExampleApplyToRule() {

	r := makeRule()

	err := indigo.ApplyToRule(r, func(r *indigo.Rule) error {
		fmt.Printf("%s ", r.ID)
		return nil
	})

	if err != nil {
		fmt.Println("Failure!")
	}

	fmt.Printf("\n")
	// Output unordered: rule1 B b1 b2 b3 b4 b4-1 b4-2 E e1 e2 e3 D d1 d2 d3
}

// Demonstrate setting the sorting function for all rules
// to be alphabetical, based on the rule ID
func ExampleSortFunc() {

	r := makeRule()

	err := indigo.ApplyToRule(r, func(r *indigo.Rule) error {
		r.EvalOptions.SortFunc = func(rules []*indigo.Rule, i, j int) bool {
			return rules[i].ID < rules[j].ID
		}
		return nil
	})

	if err != nil {
		fmt.Println("Failure!")
	}

	fmt.Println("Ok")
	//Output: Ok
}
