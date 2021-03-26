package indigo

import (
	"fmt"
	"sort"
	"strings"
)

// A Rule defines an expression that can be evaluated by the
// Engine. The language used in the expression is dependent
// on the implementation of Engine.
//
// A Rule can also have child rules, so that a Rule can serve the
// purpose of being a container for other rules. Options can be
// set via EvalOptions on each rule to determine how to process
// the child rules.
//
// Example Rule Structures
//
// A hierchy of parent/child rules, combined with evaluation options
// give many different ways of using the rules engine.
//  Rule with expression, no child rules:
//   Parent rule expression is evaluated and result returned.
//
//  Rule with expression and child rules:
//   No options specified
//   - Parent rule xpression is evaluated, and so are all the child rules.
//   - All children and their evaluation results are returned
//
//  Rule with expression and child rules
//  Option set: StopIfParentNegative
//  - Parent rule expression is evaluated
//  - If the parent rule is a boolean, and it returns FALSE,
//    the children are NOT evaluated
//  - If the parent rule returns TRUE, or if it's not a
//    boolean, all the children and their resulsts are returned
//
type Rule struct {
	// A rule identifer. (required)
	// The identifier is used identify results of evaluation
	ID string

	// The expression to evaluate (optional)
	// The expression can return a boolean (true or false), or any
	// other value the underlying expression engine can produce.
	// All values are returned in the Results.Value field.
	// Boolean values are also returned in the results as Pass = true  / false
	// If the expression is blank, the result will be true.
	// (This is common if the rule is used as a container for other rules)
	Expr string

	// Program is a compiled representation of the expression.
	// Evaluators may attach a compiled representation of the rule
	// during compilation. Depending on the Evaluator, you may serialize
	// and store the Program between evaluations. This is useful if the time
	// it takes to compile the rule set is so large that restoring a previously
	// compiled version is significantly faster.
	Program interface{}

	// The output type of the expression. Compilers with the ability to check
	// whether an expression produces the desired output should return an error
	// if the expression does not. If you are using an underlying rules engine
	// that does not support type checking, this value is for your information
	// only.
	ResultType Type

	// The schema describing the data provided in the Evaluate input. (optional)
	// Some implementations of Rules require a schema.
	Schema Schema

	// A reference to an object whose values can be used in the rule expression.
	// This is useful to allow dynamic value substitution in the expression.
	// For example, in a rule determining if the temperature is in the specified range,
	// set Self = Thermostat, where Thermostat holds the user's temperature preference.
	// When evaluating the rules, the Thermostat object will be included in the data
	// with the key selfKey (see constants). Rules can then reference the user's temperature preference
	// in the rule expression. This allows rule inputs to be changed without recompiling
	// the rule.
	Self interface{}

	// A set of child rules.
	Rules map[string]*Rule

	// Sorted list of child rule keys.
	// Child rules will be evaluated in this order.
	sortedKeys []string

	// A reference to any object.
	// Useful for external processes to attach an object to the rule.
	// Not used by the rules engine, but returned to you via the Result.
	Meta interface{}

	// Set options for how the engine should evaluate this rule and the child
	// rules. Options will be inherited by all children, but the children can
	// override with options of their own.
	EvalOpts []EvalOption
}

const (
	// If the rule includes a Self object, it will be made available in the input
	// data with this key name.
	selfKey = "self"
)

// NewRule initializes a rule with the given ID
func NewRule(id string) *Rule {
	return &Rule{
		ID:    id,
		Rules: map[string]*Rule{},
	}
}

// sortChildKeys sorts the IDs of the child rules according to the
// SortFunc set in evaluation options. If none is set, the rules are
// evaluated alphabetically.
func (r *Rule) sortChildKeys(o EvalOptions) {

	r.sortedKeys = make([]string, 0, len(r.Rules))
	for k := range r.Rules {
		r.sortedKeys = append(r.sortedKeys, k)
	}

	if o.SortFunc != nil {
		sort.Slice(r.sortedKeys, o.SortFunc)
	} else {
		sort.Strings(r.sortedKeys)
	}
}

// func (r *Rule) Copy() *Rule {

// 	nr := Rule{
// 		ID:         r.ID,
// 		Expr:       r.ID,
// 		ResultType: r.ResultType,
// 		Schema:     r.Schema,
// 		Self:       r.Self,
// 		Rules:      r.copyRules(),
// 		sortedKeys: r.copySortedKeys(),
// 		Meta:       r.Meta,
// 		EvalOpts:   []EvalOption{},
// 	}

// 	return &nr
// }

// Find a rule with the given path
// The rule path is the concatenation of the
// rule IDs in a hierarchy, separated by /
// For example, given this hierarchy of rule IDs:
//  rule1
//    b
//    c
//      c1
//      c2
//
// c1 can be identified with the path
// rule1/c/c1
// Returns a copy of the rule

// func (r *Rule) FindChild(path string) (*Rule, *Rule, bool) {

// 	elems := strings.Split(path, "/")
// 	c, ok := r.Rules[elems[0]]
// 	if !ok {
// 		return nil, nil, false
// 	}

// 	// If we're down to the last path element
// 	if len(elems) == 1 {
// 		return r, c, true
// 	}

// 	return c.FindChild(strings.Join(elems[1:len(elems)], "/"))
// }

// func (r *Rule) copyRules() map[string]*Rule {
// 	mn := make(map[string]*Rule, len(r.Rules))
// 	for k := range r.Rules {
// 		cc := r.Rules[k].Copy()
// 		mn[cc.ID] = cc
// 	}
// 	return mn
// }

// func (r *Rule) copySortedKeys() []string {
// 	b := make([]string, 0, len(r.sortedKeys))
// 	for _, s := range r.sortedKeys {
// 		b = append(b, s)
// 	}
// 	return b
// }

// DescribeStructure returns a list of all the rules in hierarchy, with
// child rules sorted in evaluation order.
// This is useful for visualizing a rule hierarchy.
func (r *Rule) DescribeStructure(n ...int) string {
	return r.describeStructure(0)
}

func (r *Rule) describeStructure(n int) string {
	s := strings.Builder{}

	s.WriteString(strings.Repeat(" ", n)) // indent
	s.WriteString(r.ID)
	s.WriteString("\n")
	if len(r.sortedKeys) != len(r.Rules) {
		s.WriteString(fmt.Sprintf("ERROR: in rule '%s', length of sortedKeys (%d) != length of rules (%d)\n", r.ID, len(r.sortedKeys), len(r.Rules)))
		s.WriteString(fmt.Sprintf("sortedKeys = %s\n", r.sortedKeys))
	}
	for _, k := range r.sortedKeys {
		if c, ok := r.Rules[k]; ok {
			s.WriteString(c.DescribeStructure(n + 1))
		} else {
			s.WriteString("ERROR: missing rule '" + k + "'\n")
		}
	}
	return s.String()
}
