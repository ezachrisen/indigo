package indigo

import (
	"fmt"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// Result of evaluating a rule.
type Result struct {
	// The Rule that was evaluated
	Rule *Rule

	// Whether the rule yielded a TRUE logical value.
	// The default is TRUE
	// This is the result of evaluating THIS rule only.
	// The result will not be affected by the results of the child rules.
	// If no rule expression is supplied for a rule, the result will be TRUE.
	Pass bool

	// The result of the evaluation. Boolean for logical expressions.
	// Calculations, object constructions or string manipulations will return the appropriate Go type.
	Value interface{}

	// Results of evaluating the child rules.
	Results map[string]*Result

	// Diagnostic data; only available if you turn on diagnostics for the evaluation
	Diagnostics *Diagnostics

	// The evaluation options used
	EvalOptions EvalOptions

	// A list of the rules evaluated, in the order they were evaluated
	// Only available if you turn on diagnostics for the evaluation
	// This may be different from the rules represented in Results, if
	// If we're discarding failed/passed rules, they will not be in the results,
	// and will not show up in diagnostics, but they will be in this list.
	RulesEvaluated []*Rule
}

// String produces a list of rules (including child rules) executed and the result of the evaluation.
func (u *Result) String() string {

	tw := table.NewWriter()
	tw.SetTitle("\nINDIGO RESULT SUMMARY\n")
	tw.AppendHeader(table.Row{"\nRule", "Result\nKey", "Pass/\nFail", "Chil-\ndren", "Output\nValue", "Diagnostics\nAvailable?",
		"Stop If\nParent Neg.", "Stop First\nPos. Child", "Stop First\nNeg. Child", "Discard\nPass", "Discard\nFail"})
	rows := u.resultsToRows("", 0)

	for _, r := range rows {
		tw.AppendRow(r)
	}
	style := table.StyleLight
	style.Format.Header = text.FormatDefault
	tw.SetStyle(style)
	return tw.Render()
}

// resultsToRows transforms the Results data to a list of resultsToRows
// for inclusion in a table.Writer table.
func (u *Result) resultsToRows(key string, n int) []table.Row {
	rows := []table.Row{}
	indent := strings.Repeat("  ", n)
	boolString := "PASS"
	if !u.Pass {
		boolString = "FAIL"
	}

	diag := false
	if u.Diagnostics != nil {
		diag = true
	}

	row := table.Row{
		fmt.Sprintf("%s%s", indent, u.Rule.ID),
		key,
		boolString,
		fmt.Sprintf("%d", len(u.Results)),
		fmt.Sprintf("%v", u.Value),
		trueFalse(fmt.Sprintf("%t", diag)),
		trueFalse(fmt.Sprintf("%t", u.EvalOptions.StopIfParentNegative)),
		trueFalse(fmt.Sprintf("%t", u.EvalOptions.StopFirstPositiveChild)),
		trueFalse(fmt.Sprintf("%t", u.EvalOptions.StopFirstNegativeChild)),
		trueFalse(fmt.Sprintf("%t", u.EvalOptions.DiscardPass)),
		trueFalse(fmt.Sprintf("%t", u.EvalOptions.DiscardFail)),
	}

	rows = append(rows, row)
	for k, cd := range u.Results {
		var childKey string
		if key == "" {
			childKey = k
		} else {
			childKey = key + "." + k
		}
		rows = append(rows, cd.resultsToRows(childKey, n+1)...)
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
