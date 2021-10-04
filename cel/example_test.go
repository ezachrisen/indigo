package cel_test

import (
	"context"
	"fmt"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ezachrisen/indigo/testdata/school"
)

func Example() {

	//Step 1: Create a schema
	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "message", Type: indigo.String{}},
		},
	}

	// Step 2: Create rules
	rule := indigo.Rule{
		ID:         "hello_check",
		Schema:     schema,
		ResultType: indigo.Bool{},
		Expr:       `message == "hello world"`,
	}

	// Step 3: Create an Indigo engine and give it an evaluator
	// In this case, CEL
	engine := indigo.NewEngine(cel.NewEvaluator())

	// Step 4: Compile the rule
	err := engine.Compile(&rule)
	if err != nil {
		fmt.Println(err)
		return
	}

	data := map[string]interface{}{
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

func Example_nativeTimestampComparison() {
	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "then", Type: indigo.String{}},
			{Name: "now", Type: indigo.Timestamp{}},
		},
	}

	data := map[string]interface{}{
		"then": "1972-01-01T10:00:20.021-05:00", //"2018-08-03T16:00:00-07:00",
		"now":  timestamppb.Now(),
	}

	rule := indigo.Rule{
		ID:     "time_check",
		Schema: schema,
		Expr:   `now > timestamp(then)`,
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println(results.ExpressionPass)
	// Output: true
}

// Demonstrates using the CEL exists function to check for a value in a slice
func Example_protoExistsOperator() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
		},
	}
	data := map[string]interface{}{
		"student": &school.Student{
			Grades: []float64{3.0, 2.9, 4.0, 2.1},
		},
	}

	rule := indigo.Rule{
		ID:     "grade_check",
		Schema: education,
		Expr:   `student.grades.exists(g, g < 2.0)`,
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println(results.ExpressionPass)
	// Output: false
}

// Demonstrates conversion between protobuf timestamps (google.protobuf.Timestamp) and time.Time
func Example_protoTimestampComparison() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
		},
	}

	data := map[string]interface{}{
		"student": &school.Student{
			// Make a protobuf timestamp from a time.Time
			EnrollmentDate: timestamppb.New(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
			Grades:         []float64{3.0, 2.9, 4.0, 2.1},
		},
		"now": timestamppb.Now(),
	}

	// The rule will return the earlier of the two dates (enrollment date or now)
	rule := indigo.Rule{
		ID:         "grade_check",
		Schema:     education,
		ResultType: indigo.Timestamp{},
		Expr: `student.enrollment_date < now
			  ?
			  student.enrollment_date
			  :
			  now
			  `,
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}

	if ts, ok := results.Value.(time.Time); ok {
		fmt.Printf("Gotime is %v\n", ts)
	}
	// Output: Gotime is 2009-11-10 23:00:00 +0000 UTC
}

// Demonstrates conversion between protobuf durations (google.protobuf.Duration) and time.Duration
func Example_protoDurationComparison() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "smry", Type: indigo.Proto{Message: &school.StudentSummary{}}},
		},
	}

	godur, _ := time.ParseDuration("10h")

	data := map[string]interface{}{
		"smry": &school.StudentSummary{
			Tenure: durationpb.New(godur),
		},
	}

	rule := indigo.Rule{
		ID:     "tenure_check",
		Schema: education,
		Expr:   `smry.tenure > duration("1h")`,
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println(results.ExpressionPass)
	// Output: true
}

// Demonstrates using the exists macro to inspect the value of nested messages in the list
func Example_protoNestedMessages() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "student_suspension", Type: indigo.Proto{Message: &school.Student_Suspension{}}},
		},
	}

	data := map[string]interface{}{
		"student": &school.Student{
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
		Expr:   `student.suspensions.exists(s, s.cause == "Fighting")`,
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println(results.Value)
	// Output: true
}

// Demonstrate constructing a proto message in an expression
func Example_protoConstruction() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "student_suspension", Type: indigo.Proto{Message: &school.Student_Suspension{}}},
			{Name: "studentSummary", Type: indigo.Proto{Message: &school.StudentSummary{}}},
		},
	}

	data := map[string]interface{}{
		"student": &school.Student{
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
		ResultType: indigo.Proto{Message: &school.StudentSummary{}},
		Expr: `
			testdata.school.StudentSummary {
				gpa: student.gpa,
				risk_factor: 2.0 + 3.0,
				tenure: duration("12h")
			}`,
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	result, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		//		fmt.Printf("Error evaluating: %v", err)
		return
	}

	summary := result.Value.(*school.StudentSummary)

	fmt.Printf("%T\n", summary)
	fmt.Printf("%0.0f\n", summary.RiskFactor)
	// Output: *school.StudentSummary
	// 5
}

// Demonstrate using the ? : operator to conditionally construct a proto message
func Example_protoConstructionConditional() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "student_suspension", Type: indigo.Proto{Message: &school.Student_Suspension{}}},
			{Name: "studentSummary", Type: indigo.Proto{Message: &school.StudentSummary{}}},
		},
	}

	data := map[string]interface{}{
		"student": &school.Student{
			Gpa:    4.0,
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
		ResultType: indigo.Proto{Message: &school.StudentSummary{}},
		Expr: `
			student.gpa > 3.0 ?
				testdata.school.StudentSummary {
					gpa: student.gpa,
					risk_factor: 0.0
				}
			:
				testdata.school.StudentSummary {
					gpa: student.gpa,
					risk_factor: 2.0 + 3.0,
					tenure: duration("12h")
				}
			`,
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Eval(context.Background(), &rule, data)
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
