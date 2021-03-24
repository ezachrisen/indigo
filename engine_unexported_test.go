package indigo

import (
	"fmt"
	"testing"

	"github.com/matryer/is"
)

func TestRuleParent(t *testing.T) {
	is := is.New(t)

	e := NewEngine(NoOpEvaluator{})
	err := e.AddRule("/", makeRuleNoOptions())
	is.NoErr(err)

	// This rule does not exist
	p, c, ok := e.root.FindRule("/x")
	is.True(!ok)
	is.True(c == nil)
	is.True(p == nil)

	p, c, ok = e.root.FindRule("/rule1")
	is.True(ok)
	is.True(c != nil)
	is.True(p != nil)

	p, c, ok = e.root.FindRule("/rule1/D")
	is.True(ok)
	is.True(c != nil)
	is.True(p != nil)
	is.Equal(c.ID, "D")
	is.Equal(p.ID, "rule1")

	p, c, ok = e.root.FindRule("/rule1/D/d2")
	is.True(ok)
	is.True(c != nil)
	is.True(p != nil)

	p, c, ok = e.root.FindRule("/rule1/D/dX")
	is.True(!ok)
	is.True(c == nil)
	is.True(p == nil)

	ruleC := &Rule{
		ID: "C",
	}

	err = e.AddRule("rule1/D", ruleC)
	//fmt.Println(e.root.DescribeStructure())

	is.NoErr(err)

}

func TestRuleAccess(t *testing.T) {
	is := is.New(t)

	e := NewEngine(NoOpEvaluator{})
	err := e.AddRule("/", makeRuleNoOptions())
	is.NoErr(err)

	// This rule does not exist
	p, c, ok := e.root.FindRule("/x")
	is.True(!ok)
	is.True(c == nil)
	is.True(p == nil)

	rule1, ok := e.root.Child("rule1")
	is.True(ok)
	is.True(rule1 != nil)

	D, ok := rule1.Child("D")
	is.True(ok)
	is.Equal(D.ID, "D")

	// p, c, ok = e.root.FindRule("/rule1/D/d2")
	// is.True(ok)
	// is.True(c != nil)
	// is.True(p != nil)

	// p, c, ok = e.root.FindRule("/rule1/D/dX")
	// is.True(!ok)
	// is.True(c == nil)
	// is.True(p == nil)

	// ruleC := &Rule{
	// 	ID: "C",
	// }

	// err = e.AddRule("rule1/D", ruleC)
	// //fmt.Println(e.root.DescribeStructure())

	// is.NoErr(err)

}

func TestAddRules(t *testing.T) {
	is := is.New(t)

	e := NewEngine(NoOpEvaluator{})

	err := e.AddRule("/", nil)
	is.True(err != nil)

	err = e.AddRule("/", &Rule{})
	if err == nil {
		is.Fail() // Adding a rule without ID should fail
	}

	err = e.AddRule("/", &Rule{ID: "ID/SOMETHING"})
	if err == nil {
		is.Fail() // Adding a rule with a / should fail
	}

	err = e.AddRule("/", &Rule{ID: "A"})
	is.NoErr(err) // Only an ID is required for a valid rule

	aCopy, ok := e.Rule("/A")
	is.True(ok) // The rule should be in the engine
	is.Equal("A", aCopy.ID)

	ruleB := Rule{
		ID:   "B",
		Meta: "B1",
		Rules: map[string]*Rule{
			"BB": &Rule{
				ID:   "BB",
				Meta: "B2",
			},
			"C": &Rule{
				ID:   "C",
				Meta: "C2",
			},
		},
	}

	err = e.AddRule("/", &ruleB)
	is.NoErr(err)

	ruleC := &Rule{
		ID: "D",
	}

	err = e.AddRule("/B", ruleC)
	is.NoErr(err)

	c, ok := e.Rule("/B/C")
	is.True(ok)
	is.Equal(c.ID, "C")
}

func TestFindMatchingChildren(t *testing.T) {
	is := is.New(t)
	e := NewEngine(NoOpEvaluator{})
	err := e.AddRule("/", makeRuleRepeatedChildren())
	is.NoErr(err)

	ps := e.AllParents("d1")
	for _, r := range ps {
		fmt.Println(r.ID)
		fmt.Println(r.Rules["d1"].ID)
	}

}

func TestReplace(t *testing.T) {
	is := is.New(t)
	ruleB := NewRule("B")
	ruleB.AddChild(NewRule("C"))
	ruleB.AddChild(NewRule("D"))
	fmt.Println(ruleB.DescribeStructure())

	ruleE := NewRule("E")
	ruleE.AddChild(NewRule("e1"))
	ruleE.AddChild(NewRule("e2"))

	ruleB.AddChild(ruleE)
	fmt.Println(ruleB.DescribeStructure())

	e := NewEngine(NoOpEvaluator{})
	err := e.AddRule("/", ruleB)
	is.NoErr(err)

	if r, ok := e.Rule("/"); ok {
		fmt.Println(r.DescribeStructure())
	} else {
		fmt.Println("Rule / not found!")
	}

	ruleCX := NewRule("CX")
	ruleCX.AddChild(NewRule("CX-1"))
	ruleCX.AddChild(NewRule("CX-2"))

	err = e.ReplaceRule("/C", ruleCX)
	is.True(err != nil)

	err = e.ReplaceRule("/B/C", ruleCX)
	is.NoErr(err)

	ruleE1New := NewRule("e5")

	err = e.ReplaceRule("/B/E/e1", ruleE1New)
	is.NoErr(err)
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

func makeRuleRepeatedChildren() *Rule {
	rule1 := &Rule{
		ID:   "rule1",
		Expr: `true`,
		Rules: map[string]*Rule{
			"d1": {
				ID:   "d1",
				Expr: `true`,
			},
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
					"d1": {
						ID:   "d1",
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
