package cel_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/ezachrisen/rules"
	"github.com/ezachrisen/rules/cel"
	"github.com/golang/protobuf/ptypes/timestamp"
)

func TestSimpleCEL(t *testing.T) {

	engine := cel.NewEngine()

	schema := rules.Schema{
		Elements: []rules.DataElement{
			{Name: "student.ID", Type: rules.String{}},
			{Name: "student.Age", Type: rules.Int{}},
			{Name: "student.GPA", Type: rules.Float{}},
			{Name: "student.Status", Type: rules.String{}},
			{Name: "student.Grades", Type: rules.List{ValueType: rules.String{}}},
			{Name: "student.EnrollmentDate", Type: rules.String{}},
			{Name: "now", Type: rules.String{}},
			{Name: "alsoNow", Type: rules.Timestamp{}},
		},
	}

	ruleSet := rules.RuleSet{
		ID:     "student_actions",
		Schema: schema,
		Rules: map[string]rules.Rule{
			"honor_student":                rules.SimpleRule{Expr: `student.GPA >= 3.6 && student.Status!="Probation" && !("C" in student.Grades)`},
			"at_risk":                      rules.SimpleRule{Expr: `student.GPA < 2.5 || student.Status == "Probation"`},
			"been_here_more_than_6_months": rules.SimpleRule{Expr: `timestamp(now) - timestamp(student.EnrollmentDate) > duration("4320h")`},
			"been_here_more_than_1_hour":   rules.SimpleRule{Expr: `alsoNow - timestamp(student.EnrollmentDate) > duration("1h")`},
		},
	}

	expectedResults := map[string]bool{
		"honor_student":                true,
		"at_risk":                      false,
		"been_here_more_than_6_months": true,
		"been_here_more_than_1_hour":   true,
	}

	err := engine.AddRuleSet(ruleSet)
	if err != nil {
		t.Errorf("Error adding ruleset: %v", err)
	}

	data := map[string]interface{}{
		"student.ID":             "12312",
		"student.Age":            16,
		"student.GPA":            3.76,
		"student.Status":         "Enrolled",
		"student.Grades":         []interface{}{"A", "B", "A"},
		"student.EnrollmentDate": "2018-08-03T16:00:00-07:00",
		"now":                    "2019-08-03T16:00:00-07:00",
		"alsoNow":                &timestamp.Timestamp{Seconds: time.Now().Unix()},
	}

	results, err := engine.EvaluateAll(data, "student_actions")
	if err != nil {
		t.Errorf("Evaluation error: %v", err)
	}

	for _, v := range results {
		exp, found := expectedResults[v.RuleID]
		if !found {
			t.Errorf("Got result for rule %s, did not expect result for it", v.RuleID)
		}
		if exp != v.Pass {
			t.Errorf("Wanted true, got false for rule %s", v.RuleID)
		}
	}
}

func TestExpressionValues(t *testing.T) {

	engine := cel.NewEngine()

	schema := rules.Schema{
		ID: "my schema",
		Elements: []rules.DataElement{
			{Name: "cost", Type: rules.Float{}},
			{Name: "score", Type: rules.Int{}},
			{Name: "name", Type: rules.String{}},
			{Name: "incidentTime", Type: rules.Timestamp{}},
			{Name: "now", Type: rules.Timestamp{}},
		},
	}

	floatRule := rules.SimpleRule{Expr: "cost + 1.22"}
	intRule := rules.SimpleRule{Expr: "score * 2"}
	stringRule := rules.SimpleRule{Expr: `"My name is " + name`}
	durationRule := rules.SimpleRule{Expr: `now-incidentTime`}

	ruleSet := rules.RuleSet{
		ID:     "myset",
		Schema: schema,
		Rules: map[string]rules.Rule{
			"float":    floatRule,
			"int":      intRule,
			"string":   stringRule,
			"duration": durationRule,
		},
	}

	err := engine.AddRuleSet(ruleSet)
	if err != nil {
		t.Fatalf("Error adding ruleset: %v", err)
	}

	data := map[string]interface{}{
		"cost":         100.00,
		"score":        6,
		"name":         "Joe",
		"incidentTime": &timestamp.Timestamp{Seconds: time.Date(2020, 4, 19, 12, 10, 30, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
		"now":          &timestamp.Timestamp{Seconds: time.Date(2020, 4, 19, 13, 15, 45, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	f, err := engine.EvaluateRule(data, "myset", "float")
	if err != nil {
		t.Fatalf("Error evaluating rule %s", floatRule.Expr)
	}
	if f.Float64Value != 101.22 {
		t.Errorf("Expected %f, got %f", 101.22, f.Float64Value)
	}

	i, err := engine.EvaluateRule(data, "myset", "int")
	if err != nil {
		t.Fatalf("Error evaluating rule %s", intRule.Expr)
	}
	if i.Int64Value != 12 {
		t.Errorf("Expected %d, got %d", 12, f.Int64Value)
	}

	s, err := engine.EvaluateRule(data, "myset", "string")
	if err != nil {
		t.Fatalf("Error evaluating rule %s", stringRule.Expr)
	}
	if s.StringValue != "My name is Joe" {
		t.Errorf("Expected '%s', got '%s'", "My name is Joe", s.StringValue)
	}

	d, err := engine.EvaluateRule(data, "myset", "duration")
	if err != nil {
		t.Fatalf("Error evaluating rule %s", durationRule.Expr)
	}
	expectedDuration, err := time.ParseDuration("1h5m15s")
	if err != nil {
		t.Fatalf("Error parsing duration")
	}
	if d.Duration != expectedDuration {
		t.Errorf("Expected '%v', got '%v'", expectedDuration, d.Duration)
	}

}

// func TestAdHoc(t *testing.T) {
// 	engine := cel.NewEngine()

// 	schema := rules.Schema{
// 		ID: "my schema",
// 		Elements: []rules.DataElement{
// 			{Name: "cost", Type: rules.Float{}},
// 			{Name: "score", Type: rules.Int{}},
// 			{Name: "name", Type: rules.String{}},
// 			{Name: "incidentTime", Type: rules.Timestamp{}},
// 			{Name: "now", Type: rules.Timestamp{}},
// 		},
// 	}

// 	data := map[string]interface{}{
// 		"cost":         100.00,
// 		"score":        6,
// 		"name":         "Joe",
// 		"incidentTime": &tpb.Timestamp{Seconds: time.Date(2020, 4, 19, 12, 10, 30, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
// 		"now":          &tpb.Timestamp{Seconds: time.Date(2020, 4, 19, 13, 15, 45, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
// 	}

// 	result, err := engine.EvaluateAdHocRule(data, schema, `cost + 1.22`)
// 	if err != nil {
// 		t.Fatalf("Error evaluating rule: %v", err)
// 	}
// 	if result.Float64Value != 101.22 {
// 		t.Errorf("Expected %f, got %f", 101.22, result.Float64Value)
// 	}

// }

func TestNestedMap(t *testing.T) {

	engine := cel.NewEngine()

	schema := rules.Schema{
		ID: "my schema",
		Elements: []rules.DataElement{
			{Name: "objectType", Type: rules.String{}},
			{Name: "state", Type: rules.String{}},
			{Name: "items", Type: rules.Map{KeyType: rules.String{}, ValueType: rules.Map{KeyType: rules.String{}, ValueType: rules.String{}}}},
		},
	}

	ruleSet := rules.RuleSet{
		ID:     "myset",
		Schema: schema,
		Rules: map[string]rules.Rule{
			"1": rules.SimpleRule{Expr: `objectType == "car" && items["one"]["color"] == "green"`},
		},
	}
	err := engine.AddRuleSet(ruleSet)
	if err != nil {
		fmt.Printf("Error adding ruleset: %v", err)
	}

	data := map[string]interface{}{
		"objectType": "car",
		"items": map[string]map[string]string{
			"one": {
				"color": "green",
				"size":  "small",
			},
			"square": {
				"color": "blue",
			},
		},
	}
	results, err := engine.EvaluateAll(data, "myset")
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
	}
	for _, v := range results {
		if !v.Pass {
			t.Errorf("Expected true, got false: %s", v.RuleID)
		}
	}

}

func BenchmarkCELSimple(b *testing.B) {
	engine := cel.NewEngine()

	schema := rules.Schema{
		Elements: []rules.DataElement{
			{Name: "student.ID", Type: rules.String{}},
			{Name: "student.Age", Type: rules.Int{}},
			{Name: "student.GPA", Type: rules.Float{}},
			{Name: "student.Status", Type: rules.String{}},
			{Name: "student.Grades", Type: rules.List{ValueType: rules.String{}}},
		},
	}

	ruleSet := rules.RuleSet{
		ID:     "student_actions",
		Schema: schema,
		Rules: map[string]rules.Rule{
			"honor_student": rules.SimpleRule{Expr: `student.GPA >= 3.6 && student.Status!="Probation" && !("C" in student.Grades)`},
			//"at_risk": rules.SimpleRule{Expr: `student.GPA < 2.5 || student.Status == "Probation"`},
		},
	}

	err := engine.AddRuleSet(ruleSet)
	if err != nil {
		b.Errorf("Error adding ruleset: %v", err)
	}

	data := map[string]interface{}{
		"student.ID":     "12312",
		"student.Age":    16,
		"student.GPA":    3.76,
		"student.Status": "Enrolled",
		"student.Grades": []interface{}{"A", "B", "A"},
	}

	for i := 0; i < b.N; i++ {
		engine.EvaluateAll(data, "student_actions")
	}

}

// 	engine := cel.NewEngine()

// 	schema := rules.Schema{
// 		ID: "my schema",
// 		Elements: []rules.DataElement{
// 			{Name: "objectType", Type: rules.String{}},
// 			{Name: "state", Type: rules.String{}},
// 			{Name: "grades", Type: rules.List{ValueType: rules.Any{}}},
// 			{Name: "claims", Type: rules.Map{KeyType: rules.String{}, ValueType: rules.Any{}}},
// 		},
// 	}

// 	ruleSet := rules.RuleSet{
// 		ID:     "myset",
// 		Schema: schema,
// 		Rules: []rules.Rule{
// 			&CustomRule{expression: `objectType == "car" && (state == "X" || state == "Y")`},
// 		},
// 	}

// 	err := engine.AddRuleSet(&ruleSet)
// 	if err != nil {
// 		fmt.Printf("Error adding ruleset: %v", err)
// 	}

// 	data := map[string]interface{}{
// 		"objectType": "car",
// 		"state":      "X",
// 	}

// 	for i := 0; i < b.N; i++ {
// 		engine.EvaluateAll(data, "myset")
// 	}
// }

// func BenchmarkCELWithLists(b *testing.B) {

// 	engine := cel.NewEngine()
// 	schema := rules.Schema{
// 		ID: "my schema",
// 		Elements: []rules.DataElement{
// 			{Name: "objectType", Type: rules.String{}},
// 			{Name: "state", Type: rules.String{}},
// 			{Name: "grades", Type: rules.List{ValueType: rules.Any{}}},
// 			{Name: "claims", Type: rules.Map{KeyType: rules.String{}, ValueType: rules.Any{}}},
// 		},
// 	}

// 	ruleSet := rules.RuleSet{
// 		ID:     "myset",
// 		Schema: schema,
// 		Rules: []rules.Rule{
// 			&CustomRule{expression: `objectType == "car" && "admin" in claims.roles && "A" in grades`},
// 		},
// 	}
// 	err := engine.AddRuleSet(&ruleSet)
// 	if err != nil {
// 		fmt.Printf("Error adding ruleset: %v", err)
// 	}

// 	data := map[string]interface{}{
// 		"objectType": "car",
// 		"grades":     []interface{}{"A", "C", "D"},
// 		"claims":     map[string]interface{}{"roles": []string{"admin", ",something", "somethingelse"}},
// 	}

// 	for i := 0; i < b.N; i++ {
// 		engine.EvaluateAll(data, "myset")
// 	}
// }
