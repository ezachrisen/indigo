package indigo

import (
	"fmt"
	"iter"
	"maps"
	"slices"
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
	// This may be different from the rules represented in Results, if
	// If we're discarding failed/passed rules, they will not be in the results,
	// and will not show up in diagnostics, but they will be in this list.
	RulesEvaluated []*Rule
}

// Unshard reorganizes the results into the "original" structure of the rule
// without shards applied. If you defined shards and applied them with BuildShards,
// results will come organized by shard. This is normally not desired, since the client
// will need to know about the shard structure, which typically is used for performance reasons.
// This function removes the sharding from the results, returning a structure that the client
// is familiar with.
func (u *Result) Unshard() error {
	if u == nil || u.Rule == nil || u.Rule.Shards == nil {
		// by default, if there are no shards, results are unchanged
		return nil
	}

	detached := u.Results
	nr := map[string]*Result{}

	for _, d := range detached {
		if d.Rule.shard {
			for _, dr := range d.Results {
				err := dr.Unshard()
				if err != nil {
					return err
				}
				switch dr.Rule.shard {
				case true:
					for _, drc := range dr.Results {
						nr[drc.Rule.ID] = drc
					}
				default:
					nr[dr.Rule.ID] = dr
				}
			}
		}
	}
	u.Results = nr
	return nil
}

// Flat returns all results from r as a single iterable list,
// without rule hierarchy. Skips all shard rules, but results
// from the shards is included. This is useful when you only care about which rules
// passed, and you don't care about the hierarchy of parent/child rules.
func (r *Result) Flat() iter.Seq[*Result] {
	return func(yield func(*Result) bool) {
		var dfs func(*Result) bool
		dfs = func(node *Result) bool {
			if node == nil {
				return true
			}
			if !node.Rule.shard {
				if !yield(node) {
					return false
				}
			}
			// Push children in reverse order for deterministic traversal (pre-order)
			keys := make([]string, 0, len(node.Results))
			for k := range node.Results {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			for i := len(keys) - 1; i >= 0; i-- {
				dfs(node.Results[keys[i]])
			}
			return true
		}
		dfs(r)
	}
}

// String produces a list of rules (including child rules) executed and the result of the evaluation.
func (u *Result) String() string {
	tw := table.NewWriter()
	tw.SetTitle("\nINDIGO RESULTS\n")
	tw.AppendHeader(table.Row{
		"\nRule", "Pass/\nFail", "Expr.\nPass/\nFail", "Chil-\ndren", "Output\nValue", "Diagnostics\nAvailable?", "True\nIf Any?",
		"Stop If\nParent Neg.", "Stop First\nPos. Child", "Stop First\nNeg. Child", "Discard\nPass", "Discard\nFail",
	})
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
	}

	rows = append(rows, row)
	keys := slices.Sorted(maps.Keys(u.Results))
	for _, k := range keys {
		cd := u.Results[k]
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
