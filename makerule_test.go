package indigo_test

import "github.com/ezachrisen/indigo"

// -------------------------------------------------- RULE CREATION HELPERS
// This file has functions that make rules.
// The rules are reused in multiple tests.
//

// Make a rule that incldues a reference to a "self" value
func makeRuleWithSelf(id string) *indigo.Rule {

	return &indigo.Rule{
		ID:   id,
		Expr: `true`,
		Rules: map[string]*indigo.Rule{
			"a": &indigo.Rule{
				ID:   "a",
				Expr: `self`,
				Self: 22,
				Rules: map[string]*indigo.Rule{
					"a1": &indigo.Rule{
						ID:   "a1",
						Expr: `self`,
					},
				},
			},
		},
	}
}

// Make a nested rule tree where the rules
// do not have any evaluation options set locally
func makeRuleNoOptions() *indigo.Rule {
	rule1 := &indigo.Rule{
		ID:   "rule1",
		Expr: `true`,
		Rules: map[string]*indigo.Rule{
			"D": &indigo.Rule{
				ID:   "D",
				Expr: `true`,
				Rules: map[string]*indigo.Rule{
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
				Rules: map[string]*indigo.Rule{
					"b1": {
						ID:   "b1",
						Expr: `true`,
					},
					"b2": {
						ID:   "b2",
						Expr: `false`,
					},
					"b3": {
						ID:   "b3",
						Expr: `true`,
					},
					"b4": {
						ID:   "b4",
						Expr: `false`,
						Rules: map[string]*indigo.Rule{
							"b4-1": {
								ID:   "b4-1",
								Expr: `true`,
							},
							"b4-2": {
								ID:   "b4-2",
								Expr: `false`,
							},
						},
					},
				},
			},
			"E": {
				ID:   "E",
				Expr: `false`,
				Rules: map[string]*indigo.Rule{
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

// Make a nested rule tree where some rules have local
// evaluation options set
func makeRuleWithOptions() *indigo.Rule {
	rule1 := &indigo.Rule{
		ID:   "rule1",
		Expr: `true`,
		Rules: map[string]*indigo.Rule{
			"D": {
				ID:   "D",
				Expr: `true`,
				Rules: map[string]*indigo.Rule{
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
				ID:      "B",
				Expr:    `false`,
				Options: indigo.RuleOptions{DiscardPass: true},
				Rules: map[string]*indigo.Rule{
					"b1": {
						ID:   "b1",
						Expr: `true`,
					},
					"b2": {
						ID:   "b2",
						Expr: `false`,
					},
					"b3": {
						ID:   "b3",
						Expr: `true`,
					},
					"b4": {
						ID:      "b4",
						Expr:    `false`,
						Options: indigo.RuleOptions{DiscardPass: true},
						Rules: map[string]*indigo.Rule{
							"b4-1": {
								ID:   "b4-1",
								Expr: `true`,
							},
							"b4-2": {
								ID:   "b4-2",
								Expr: `false`,
							},
						},
					},
				},
			},
			"E": {
				ID:      "E",
				Expr:    `false`,
				Options: indigo.RuleOptions{StopIfParentNegative: true},
				Rules: map[string]*indigo.Rule{
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

// SortRulesAlpha will sort rules alphabetically by their rule ID
func sortRulesAlpha(rules []*indigo.Rule, i, j int) bool {
	return rules[i].ID < rules[j].ID
}

// Make a nested rule tree where some rules have local
// evaluation options set
func makeRuleSorted() *indigo.Rule {
	rule1 := &indigo.Rule{
		ID:      "rule1",
		Expr:    `true`,
		Options: indigo.RuleOptions{SortFunc: sortRulesAlpha},
		Rules: map[string]*indigo.Rule{
			"D": {
				ID:      "D",
				Expr:    `true`,
				Options: indigo.RuleOptions{SortFunc: sortRulesAlpha},
				Rules: map[string]*indigo.Rule{
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
				ID:      "B",
				Expr:    `false`,
				Options: indigo.RuleOptions{SortFunc: sortRulesAlpha},
				Rules: map[string]*indigo.Rule{
					"b1": {
						ID:   "b1",
						Expr: `true`,
					},
					"b2": {
						ID:   "b2",
						Expr: `false`,
					},
					"b3": {
						ID:   "b3",
						Expr: `true`,
					},
					"b4": {
						ID:      "b4",
						Expr:    `false`,
						Options: indigo.RuleOptions{SortFunc: sortRulesAlpha},
						Rules: map[string]*indigo.Rule{
							"b4-1": {
								ID:   "b4-1",
								Expr: `true`,
							},
							"b4-2": {
								ID:   "b4-2",
								Expr: `false`,
							},
						},
					},
				},
			},
			"E": {
				ID:      "E",
				Expr:    `false`,
				Options: indigo.RuleOptions{SortFunc: sortRulesAlpha},
				Rules: map[string]*indigo.Rule{
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
