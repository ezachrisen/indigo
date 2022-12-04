package indigo_test

import "github.com/ezachrisen/indigo"

// -------------------------------------------------- RULE CREATION HELPERS
// This file has functions that make rules.
// The rules are reused in multiple tests.
//
// Make a nested rule tree where the rules
// do not have any evaluation options set locally

// ┌───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
// │                                                                                                                       │
// │ INDIGO RESULT SUMMARY                                                                                                 │
// │                                                                                                                       │
// ├────────────┬───────┬───────┬───────┬────────┬─────────────┬─────────────┬────────────┬────────────┬─────────┬─────────┤
// │            │ Pass/ │ Expr. │ Chil- │ Output │ Diagnostics │ Stop If     │ Stop First │ Stop First │ Discard │ Discard │
// │ Rule       │ Fail  │ Pass/ │ dren  │ Value  │ Available?  │ Parent Neg. │ Pos. Child │ Neg. Child │ Pass    │ Fail    │
// │            │       │ Fail  │       │        │             │             │            │            │         │         │
// ├────────────┼───────┼───────┼───────┼────────┼─────────────┼─────────────┼────────────┼────────────┼─────────┼─────────┤
// │ rule1      │ FAIL  │ PASS  │ 3     │ true   │             │             │            │            │         │         │
// │   B        │ FAIL  │ FAIL  │ 4     │ false  │             │             │            │            │         │         │
// │     b1     │ PASS  │ PASS  │ 0     │ true   │             │             │            │            │         │         │
// │     b2     │ FAIL  │ FAIL  │ 0     │ false  │             │             │            │            │         │         │
// │     b3     │ PASS  │ PASS  │ 0     │ true   │             │             │            │            │         │         │
// │     b4     │ FAIL  │ FAIL  │ 2     │ false  │             │             │            │            │         │         │
// │       b4-1 │ PASS  │ PASS  │ 0     │ true   │             │             │            │            │         │         │
// │       b4-2 │ FAIL  │ FAIL  │ 0     │ false  │             │             │            │            │         │         │
// │   E        │ FAIL  │ FAIL  │ 3     │ false  │             │             │            │            │         │         │
// │     e1     │ PASS  │ PASS  │ 0     │ true   │             │             │            │            │         │         │
// │     e2     │ FAIL  │ FAIL  │ 0     │ false  │             │             │            │            │         │         │
// │     e3     │ PASS  │ PASS  │ 0     │ true   │             │             │            │            │         │         │
// │   D        │ FAIL  │ PASS  │ 3     │ true   │             │             │            │            │         │         │
// │     d3     │ PASS  │ PASS  │ 0     │ true   │             │             │            │            │         │         │
// │     d1     │ PASS  │ PASS  │ 0     │ true   │             │             │            │            │         │         │
// │     d2     │ FAIL  │ FAIL  │ 0     │ false  │             │             │            │            │         │         │
// └────────────┴───────┴───────┴───────┴────────┴─────────────┴─────────────┴────────────┴────────────┴─────────┴─────────┘

func makeRule() *indigo.Rule {
	return &indigo.Rule{
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
}
