package examples_test

import (
	"fmt"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"github.com/ezachrisen/indigo/cel/examples"
)

func ExampleSchool() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Struct{Name: "examples.Student", Struct: &examples.Student{}}},
		},
	}

	data := map[string]interface{}{
		"student": examples.Student{
			GPA: 2.6,
			Age: 16,
		},
	}

	// Check if the student was ever suspended for fighting
	rule := indigo.Rule{
		ID:     "fighting_check",
		Schema: education,
		Expr:   `student.GPA > 2.0 && (student.Age + 1) > 16`,
	}

	ap := cel.NewAttributeProvider()
	//	ap.RegisterType(&examples.Student{})

	evaluator := cel.NewEvaluator(ap)
	engine := indigo.NewEngine(evaluator)
	err := engine.AddRule(&rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Evaluate(data, "fighting_check")
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println(results.Value)
	// Output: true
}
