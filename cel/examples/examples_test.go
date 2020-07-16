// This provides *EXPERIMENTAL* support for native Go structs in CEL expressions.
// Only some features of Go structs are implemented.

package examples_test

import (
	"fmt"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"github.com/ezachrisen/indigo/cel/examples"
	"github.com/golang/protobuf/ptypes"
)

func ExampleSchool() {

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

	results, err := engine.Evaluate(data, "checks")
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	//	fmt.Println("Evaluated ", rule.Expr, "\nResult: ", results.Value)
	fmt.Println(results.Value)
	// Output: true
}

func ExampleCreateStruct() {
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

	rule := indigo.Rule{
		ID:     "makestruct",
		Schema: education,
		Expr: `examples.Student {
			GPA: 1.2,
			Age: 17,
			Grades: [2.45, 3.14],
			EnrollmentDate: timestamp("2019-03-12T12:11:20.021-04:00"),
			Teachers: {"gym":"Carlson", "math":"Johnson"},
			Summary: examples.StudentSummary{ RiskFactor: 0.2, ClassesTaken: 99, Tenure: duration("24h") },
			GradeBook: [
					examples.Grade{
							NumericGrade: 2.0,
							LetterGrade: "D"
						}
					]
			}`,
	}

	ap := cel.NewAttributeProvider()
	evaluator := cel.NewEvaluator(ap)
	engine := indigo.NewEngine(evaluator)

	err := engine.AddRule(&rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Evaluate(map[string]interface{}{}, "makestruct")
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}

	//	fmt.Println("Evaluated ", rule.Expr, "\nResult: ", results.Value)
	if snew, ok := results.Value.(examples.Student); ok {
		fmt.Printf("Got student with GPA %f, first grade is %f, the math teacher is %s, the first letter grade is  %s, has taken %d classes, %v, enrolled on %s\n",
			snew.GPA, snew.Grades[0], snew.Teachers["math"],
			snew.GradeBook[0].LetterGrade,
			snew.Summary.ClassesTaken,
			snew.Summary.Tenure,
			snew.EnrollmentDate,
		)
	} else {
		fmt.Printf("Got object of type %T, %v\n", results.Value, results.Value)
	}
	// Output: Got student with GPA 1.200000, first grade is 2.450000, the math teacher is Johnson, the first letter grade is  D, has taken 99 classes, 24h0m0s, enrolled on 2019-03-12 16:11:20.021 +0000 UTC

}
