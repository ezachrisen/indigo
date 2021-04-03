package indigo

import (
	"fmt"
	"strings"

	"github.com/ezachrisen/indigo/schema"
)

// Result of evaluating a rule.
type Result struct {
	// The Rule that was evaluated
	RuleID string

	// Reference to the Rule.Meta value set when the rule was evaluated.
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
	Diagnostics string

	// A count of the number of rules evaluated, including the parent rule
	// and any children
	RulesEvaluated int
}

// The value returned in the Result.
// Inspect the Typ to determine what it is.
type Value struct {
	Val interface{}
	Typ schema.Type
}

// SummarizeResults produces a list of rules (including child rules) executed and the result of the evaluation.
func (u *Result) Summarize() string {
	return u.summarizeResults(0)
}

func (u *Result) summarizeResults(n int) string {
	s := strings.Builder{}

	if n == 0 {
		s.WriteString("\n---------- Result Diagnostic --------------------------\n")
		s.WriteString("                                         Pass Chil-\n")
		s.WriteString("Rule                                     Fail dren Value\n")
		s.WriteString("--------------------------------------------------------\n")
	}
	indent := strings.Repeat(" ", n)
	boolString := "PASS"
	if !u.Pass {
		boolString = "FAIL"
	}
	s.WriteString(fmt.Sprintf("%-40s %-4s %4d %v\n", fmt.Sprintf("%s%s", indent, u.RuleID), boolString, len(u.Results), u.Value))
	for _, c := range u.Results {
		s.WriteString(c.summarizeResults(n + 1))
	}
	return s.String()
}