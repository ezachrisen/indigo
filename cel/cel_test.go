package cel_test

import (
	"fmt"
	"github.com/ezachrisen/rules"
	"github.com/ezachrisen/rules/cel"
	"testing"
)

func TestSimpleCEL(t *testing.T) {

	engine := cel.NewEngine()

	schema := rules.Schema{
		ID: "my schema",
		Elements: []rules.DataElement{
			{Name: "objectType", Type: rules.String{}},
			{Name: "state", Type: rules.String{}},
			{Name: "grades", Type: rules.List{ValueType: rules.Any{}}},
			{Name: "claims", Type: rules.Map{KeyType: rules.String{}, ValueType: rules.Any{}}},
		},
	}

	ruleSet := rules.RuleSet{
		ID:     "myset",
		Schema: schema,
		Rules: []rules.Rule{
			{ID: "1", Expression: `objectType == "car" && "admin" in claims.roles && "A" in grades`},
			{ID: "2", Expression: `objectType == "xya" || ( "admin" in claims.roles && "A" in grades)`},
		},
	}

	err := engine.AddRuleSet(&ruleSet)
	if err != nil {
		fmt.Printf("Error adding ruleset: %v", err)
	}

	data := map[string]interface{}{
		"objectType": "car",
		"grades":     []interface{}{"A", "C", "D"},
		"claims":     map[string]interface{}{"roles": []string{"admin", ",something", "somethingelse"}},
	}

	results, err := engine.EvaluateAll(data, "myset")
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
	}
	for _, v := range results {
		if !v.Pass {
			t.Errorf("Expected true, got false: %s: %s", v.Rule.ID, v.Rule.Expression)
		}
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
		Rules: []rules.Rule{
			{ID: "1", Expression: `objectType == "car" && items["one"]["color"] == "green"`},
		},
	}
	err := engine.AddRuleSet(&ruleSet)
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
			t.Errorf("Expected true, got false: %s: %s", v.Rule.ID, v.Rule.Expression)
		}
	}

}

func BenchmarkCELSimple(b *testing.B) {

	engine := cel.NewEngine()

	schema := rules.Schema{
		ID: "my schema",
		Elements: []rules.DataElement{
			{Name: "objectType", Type: rules.String{}},
			{Name: "state", Type: rules.String{}},
			{Name: "grades", Type: rules.List{ValueType: rules.Any{}}},
			{Name: "claims", Type: rules.Map{KeyType: rules.String{}, ValueType: rules.Any{}}},
		},
	}

	ruleSet := rules.RuleSet{
		ID:     "myset",
		Schema: schema,
		Rules: []rules.Rule{
			{ID: "1", Expression: `objectType == "car" && (state == "X" || state == "Y")`},
		},
	}

	err := engine.AddRuleSet(&ruleSet)
	if err != nil {
		fmt.Printf("Error adding ruleset: %v", err)
	}

	data := map[string]interface{}{
		"objectType": "car",
		"state":      "X",
	}

	for i := 0; i < b.N; i++ {
		engine.EvaluateAll(data, "myset")
	}
}

func BenchmarkCELWithLists(b *testing.B) {

	engine := cel.NewEngine()
	schema := rules.Schema{
		ID: "my schema",
		Elements: []rules.DataElement{
			{Name: "objectType", Type: rules.String{}},
			{Name: "state", Type: rules.String{}},
			{Name: "grades", Type: rules.List{ValueType: rules.Any{}}},
			{Name: "claims", Type: rules.Map{KeyType: rules.String{}, ValueType: rules.Any{}}},
		},
	}

	ruleSet := rules.RuleSet{
		ID:     "myset",
		Schema: schema,
		Rules: []rules.Rule{
			{ID: "1", Expression: `objectType == "car" && "admin" in claims.roles && "A" in grades`},
		},
	}
	err := engine.AddRuleSet(&ruleSet)
	if err != nil {
		fmt.Printf("Error adding ruleset: %v", err)
	}

	data := map[string]interface{}{
		"objectType": "car",
		"grades":     []interface{}{"A", "C", "D"},
		"claims":     map[string]interface{}{"roles": []string{"admin", ",something", "somethingelse"}},
	}

	for i := 0; i < b.N; i++ {
		engine.EvaluateAll(data, "myset")
	}
}
