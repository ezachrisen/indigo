package cel_test

import (
	"context"
	"fmt"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"

	"github.com/ezachrisen/indigo/testdata/school"
)

// Example_manual demonstrates evaluating multiple rules and
// processing the results manually
func Example_manual() {

	// In this example we're going to determine which, if any,
	// communications the administration should send to the student.
	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "s", Type: indigo.Proto{Message: &school.Student{}}},
		},
	}

	data := map[string]interface{}{
		"s": &school.Student{
			Id:      927312,
			Age:     21,
			Credits: 16,
			Gpa:     3.1,
			Attrs:   map[string]string{"major": "Accounting", "home_town": "Chicago"},
			Status:  school.Student_ENROLLED,
			Grades:  []float64{3, 3, 4, 2, 3, 3.5, 4},
		},
	}

	accounting_honors := indigo.Rule{
		Schema: education,
		Expr:   `s.attrs.exists(k, k == "major" && s.attrs[k] == "Accounting") && s.gpa > 3`,
	}

	arts_honors := indigo.Rule{
		Schema: education,
		Expr:   `s.attrs.exists(k, k == "major" && s.attrs[k] == "Arts") && s.gpa > 3`,
	}

	last_3_grades_3_or_above := indigo.Rule{
		Schema: education,
		Expr: `size(s.grades) >=3 
                 && s.grades[size(s.grades)-1] >= 3.0 
                 && s.grades[size(s.grades)-2] >= 3.0 
                 && s.grades[size(s.grades)-3] >= 3.0 `,
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&accounting_honors)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Eval(context.Background(), &accounting_honors, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println("accounting_honors?", results.ExpressionPass)

	err = engine.Compile(&arts_honors)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err = engine.Eval(context.Background(), &arts_honors, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println("arts_honors?", results.ExpressionPass)

	err = engine.Compile(&last_3_grades_3_or_above)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err = engine.Eval(context.Background(), &last_3_grades_3_or_above, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println("last_3_grades_above_3?", results.ExpressionPass)

	// Output: accounting_honors? true
	// arts_honors? false
	// last_3_grades_above_3? true
}

func Example_indigo() {

	// In this example we're going to determine which, if any,
	// communications the administration should send to the student.
	education := indigo.Schema{
		ID: "education",
		Elements: []indigo.DataElement{
			{Name: "s", Type: indigo.Proto{Message: &school.Student{}}},
		},
	}

	data := map[string]interface{}{
		"s": &school.Student{
			Id:      927312,
			Age:     21,
			Credits: 16,
			Gpa:     3.1,
			Attrs:   map[string]string{"major": "Accounting", "home_town": "Chicago"},
			Status:  school.Student_ENROLLED,
			Grades:  []float64{3, 3, 4, 2, 3, 3.5, 4},
		},
	}

	root := indigo.NewRule("root", "")
	root.Schema = education

	root.Add(indigo.NewRule("accounting_honors",
		`s.attrs.exists(k, k == "major" && s.attrs[k] == "Accounting") 
         && s.gpa > 3`))

	root.Add(indigo.NewRule("arts_honors",
		`s.attrs.exists(k, k == "major" && s.attrs[k] == "Arts") 
         && s.gpa > 3`))

	root.Add(indigo.NewRule("last_3_grades_above_3",
		`size(s.grades) >=3 
         && s.grades[size(s.grades)-1] >= 3.0 
         && s.grades[size(s.grades)-2] >= 3.0 
         && s.grades[size(s.grades)-3] >= 3.0 `))

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(root)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Eval(context.Background(), root, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}

	for k, v := range results.Results {
		fmt.Printf("%s? %t\n", k, v.ExpressionPass)
	}

	fmt.Println(results)

	// Output: accounting_honors? true
	// arts_honors? false
	// last_3_grades_above_3? true
}
