package cel_test

import (
	"testing"
	"time"

	"github.com/ezachrisen/rules/school"

	"github.com/ezachrisen/rules"
	"github.com/ezachrisen/rules/cel"
	"github.com/golang/protobuf/ptypes/timestamp"
)

// func TestConstruction(t *testing.T) {

// 	schema := rules.Schema{
// 		Elements: []rules.DataElement{
// 			{Name: "student", Type: rules.Proto{Protoname: "school.Student", Message: &school.Student{}}},
// 			{Name: "now", Type: rules.Timestamp{}},
// 		},
// 	}

// 	engine := cel.NewEngine()

// 	ruleSet := rules.RuleSet{
// 		ID:     "student_actions",
// 		Schema: schema,
// 		Rules: map[string]rules.Rule{
// 			"construct": rules.SimpleRule{Expr: `school.Student{ GPA: 1.0}`},
// 		},
// 	}

// 	err := engine.AddRuleSet(ruleSet)
// 	if err != nil {
// 		t.Errorf("Error adding ruleset: %v", err)
// 	}

// 	s := school.Student{
// 		Age:            16,
// 		GPA:            3.76,
// 		Status:         school.Student_ENROLLED,
// 		Grades:         []float64{4.0, 4.0, 3.7},
// 		Attrs:          map[string]string{"Nickname": "Joey"},
// 		EnrollmentDate: &timestamp.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
// 	}

// 	data := map[string]interface{}{
// 		"student": &s,
// 		"now":     &timestamp.Timestamp{Seconds: time.Now().Unix()},
// 	}

// 	results, err := engine.EvaluateAll(data, "student_actions")
// 	if err != nil {
// 		t.Errorf("Evaluation error: %v", err)
// 	}

// 	for _, v := range results {
// 		fmt.Println(v.RawValue)
// 	}

// }

func TestProto(t *testing.T) {

	//	pb.DefaultDb.RegisterMessage(&school.Student{})

	schema := rules.Schema{
		Elements: []rules.DataElement{
			{Name: "student", Type: rules.Proto{Protoname: "school.Student", Message: &school.Student{}}},
			{Name: "now", Type: rules.Timestamp{}},
		},
	}

	engine := cel.NewEngine()

	ruleSet := rules.RuleSet{
		ID:     "student_actions",
		Schema: schema,
		Rules: map[string]rules.Rule{
			"honor_student":     rules.SimpleRule{Expr: `student.GPA >= 3.6 && student.Status != school.Student.status_type.PROBATION && student.Grades.all(g, g>=3.0)`},
			"at_risk":           rules.SimpleRule{Expr: `student.GPA < 2.5 || student.Status == school.Student.status_type.PROBATION`},
			"tenure_gt_6months": rules.SimpleRule{Expr: `now - student.EnrollmentDate > duration("4320h")`}, // 6 months = 4320 hours
		},
	}

	expectedResults := map[string]bool{
		"honor_student":     true,
		"at_risk":           false,
		"tenure_gt_6months": true,
	}

	err := engine.AddRuleSet(ruleSet)
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
