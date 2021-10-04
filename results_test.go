package indigo_test

import (
	"fmt"

	"github.com/ezachrisen/indigo"
)

// --------------------------------------------------
// Functions to manipulate and compare rule evaluation results
// and expected results

// flattenResultsExprResult takes a hierarchy of Result objects and flattens it
// to a map of rule ID to pass/fail. This is so that it's easy to
// compare the results to expected.
func flattenResultsExprResult(result *indigo.Result) map[string]bool {
	m := map[string]bool{}
	m[result.Rule.ID] = result.ExpressionPass
	for k := range result.Results {
		r := result.Results[k]
		mc := flattenResultsExprResult(r)
		for k := range mc {
			m[k] = mc[k]
		}
	}
	return m
}

// flattenResultsExprResult takes a hierarchy of Result objects and flattens it
// to a map of rule ID to pass/fail. This is so that it's easy to
// compare the results to expected.
func flattenResultsRuleResult(result *indigo.Result) map[string]bool {
	m := map[string]bool{}
	m[result.Rule.ID] = result.Pass
	for k := range result.Results {
		r := result.Results[k]
		mc := flattenResultsRuleResult(r)
		for k := range mc {
			m[k] = mc[k]
		}
	}
	return m
}

// flattenResults takes a hierarchy of Result objects and flattens it
// to a map of rule ID to pass/fail. This is so that it's easy to
// compare the results to expected.
func flattenResultsEvaluated(result *indigo.Result) []string {
	m := []string{}
	m = append(m, result.Rule.ID)
	for _, c := range result.RulesEvaluated {
		u := result.Results[c.ID]
		//		fmt.Println("About to flatten reuslts for u=", c.ID, "in parent ", result.Rule.ID, "u=", u)
		mc := flattenResultsEvaluated(u)
		m = append(m, mc...)
	}
	return m
}

// flattenResults takes a hierarchy of Result objects and flattens it
// to a map of rule ID to pass/fail. This is so that it's easy to
// compare the results to expected.
func flattenResultsDiagnostics(result *indigo.Result) map[string]*indigo.Diagnostics {
	m := map[string]*indigo.Diagnostics{}
	m[result.Rule.ID] = result.Diagnostics
	for k := range result.Results {
		r := result.Results[k]
		mc := flattenResultsDiagnostics(r)
		for ck := range mc {
			m[ck] = mc[ck]
		}
	}
	return m
}

// anyNotEmpty checks if any of the flattened result diagnostics are missing
// Input is the result of flattenResultsDiagnostics
func anyNotEmpty(m map[string]*indigo.Diagnostics) error {
	for k, v := range m {
		if v != nil {
			return fmt.Errorf("diagnostics for rule '%s' is not empty", k)
		}
	}
	return nil
}

// allNotEmpty checks if all entries in the map are populated
// Input is the result of flattenResultsDiagnostics
func allNotEmpty(m map[string]*indigo.Diagnostics) error {
	for k, v := range m {
		if v == nil {
			return fmt.Errorf("diagnostics missing for rule '%s'", k)
		}
	}
	return nil
}

// Match compares the results to expected results.
// Call flattenResults on the *indigo.Result first.
func match(result map[string]bool, expected map[string]bool) error {
	for k, v := range result {
		ev, ok := expected[k]
		if !ok {
			return fmt.Errorf("received result for rule %s ( %v ); no result was expected", k, v)
		}

		if v != ev {
			return fmt.Errorf("result mismatch: rule %s: got %v, wanted %v", k, v, ev)
		}
	}

	for k := range expected {
		if _, ok := result[k]; !ok {
			return fmt.Errorf("expected result for rule %s: no result found", k)
		}
	}

	return nil
}

// copyMap duplicates an expected results map (rule ID to rule output)
func copyMap(m map[string]bool) map[string]bool {

	n := make(map[string]bool, len(m))

	for k, v := range m {
		n[k] = v
	}
	return n
}

// deleteKeys removes entries from an expected results map
func deleteKeys(m map[string]bool, keys ...string) map[string]bool {

	for _, k := range keys {
		delete(m, k)
	}
	return m
}
