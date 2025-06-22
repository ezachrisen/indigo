package cel_test

import (
	"context"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"github.com/ezachrisen/indigo/testdata/school"
	"github.com/google/cel-go/common/types/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func makeStudentData() map[string]any {
	return map[string]any{
		"student.ID":             "12312",
		"student.Age":            16,
		"student.GPA":            2.2,
		"student.Status":         "Enrolled",
		"student.Grades":         []any{"A", "B", "A"},
		"student.EnrollmentDate": "2018-08-03T16:00:00-07:00",
		"student.Adjustment":     2.1,
		"isSummer":               false,
		"now":                    "2019-08-03T16:00:00-07:00",
		"specificTime":           &timestamppb.Timestamp{Seconds: time.Now().Unix()},
	}
}

func makeEducationSchema() indigo.Schema {
	return indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student.ID", Type: indigo.String{}},
			{Name: "student.Age", Type: indigo.Int{}},
			{Name: "student.GPA", Type: indigo.Float{}},
			{Name: "student.Adjustment", Type: indigo.Float{}},
			{Name: "student.Status", Type: indigo.String{}},
			{Name: "student.Grades", Type: indigo.List{ValueType: indigo.String{}}},
			{Name: "student.EnrollmentDate", Type: indigo.String{}},
			{Name: "now", Type: indigo.String{}},
			{Name: "alsoNow", Type: indigo.Timestamp{}},
			{Name: "isSummer", Type: indigo.Bool{}},
		},
	}
}

func makeEducationRules1() *indigo.Rule {
	rule1 := &indigo.Rule{
		ID:     "student_actions",
		Meta:   "d04ab6d9-f59d-9474-5c38-34d65380c612",
		Schema: makeEducationSchema(),
		Rules: map[string]*indigo.Rule{
			"honors_student": {
				ID:         "honors_student",
				Expr:       `student.GPA >= 3.6 && student.Status!="Probation" && !("C" in student.Grades)`,
				ResultType: indigo.Bool{},
				Schema:     makeEducationSchema(),
			},
			"at_risk": {
				ID: "at_risk",
				//				Expr:       `student.GPA < 2.5 || student.Status == "Probation"`,
				Expr:       `student.GPA < 2.5 || student.Status == "Probation"`,
				Schema:     makeEducationSchema(),
				ResultType: indigo.Bool{},
				Rules: map[string]*indigo.Rule{
					"risk_factor": {
						ID:         "risk_factor",
						Expr:       `2.0+6.0`,
						ResultType: indigo.Float{},
					},
				},
			},
			"notsummer": {
				ID:     "notsummer",
				Expr:   "isSummer && student.GPA < 2.5",
				Schema: makeEducationSchema(),
			},
			"timecheck": {
				ID:     "timecheck",
				Expr:   "now > student.EnrollmentDate",
				Schema: makeEducationSchema(),
			},
		},
	}

	rule2 := &indigo.Rule{
		ID:     "depthRules",
		Schema: makeEducationSchema(),
		Expr:   `student.GPA > 3.5`, // false
		Rules: map[string]*indigo.Rule{
			"a": {
				ID:     "c1",
				Expr:   `student.Adjustment > 0.0`, // true
				Schema: makeEducationSchema(),
			},
			"b": {
				ID:     "c2",
				Expr:   `student.Adjustment > 3.0`, // false
				Schema: makeEducationSchema(),
			},
			"c": {
				ID:     "c3",
				Expr:   `student.Adjustment < 2.6`, // true
				Schema: makeEducationSchema(),
			},
			"d": {
				ID:     "c4",
				Expr:   `student.Adjustment > 3.0`, // false
				Schema: makeEducationSchema(),
			},
		},
	}

	rule3 := &indigo.Rule{
		ID:     "ruleOptions",
		Schema: makeEducationSchema(),
		Expr:   `student.GPA > 3.5`, // false
		Rules: map[string]*indigo.Rule{
			"A": {
				ID:          "D",
				Expr:        `student.Adjustment > 0.0`, // true
				EvalOptions: indigo.EvalOptions{StopFirstPositiveChild: true},
				Schema:      makeEducationSchema(),
				Rules: map[string]*indigo.Rule{
					"d1": {
						ID:     "d1",
						Expr:   `student.Adjustment < 2.6`, // true
						Schema: makeEducationSchema(),
					},
					"d2": {
						ID:     "d2",
						Expr:   `student.Adjustment > 3.0`, // false
						Schema: makeEducationSchema(),
					},
					"d3": {
						ID:     "d3",
						Expr:   `student.Adjustment < 2.6`, // true
						Schema: makeEducationSchema(),
					},
				},
			},
			"B": {
				ID:     "b1",
				Expr:   `student.Adjustment > 3.0`, // false
				Schema: makeEducationSchema(),
			},
			"E": {
				ID:     "E",
				Expr:   `student.Adjustment > 0.0`, // true
				Schema: makeEducationSchema(),
				Rules: map[string]*indigo.Rule{
					"e1": {
						ID:     "e1",
						Expr:   `student.Adjustment < 2.6`, // true
						Schema: makeEducationSchema(),
					},
					"e2": {
						ID:     "e2",
						Expr:   `student.Adjustment > 3.0`, // false
						Schema: makeEducationSchema(),
					},
					"e3": {
						ID:     "e3",
						Expr:   `student.Adjustment < 2.6`, // true
						Schema: makeEducationSchema(),
					},
				},
			},
		},
	}

	root := indigo.NewRule("root", "")

	root.Rules[rule1.ID] = rule1
	root.Rules[rule2.ID] = rule2
	root.Rules[rule3.ID] = rule3
	return root
}

func makeEducationRulesWithIncorrectTypes() *indigo.Rule {
	rule1 := &indigo.Rule{
		ID:     "student_actions",
		Meta:   "d04ab6d9-f59d-9474-5c38-34d65380c612",
		Schema: makeEducationSchema(),
		Rules: map[string]*indigo.Rule{
			"a": {
				ID:         "honors_student",
				Expr:       `student.GPA != "3.6" && student.Status > 2.0`,
				ResultType: indigo.Bool{},
				Schema:     makeEducationSchema(),
			},
		},
	}
	return rule1
}

func makeEducationProtoSchema() indigo.Schema {
	return indigo.Schema{
		ID: "educationProto",
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
			{Name: "honors", Type: indigo.Proto{Message: &school.HonorsConfiguration{}}},
		},
	}
}

func makeEducationProtoRules(id string) *indigo.Rule {
	return &indigo.Rule{
		ID:     id,
		Schema: makeEducationProtoSchema(),
		Rules: map[string]*indigo.Rule{
			"honor_student": {
				ID:     "honor_student",
				Expr:   `student.gpa >= honors.Minimum_GPA && student.status != testdata.school.Student.status_type.PROBATION && student.grades.all(g, g>=3.0)`,
				Meta:   true,
				Schema: makeEducationProtoSchema(),
			},
			"at_risk": {
				ID:     "at_risk",
				Expr:   `student.gpa < 2.5 || student.status == testdata.school.Student.status_type.PROBATION`,
				Meta:   false,
				Schema: makeEducationProtoSchema(),
			},
			"tenure_gt_6months": {
				ID:     "tenure_gt_6months",
				Expr:   `now - student.enrollment_date > duration("4320h")`, // 6 months = 4320 hours
				Meta:   true,
				Schema: makeEducationProtoSchema(),
			},
		},
	}
}

func makeStudentProtoData() map[string]any {
	s := school.Student{
		Age:            16,
		Gpa:            3.76,
		Status:         school.Student_ENROLLED,
		Grades:         []float64{4.0, 4.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamppb.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	return map[string]any{
		"student": &s,
		"now":     &timestamppb.Timestamp{Seconds: time.Now().Unix()},
		"honors":  &school.HonorsConfiguration{Minimum_GPA: 3.7},
	}
}

func TestBasicRules(t *testing.T) {
	e := indigo.NewEngine(cel.NewEvaluator())
	r := makeEducationRules1()
	err := e.Compile(r, indigo.CollectDiagnostics(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sa := r.Rules["student_actions"]

	results, err := e.Eval(context.Background(), sa, makeStudentData(), indigo.ReturnDiagnostics(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results.Rule != sa {
		t.Errorf("expected %v, got %v", sa, results.Rule)
	}
	if !results.ExpressionPass {
		t.Error("condition should be true")
	}
	if results.Results["honors_student"].ExpressionPass {
		t.Error("condition should be true")
	}
	if !results.Results["at_risk"].ExpressionPass {
		t.Error("condition should be true")
	}
	if results.Results["at_risk"].Results["risk_factor"].Value.(float64) != 8.0 {
		t.Errorf("expected %v, got %v", 8.0, results.Results["at_risk"].Results["risk_factor"].Value.(float64))
	}
}

// Make sure that type mismatches between schema and rule are caught at compile time
func TestCompileErrors(t *testing.T) {
	e := indigo.NewEngine(cel.NewEvaluator())
	r := makeEducationRulesWithIncorrectTypes()

	err := e.Compile(r)
	if err == nil {
		t.Fatal("test should fail") // expected compile error here
	}
	if !strings.Contains(err.Error(), "1:13: found no matching overload for '_!=_' applied to '(double, string)'") {
		t.Error("condition should be true")
	}
	if !strings.Contains(err.Error(), "1:40: found no matching overload for '_>_' applied to '(string, double)'") {
		t.Error("condition should be true")
	}
}

func TestProtoMessage(t *testing.T) {
	e := indigo.NewEngine(cel.NewEvaluator())

	r := makeEducationProtoRules("student_actions")
	err := e.Compile(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results, err := e.Eval(context.Background(), r, makeStudentProtoData())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results.Results) != 3 {
		t.Errorf("expected %v, got %v", 3, len(results.Results))
	}
	for _, v := range results.Results {
		if v.Rule.Meta != v.ExpressionPass {
			t.Errorf("expected %v, got %v", v.Rule.Meta, v.ExpressionPass)
		}
	}
}

func TestDiagnosticOptions(t *testing.T) {
	e := indigo.NewEngine(cel.NewEvaluator())
	r2 := makeEducationProtoRules("student_actions")
	err := e.Compile(r2, indigo.CollectDiagnostics(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := e.Eval(context.Background(), r2, makeStudentProtoData(), indigo.ReturnDiagnostics(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	//	fmt.Println(r2)
	//	fmt.Println(indigo.DiagnosticsReport(u, makeStudentProtoData()))
	//	fmt.Println(u.Diagnostics.String())
	for _, c := range u.Results {
		if c.Diagnostics == nil {
			t.Errorf("Wanted diagnostics for rule %s", c.Rule.ID)
		} // else {
		// 	fmt.Println(u)
		// 	fmt.Println(c.Diagnostics)
		// }
	}
}

func TestDiagnosticsWithEmptyRule(t *testing.T) {
	e := indigo.NewEngine(cel.NewEvaluator())
	d := map[string]any{"a": "a"} // dummy data, not important

	r := &indigo.Rule{
		ID: "root",
	}

	err := e.Compile(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := e.Eval(context.Background(), r, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Diagnostics != nil {
		t.Errorf("expected %v, got %v", nil, u.Diagnostics)
	}
}

func TestRuleResultTypes(t *testing.T) {
	cases := []struct {
		rule indigo.Rule
		err  error
	}{
		{
			indigo.Rule{
				ID:         "shouldBeBool",
				Schema:     makeEducationProtoSchema(),
				ResultType: indigo.Bool{},
				Expr:       `student.gpa >= 3.6`,
			},
			nil,
		},
		{
			indigo.Rule{
				ID:         "shouldBeBFloat",
				Schema:     makeEducationProtoSchema(),
				ResultType: indigo.Float{},
				Expr:       `student.gpa + 1.0`,
			},
			nil,
		},
		{
			indigo.Rule{
				ID:         "shouldBeInt",
				Schema:     makeEducationProtoSchema(),
				ResultType: indigo.Int{},
				Expr:       `4`,
			},
			nil,
		},
		{
			indigo.Rule{
				ID:         "shouldBeIntList",
				Schema:     makeEducationProtoSchema(),
				ResultType: indigo.List{ValueType: indigo.Int{}},
				Expr:       `[1,2,3]`,
			},
			nil,
		},
		{
			indigo.Rule{
				ID:         "shouldBeAnyList",
				Schema:     makeEducationProtoSchema(),
				ResultType: indigo.List{ValueType: indigo.Any{}},
				Expr:       `[1,"2",3]`,
			},
			nil,
		},
		{
			indigo.Rule{
				ID:         "shouldBeList",
				Schema:     makeEducationProtoSchema(),
				ResultType: indigo.List{ValueType: indigo.Float{}},
				Expr:       `student.grades`,
			},
			nil,
		},
		{
			indigo.Rule{
				ID:         "shouldBeMap",
				Schema:     makeEducationProtoSchema(),
				ResultType: indigo.Map{KeyType: indigo.String{}, ValueType: indigo.Int{}},
				Expr:       `{'blue': 1, 'red': 2}`,
			},
			nil,
		},

		{
			indigo.Rule{
				ID:         "shouldBeStudent",
				Schema:     makeEducationProtoSchema(),
				ResultType: indigo.Proto{Message: &school.Student{}},
				Expr:       `testdata.school.Student { gpa: 1.2 }`,
			},
			nil,
		},
		{
			indigo.Rule{
				ID:         "shouldBeTimestamp",
				Schema:     makeEducationProtoSchema(),
				ResultType: indigo.Timestamp{},
				Expr:       `now`,
			},
			nil,
		},
		{
			indigo.Rule{
				ID:         "shouldBeDuration",
				Schema:     makeEducationProtoSchema(),
				ResultType: indigo.Duration{},
				Expr:       `duration("12h")`,
			},
			nil,
		},

		{
			indigo.Rule{
				ID:         "NEGATIVEshouldBeBFloat",
				Schema:     makeEducationProtoSchema(),
				ResultType: indigo.Bool{},
				Expr:       `student.gpa + 1.0`,
			},
			fmt.Errorf("Should be an error"),
		},
		{
			indigo.Rule{
				ID:         "NEGATIVEshouldBeStudent",
				Schema:     makeEducationProtoSchema(),
				ResultType: indigo.Proto{Message: &school.HonorsConfiguration{}},
				Expr:       `testdata.school.Student { gpa: 1.2 }`,
			},
			fmt.Errorf("Should be an error"),
		},
	}

	for i := range cases {
		for _, b := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s_fixed_schema_%t", cases[i].rule.ID, b), func(t *testing.T) {
				var e *indigo.DefaultEngine
				switch b {
				case true:
					s := makeEducationProtoSchema()
					e = indigo.NewEngine(cel.NewEvaluator(cel.FixedSchema(&s)))
				case false:
					e = indigo.NewEngine(cel.NewEvaluator())
				}

				//		lc := c
				//		fmt.Println("Case: ", cases[i].rule.ID)
				err := e.Compile(&cases[i].rule)
				if cases[i].err == nil && err != nil {
					t.Errorf("compiling rule %s, wanted err = %v, got %v", cases[i].rule.ID, cases[i].err, err)
				}

				_, err = e.Eval(context.Background(), &cases[i].rule, makeStudentProtoData())

				if err != nil && cases[i].err == nil {
					t.Fatalf("For rule %s, wanted err = %v, got %v", cases[i].rule.ID, cases[i].err, err)
				} /* else {
						if res != nil {
						 	fmt.Printf("%v\nn", res)
						 }
				} */
			})
		}
	}
}

// Generate all diagnostics for both sets of rules to make sure
// no panics
func TestDiagnosticGeneration(t *testing.T) {
	e := indigo.NewEngine(cel.NewEvaluator())

	red1 := makeEducationRules1()
	err := e.Compile(red1, indigo.CollectDiagnostics(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := e.Eval(context.Background(), red1, makeStudentData(), indigo.ReturnDiagnostics(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	//	fmt.Println(indigo.DiagnosticsReport(u, nil))
	_ = u.Diagnostics.String()

	red2 := makeEducationProtoRules("red2")
	err = e.Compile(red2, indigo.CollectDiagnostics(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	u, err = e.Eval(context.Background(), red2, makeStudentProtoData(), indigo.ReturnDiagnostics(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = u.Diagnostics.String()
	_ = indigo.DiagnosticsReport(u, makeStudentData())
}

// ------------------------------------------------------------------------------------------
// BENCHMARKS
//
//
//
//
//

func BenchmarkSimpleRule(b *testing.B) {
	education := makeEducationSchema()
	data := makeStudentData()

	r := indigo.Rule{
		ID:     "student_actions",
		Schema: education,
		Rules: map[string]*indigo.Rule{
			"a": {
				ID:     "at_risk",
				Schema: education,
				Expr:   `student.GPA < 2.5 || student.Status == "Probation"`,
			},
		},
	}
	e := indigo.NewEngine(cel.NewEvaluator())
	err := e.Compile(&r)
	if err != nil {
		b.Errorf("Error adding ruleset: %v", err)
	}

	for i := 0; i < b.N; i++ {
		_, err := e.Eval(context.Background(), &r, data)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkSimpleRuleWithDiagnostics(b *testing.B) {
	e := indigo.NewEngine(cel.NewEvaluator())
	education := makeEducationSchema()
	data := makeStudentData()

	r := indigo.Rule{
		ID:     "student_actions",
		Schema: education,
		Rules: map[string]*indigo.Rule{
			"a": {
				ID:     "at_risk",
				Schema: education,
				Expr:   `student.GPA < 2.5 || student.Status == "Probation"`,
			},
		},
	}

	err := e.Compile(&r, indigo.CollectDiagnostics(true))
	if err != nil {
		b.Errorf("Error adding ruleset: %v", err)
	}

	for i := 0; i < b.N; i++ {
		_, err := e.Eval(context.Background(), &r, data, indigo.ReturnDiagnostics(true))
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkRuleWithArray(b *testing.B) {
	e := indigo.NewEngine(cel.NewEvaluator())

	education := makeEducationSchema()

	r := &indigo.Rule{
		ID:     "student_actions",
		Schema: education,
		Rules: map[string]*indigo.Rule{
			"a": {
				ID:     "honors_student",
				Schema: education,
				Expr:   `student.GPA >= 3.6 && student.Status!="Probation" && !("C" in student.Grades)`,
			},
		},
	}

	err := e.Compile(r)
	if err != nil {
		b.Errorf("Error adding ruleset: %v", err)
	}

	data := makeStudentData()
	for i := 0; i < b.N; i++ {
		_, err := e.Eval(context.Background(), r, data)
		if err != nil {
			b.Error(err)
		}

	}
}

func BenchmarkProtoWithHonorsX(b *testing.B) {
	b.StopTimer()

	_, err := pb.DefaultDb.RegisterMessage(&school.Student{})
	if err != nil {
		b.Error(err)
	}

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
			{Name: "honors", Type: indigo.Proto{Message: &school.HonorsConfiguration{}}},
		},
	}

	e := indigo.NewEngine(cel.NewEvaluator())

	r := &indigo.Rule{
		ID:     "student_actions",
		Schema: schema,
		Rules: map[string]*indigo.Rule{
			"a": {
				ID:     "at_risk",
				Expr:   `student.gpa < honors.Minimum_GPA && student.status == testdata.school.Student.status_type.PROBATION`,
				Schema: schema,
				Meta:   false,
			},
		},
	}

	err = e.Compile(r)
	if err != nil {
		log.Fatalf("Error adding ruleset: %v", err)
	}

	s := school.Student{
		Age:            16,
		Gpa:            3,
		Status:         school.Student_PROBATION,
		Grades:         []float64{4.0, 4.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamppb.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	data := map[string]any{
		"student": &s,
		"now":     &timestamppb.Timestamp{Seconds: time.Now().Unix()},
		"honors":  &school.HonorsConfiguration{Minimum_GPA: 3.7},
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_, err := e.Eval(context.Background(), r, data)
		if err != nil {
			b.Error(err)
		}

	}
}

func BenchmarkProtoWithoutHonors(b *testing.B) {
	_, err := pb.DefaultDb.RegisterMessage(&school.Student{})
	if err != nil {
		b.Error(err)
	}

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
		},
	}
	e := indigo.NewEngine(cel.NewEvaluator())

	r := &indigo.Rule{
		ID:     "student_actions",
		Schema: schema,
		Rules: map[string]*indigo.Rule{
			"a": {
				ID:     "at_risk",
				Expr:   `student.gpa < 2.5 || student.status == testdata.school.Student.status_type.PROBATION`,
				Schema: schema,
				Meta:   false,
			},
		},
	}

	err = e.Compile(r)
	if err != nil {
		log.Fatalf("Error adding ruleset: %v", err)
	}

	s := school.Student{
		Age:            16,
		Gpa:            3.76,
		Status:         school.Student_ENROLLED,
		Grades:         []float64{4.0, 4.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamppb.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	data := map[string]any{
		"student": &s,
		"now":     &timestamppb.Timestamp{Seconds: time.Now().Unix()},
	}

	for i := 0; i < b.N; i++ {
		_, err := e.Eval(context.Background(), r, data)
		if err != nil {
			b.Error(err)
		}

	}
}

func BenchmarkProtoCreation(b *testing.B) {
	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "student_suspension", Type: indigo.Proto{Message: &school.Student_Suspension{}}},
			{Name: "studentSummary", Type: indigo.Proto{Message: &school.StudentSummary{}}},
		},
	}

	r := &indigo.Rule{
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

	e := indigo.NewEngine(cel.NewEvaluator())
	err := e.Compile(r)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	for i := 0; i < b.N; i++ {
		_, err := e.Eval(context.Background(), r, makeStudentProtoData())
		if err != nil {
			b.Error(err)
		}

	}
}

func BenchmarkEval2000Rules(b *testing.B) {
	b.StopTimer()
	_, err := pb.DefaultDb.RegisterMessage(&school.Student{})
	if err != nil {
		b.Error(err)
	}

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
			{Name: "honors", Type: indigo.Proto{Message: &school.HonorsConfiguration{}}},
		},
	}

	e := indigo.NewEngine(cel.NewEvaluator())

	r := &indigo.Rule{
		ID:     "student_actions",
		Schema: schema,
		Rules:  map[string]*indigo.Rule{},
	}

	for i := 0; i < 2_000; i++ {
		cr := &indigo.Rule{
			ID:     fmt.Sprintf("at_risk_%d", i),
			Expr:   `student.gpa < honors.Minimum_GPA && student.status == testdata.school.Student.status_type.PROBATION`,
			Schema: schema,
			Meta:   false,
		}
		r.Rules[cr.ID] = cr
	}

	err = e.Compile(r)
	if err != nil {
		log.Fatalf("Error adding ruleset: %v", err)
	}

	s := school.Student{
		Age:            16,
		Gpa:            3,
		Status:         school.Student_PROBATION,
		Grades:         []float64{2.0, 2.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamppb.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	data := map[string]any{
		"student": &s,
		"now":     &timestamppb.Timestamp{Seconds: time.Now().Unix()},
		"honors":  &school.HonorsConfiguration{Minimum_GPA: 3.7},
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := e.Eval(context.Background(), r, data)
		if err != nil {
			b.Error(err)
		}

	}
}

func BenchmarkEval2000WithSelfRules(b *testing.B) {
	b.StopTimer()
	_, err := pb.DefaultDb.RegisterMessage(&school.Student{})
	if err != nil {
		b.Error(err)
	}

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
			{Name: "honors", Type: indigo.Proto{Message: &school.HonorsConfiguration{}}},
			{Name: "self", Type: indigo.Int{}},
		},
	}

	e := indigo.NewEngine(cel.NewEvaluator())

	r := &indigo.Rule{
		ID:     "student_actions",
		Schema: schema,
		Rules:  map[string]*indigo.Rule{},
	}

	for i := 0; i < 2_000; i++ {
		cr := &indigo.Rule{
			ID:     fmt.Sprintf("at_risk_%d", i),
			Expr:   `student.gpa < honors.Minimum_GPA && student.status == testdata.school.Student.status_type.PROBATION && self == 42`,
			Schema: schema,
			Meta:   false,
			Self:   42,
		}
		r.Rules[cr.ID] = cr
	}

	err = e.Compile(r)
	if err != nil {
		log.Fatalf("Error adding ruleset: %v", err)
	}

	s := school.Student{
		Age:            16,
		Gpa:            3,
		Status:         school.Student_PROBATION,
		Grades:         []float64{2.0, 2.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamppb.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	data := map[string]any{
		"student": &s,
		"now":     &timestamppb.Timestamp{Seconds: time.Now().Unix()},
		"honors":  &school.HonorsConfiguration{Minimum_GPA: 3.7},
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := e.Eval(context.Background(), r, data)
		if err != nil {
			b.Error(err)
		}

	}
}

func BenchmarkEval2000RulesWithSelfParallel(b *testing.B) {
	b.StopTimer()
	_, err := pb.DefaultDb.RegisterMessage(&school.Student{})
	if err != nil {
		b.Error(err)
	}

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
			{Name: "honors", Type: indigo.Proto{Message: &school.HonorsConfiguration{}}},
			{Name: "self", Type: indigo.Int{}},
		},
	}

	e := indigo.NewEngine(cel.NewEvaluator())

	r := &indigo.Rule{
		ID:     "student_actions",
		Schema: schema,
		Rules:  map[string]*indigo.Rule{},
	}

	for i := 0; i < 2_000; i++ {
		cr := &indigo.Rule{
			ID:     fmt.Sprintf("at_risk_%d", i),
			Expr:   `student.gpa < honors.Minimum_GPA && student.status == testdata.school.Student.status_type.PROBATION && self == 42`,
			Schema: schema,
			Meta:   false,
			Self:   42,
		}
		r.Rules[cr.ID] = cr
	}

	err = e.Compile(r)
	if err != nil {
		log.Fatalf("Error adding ruleset: %v", err)
	}

	s := school.Student{
		Age:            16,
		Gpa:            3,
		Status:         school.Student_PROBATION,
		Grades:         []float64{2.0, 2.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamppb.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	data := map[string]any{
		"student": &s,
		"now":     &timestamppb.Timestamp{Seconds: time.Now().Unix()},
		"honors":  &school.HonorsConfiguration{Minimum_GPA: 3.7},
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err = e.Eval(context.Background(), r, data, indigo.Parallel(200, 10))
		if err != nil {
			b.Error(err)
		}

	}
}

func BenchmarkEval2000RulesParallel(b *testing.B) {
	b.StopTimer()
	_, err := pb.DefaultDb.RegisterMessage(&school.Student{})
	if err != nil {
		b.Error(err)
	}

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
			{Name: "honors", Type: indigo.Proto{Message: &school.HonorsConfiguration{}}},
		},
	}

	e := indigo.NewEngine(cel.NewEvaluator())

	r := &indigo.Rule{
		ID:     "student_actions",
		Schema: schema,
		Rules:  map[string]*indigo.Rule{},
	}

	for i := 0; i < 2_000; i++ {
		cr := &indigo.Rule{
			ID:     fmt.Sprintf("at_risk_%d", i),
			Expr:   `student.gpa < honors.Minimum_GPA && student.status == testdata.school.Student.status_type.PROBATION`,
			Schema: schema,
			Meta:   false,
		}
		r.Rules[cr.ID] = cr
	}

	err = e.Compile(r)
	if err != nil {
		log.Fatalf("Error adding ruleset: %v", err)
	}

	s := school.Student{
		Age:            16,
		Gpa:            3,
		Status:         school.Student_PROBATION,
		Grades:         []float64{2.0, 2.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamppb.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	data := map[string]any{
		"student": &s,
		"now":     &timestamppb.Timestamp{Seconds: time.Now().Unix()},
		"honors":  &school.HonorsConfiguration{Minimum_GPA: 3.7},
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err = e.Eval(context.Background(), r, data, indigo.Parallel(200, 10))
		if err != nil {
			b.Error(err)
		}

	}
}

func BenchmarkEval2000RulesWithSort(b *testing.B) {
	b.StopTimer()
	_, err := pb.DefaultDb.RegisterMessage(&school.Student{})
	if err != nil {
		b.Error(err)
	}

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
			{Name: "honors", Type: indigo.Proto{Message: &school.HonorsConfiguration{}}},
		},
	}

	e := indigo.NewEngine(cel.NewEvaluator())

	r := &indigo.Rule{
		ID:     "student_actions",
		Schema: schema,
		Rules:  map[string]*indigo.Rule{},
	}
	r.EvalOptions.SortFunc = indigo.SortRulesAlpha // <-- we'll sort all 2000 child rules alphabetically

	for i := 0; i < 2_000; i++ {
		cr := &indigo.Rule{
			ID:     fmt.Sprintf("at_risk_%d", i),
			Expr:   `student.gpa < honors.Minimum_GPA && student.status == testdata.school.Student.status_type.PROBATION`,
			Schema: schema,
			Meta:   false,
		}
		r.Rules[cr.ID] = cr
	}

	err = e.Compile(r)
	if err != nil {
		log.Fatalf("Error adding ruleset: %v", err)
	}

	s := school.Student{
		Age:            16,
		Gpa:            3,
		Status:         school.Student_PROBATION,
		Grades:         []float64{2.0, 2.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamppb.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	data := map[string]any{
		"student": &s,
		"now":     &timestamppb.Timestamp{Seconds: time.Now().Unix()},
		"honors":  &school.HonorsConfiguration{Minimum_GPA: 3.7},
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := e.Eval(context.Background(), r, data)
		if err != nil {
			b.Error(err)
		}

	}
}

func BenchmarkCompileRule(b *testing.B) {
	b.StopTimer()
	e := indigo.NewEngine(cel.NewEvaluator())
	r := makeEducationProtoRules("rule")
	b.StartTimer()
	for i := 1; i < b.N; i++ {
		err := e.Compile(r)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkCompileRuleWithFixedSchema(b *testing.B) {
	b.StopTimer()
	r := makeEducationProtoRules("rule")
	e := indigo.NewEngine(cel.NewEvaluator(cel.FixedSchema(&r.Schema)))
	b.StartTimer()
	for i := 1; i < b.N; i++ {
		err := e.Compile(r)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestEval2000RulesParallel(t *testing.T) {

	_, err := pb.DefaultDb.RegisterMessage(&school.Student{})
	if err != nil {
		t.Error(err)
	}

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
			{Name: "honors", Type: indigo.Proto{Message: &school.HonorsConfiguration{}}},
		},
	}

	e := indigo.NewEngine(cel.NewEvaluator())

	r := &indigo.Rule{
		ID:     "student_actions",
		Schema: schema,
		Rules:  map[string]*indigo.Rule{},
	}

	for i := 0; i < 2_000; i++ {
		cr := &indigo.Rule{
			ID:     fmt.Sprintf("at_risk_%d", i),
			Expr:   `student.gpa < honors.Minimum_GPA && student.status == testdata.school.Student.status_type.PROBATION`,
			Schema: schema,
			Meta:   false,
		}
		r.Rules[cr.ID] = cr
	}

	err = e.Compile(r)
	if err != nil {
		log.Fatalf("Error adding ruleset: %v", err)
	}

	s := school.Student{
		Age:            16,
		Gpa:            3,
		Status:         school.Student_PROBATION,
		Grades:         []float64{2.0, 2.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamppb.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	data := map[string]any{
		"student": &s,
		"now":     &timestamppb.Timestamp{Seconds: time.Now().Unix()},
		"honors":  &school.HonorsConfiguration{Minimum_GPA: 3.7},
	}
	//	start := time.Now()
	_, err = e.Eval(context.Background(), r, data, indigo.Parallel(200, 10))
	if err != nil {
		t.Error(err)
	}
	//	fmt.Println("Took ", time.Since(start))

}

func TestEval2000Rules(t *testing.T) {

	_, err := pb.DefaultDb.RegisterMessage(&school.Student{})
	if err != nil {
		t.Error(err)
	}

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
			{Name: "honors", Type: indigo.Proto{Message: &school.HonorsConfiguration{}}},
		},
	}

	e := indigo.NewEngine(cel.NewEvaluator())

	r := &indigo.Rule{
		ID:     "student_actions",
		Schema: schema,
		Rules:  map[string]*indigo.Rule{},
	}

	for i := 0; i < 2_000; i++ {
		cr := &indigo.Rule{
			ID:     fmt.Sprintf("at_risk_%d", i),
			Expr:   `student.gpa < honors.Minimum_GPA && student.status == testdata.school.Student.status_type.PROBATION`,
			Schema: schema,
			Meta:   false,
		}
		r.Rules[cr.ID] = cr
	}

	err = e.Compile(r)
	if err != nil {
		log.Fatalf("Error adding ruleset: %v", err)
	}

	s := school.Student{
		Age:            16,
		Gpa:            3,
		Status:         school.Student_PROBATION,
		Grades:         []float64{2.0, 2.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamppb.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	data := map[string]any{
		"student": &s,
		"now":     &timestamppb.Timestamp{Seconds: time.Now().Unix()},
		"honors":  &school.HonorsConfiguration{Minimum_GPA: 3.7},
	}
	//	start := time.Now()
	_, err = e.Eval(context.Background(), r, data)
	if err != nil {
		t.Error(err)
	}
	//	fmt.Println("Took ", time.Since(start))

}

func TestRuleSelfFunctionality(t *testing.T) {
	e := indigo.NewEngine(cel.NewEvaluator())

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "score", Type: indigo.Int{}},
			{Name: "category", Type: indigo.String{}},
			{Name: "self", Type: indigo.Any{}},
		},
	}

	testCases := []struct {
		name     string
		parallel bool
	}{
		{"sequential", false},
		{"parallel", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &indigo.Rule{
				ID:     "root",
				Schema: schema,
				Expr:   "true",
				Self: map[string]any{
					"priority": 10,
					"category": "high",
					"weight":   2.5,
				},
				Rules: map[string]*indigo.Rule{
					"priority_check": {
						ID:     "priority_check",
						Schema: schema,
						Expr:   `self.priority > 5`,
						Self: map[string]any{
							"priority": 8,
							"category": "medium",
							"weight":   1.8,
						},
					},
					"category_check": {
						ID:     "category_check",
						Schema: schema,
						Expr:   `self.category == "high" && score > 90`,
						Self: map[string]any{
							"priority": 15,
							"category": "high",
							"weight":   3.0,
						},
					},
					"weight_check": {
						ID:     "weight_check",
						Schema: schema,
						Expr:   `self.weight * double(score) > 150.0`,
						Self: map[string]any{
							"priority": 5,
							"category": "low",
							"weight":   2.0,
						},
					},
					"no_self": {
						ID:     "no_self",
						Schema: schema,
						Expr:   `score > 80`,
					},
				},
			}

			err := e.Compile(r)
			if err != nil {
				t.Fatalf("compilation failed: %v", err)
			}

			data := map[string]any{
				"score":    95,
				"category": "premium",
			}

			var result *indigo.Result
			if tc.parallel {
				result, err = e.Eval(context.Background(), r, data, indigo.Parallel(2, 2))
			} else {
				result, err = e.Eval(context.Background(), r, data)
			}

			if err != nil {
				t.Fatalf("evaluation failed: %v", err)
			}

			if !result.Pass {
				t.Error("root rule should pass")
			}

			if !result.Results["priority_check"].Pass {
				t.Error("priority_check should pass (self.priority=8 > 5)")
			}

			if !result.Results["category_check"].Pass {
				t.Error("category_check should pass (self.category='high' and score=95 > 90)")
			}

			if !result.Results["weight_check"].Pass {
				t.Error("weight_check should pass (self.weight=2.0 * score=95 = 190 > 150)")
			}

			if !result.Results["no_self"].Pass {
				t.Error("no_self should pass (score=95 > 80)")
			}

			if len(result.Results) != 4 {
				t.Errorf("expected 4 child results, got %d", len(result.Results))
			}
		})
	}
}

func TestRuleSelfInheritance(t *testing.T) {
	e := indigo.NewEngine(cel.NewEvaluator())

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "value", Type: indigo.Int{}},
			{Name: "self", Type: indigo.Any{}},
		},
	}

	testCases := []struct {
		name     string
		parallel bool
	}{
		{"sequential_inheritance", false},
		{"parallel_inheritance", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &indigo.Rule{
				ID:     "parent",
				Schema: schema,
				Expr:   "true",
				Self: map[string]any{
					"threshold": 50,
					"mode":      "strict",
				},
				Rules: map[string]*indigo.Rule{
					"child_with_self": {
						ID:     "child_with_self",
						Schema: schema,
						Expr:   `self.mode == "strict"`,
						Self: map[string]any{
							"threshold": 75,
							"mode":      "strict",
						},
					},
					"child_no_self": {
						ID:     "child_no_self",
						Schema: schema,
						Expr:   `value > 50`,
						// No Self field - should not inherit parent's self
					},
					"grandchild_test": {
						ID:     "grandchild_test",
						Schema: schema,
						Expr:   `value > self.threshold`,
						Self: map[string]any{
							"threshold": 30,
							"mode":      "relaxed",
						},
					},
				},
			}

			err := e.Compile(r)
			if err != nil {
				t.Fatalf("compilation failed: %v", err)
			}

			data := map[string]any{
				"value": 60,
			}

			var result *indigo.Result
			if tc.parallel {
				result, err = e.Eval(context.Background(), r, data, indigo.Parallel(2, 2))
			} else {
				result, err = e.Eval(context.Background(), r, data)
			}

			if err != nil {
				t.Fatalf("evaluation failed: %v", err)
			}

			if !result.Pass {
				t.Error("parent rule should pass")
			}

			if !result.Results["child_with_self"].Pass {
				t.Error("child_with_self should pass (self.mode='strict')")
			}

			if !result.Results["child_no_self"].Pass {
				t.Error("child_no_self should pass (value=60 > 50)")
			}

			if !result.Results["grandchild_test"].Pass {
				t.Error("grandchild_test should pass (value=60 > self.threshold=30)")
			}
		})
	}
}

func TestRuleSelfUndefinedReference(t *testing.T) {
	e := indigo.NewEngine(cel.NewEvaluator())

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "value", Type: indigo.Int{}},
			{Name: "self", Type: indigo.Any{}},
		},
	}

	r := &indigo.Rule{
		ID:     "parent",
		Schema: schema,
		Expr:   "true",
		Self: map[string]any{
			"threshold": 50,
		},
		Rules: map[string]*indigo.Rule{
			"child_undefined_self": {
				ID:     "child_undefined_self",
				Schema: schema,
				Expr:   `value > self.threshold`,
				// No Self field - should cause error when trying to access self
			},
		},
	}

	err := e.Compile(r)
	if err != nil {
		t.Fatalf("compilation failed: %v", err)
	}

	data := map[string]any{
		"value": 60,
	}

	_, err = e.Eval(context.Background(), r, data)

	// We expect this to fail because child tries to access self.threshold but has no Self field
	if err == nil {
		t.Error("evaluation should fail because child references undefined self")
	}
	if err != nil && !strings.Contains(err.Error(), "no such key: threshold") {
		t.Fatalf("unexpected error: %v", err)
	}
}
