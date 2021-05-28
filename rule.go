package indigo

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
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
	// If no type is provided, evaluation and compilation will default to Bool
	ResultType Type `json:"result_type,omitempty"`

	// The schema describing the data provided in the Evaluate input. (optional)
	// Some implementations of Evaluator require a schema.
	Schema Schema `json:"schema,omitempty"`

	// A reference to an object whose values can be used in the rule expression.
	// Add the corresponding object in the data with the reserved key name selfKey
	// (see constants).
	// Child rules do not inherit the self value.
	// See example for usage. TODO: example
	Self interface{} `json:"-"`

	// A set of child rules.
	Rules map[string]*Rule `json:"rules,omitempty"`

	// Reference to intermediate compilation / evaluation data.
	Program interface{} `json:"-"`

	// A reference to any object.
	// Not used by the rules engine.
	Meta interface{} `json:"-"`

	// Options determining how the child rules should be handled.
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

// ApplyToRule applies the function f to the rule r and its children recursively.
func ApplyToRule(r *Rule, f func(r *Rule) error) error {
	err := f(r)
	if err != nil {
		return err
	}
	for _, c := range r.Rules {
		err := ApplyToRule(c, f)
		if err != nil {
			return err
		}
	}
	return nil
}

// String returns a list of all the rules in hierarchy, with
// child rules sorted in evaluation order.
func (r *Rule) String() string {
	tw := table.NewWriter()
	tw.SetTitle("\nINDIGO RULES\n")
	tw.AppendHeader(table.Row{"\nRule", "\nSchema", "\nExpression", "Result\nType", "\nMeta"})

	maxWidthOfExpressionColumn := 40
	rows, maxExprLength := r.rulesToRows(0)
	for _, r := range rows {
		tw.AppendRow(r)
	}

	tw.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1},
		{Number: 2},
		{Number: 3, WidthMax: maxWidthOfExpressionColumn},
		{Number: 4},
		{Number: 5},
	})

	style := table.StyleLight
	style.Format.Header = text.FormatDefault
	// Only add the row separator if the expression is wide enough to wrap.
	if maxExprLength > maxWidthOfExpressionColumn {
		style.Options.SeparateRows = true
	}
	tw.SetStyle(style)
	return tw.Render()

}

func (r *Rule) rulesToRows(n int) ([]table.Row, int) {
	rows := []table.Row{}
	indent := strings.Repeat("  ", n)

	row := table.Row{
		fmt.Sprintf("%s%s", indent, r.ID),
		r.Schema.ID,
		r.Expr,
		fmt.Sprintf("%v", r.ResultType),
		fmt.Sprintf("%T", r.Meta),
	}
	rows = append(rows, row)
	maxExprLength := len(r.Expr)

	for _, c := range r.Rules {
		cr, max := c.rulesToRows(n + 1)
		if max > maxExprLength {
			maxExprLength = max
		}
		rows = append(rows, cr...)
	}
	return rows, maxExprLength
}

// sortChildKeys sorts the IDs of the child rules according to the
// SortFunc set in evaluation options. If no SortFunc is set, the evaluation
// order is not specified.
func (r *Rule) sortChildKeys(o EvalOptions) []*Rule {
	keys := make([]*Rule, 0, len(r.Rules))
	for k := range r.Rules {
		keys = append(keys, r.Rules[k])
	}

	if o.SortFunc != nil && sortOrderMatters(o) {
		sort.Slice(keys, func(i, j int) bool {
			return o.SortFunc(keys, i, j)
		})
	}
	return keys
}

// Based on the evaluation options, determine if the order of evaluation matters
func sortOrderMatters(o EvalOptions) bool {

	if o.StopFirstNegativeChild || o.StopFirstPositiveChild {
		return true
	}

	return false

}
