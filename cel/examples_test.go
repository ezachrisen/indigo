package cel_test

import (
	"fmt"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"github.com/ezachrisen/indigo/schema"
	"github.com/ezachrisen/indigo/testdata/school"
	"github.com/golang/protobuf/ptypes"
)

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

	// Step 3: Create Indigo and give it an evaluator
	// In this case, CEL
	evaluator := cel.NewEvaluator()

	// Step 4: Compile the rule
	err := rule.Compile(evaluator)
	if err != nil {
		fmt.Println(err)
		return
	}

	data := map[string]interface{}{
		"message": "hello world",
	}

	// Step 5: Evaluate and check the results
	results, err := rule.Evaluate(evaluator, data)
	fmt.Println(results.Pass)
	// Output: true
}

func ExampleCELEvaluator_timestamps() {
	schema := schema.Schema{
		Elements: []schema.DataElement{
			{Name: "then", Type: schema.String{}},
			{Name: "now", Type: schema.Timestamp{}},
		},
	}

	data := map[string]interface{}{
		"then": "1972-01-01T10:00:20.021-05:00", //"2018-08-03T16:00:00-07:00",
		"now":  ptypes.TimestampNow(),
	}

	rule := indigo.Rule{
		ID:     "time_check",
		Schema: schema,
		Expr:   `now > timestamp(then)`,
	}

	evaluator := cel.NewEvaluator()
	err := rule.Compile(evaluator)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := rule.Evaluate(evaluator, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println(results.Pass)
	// Output: true
}

func ExampleCELEvaluator_exists() {

	education := schema.Schema{
		Elements: []schema.DataElement{
			{Name: "student", Type: schema.Proto{Protoname: "school.Student", Message: &school.Student{}}},
		},
	}

	data := map[string]interface{}{
		"student": school.Student{
			Grades: []float64{3.0, 2.9, 4.0, 2.1},
		},
	}

	rule := indigo.Rule{
		ID:     "grade_check",
		Schema: education,
		Expr:   `student.Grades.exists(g, g < 2.0)`,
	}

	evaluator := cel.NewEvaluator()
	err := rule.Compile(evaluator)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := rule.Evaluate(evaluator, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println(results.Pass)
	// Output: false
}

// Demonstrates using the exists macro to inspect the value of nested messages in the list
func ExampleCELEvaluator_nested() {

	education := schema.Schema{
		Elements: []schema.DataElement{
			{Name: "student", Type: schema.Proto{Protoname: "school.Student", Message: &school.Student{}}},
			{Name: "student_suspension", Type: schema.Proto{Protoname: "school.Student.Suspension", Message: &school.Student_Suspension{}}},
		},
	}

	data := map[string]interface{}{
		"student": school.Student{
			Grades: []float64{3.0, 2.9, 4.0, 2.1},
			Suspensions: []*school.Student_Suspension{
				&school.Student_Suspension{Cause: "Cheating"},
				&school.Student_Suspension{Cause: "Fighting"},
			},
		},
	}

	// Check if the student was ever suspended for fighting
	rule := indigo.Rule{
		ID:     "fighting_check",
		Schema: education,
		Expr:   `student.Suspensions.exists(s, s.Cause == "Fighting")`,
	}

	evaluator := cel.NewEvaluator()
	err := rule.Compile(evaluator)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := rule.Evaluate(evaluator, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println(results.Value)
	// Output: true
}

// Demonstrate constructing a proto message in an expression
func ExampleCELEvaluator_construction() {

	education := schema.Schema{
		Elements: []schema.DataElement{
			{Name: "student", Type: schema.Proto{Protoname: "school.Student", Message: &school.Student{}}},
			{Name: "student_suspension", Type: schema.Proto{Protoname: "school.Student.Suspension", Message: &school.Student_Suspension{}}},
			{Name: "studentSummary", Type: schema.Proto{Protoname: "school.StudentSummary", Message: &school.StudentSummary{}}},
		},
	}

	data := map[string]interface{}{
		"student": school.Student{
			Grades: []float64{3.0, 2.9, 4.0, 2.1},
			Suspensions: []*school.Student_Suspension{
				&school.Student_Suspension{Cause: "Cheating"},
				&school.Student_Suspension{Cause: "Fighting"},
			},
		},
	}

	rule := indigo.Rule{
		ID:         "create_summary",
		Schema:     education,
		ResultType: schema.Proto{Protoname: "school.StudentSummary", Message: &school.StudentSummary{}},
		Expr: `
			school.StudentSummary {
				GPA: student.GPA,
				RiskFactor: 2.0 + 3.0,
				Tenure: duration("12h")
			}`,
	}

	evaluator := cel.NewEvaluator()
	err := rule.Compile(evaluator)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := rule.Evaluate(evaluator, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}

	// The result is a fully-formed school.StudentSummary message.
	// There is no need to convert it.
	fmt.Printf("%T\n", results.Value)
	summ := results.Value.(*school.StudentSummary)
	fmt.Println(summ.RiskFactor)
	// Output: *school.StudentSummary
	// 5
}

// Demonstrate using the ? : operator to conditionally construct a proto message
func ExampleConditionalProtoConstruction() {

	education := schema.Schema{
		Elements: []schema.DataElement{
			{Name: "student", Type: schema.Proto{Protoname: "school.Student", Message: &school.Student{}}},
			{Name: "student_suspension", Type: schema.Proto{Protoname: "school.Student.Suspension", Message: &school.Student_Suspension{}}},
			{Name: "studentSummary", Type: schema.Proto{Protoname: "school.StudentSummary", Message: &school.StudentSummary{}}},
		},
	}

	data := map[string]interface{}{
		"student": school.Student{
			GPA:    3.6,
			Grades: []float64{3.0, 2.9, 4.0, 2.1},
			Suspensions: []*school.Student_Suspension{
				&school.Student_Suspension{Cause: "Cheating"},
				&school.Student_Suspension{Cause: "Fighting"},
			},
		},
	}

	rule := indigo.Rule{
		ID:         "create_summary",
		Schema:     education,
		ResultType: schema.Proto{Protoname: "school.StudentSummary", Message: &school.StudentSummary{}},
		Expr: `
			student.GPA > 3.0 ? 
				school.StudentSummary {
					GPA: student.GPA,
					RiskFactor: 0.0
				}
			:
				school.StudentSummary {
					GPA: student.GPA,
					RiskFactor: 2.0 + 3.0,
					Tenure: duration("12h")
				}
			`,
	}

	evaluator := cel.NewEvaluator()
	err := rule.Compile(evaluator)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := rule.Evaluate(evaluator, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}

	// The result is a fully-formed school.StudentSummary message.
	// There is no need to convert it.
	fmt.Printf("%T\n", results.Value)
	summ := results.Value.(*school.StudentSummary)
	fmt.Println(summ.RiskFactor)
	// Output: *school.StudentSummary
	// 0
}
