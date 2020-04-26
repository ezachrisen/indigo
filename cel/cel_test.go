package cel_test

import (
	"fmt"
	"testing"

	"github.com/ezachrisen/rules"
	"github.com/ezachrisen/rules/cel"
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
		},
	}

	ruleSet := rules.RuleSet{
		ID:     "student_actions",
		Schema: schema,
		Rules: map[string]rules.Rule{
			"honor_student": rules.SimpleRule{Expr: `student.GPA >= 3.6 && student.Status!="Probation" && !("C" in student.Grades)`},
			"at_risk":       rules.SimpleRule{Expr: `student.GPA < 2.5 || student.Status == "Probation"`},
		},
	}

	expectedResults := map[string]bool{
		"honor_student": true,
		"at_risk":       false,
	}

	err := engine.AddRuleSet(ruleSet)
	if err != nil {
		t.Errorf("Error adding ruleset: %v", err)
	}

	data := map[string]interface{}{
		"student.ID":     "12312",
		"student.Age":    16,
		"student.GPA":    3.76,
		"student.Status": "Enrolled",
		"student.Grades": []interface{}{"A", "B", "A"},
		//		"claims":         map[string]interface{}{"roles": []string{"admin", ",something", "somethingelse"}},
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
		},
	}

	floatRule := rules.SimpleRule{Expr: "cost + 1.22"}
	intRule := rules.SimpleRule{Expr: "score * 2"}
	stringRule := rules.SimpleRule{Expr: `"My name is " + name`}

	ruleSet := rules.RuleSet{
		ID:     "myset",
		Schema: schema,
		Rules: map[string]rules.Rule{
			"float":  floatRule,
			"int":    intRule,
			"string": stringRule,
		},
	}

	err := engine.AddRuleSet(ruleSet)
	if err != nil {
		t.Fatalf("Error adding ruleset: %v", err)
	}

	data := map[string]interface{}{
		"cost":  100.00,
		"score": 6,
		"name":  "Joe",
	}

	f, err := engine.EvaluateRule(data, "myset", "float")
	if err != nil {
		t.Errorf("Error evaluating rule %s", floatRule.Expr)
	}
	if f.Float64Value != 101.22 {
		t.Errorf("Expected %f, got %f", 101.22, f.Float64Value)
	}

	i, err := engine.EvaluateRule(data, "myset", "int")
	if err != nil {
		t.Errorf("Error evaluating rule %s", intRule.Expr)
	}
	if i.Int64Value != 12 {
		t.Errorf("Expected %d, got %d", 12, f.Int64Value)
	}

	s, err := engine.EvaluateRule(data, "myset", "string")
	if err != nil {
		t.Errorf("Error evaluating rule %s", stringRule.Expr)
	}
	if s.StringValue != "My name is Joe" {
		t.Errorf("Expected '%s', got '%s'", "My name is Joe", s.StringValue)
	}

}

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
			"at_risk":       rules.SimpleRule{Expr: `student.GPA < 2.5 || student.Status == "Probation"`},
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
