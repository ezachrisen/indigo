package indigo

import (
	"fmt"
	"testing"

	"github.com/matryer/is"
)

func TestRuleParent(t *testing.T) {
	is := is.New(t)

	e := NewEngine(NoOpEvaluator{})

	err := e.AddRule(makeRuleNoOptions())
	is.NoErr(err)

	r := e.ruleParent(nil, "rule1/D/d2")
	is.True(r != nil)
	fmt.Println("r = ", r.ID)
}

func makeRuleNoOptions() *Rule {
	rule1 := &Rule{
		ID:   "rule1",
		Expr: `true`,
		Rules: map[string]*Rule{
			"D": &Rule{
				ID:   "D",
				Expr: `true`,
				Rules: map[string]*Rule{
					"d1": {
						ID:   "d1",
						Expr: `true`,
					},
					"d2": {
						ID:   "d2",
						Expr: `false`,
					},
					"d3": {
						ID:   "d3",
						Expr: `true`,
					},
				},
			},
			"B": {
				ID:   "B",
				Expr: `false`,
			},
			"E": {
				ID:   "E",
				Expr: `false`,
				Rules: map[string]*Rule{
					"e1": {
						ID:   "e1",
						Expr: `true`,
					},
					"e2": {
						ID:   "e2",
						Expr: `false`,
					},
					"e3": {
						ID:   "e3",
						Expr: `true`,
					},
				},
			},
		},
	}
	return rule1
}

type NoOpEvaluator struct{}

func (n NoOpEvaluator) Compile(rule *Rule, collectDiagnostics bool, dryRun bool) error {
	return nil
}

func (n NoOpEvaluator) Eval(data map[string]interface{}, rule *Rule, opt EvalOptions) (Value, string, error) {
	return Value{
		Val: false,
		Typ: Bool{},
	}, "", nil

}
