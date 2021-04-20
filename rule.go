package indigo

import (
	"sort"
	"strings"

	"github.com/ezachrisen/indigo/schema"
)

// A Rule defines logic that can be evaluated by an Evaluator.
// The logic for evaluation is specified by an expression.
// A rule can have child rules. Rule options specify to the Evaluator
// how child rules should be handled. Child rules can in turn have children,
// enabling you to create a hierarchy of rules.
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
	ID string `json:"id"`

	// The expression to evaluate (optional)
	// The expression can return a boolean (true or false), or any
	// other value the underlying expression engine can produce.
	// All values are returned in the Results.Value field.
	// Boolean values are also returned in the results as Pass = true  / false
	// If the expression is blank, the result will be true.
	Expr string `json:"expr"`

	// The output type of the expression. Evaluators with the ability to check
	// whether an expression produces the desired output should return an error
	// if the expression does not.
	ResultType schema.Type `json:"result_type,omitempty"`

	// The schema describing the data provided in the Evaluate input. (optional)
	// Some implementations of Evaluator require a schema.
	Schema schema.Schema `json:"schema,omitempty"`

	// A reference to an object whose values can be used in the rule expression.
	// Add the corresponding object in the data with the reserved key name selfKey
	// (see constants).
	// See example for usage. TODO: example
	Self interface{} `json:"-"`

	// A set of child rules.
	Rules map[string]*Rule `json:"rules,omitempty"`

	// Reference to intermediate compilation / evaluation data.
	Program interface{} `json:"-"`

	// A reference to any object.
	// Not used by the rules engine.
	Meta interface{} `json:"-"`

	// Options determining how the rule should be evaluated
	EvalOptions EvalOptions `json:"eval_options"`
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

// String returns a list of all the rules in hierarchy, with
// child rules sorted in evaluation order.
func (r *Rule) String() string {
	return r.describe(0)
}

func (r *Rule) describe(n int) string {
	s := strings.Builder{}
	s.WriteString(strings.Repeat("  ", n)) // indent
	s.WriteString(r.ID)
	s.WriteString("\n")
	for _, c := range r.Rules {
		s.WriteString(c.describe(n + 1))
	}
	return s.String()
}

// sortChildKeys sorts the IDs of the child rules according to the
// SortFunc set in evaluation options. If no SortFunc is set, the evaluation
// order is not specified.
func (r *Rule) sortChildKeys() []*Rule {
	keys := make([]*Rule, 0, len(r.Rules))
	for k := range r.Rules {
		keys = append(keys, r.Rules[k])
	}

	if r.EvalOptions.SortFunc != nil {
		sort.Slice(keys, func(i, j int) bool {
			return r.EvalOptions.SortFunc(keys, i, j)
		})
	}
	return keys
}
