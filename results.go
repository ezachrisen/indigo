package indigo

import (
	"fmt"
	"strings"
)

// Result of evaluating a rule.
type Result struct {
	// The Rule that was evaluated
	RuleID string

	// Reference to a value set when the rule was added to the engine.
	Meta interface{}

	// Whether the rule yielded a TRUE logical value.
	// The default is FALSE
	// This is the result of evaluating THIS rule only.
	// The result will not be affected by the results of the child rules.
	// If no rule expression is supplied for a rule, the result will be TRUE.
	Pass bool

	// The result of the evaluation. Boolean for logical expressions.
	// Calculations or string manipulations will return the appropriate type.
	Value interface{}

	// Results of evaluating the child rules.
	Results map[string]*Result

	// Diagnostic data
	Diagnostics    string
	RulesEvaluated int
}

type Value struct {
	Val interface{}
	Typ Type
}

// SummarizeResults produces a list of rules (including child rules) executed and the result of the evaluation.
// n[0] is the indent level, passed as a variadic solely to allow callers to omit it
func SummarizeResults(r *Result, n ...int) string {
	s := strings.Builder{}

	if len(n) == 0 {
		s.WriteString("\n---------- Result Diagnostic --------------------------\n")
		s.WriteString("                                         Pass Chil-\n")
		s.WriteString("Rule                                     Fail dren Value\n")
		s.WriteString("--------------------------------------------------------\n")
		n = append(n, 0)
	}
	indent := strings.Repeat(" ", (n[0]))
	boolString := "PASS"
	if !r.Pass {
		boolString = "FAIL"
	}
	s.WriteString(fmt.Sprintf("%-40s %-4s %4d %v\n", fmt.Sprintf("%s%s", indent, r.RuleID), boolString, len(r.Results), r.Value))
	for _, c := range r.Results {
		s.WriteString(SummarizeResults(c, n[0]+1))
	}
	return s.String()
}
