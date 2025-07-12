package indigo

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// Result of evaluating a rule.
type Result struct {
	// The Rule that was evaluated
	Rule *Rule

	// Whether the rule is true.
	// The default is TRUE.
	// Pass is the result of rolling up all child rules and evaluating the
	// rule's own expression. All child rules and the rule's expression must be
	// true for Pass to be true.
	Pass bool

	// Whether evaluating the rule expression yielded a TRUE logical value.
	// The default is TRUE.
	// The result will not be affected by the results of the child rules.
	// If no rule expression is supplied for a rule, the result will be TRUE.
	ExpressionPass bool

	// The raw result of evaluating the expression. Boolean for logical expressions.
	// Calculations, object constructions or string manipulations will return the appropriate Go type.
	// This value is never affected by child rules.
	Value any

	// Results of evaluating the child rules.
	Results map[string]*Result

	// Diagnostic data; only available if you turn on diagnostics for the evaluation
	Diagnostics *Diagnostics

	// The evaluation options used
	EvalOptions EvalOptions

	// A list of the rules evaluated, in the order they were evaluated
	// Only available if you turn on diagnostics for the evaluation
	// This may be different from the rules represented in Results.
	// If we're discarding failed/passed rules, they will not be in the results,
	// and will not show up in diagnostics, but they will be in this list.
	RulesEvaluated []*Rule

	// EvalCount is the number of rules evaluated, including the parent rule
	EvalCount int

	// EvalParallel indicates whether the children were evaluated in parallel or in sequence
	EvalParallelCount int
}

// String produces a list of rules (including child rules) executed and the result of the evaluation.
func (u *Result) String() string {

	tw := table.NewWriter()
	tw.SetTitle("\nINDIGO RESULTS\n")
	tw.AppendHeader(table.Row{"\nRule", "Pass/\nFail", "Expr.\nPass/\nFail", "Chil-\ndren", "Output\nValue", "Diagnostics\nAvailable?", "True\nIf Any?",
		"Stop If\nParent Neg.", "Stop First\nPos. Child", "Stop First\nNeg. Child", "Discard\nPass", "Discard\nFail", "Eval\nCount", "Eval\nParallel"})
	rows := u.resultsToRows(0)

	for _, r := range rows {
		tw.AppendRow(r)
	}
	style := table.StyleLight
	style.Format.Header = text.FormatDefault
	tw.SetStyle(style)
	return tw.Render()
}

func boolString(b bool) string {
	switch b {
	case true:
		return "PASS"
	default:
		return "FAIL"
	}
}

// resultsToRows transforms the Results data to a list of resultsToRows
// for inclusion in a table.Writer table.
func (u *Result) resultsToRows(n int) []table.Row {
	rows := []table.Row{}
	indent := strings.Repeat("  ", n)

	diag := false
	if u.Diagnostics != nil {
		diag = true
	}

	row := table.Row{
		fmt.Sprintf("%s%s", indent, u.Rule.ID),
		boolString(u.Pass),
		boolString(u.ExpressionPass),
		strconv.Itoa(len(u.Results)),
		fmt.Sprintf("%v", u.Value),
		trueFalse(strconv.FormatBool(diag)),
		trueFalse(strconv.FormatBool(u.EvalOptions.TrueIfAny)),
		trueFalse(strconv.FormatBool(u.EvalOptions.StopIfParentNegative)),
		trueFalse(strconv.FormatBool(u.EvalOptions.StopFirstPositiveChild)),
		trueFalse(strconv.FormatBool(u.EvalOptions.StopFirstNegativeChild)),
		trueFalse(strconv.FormatBool(u.EvalOptions.DiscardPass)),
		trueFalse(strconv.Itoa(int(u.EvalOptions.DiscardFail))),
		strconv.Itoa(u.EvalCount),
		strconv.Itoa(u.EvalParallelCount),
	}

	rows = append(rows, row)
	for _, cd := range u.Results {
		rows = append(rows, cd.resultsToRows(n+1)...)
	}
	return rows
}

func trueFalse(t string) string {
	switch t {
	case "false":
		return ""
	case "true":
		return "yes"
	default:
		return t
	}
}

// String produces a list of rules (including child rules) executed and the result of the evaluation.
func (u *Result) Summary() string {

	tw := table.NewWriter()
	tw.SetTitle("\nINDIGO RESULT SUMMARY\n")
	tw.AppendHeader(table.Row{"\nRule", "Pass/\nFail", "Expr.\nPass/\nFail", "Output\nValue"})
	rows := u.summaryResultsToRows(0)

	for _, r := range rows {
		tw.AppendRow(r)
	}
	style := table.StyleLight
	style.Format.Header = text.FormatDefault
	tw.SetStyle(style)
	return tw.Render()
}

// summaryResultsToRows transforms the Results data to a list of resultsToRows
// for inclusion in a table.Writer table.
func (u *Result) summaryResultsToRows(n int) []table.Row {
	rows := []table.Row{}
	indent := strings.Repeat("  ", n)

	row := table.Row{
		fmt.Sprintf("%s%s", indent, u.Rule.ID),
		boolString(u.Pass),
		boolString(u.ExpressionPass),
		fmt.Sprintf("%v", u.Value),
	}

	rows = append(rows, row)
	for _, cd := range u.Results {
		rows = append(rows, cd.summaryResultsToRows(n+1)...)
	}
	return rows
}
