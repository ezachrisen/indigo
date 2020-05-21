package cel_test

import (
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/ezachrisen/rules"
	"github.com/ezachrisen/rules/cel"
	"github.com/ezachrisen/rules/testdata/school"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/cel-go/common/types/pb"
)

func TestBasicRules(t *testing.T) {

	engine := cel.NewEngine()

	education := rules.Schema{
		Elements: []rules.DataElement{
			{Name: "student.ID", Type: rules.String{}},
			{Name: "student.Age", Type: rules.Int{}},
			{Name: "student.GPA", Type: rules.Float{}},
			{Name: "student.Adjustment", Type: rules.Float{}},
			{Name: "student.Status", Type: rules.String{}},
			{Name: "student.Grades", Type: rules.List{ValueType: rules.String{}}},
			{Name: "student.EnrollmentDate", Type: rules.String{}},
			{Name: "now", Type: rules.String{}},
			{Name: "alsoNow", Type: rules.Timestamp{}},
		},
	}

	myref := "d04ab6d9-f59d-9474-5c38-34d65380c612"
	rule := rules.Rule{
		ID:     "student_actions",
		Meta:   myref,
		Schema: education,
		Rules: map[string]rules.Rule{
			"honors_student": {
				ID:   "honors_student",
				Expr: `student.GPA >= 3.6 && student.Status!="Probation" && !("C" in student.Grades)`,
			},
			"at_risk": {
				ID:   "at_risk",
				Expr: `student.GPA < 2.5 || student.Status == "Probation"`,
				Rules: map[string]rules.Rule{
					"risk_factor": {
						ID:   "risk_factor",
						Expr: `2.0+6.0`,
					},
				},
			},
		},
	}

	err := engine.AddRule(rule)
	if err != nil {
		t.Error(err)
	}

	data := map[string]interface{}{
		"student.ID":             "12312",
		"student.Age":            16,
		"student.GPA":            2.2,
		"student.Status":         "Enrolled",
		"student.Grades":         []interface{}{"A", "B", "A"},
		"student.EnrollmentDate": "2018-08-03T16:00:00-07:00",
		"student.Adjustment":     2.1,
		"now":                    "2019-08-03T16:00:00-07:00",
		"specificTime":           &timestamp.Timestamp{Seconds: time.Now().Unix()},
	}

	results, err := engine.EvaluateN(data, "student_actions", 2)
	// PrintResults(results, 0)
	// if results.XRef == nil {
	// 	t.Errorf("No xref. Expected %s", myref)
	// }
	if results.Meta != myref {
		t.Errorf("Expected Xref %v, got %v (type %T)", myref, results.Meta, results.Meta)
	}

	if results.Pass != true {
		t.Errorf("Expected true, got false: %v", results.RuleID)
	}

	if results.Results["honors_student"].Pass != false {
		t.Errorf("Expected false, got true: %v", "honors_student")
	}

	if results.Results["at_risk"].Pass != true {
		t.Errorf("Expected true, got false: %v", "at_risk")
	}

	if results.Results["at_risk"].Results["risk_factor"].Value.(float64) != 8.0 {
		t.Errorf("Expected %f, got false: %v", 8.0, results.Results["at_risk"].Results["risk_factor"].Value)

	}
}

func PrintResults(res *rules.Result, tabs int) {

	fmt.Printf("%-30s %v ", fmt.Sprintf("%s%s", strings.Repeat(" ", tabs), res.RuleID), res.Pass)
	// r := *res.Rule
	// test_rule := r.(Rule)
	// if res.Value != test_rule.expected.Value {
	// 	fmt.Printf(" --- expected %v (%T), got %v (%T)\n", test_rule.expected.Value, test_rule.expected.Value, res.Value, res.Value)
	// } else {
	fmt.Printf("\n")
	//	}

	for _, cres := range res.Results {
		PrintResults(&cres, tabs+1)
	}
}

func TestCalculation(t *testing.T) {

	engine := cel.NewEngine()

	education := rules.Schema{
		Elements: []rules.DataElement{
			{Name: "student.ID", Type: rules.String{}},
			{Name: "student.Age", Type: rules.Int{}},
			{Name: "student.GPA", Type: rules.Float{}},
			{Name: "student.Adjustment", Type: rules.Float{}},
			{Name: "student.Status", Type: rules.String{}},
			{Name: "student.Grades", Type: rules.List{ValueType: rules.String{}}},
			{Name: "student.EnrollmentDate", Type: rules.String{}},
			{Name: "now", Type: rules.String{}},
			{Name: "alsoNow", Type: rules.Timestamp{}},
		},
	}

	data := map[string]interface{}{
		"student.ID":             "12312",
		"student.Age":            16,
		"student.GPA":            2.2,
		"student.Adjustment":     1.6,
		"student.Status":         "Enrolled",
		"student.Grades":         []interface{}{"A", "B", "A"},
		"student.EnrollmentDate": "2018-08-03T16:00:00-07:00",
		"now":                    "2019-08-03T16:00:00-07:00",
	}

	v, err := engine.Calculate(data, `2.0+student.GPA + (1.344 * student.Adjustment)/3.3`, education)
	if err != nil {
		t.Error(err)
	}
	if v != 4.851636363636364 {
		t.Errorf("Expected %f, got %f", 4.851636363636364, v)
	}
}

func BenchmarkCalculation(b *testing.B) {

	engine := cel.NewEngine()

	education := rules.Schema{
		Elements: []rules.DataElement{
			{Name: "student.ID", Type: rules.String{}},
			{Name: "student.Age", Type: rules.Int{}},
			{Name: "student.GPA", Type: rules.Float{}},
			{Name: "student.Adjustment", Type: rules.Float{}},
			{Name: "student.Status", Type: rules.String{}},
			{Name: "student.Grades", Type: rules.List{ValueType: rules.String{}}},
			{Name: "student.EnrollmentDate", Type: rules.String{}},
			{Name: "now", Type: rules.String{}},
		},
	}

	data := map[string]interface{}{
		"student.ID":             "12312",
		"student.Age":            16,
		"student.GPA":            2.2,
		"student.Adjustment":     1.6,
		"student.Status":         "Enrolled",
		"student.Grades":         []interface{}{"A", "B", "A"},
		"student.EnrollmentDate": "2018-08-03T16:00:00-07:00",
		"now":                    "2019-08-03T16:00:00-07:00",
	}

	for i := 0; i < b.N; i++ {
		_, err := engine.Calculate(data, `2.0+student.GPA + (1.344 * student.Adjustment)/3.3`, education)
		if err != nil {
			b.Fatalf("Could not calculate risk factor: %v", err)
		}
	}
}

func TestProtoMessage(t *testing.T) {

	pb.DefaultDb.RegisterMessage(&school.Student{})

	schema := rules.Schema{
		Elements: []rules.DataElement{
			{Name: "student", Type: rules.Proto{Protoname: "school.Student", Message: &school.Student{}}},
			{Name: "now", Type: rules.Timestamp{}},
			{Name: "self", Type: rules.Proto{Protoname: "school.HonorsConfiguration", Message: &school.HonorsConfiguration{}}},
		},
	}

	engine := cel.NewEngine()

	rule := rules.Rule{
		ID:     "student_actions",
		Schema: schema,
		Rules: map[string]rules.Rule{
			"honor_student": {
				ID:   "honor_student",
				Expr: `student.GPA >= self.Minimum_GPA && student.Status != school.Student.status_type.PROBATION && student.Grades.all(g, g>=3.0)`,
				Self: &school.HonorsConfiguration{Minimum_GPA: 3.7},
				Meta: true,
			},
			"at_risk": {
				ID:   "at_risk",
				Expr: `student.GPA < 2.5 || student.Status == school.Student.status_type.PROBATION`,
				Meta: false,
			},
			"tenure_gt_6months": {
				ID:   "tenure_gt_6months",
				Expr: `now - student.EnrollmentDate > duration("4320h")`, // 6 months = 4320 hours
				Meta: true,
			},
		},
	}

	err := engine.AddRule(rule)
	if err != nil {
		t.Errorf("Error adding ruleset: %v", err)
	}

	s := school.Student{
		Age:            16,
		GPA:            3.76,
		Status:         school.Student_ENROLLED,
		Grades:         []float64{4.0, 4.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamp.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	data := map[string]interface{}{
		"student": &s,
		"now":     &timestamp.Timestamp{Seconds: time.Now().Unix()},
	}

	results, err := engine.Evaluate(data, "student_actions")
	if err != nil {
		t.Fatalf("Error evaluating: %v", err)
	}
	for _, v := range results.Results {
		br, found := rule.Rules[v.RuleID]
		if !found {
			t.Errorf("Unexpected rule ID in results; no corresponding rule: %s", v.RuleID)
		}
		if br.Meta != v.Pass {
			t.Errorf("Expected %t, got %t, rule %s", br.Meta, v.Pass, v.RuleID)
		}
	}
}

func BenchmarkSimpleRule(b *testing.B) {

	engine := cel.NewEngine()

	education := rules.Schema{
		Elements: []rules.DataElement{
			{Name: "student.ID", Type: rules.String{}},
			{Name: "student.Age", Type: rules.Int{}},
			{Name: "student.GPA", Type: rules.Float{}},
			{Name: "student.Status", Type: rules.String{}},
			{Name: "student.Grades", Type: rules.List{ValueType: rules.String{}}},
			{Name: "student.EnrollmentDate", Type: rules.String{}},
			{Name: "now", Type: rules.String{}},
		},
	}

	rule := rules.Rule{
		ID:     "student_actions",
		Schema: education,
		Rules: map[string]rules.Rule{
			"at_risk": {
				ID:     "at_risk",
				Schema: education,
				Expr:   `student.GPA < 2.5 || student.Status == "Probation"`,
			},
		},
	}

	err := engine.AddRule(rule)
	if err != nil {
		b.Errorf("Error adding ruleset: %v", err)
	}

	data := map[string]interface{}{
		"student.ID":             "12312",
		"student.Age":            16,
		"student.GPA":            2.2,
		"student.Status":         "Enrolled",
		"student.Grades":         []interface{}{"A", "B", "A"},
		"student.EnrollmentDate": "2018-08-03T16:00:00-07:00",
		"now":                    "2019-08-03T16:00:00-07:00",
		"alsoNow":                &timestamp.Timestamp{Seconds: time.Now().Unix()},
	}

	for i := 0; i < b.N; i++ {
		engine.Evaluate(data, "student_actions")
	}
}

func BenchmarkRuleWithArray(b *testing.B) {

	engine := cel.NewEngine()

	education := rules.Schema{
		Elements: []rules.DataElement{
			{Name: "student.ID", Type: rules.String{}},
			{Name: "student.Age", Type: rules.Int{}},
			{Name: "student.GPA", Type: rules.Float{}},
			{Name: "student.Status", Type: rules.String{}},
			{Name: "student.Grades", Type: rules.List{ValueType: rules.String{}}},
			{Name: "student.EnrollmentDate", Type: rules.String{}},
			{Name: "now", Type: rules.String{}},
		},
	}

	rule := rules.Rule{
		ID:     "student_actions",
		Schema: education,
		Rules: map[string]rules.Rule{
			"honors_student": {
				ID:     "honors_student",
				Schema: education,
				Expr:   `student.GPA >= 3.6 && student.Status!="Probation" && !("C" in student.Grades)`,
			},
		},
	}

	err := engine.AddRule(rule)
	if err != nil {
		b.Errorf("Error adding ruleset: %v", err)
	}

	data := map[string]interface{}{
		"student.ID":             "12312",
		"student.Age":            16,
		"student.GPA":            2.2,
		"student.Status":         "Enrolled",
		"student.Grades":         []interface{}{"A", "B", "A"},
		"student.EnrollmentDate": "2018-08-03T16:00:00-07:00",
		"now":                    "2019-08-03T16:00:00-07:00",
		"alsoNow":                &timestamp.Timestamp{Seconds: time.Now().Unix()},
	}

	for i := 0; i < b.N; i++ {
		engine.Evaluate(data, "student_actions")
	}
}

func BenchmarkProtoWithSelf(b *testing.B) {

	pb.DefaultDb.RegisterMessage(&school.Student{})

	schema := rules.Schema{
		Elements: []rules.DataElement{
			{Name: "student", Type: rules.Proto{Protoname: "school.Student", Message: &school.Student{}}},
			{Name: "now", Type: rules.Timestamp{}},
			{Name: "self", Type: rules.Proto{Protoname: "school.HonorsConfiguration", Message: &school.HonorsConfiguration{}}},
		},
	}

	engine := cel.NewEngine()

	rule := rules.Rule{
		ID:     "student_actions",
		Schema: schema,
		Rules: map[string]rules.Rule{
			"at_risk": {
				ID:   "at_risk",
				Expr: `student.GPA < self.Minimum_GPA || student.Status == school.Student.status_type.PROBATION`,
				Self: &school.HonorsConfiguration{Minimum_GPA: 3.7},
				Meta: false,
			},
		},
	}

	err := engine.AddRule(rule)
	if err != nil {
		log.Fatalf("Error adding ruleset: %v", err)
	}

	s := school.Student{
		Age:            16,
		GPA:            3.76,
		Status:         school.Student_ENROLLED,
		Grades:         []float64{4.0, 4.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamp.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	data := map[string]interface{}{
		"student": &s,
		"now":     &timestamp.Timestamp{Seconds: time.Now().Unix()},
	}

	for i := 0; i < b.N; i++ {
		engine.Evaluate(data, "student_actions")
	}

}

func BenchmarkProtoWithoutSelf(b *testing.B) {

	pb.DefaultDb.RegisterMessage(&school.Student{})

	schema := rules.Schema{
		Elements: []rules.DataElement{
			{Name: "student", Type: rules.Proto{Protoname: "school.Student", Message: &school.Student{}}},
			{Name: "now", Type: rules.Timestamp{}},
		},
	}

	engine := cel.NewEngine()

	rule := rules.Rule{
		ID:     "student_actions",
		Schema: schema,
		Rules: map[string]rules.Rule{
			"at_risk": {
				ID:   "at_risk",
				Expr: `student.GPA < 2.5 || student.Status == school.Student.status_type.PROBATION`,
				Meta: false,
			},
		},
	}

	err := engine.AddRule(rule)
	if err != nil {
		log.Fatalf("Error adding ruleset: %v", err)
	}

	s := school.Student{
		Age:            16,
		GPA:            3.76,
		Status:         school.Student_ENROLLED,
		Grades:         []float64{4.0, 4.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamp.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	data := map[string]interface{}{
		"student": &s,
		"now":     &timestamp.Timestamp{Seconds: time.Now().Unix()},
	}

	for i := 0; i < b.N; i++ {
		engine.Evaluate(data, "student_actions")
	}

}
