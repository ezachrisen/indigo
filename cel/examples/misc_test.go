package examples_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"github.com/ezachrisen/indigo/cel/examples"
	"github.com/golang/protobuf/ptypes"
)

func BenchmarkStruct(b *testing.B) {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student",
				Type: indigo.Struct{
					Struct: &examples.Student{},
					Imports: []interface{}{
						&examples.Grade{},
						&examples.StudentSummary{},
					},
				},
			},
			{Name: "now", Type: indigo.Timestamp{}},
		},
	}

	dur, err := time.ParseDuration("12h")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	data := map[string]interface{}{
		"student": examples.Student{
			GPA:            2.6,
			Age:            16,
			EnrollmentDate: time.Date(2018, 5, 10, 8, 59, 59, 0, time.Local),
			Grades:         []float64{2.0, 3.1, 3.5},
			GradeBook: []examples.Grade{
				{NumericGrade: 2.0, LetterGrade: "D"},
				{NumericGrade: 3.1, LetterGrade: "F"},
				{NumericGrade: 3.5, LetterGrade: "B+"},
			},
			Teachers: map[string]string{"Math": "Smith", "History": "Johnson"},
			Summary: examples.StudentSummary{
				ClassesTaken: 12,
				RiskFactor:   12.5,
				Tenure:       dur,
			},
		},
		"now": ptypes.TimestampNow(),
	}

	rule := indigo.Rule{
		ID:     "checks",
		Schema: education,
		Expr: `student.GPA > 2.0 && (student.Age + 1) > 16 && now - student.EnrollmentDate > duration("4320h") && student.Teachers["Math"] == "Smith" && (2.0 in student.Grades)
&& student.GradeBook.all(g, g.LetterGrade != "A") && student.Summary.ClassesTaken == 12 && student.Summary.Tenure < duration("13h")`,
	}

	ap := cel.NewAttributeProvider()
	evaluator := cel.NewEvaluator(ap)
	engine := indigo.NewEngine(evaluator)

	err = engine.AddRule(&rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	for i := 0; i < b.N; i++ {
		engine.Evaluate(data, "checks")
	}
}
