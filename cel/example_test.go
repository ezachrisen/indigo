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

func Example_basic() {

	//Step 1: Create a schema
	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "x", Type: indigo.Int{}},
			{Name: "y", Type: indigo.String{}},
		},
	}

	// Step 2: Create rules
	rule := indigo.Rule{
		Schema:     schema,
		ResultType: indigo.Bool{},
		Expr:       `x > 10 && y != "blue"`,
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
		"x": 11,
		"y": "red",
	}

	// Step 5: Evaluate and check the results
	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(results.ExpressionPass)

	// Output: true
}

func Example_list() {

	//Step 1: Create a schema
	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "grades", Type: indigo.List{ValueType: indigo.Float{}}},
		},
	}

	// Step 2: Create rules
	rule := indigo.Rule{
		Schema:     schema,
		ResultType: indigo.Bool{},
		Expr:       `size(grades) > 3`,
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
		"grades": []float64{3.4, 3.6, 3.8, 2.9},
	}

	// Step 5: Evaluate and check the results
	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Is size(grades) > 3? ", results.ExpressionPass)
	}

	// Check the value of a specific element
	rule.Expr = `grades[1] == 3.6`
	engine.Compile(&rule)
	results, _ = engine.Eval(context.Background(), &rule, data)
	fmt.Println("Is element 1 == 3.6? ", results.ExpressionPass)

	// Check if the list contains a value
	rule.Expr = `grades.exists(g, g < 3.0)`
	engine.Compile(&rule)
	results, _ = engine.Eval(context.Background(), &rule, data)
	fmt.Println("Any grades below 3.0? ", results.ExpressionPass)

	// Output: Is size(grades) > 3?  true
	// Is element 1 == 3.6?  true
	// Any grades below 3.0?  true
}

func Example_map() {

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "flights", Type: indigo.Map{KeyType: indigo.String{}, ValueType: indigo.String{}}},
		},
	}

	rule := indigo.Rule{
		Schema:     schema,
		ResultType: indigo.Bool{},
		Expr:       `flights.exists(k, flights[k] == "Delayed")`,
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	if err != nil {
		fmt.Println(err)
		return
	}

	data := map[string]interface{}{
		"flights": map[string]string{"UA1500": "On Time", "DL232": "Delayed", "AA1622": "Delayed"},
	}

	// Step 5: Evaluate and check the results
	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Are any flights delayed?", results.ExpressionPass)
	}

	// Output: Are any flights delayed? true
}

// Demonstrates using the in operator on lists and maps
func Example_in() {

	schema := indigo.Schema{
		Elements: []indigo.DataElement{

			{Name: "x", Type: indigo.String{}},
			{Name: "flights", Type: indigo.Map{KeyType: indigo.String{}, ValueType: indigo.String{}}},
			{Name: "holding", Type: indigo.List{ValueType: indigo.String{}}},
		},
	}

	rule := indigo.Rule{
		Schema:     schema,
		ResultType: indigo.Bool{},
		Expr:       `x in ["UA1500", "ABC123"]`,
		//		Expr:       `"UA1500" in flights && "SW123" in holding`,
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	if err != nil {
		fmt.Println(err)
		return
	}

	data := map[string]interface{}{
		"x":       "ABC123",
		"flights": map[string]string{"UA1500": "On Time", "DL232": "Delayed", "AA1622": "Delayed"},
		"holding": []string{"SW123", "BA355", "UA91"},
	}

	// Step 5: Evaluate and check the results
	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(results.Pass)

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

// Demonstrates basic protocol buffer usage in a rule
func Example_protoBasic() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
		},
	}
	data := map[string]interface{}{
		"student": &school.Student{
			Age: 21,
		},
	}

	rule := indigo.Rule{
		Schema: education,
		Expr:   `student.age > 21`,
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

// Demonstrates using a protocol buffer enum value in a rule
func Example_protoEnum() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
		},
	}
	data := map[string]interface{}{
		"student": &school.Student{
			Status: school.Student_ENROLLED,
		},
	}

	rule := indigo.Rule{
		Schema: education,
		Expr:   `student.status == testdata.school.Student.status_type.PROBATION`,
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

// Demonstrates using a protocol buffer oneof value in a rule
func Example_protoOneof() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
		},
	}
	data := map[string]interface{}{
		"student": &school.Student{
			Status: school.Student_ENROLLED,
			HousingAddress: &school.Student_OnCampus{
				&school.Student_CampusAddress{
					Building: "Hershey",
					Room:     "308",
				},
			},
		},
	}

	rule := indigo.Rule{
		Schema: education,
		Expr:   `has(student.on_campus) && student.on_campus.building == "Hershey"`,
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

// Demonstrates writing rules on timestamps and durations
func Example_protoTimestampAndDurationComparison() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "s", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
			{Name: "ssum", Type: indigo.Proto{Message: &school.StudentSummary{}}},
		},
	}

	data := map[string]interface{}{
		"s": &school.Student{
			EnrollmentDate: timestamppb.New(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
		},
		"ssum": &school.StudentSummary{
			Tenure: durationpb.New(time.Duration(time.Hour * 24 * 451)),
		},
		"now": timestamppb.Now(),
	}

	rule := indigo.Rule{
		ID:     "",
		Schema: education,
		Expr: `s.enrollment_date < now
               && 
               // 12,000h = 500 days * 24 hours
               ssum.tenure > duration("12000h")
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

	fmt.Println(results.ExpressionPass)
	// Output: false
}

// Demonstrates writing rules on timestamps and durations
func Example_protoTimestampPart() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "s", Type: indigo.Proto{Message: &school.Student{}}},
		},
	}

	data := map[string]interface{}{
		"s": &school.Student{
			EnrollmentDate: timestamppb.New(time.Date(2022, time.April, 8, 23, 0, 0, 0, time.UTC)),
		},
	}

	rule := indigo.Rule{
		ID:         "",
		ResultType: indigo.Bool{},
		Schema:     education,
		Expr:       `s.enrollment_date.getDayOfWeek() == 5 // Friday`,
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

// Demonstrates writing rules on timestamps and durations
func Example_protoTimestampPartTZ() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "s", Type: indigo.Proto{Message: &school.Student{}}},
		},
	}

	data := map[string]interface{}{
		"s": &school.Student{
			EnrollmentDate: timestamppb.New(time.Date(2022, time.April, 8, 23, 0, 0, 0, time.UTC)),
		},
	}

	rule := indigo.Rule{
		ID:         "",
		ResultType: indigo.Bool{},
		Schema:     education,
		Expr:       `s.enrollment_date.getDayOfWeek("Asia/Kolkata") == 6 // Saturday`,
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

// Demonstrates writing rules on timestamps and durations
func Example_protoDurationCalculation() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "s", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
			{Name: "ssum", Type: indigo.Proto{Message: &school.StudentSummary{}}},
		},
	}

	data := map[string]interface{}{
		"s": &school.Student{
			EnrollmentDate: timestamppb.New(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
		},
		"now": timestamppb.Now(),
	}

	rule := indigo.Rule{
		ID:     "",
		Schema: education,
		Expr: `// 2,400h = 100 days * 24 hours
               now - s.enrollment_date  > duration("2400h")
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

	fmt.Println(results.ExpressionPass)
	// Output: true
}

// Demonstrates using the exists macro to inspect the value of nested messages in the list
func Example_protoNestedMessages() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "x", Type: indigo.Proto{Message: &school.Student{}}}},
	}

	data := map[string]interface{}{
		"x": &school.Student{
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
		Expr:   `x.suspensions.exists(s, s.cause == "Fighting")`,
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
			{Name: "s", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "student_suspension", Type: indigo.Proto{Message: &school.Student_Suspension{}}},
			{Name: "studentSummary", Type: indigo.Proto{Message: &school.StudentSummary{}}},
		},
	}

	data := map[string]interface{}{
		"s": &school.Student{
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
				gpa: s.gpa,
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

// Example_alarms illustrates using the FailAction option to only return
// true rules from evaluation
func Example_alarms() {

	sysmetrics := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "cpu_utilization", Type: indigo.Int{}},
			{Name: "disk_free_space", Type: indigo.Int{}},
			{Name: "memory_utilization", Type: indigo.Int{}},
		},
	}

	rule := indigo.Rule{
		ID:    "alarm_check",
		Rules: map[string]*indigo.Rule{},
	}

	// Setting this option so we only get back
	// rules that evaluate to 'true'
	rule.EvalOptions.DiscardFail = indigo.Discard

	rule.Rules["cpu_alarm"] = &indigo.Rule{
		ID:     "cpu_alarm",
		Schema: sysmetrics,
		Expr:   "cpu_utilization > 90",
	}

	rule.Rules["disk_alarm"] = &indigo.Rule{
		ID:     "disk_alarm",
		Schema: sysmetrics,
		Expr:   "disk_free_space < 70",
	}

	rule.Rules["memory_alarm"] = &indigo.Rule{
		ID:     "memory_alarm",
		Schema: sysmetrics,
		Expr:   "memory_utilization > 90",
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	if err != nil {
		fmt.Println(err)
		return
	}

	data := map[string]interface{}{
		"cpu_utilization":    99,
		"disk_free_space":    85,
		"memory_utilization": 89,
	}

	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Println(err)
	}

	for k := range results.Results {
		fmt.Println(k)
	}

	// Unordered output: cpu_alarm
}

// Example_alarms illustrates using the FailAction option to only return
// true rules from evaluation with a multi-level hierarchy
func Example_alarmsTwoLevel() {

	sysmetrics := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "cpu_utilization", Type: indigo.Int{}},
			{Name: "disk_free_space", Type: indigo.Int{}},
			{Name: "memory_utilization", Type: indigo.Int{}},
			{Name: "memory_mb_remaining", Type: indigo.Int{}},
		},
	}

	rule := indigo.Rule{
		ID:    "alarm_check",
		Rules: map[string]*indigo.Rule{},
	}

	// Setting this option so we only get back
	// rules that evaluate to 'true'
	rule.EvalOptions.DiscardFail = indigo.Discard

	rule.Rules["cpu_alarm"] = &indigo.Rule{
		ID:     "cpu_alarm",
		Schema: sysmetrics,
		Expr:   "cpu_utilization > 90",
	}

	rule.Rules["disk_alarm"] = &indigo.Rule{
		ID:     "disk_alarm",
		Schema: sysmetrics,
		Expr:   "disk_free_space < 70",
	}

	memory_alarm := &indigo.Rule{
		ID:    "memory_alarm",
		Rules: map[string]*indigo.Rule{},
		EvalOptions: indigo.EvalOptions{
			DiscardFail: indigo.KeepAll,
			TrueIfAny:   true,
		},
	}

	memory_alarm.Rules["memory_utilization_alarm"] = &indigo.Rule{
		ID:     "memory_utilization_alarm",
		Schema: sysmetrics,
		Expr:   "memory_utilization > 90",
	}

	memory_alarm.Rules["memory_remaining_alarm"] = &indigo.Rule{
		ID:     "memory_remaining_alarm",
		Schema: sysmetrics,
		Expr:   "memory_mb_remaining < 16",
	}

	rule.Rules["memory_alarm"] = memory_alarm

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	if err != nil {
		fmt.Println(err)
		return
	}

	data := map[string]interface{}{
		"cpu_utilization":     99,
		"disk_free_space":     85,
		"memory_utilization":  89,
		"memory_mb_remaining": 7,
	}

	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Println(err)
	}

	for k := range results.Results {
		fmt.Println(k)
	}

	// Unordered output: cpu_alarm
	// memory_alarm
}
