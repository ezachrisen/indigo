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

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/matryer/is"
)

func makeStudentData() map[string]interface{} {
	return map[string]interface{}{
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
				ID:     "at_risk",
				Expr:   `student.GPA < 2.5 || student.Status == "Probation"`,
				Schema: makeEducationSchema(),
				Rules: map[string]*indigo.Rule{
					"risk_factor": {
						ID:   "risk_factor",
						Expr: `2.0+6.0`,
					},
				},
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

	root := indigo.NewRule("root")

	root.Rules[rule1.ID] = rule1
	root.Rules[rule2.ID] = rule2
	root.Rules[rule3.ID] = rule3
	return root
}

func makeEducationRules() map[string]*indigo.Rule {

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
				ID:     "at_risk",
				Expr:   `student.GPA < 2.5 || student.Status == "Probation"`,
				Schema: makeEducationSchema(),
				Rules: map[string]*indigo.Rule{
					"risk_factor": {
						ID:   "risk_factor",
						Expr: `2.0+6.0`,
					},
				},
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

	return map[string]*indigo.Rule{
		rule1.ID: rule1,
		rule2.ID: rule2,
		rule3.ID: rule3,
	}
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

func TestBasicRules(t *testing.T) {

	is := is.New(t)

	e := indigo.NewEngine(cel.NewEvaluator())
	r := makeEducationRules1()
	e.Compile(r)

	sa := r.Rules["student_actions"]

	results, err := e.Eval(context.Background(), sa, makeStudentData())
	is.NoErr(err)
	is.Equal(results.Rule, sa)
	is.True(results.Pass)
	is.True(!results.Results["honors_student"].Pass)
	is.True(results.Results["at_risk"].Pass)
	is.Equal(results.Results["at_risk"].Results["risk_factor"].Value.(float64), 8.0)
}

func makeEducationProtoSchema() indigo.Schema {
	return indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Protoname: "school.Student", Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
			{Name: "self", Type: indigo.Proto{Protoname: "school.HonorsConfiguration", Message: &school.HonorsConfiguration{}}},
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
				Expr:   `student.GPA >= self.Minimum_GPA && student.Status != school.Student.status_type.PROBATION && student.Grades.all(g, g>=3.0)`,
				Self:   &school.HonorsConfiguration{Minimum_GPA: 3.7},
				Meta:   true,
				Schema: makeEducationProtoSchema(),
			},
			"at_risk": {
				ID:     "at_risk",
				Expr:   `student.GPA < 2.5 || student.Status == school.Student.status_type.PROBATION`,
				Meta:   false,
				Schema: makeEducationProtoSchema(),
			},
			"tenure_gt_6months": {
				ID:     "tenure_gt_6months",
				Expr:   `now - student.EnrollmentDate > duration("4320h")`, // 6 months = 4320 hours
				Meta:   true,
				Schema: makeEducationProtoSchema(),
			},
		},
	}

}

func makeEducationProtoRulesSimple(id string) *indigo.Rule {
	return &indigo.Rule{
		ID:     id,
		Schema: makeEducationProtoSchema(),
		Rules: map[string]*indigo.Rule{
			"honor_student": {
				ID:   "honor_student",
				Expr: `student.GPA >= self.Minimum_GPA && student.Status != school.Student.status_type.PROBATION && student.Grades.all(g, g>=3.0)`,
				Self: &school.HonorsConfiguration{Minimum_GPA: 3.7},
				Meta: true,
			},
		},
	}

}

func makeStudentProtoData() map[string]interface{} {
	s := school.Student{
		Age:            16,
		GPA:            3.76,
		Status:         school.Student_ENROLLED,
		Grades:         []float64{4.0, 4.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamp.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	s.ProtoReflect()

	return map[string]interface{}{
		"student": &s,
		"now":     &timestamp.Timestamp{Seconds: time.Now().Unix()},
	}

}

// Make sure that type mismatches between schema and rule are caught at compile time
func TestCompileErrors(t *testing.T) {

	is := is.New(t)

	e := indigo.NewEngine(cel.NewEvaluator())
	r := makeEducationRulesWithIncorrectTypes()

	err := e.Compile(r)
	if err == nil {
		is.Fail() // expected compile error here
	}
	is.True(strings.Contains(err.Error(), "1:13: found no matching overload for '_!=_' applied to '(double, string)'"))
	is.True(strings.Contains(err.Error(), "1:40: found no matching overload for '_>_' applied to '(string, double)'"))
}

func TestProtoMessage(t *testing.T) {

	is := is.New(t)
	e := indigo.NewEngine(cel.NewEvaluator())

	r := makeEducationProtoRules("student_actions")
	err := e.Compile(r)
	is.NoErr(err)

	results, err := e.Eval(context.Background(), r, makeStudentProtoData())
	is.NoErr(err)
	is.Equal(len(results.Results), 3)
	for _, v := range results.Results {
		is.Equal(v.Rule.Meta, v.Pass)
	}
}

func TestDiagnosticOptions(t *testing.T) {

	is := is.New(t)
	e := indigo.NewEngine(cel.NewEvaluator())
	r2 := makeEducationProtoRules("student_actions")
	err := e.Compile(r2, indigo.CollectDiagnostics(true))
	is.NoErr(err)

	results, err := e.Eval(context.Background(), r2, makeStudentProtoData(), indigo.ReturnDiagnostics(true))
	is.NoErr(err)

	for _, c := range results.Results {
		if len(c.Diagnostics) < 100 {
			t.Errorf("Wanted diagnostics for rule %s, got %s", c.Rule.ID, c.Diagnostics)
		}
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
				Schema:     makeEducationSchema(),
				ResultType: indigo.Bool{},
				Expr:       `student.GPA >= 3.6 && student.Status!="Probation" && !("C" in student.Grades)`,
			},
			nil,
		},
		{
			indigo.Rule{
				ID:         "shouldBeBFloat",
				Schema:     makeEducationSchema(),
				ResultType: indigo.Float{},
				Expr:       `student.GPA + 1.0`,
			},
			nil,
		},
		{
			indigo.Rule{
				ID:         "shouldBeStudent",
				Schema:     makeEducationProtoSchema(),
				ResultType: indigo.Proto{Protoname: "school.Student"},
				Expr:       `school.Student { GPA: 1.2 }`,
			},
			nil,
		},
		{
			indigo.Rule{
				ID:         "NEGATIVEshouldBeBFloat",
				Schema:     makeEducationSchema(),
				ResultType: indigo.Bool{},
				Expr:       `student.GPA + 1.0`,
			},
			fmt.Errorf("Should be an error"),
		},
		{
			indigo.Rule{
				ID:         "NEGATIVEshouldBeStudent",
				Schema:     makeEducationProtoSchema(),
				ResultType: indigo.Proto{Protoname: "school.HonorsConfiguration"},
				Expr:       `school.Student { GPA: 1.2 }`,
			},
			fmt.Errorf("Should be an error"),
		},
	}

	e := indigo.NewEngine(cel.NewEvaluator())

	for _, c := range cases {
		err := e.Compile(&c.rule)
		if c.err == nil && err != nil {
			t.Errorf("For rule %s, wanted err = %v, got %v", c.rule.ID, c.err, err)
		}
	}
}

// func TestConcurrencyNoDiagnostics(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping test in short mode.")
// 	}

// 	is := is.New(t)
// 	e := indigo.NewEngine(cel.NewEvaluator(), indigo.DryRun(true))

// 	// How large to make the channel depends on the capacity of the system
// 	// Because AddRule is slow, a large capacity channel will exhaust the number of
// 	// allowable goroutines.
// 	buf := make(chan int, 8_000)
// 	start := time.Now()
// 	printDebug := false

// 	go func() {
// 		for i := 1; i < 50_000; i++ {
// 			if i%1000 == 0 && printDebug {
// 				fmt.Println("---- Sent ", i)
// 			}
// 			buf <- i
// 		}
// 		close(buf)
// 	}()

// 	for i := range buf {
// 		err := e.AddRule("/", makeEducationProtoRulesSimple(fmt.Sprintf("rule%d", i)))
// 		is.NoErr(err)
// 		r, err := e.Eval(context.Background(), makeStudentProtoData(), fmt.Sprintf("rule%d", i), indigo.ReturnDiagnostics(false))
// 		is.NoErr(err)
// 		is.Equal(r.RulesEvaluated, 2)
// 		if i%1000 == 0 && printDebug {
// 			fmt.Println("---- Done ", i, " in ", time.Since(start))
// 		}

// 	}
// }

// func TestConcurrencyCollectDiagnostics(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping test in short mode.")
// 	}

// 	is := is.New(t)
// 	e := indigo.NewEngine(cel.NewEvaluator(), indigo.DryRun(true), indigo.ForceDiagnosticsAllRules(true), indigo.CollectDiagnostics(true))

// 	// How large to make the channel depends on the capacity of the system
// 	// Because AddRule is slow, a large capacity channel will exhaust the number of
// 	// allowable goroutines.
// 	buf := make(chan int, 8_000)
// 	start := time.Now()
// 	printDebug := false

// 	go func() {
// 		for i := 1; i < 50_000; i++ {
// 			if i%1000 == 0 && printDebug {
// 				fmt.Println("---- Sent ", i)
// 			}
// 			buf <- i
// 		}
// 		close(buf)
// 	}()

// 	for i := range buf {
// 		err := e.AddRule("/", makeEducationProtoRulesSimple(fmt.Sprintf("rule%d", i)))
// 		is.NoErr(err)
// 		r, err := e.Eval(context.Background(), makeStudentProtoData(), fmt.Sprintf("rule%d", i), indigo.ReturnDiagnostics(false))
// 		is.NoErr(err)
// 		is.Equal(r.RulesEvaluated, 2)
// 		if i%1000 == 0 && printDebug {
// 			fmt.Println("---- Done ", i, " in ", time.Since(start))
// 		}

// 	}
// }

// func TestConcurrencyMixed(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping test in short mode.")
// 	}

// 	dryRun := true
// 	n := 100_000
// 	randomDelay := true
// 	maxDelayMicroseconds := 1000
// 	printDebug := true
// 	printDebugInterval := 1000
// 	rand.Seed(time.Now().Unix())

// 	is := is.New(t)
// 	e := indigo.NewEngine(cel.NewEvaluator(), indigo.DryRun(dryRun))

// 	// Set up a rule that we're going to evaluate over and over again
// 	err := e.AddRule("/", makeEducationProtoRulesSimple("rule-1"))
// 	is.NoErr(err)

// 	// Set up a rule that we're going to replace over and over again
// 	err = e.AddRule("/", makeEducationProtoRulesSimple("rule-2"))
// 	is.NoErr(err)

// 	//	fmt.Println(e.RootRuleUnsafe().DescribeStructure())
// 	// How large to make the channels depends on the capacity of the system
// 	// Because AddRule is slow, a large capacity channel will exhaust the number of
// 	// allowable goroutines.
// 	add := make(chan int, 2000)
// 	eval := make(chan int, 2000)
// 	replace := make(chan int, 2000)
// 	del := make(chan int, 2000)

// 	var adds int
// 	var evals int
// 	var replaces int
// 	var dels int

// 	// start := time.Now()

// 	// AddRule requests
// 	go func() {
// 		for i := 1; i < n; i++ {
// 			if i%printDebugInterval == 0 && printDebug {
// 				fmt.Println("---- Add ", i)
// 			}
// 			add <- i
// 			if randomDelay {
// 				time.Sleep(time.Duration(rand.Intn(maxDelayMicroseconds)) * time.Microsecond)
// 			}
// 		}
// 		close(add)
// 	}()

// 	// Evaluation requests
// 	go func() {
// 		for i := 1; i < n; i++ {
// 			if i%printDebugInterval == 0 && printDebug {
// 				fmt.Println("---- Eval ", i)
// 			}
// 			eval <- i
// 			if randomDelay {
// 				time.Sleep(time.Duration(rand.Intn(maxDelayMicroseconds)) * time.Microsecond)
// 			}

// 		}
// 		close(eval)
// 	}()

// 	// Replace rule requests
// 	go func() {
// 		for i := 1; i < n; i++ {
// 			if i%printDebugInterval == 0 && printDebug {
// 				fmt.Println("---- Replace ", i)
// 			}
// 			replace <- i
// 			if randomDelay {
// 				time.Sleep(time.Duration(rand.Intn(maxDelayMicroseconds)) * time.Microsecond)
// 			}

// 		}
// 		close(replace)
// 	}()

// 	// Delete rule requests
// 	go func() {
// 		for i := 1; i < n; i++ {
// 			if i%printDebugInterval == 0 && printDebug {
// 				fmt.Println("---- Delete ", i)
// 			}
// 			del <- i
// 			if randomDelay {
// 				time.Sleep(time.Duration(rand.Intn(maxDelayMicroseconds)) * time.Microsecond)
// 			}

// 		}
// 		close(del)
// 	}()

// 	for eval != nil || add != nil || replace != nil || del != nil {

// 		select {

// 		case i, ok := <-add:
// 			if !ok {
// 				add = nil
// 			} else {
// 				err := e.AddRule("/", makeEducationProtoRulesSimple(fmt.Sprintf("rule%d", i)))
// 				is.NoErr(err)
// 				adds++
// 			}
// 			if i%printDebugInterval == 0 && printDebug {
// 				fmt.Println("---- Processed Add ", i)
// 			}

// 		case i, ok := <-eval:
// 			if !ok {
// 				eval = nil
// 			} else {
// 				results, err := e.Eval(context.Background(), makeStudentProtoData(), fmt.Sprintf("rule%d", -1))
// 				is.NoErr(err)
// 				if !dryRun {
// 					is.Equal(len(results.Results), 1)
// 					for _, v := range results.Results {
// 						is.Equal(v.Meta, v.Pass)
// 					}
// 				}
// 				evals++
// 				if i%printDebugInterval == 0 && printDebug {
// 					fmt.Println("---- Processed Eval ", i)
// 				}
// 			}
// 		case i, ok := <-replace:
// 			if !ok {
// 				replace = nil
// 			} else {
// 				err = e.ReplaceRule("/rule-2", &indigo.Rule{
// 					ID:   "rule-2",
// 					Expr: `student.GPA < 1000.0 || student.Status == school.Student.status_type.PROBATION`,
// 					Meta: true,
// 				})
// 				is.NoErr(err)
// 				replaces++
// 				if i%printDebugInterval == 0 && printDebug {
// 					fmt.Println("---- Processed Replace ", i)
// 				}

// 			}
// 		case i, ok := <-del:
// 			if !ok {
// 				del = nil
// 			} else {
// 				dels++
// 				if i%printDebugInterval == 0 && printDebug {
// 					fmt.Println("---- Processed Delete ", i)
// 				}
// 				ruleID := fmt.Sprintf("ruled%d", i)
// 				err := e.AddRule("/", makeEducationProtoRulesSimple(ruleID))
// 				is.NoErr(err)
// 				err = e.DeleteRule(ruleID)
// 				is.NoErr(err)
// 			}
// 		}
// 	}
// 	is.True(adds == replaces && dels == replaces && replaces == evals && adds == (n-1))

// }

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
		e.Eval(context.Background(), &r, data)
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
		e.Eval(context.Background(), &r, data, indigo.ReturnDiagnostics(true))
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
		e.Eval(context.Background(), r, data)
	}
}

func BenchmarkProtoWithSelfX(b *testing.B) {
	b.StopTimer()

	pb.DefaultDb.RegisterMessage(&school.Student{})

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Protoname: "school.Student", Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
			{Name: "self", Type: indigo.Proto{Protoname: "school.HonorsConfiguration", Message: &school.HonorsConfiguration{}}},
		},
	}

	e := indigo.NewEngine(cel.NewEvaluator())

	r := &indigo.Rule{
		ID:     "student_actions",
		Schema: schema,
		Rules: map[string]*indigo.Rule{
			"a": {
				ID:     "at_risk",
				Expr:   `student.GPA < self.Minimum_GPA && student.Status == school.Student.status_type.PROBATION`,
				Self:   &school.HonorsConfiguration{Minimum_GPA: 3.7},
				Schema: schema,
				Meta:   false,
			},
		},
	}

	err := e.Compile(r)
	if err != nil {
		log.Fatalf("Error adding ruleset: %v", err)
	}

	s := school.Student{
		Age:            16,
		GPA:            3,
		Status:         school.Student_PROBATION,
		Grades:         []float64{4.0, 4.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamp.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	data := map[string]interface{}{
		"student": &s,
		"now":     &timestamp.Timestamp{Seconds: time.Now().Unix()},
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		e.Eval(context.Background(), r, data)
	}

}

func BenchmarkProtoWithoutSelf(b *testing.B) {

	pb.DefaultDb.RegisterMessage(&school.Student{})

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Protoname: "school.Student", Message: &school.Student{}}},
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
				Expr:   `student.GPA < 2.5 || student.Status == school.Student.status_type.PROBATION`,
				Schema: schema,
				Meta:   false,
			},
		},
	}

	err := e.Compile(r)
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
		e.Eval(context.Background(), r, data)
	}

}

func BenchmarkProtoCreation(b *testing.B) {
	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Protoname: "school.Student", Message: &school.Student{}}},
			{Name: "student_suspension", Type: indigo.Proto{Protoname: "school.Student.Suspension", Message: &school.Student_Suspension{}}},
			{Name: "studentSummary", Type: indigo.Proto{Protoname: "school.StudentSummary", Message: &school.StudentSummary{}}},
		},
	}

	r := &indigo.Rule{
		ID:         "create_summary",
		Schema:     education,
		ResultType: indigo.Proto{Protoname: "school.StudentSummary", Message: &school.StudentSummary{}},
		Expr: `
			school.StudentSummary {
				GPA: student.GPA,
				RiskFactor: 2.0 + 3.0,
				Tenure: duration("12h")
			}`,
	}

	e := indigo.NewEngine(cel.NewEvaluator())
	err := e.Compile(r)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	for i := 0; i < b.N; i++ {
		e.Eval(context.Background(), r, map[string]interface{}{})
	}

}

func BenchmarkEval2000Rules(b *testing.B) {
	b.StopTimer()
	pb.DefaultDb.RegisterMessage(&school.Student{})

	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Protoname: "school.Student", Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
			{Name: "self", Type: indigo.Proto{Protoname: "school.HonorsConfiguration", Message: &school.HonorsConfiguration{}}},
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
			Expr:   `student.GPA < self.Minimum_GPA && student.Status == school.Student.status_type.PROBATION`,
			Schema: schema,
			Self:   &school.HonorsConfiguration{Minimum_GPA: 3.7},
			Meta:   false,
		}
		r.Rules[cr.ID] = cr
	}

	err := e.Compile(r)
	if err != nil {
		log.Fatalf("Error adding ruleset: %v", err)
	}

	s := school.Student{
		Age:            16,
		GPA:            3,
		Status:         school.Student_PROBATION,
		Grades:         []float64{4.0, 4.0, 3.7},
		Attrs:          map[string]string{"Nickname": "Joey"},
		EnrollmentDate: &timestamp.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()},
	}

	data := map[string]interface{}{
		"student": &s,
		"now":     &timestamp.Timestamp{Seconds: time.Now().Unix()},
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		e.Eval(context.Background(), r, data)
	}
}

func BenchmarkCompileRule(b *testing.B) {
	is := is.New(b)
	e := indigo.NewEngine(cel.NewEvaluator())

	r := makeEducationProtoRules(fmt.Sprintf("rule"))
	for i := 1; i < b.N; i++ {
		err := e.Compile(r)
		is.NoErr(err)
	}
}
