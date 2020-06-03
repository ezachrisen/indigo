package cel_test

import (
	"fmt"

	"github.com/ezachrisen/rules"
	"github.com/ezachrisen/rules/cel"
	"github.com/ezachrisen/rules/testdata/school"
	"github.com/golang/protobuf/ptypes"
)

// Calculate a student's 2-semester GPA
func ExampleCalculation() {
	education := rules.Schema{
		Elements: []rules.DataElement{
			{Name: "student", Type: rules.Proto{Protoname: "school.Student", Message: &school.Student{}}},
		},
	}

	data := map[string]interface{}{
		"student": school.Student{
			Grades: []float64{3.0, 2.9, 4.0, 2.1},
		},
	}

	engine := cel.NewEngine()
	gpa, err := engine.Calculate(data, `(student.Grades[0] + student.Grades[1] ) /2.0`, education)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println(gpa)
	// Output: 2.95
}

func ExampleTimestampComparison() {
	schema := rules.Schema{
		Elements: []rules.DataElement{
			{Name: "then", Type: rules.String{}},
			{Name: "now", Type: rules.Timestamp{}},
		},
	}

	data := map[string]interface{}{
		"then": "1972-01-01T10:00:20.021-05:00", //"2018-08-03T16:00:00-07:00",
		"now":  ptypes.TimestampNow(),
	}

	rule := rules.Rule{
		ID:     "time_check",
		Schema: schema,
		Expr:   `now > timestamp(then)`,
	}

	engine := cel.NewEngine()
	err := engine.AddRule(rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Evaluate(data, "time_check")
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println(results.Pass)
	// Output: true
}

func ExampleExists() {

	education := rules.Schema{
		Elements: []rules.DataElement{
			{Name: "student", Type: rules.Proto{Protoname: "school.Student", Message: &school.Student{}}},
		},
	}

	data := map[string]interface{}{
		"student": school.Student{
			Grades: []float64{3.0, 2.9, 4.0, 2.1},
		},
	}

	rule := rules.Rule{
		ID:     "grade_check",
		Schema: education,
		Expr:   `student.Grades.exists(g, g < 2.0)`,
	}

	engine := cel.NewEngine()
	err := engine.AddRule(rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Evaluate(data, "grade_check")
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println(results.Pass)
	// Output: false
}

func ExampleExistsNested() {

	education := rules.Schema{
		Elements: []rules.DataElement{
			{Name: "student", Type: rules.Proto{Protoname: "school.Student", Message: &school.Student{}}},
			{Name: "student_suspension", Type: rules.Proto{Protoname: "school.Student.Suspension", Message: &school.Student_Suspension{}}},
		},
	}

	data := map[string]interface{}{
		"student": school.Student{
			Grades: []float64{3.0, 2.9, 4.0, 2.1},
			XYZ:    &school.Student_Suspension{Cause: "Being funny"},
			Suspensions: []*school.Student_Suspension{
				&school.Student_Suspension{Cause: "Cheating"},
				&school.Student_Suspension{Cause: "Fighting"},
			},
		},
	}

	rule := rules.Rule{
		ID:     "fighting_check",
		Schema: education,
		Expr:   `student.XYZ.Cause == "Blah"`,
	}

	engine := cel.NewEngine()
	err := engine.AddRule(rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Evaluate(data, "fighting_check")
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println("RESULT: ", results.Value)
	// Output: false
}
