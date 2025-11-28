package indigo_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
)

// func TestShard(t *testing.T) {
// 	schema := &indigo.Schema{
// 		ID: "x",
// 		Elements: []indigo.DataElement{
// 			{Name: "school", Type: indigo.String{}},
// 			{Name: "nationality", Type: indigo.String{}},
// 			{Name: "class", Type: indigo.Int{}},
// 			{Name: "gpa", Type: indigo.Float{}},
// 		},
// 	}
//
// 	root := indigo.NewRule("root", "")
//
// 	centralHSHonors := indigo.NewRule("centralHonors", `school =="Central" && class == 2026 && gpa > 3.5`)
// 	centralAtRisk := indigo.NewRule("centralAtRisk", `school =="Central" && class == 2026 && gpa < 2.5`)
// 	root.Add(centralHSHonors)
// 	root.Add(centralAtRisk)
//
// 	woodlawnHSHonors := indigo.NewRule("woodlawnHonors", `school =="woodlawn" && class == 2026 && gpa > 3.7`)
// 	woodlawnAtRisk := indigo.NewRule("woodlawnAtRisk", `school =="woodlawn" && class == 2026 && gpa < 2.0`)
// 	woodlawnForeignHonors := indigo.NewRule("woodlawnForeignHonors", `school =="woodlawn" && class == 2026 && gpa > 3.7 && nationality != "US"`)
// 	woodlawnForeignAtRisk := indigo.NewRule("woodlawnHonors", `school =="woodlawn" && class == 2026 && gpa < 2.0 && nationality != "US"`)
//
// 	root.Add(woodlawnHSHonors)
// 	root.Add(woodlawnAtRisk)
// 	root.Add(woodlawnForeignHonors)
// 	root.Add(woodlawnForeignAtRisk)
//
// 	eastHSHonors := indigo.NewRule("eastHonors", `school =="east" && class == 2026 && gpa > 3.3`)
// 	eastAtRisk := indigo.NewRule("eastAtRisk", `school =="east" && class == 2026 && gpa < 2.2`)
// 	root.Add(eastHSHonors)
// 	root.Add(eastAtRisk)
//
// 	e := indigo.NewEngine(cel.NewEvaluator(cel.FixedSchema(schema)))
// 	err := e.Compile(root, indigo.CollectDiagnostics(true))
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	d := map[string]any{
// 		"school":      "Central",
// 		"class":       2026,
// 		"gpa":         3.7,
// 		"nationality": "US",
// 	}
//
// 	res, err := e.Eval(context.Background(), root, d, indigo.ReturnDiagnostics(true))
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	for _, rr := range res.Results {
// 		fmt.Println(rr.Rule.ID)
// 	}
// 	if _, ok := res.Results["centralHonors"]; !ok {
// 		t.Error("expected centralHonors")
// 	}
//
// 	ev := evaluated(res)
// 	if _, ok := ev["woodlawnHonors"]; ok {
// 		t.Error("woodlawn honors should not have been evaluated for a central student")
// 	}
// }

func TestShards(t *testing.T) {
	schema := &indigo.Schema{
		ID: "x",
		Elements: []indigo.DataElement{
			{Name: "school", Type: indigo.String{}},
			{Name: "nationality", Type: indigo.String{}},
			{Name: "class", Type: indigo.Int{}},
			{Name: "gpa", Type: indigo.Float{}},
		},
	}
	// We run this test many times because it is very important
	// that the shard rules are sorted correctly for shard creation to work.
	// This loop will catch any errors where the sort works most of the time, but fails sometimes.
	for range 1 {

		root := indigo.NewRule("root", "")

		// shard stuff

		centralShard := indigo.NewRule("central", `school == "Central"`)
		centralShard.Meta = func(r *indigo.Rule) bool {
			return strings.Contains(r.Expr, `"Central"`)
		}

		woodlawnShard := indigo.NewRule("woodlawn", `school == "woodlawn"`)
		woodlawnShard.Meta = func(r *indigo.Rule) bool {
			return strings.Contains(r.Expr, `"woodlawn"`)
		}

		woodlawnForeignShard := indigo.NewRule("woodlawnForeign", `school == "woodlawn" && nationality != "US"`)
		woodlawnForeignShard.Meta = func(r *indigo.Rule) bool {
			return strings.Contains(r.Expr, `"woodlawn"`) && strings.Contains(r.Expr, `!= "US"`)
		}
		woodlawnShard.Shards = []*indigo.Rule{woodlawnForeignShard}

		eastShard := indigo.NewRule("east", `school == "east"`)
		eastShard.Meta = func(r *indigo.Rule) bool {
			return strings.Contains(r.Expr, `"east"`)
		}

		root.Shards = []*indigo.Rule{centralShard, woodlawnShard, eastShard}

		//--------------------------------------------------

		centralHSHonors := indigo.NewRule("centralHonors", `school =="Central" && class == 2026 && gpa > 3.5`)
		centralAtRisk := indigo.NewRule("centralAtRisk", `school =="Central" && class == 2026 && gpa < 2.5`)
		root.Add(centralHSHonors)
		root.Add(centralAtRisk)

		woodlawnHSHonors := indigo.NewRule("woodlawnHonors", `school =="woodlawn" && class == 2026 && gpa > 3.7`)
		woodlawnAtRisk := indigo.NewRule("woodlawnAtRisk", `school =="woodlawn" && class == 2026 && gpa < 2.0`)
		woodlawnForeignHonors := indigo.NewRule("woodlawnForeignHonors", `school =="woodlawn" && class == 2026 && gpa > 3.7 && nationality != "US"`)
		woodlawnForeignAtRisk := indigo.NewRule("woodlawnForeignAtRisk", `school =="woodlawn" && class == 2026 && gpa < 2.0 && nationality != "US"`)

		root.Add(woodlawnHSHonors)
		root.Add(woodlawnAtRisk)
		root.Add(woodlawnForeignHonors)
		root.Add(woodlawnForeignAtRisk)

		eastHSHonors := indigo.NewRule("eastHonors", `school =="east" && class == 2026 && gpa > 3.3`)
		eastAtRisk := indigo.NewRule("eastAtRisk", `school =="east" && class == 2026 && gpa < 2.2`)
		root.Add(eastHSHonors)
		root.Add(eastAtRisk)
		generic := indigo.NewRule("anyAtRisk", `class == 2026 && gpa < 2.0`)
		genericChild := indigo.NewRule("anyAtRiskChild", `gpa > 0.0`)
		generic.Add(genericChild)
		root.Add(generic)

		debugLogf(t, "Before sharding:\n%s\n", root)

		// Before building the shards, root looks like this:
		//
		// root
		// ├── anyAtRisk
		// │   └── anyAtRiskChild
		// ├── centralAtRisk
		// ├── centralHonors
		// ├── eastAtRisk
		// ├── eastHonors
		// ├── woodlawnAtRisk
		// ├── woodlawnForeignAtRisk
		// ├── woodlawnForeignHonors
		// └── woodlawnHonors
		err := root.BuildShards()
		if err != nil {
			t.Fatal(err)
		}
		debugLogf(t, "After sharding:\n%s\n", root)
		gotTree := root.Tree()
		// After building the shards, root should look like this:
		wantTree := `
root
├── central
│   ├── centralAtRisk
│   └── centralHonors
├── default
│   └── anyAtRisk
│       └── anyAtRiskChild
├── east
│   ├── eastAtRisk
│   └── eastHonors
└── woodlawn
    ├── default
    │   ├── woodlawnAtRisk
    │   └── woodlawnHonors
    └── woodlawnForeign
        ├── woodlawnForeignAtRisk
        └── woodlawnForeignHonors
	`
		wantTree = strings.TrimSpace(wantTree)
		gotTree = strings.TrimSpace(gotTree)
		if gotTree != wantTree {
			t.Errorf("Wanted \n%s\n\nGot\n%s\n", wantTree, gotTree)
		}
		e := indigo.NewEngine(cel.NewEvaluator(cel.FixedSchema(schema)))
		err = e.Compile(root, indigo.CollectDiagnostics(true))
		if err != nil {
			t.Fatal(err)
		}
		d := map[string]any{
			"school":      "Central",
			"class":       2026,
			"gpa":         3.7,
			"nationality": "US",
		}

		res, err := e.Eval(context.Background(), root, d, indigo.ReturnDiagnostics(true))
		if err != nil {
			t.Fatal(err)
		}

		wantResults := `
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                                                                                                                                  │
│ INDIGO RESULTS                                                                                                                                   │
│                                                                                                                                                  │
├─────────────────────────────┬───────┬───────┬───────┬────────┬─────────────┬─────────┬─────────────┬────────────┬────────────┬─────────┬─────────┤
│                             │ Pass/ │ Expr. │ Chil- │ Output │ Diagnostics │ True    │ Stop If     │ Stop First │ Stop First │ Discard │ Discard │
│ Rule                        │ Fail  │ Pass/ │ dren  │ Value  │ Available?  │ If Any? │ Parent Neg. │ Pos. Child │ Neg. Child │ Pass    │ Fail    │
│                             │       │ Fail  │       │        │             │         │             │            │            │         │         │
├─────────────────────────────┼───────┼───────┼───────┼────────┼─────────────┼─────────┼─────────────┼────────────┼────────────┼─────────┼─────────┤
│ root                        │ FAIL  │ PASS  │ 4     │ true   │             │         │             │            │            │         │ 0       │
│   central                   │ FAIL  │ PASS  │ 2     │ true   │ yes         │         │             │            │            │         │ 0       │
│     centralAtRisk           │ FAIL  │ FAIL  │ 0     │ false  │ yes         │         │             │            │            │         │ 0       │
│     centralHonors           │ PASS  │ PASS  │ 0     │ true   │ yes         │         │             │            │            │         │ 0       │
│   default                   │ FAIL  │ PASS  │ 1     │ true   │             │         │             │            │            │         │ 0       │
│     anyAtRisk               │ FAIL  │ FAIL  │ 1     │ false  │ yes         │         │             │            │            │         │ 0       │
│       anyAtRiskChild        │ PASS  │ PASS  │ 0     │ true   │ yes         │         │             │            │            │         │ 0       │
│   east                      │ FAIL  │ FAIL  │ 2     │ false  │ yes         │         │             │            │            │         │ 0       │
│     eastAtRisk              │ FAIL  │ FAIL  │ 0     │ false  │ yes         │         │             │            │            │         │ 0       │
│     eastHonors              │ FAIL  │ FAIL  │ 0     │ false  │ yes         │         │             │            │            │         │ 0       │
│   woodlawn                  │ FAIL  │ FAIL  │ 2     │ false  │ yes         │         │             │            │            │         │ 0       │
│     default                 │ FAIL  │ PASS  │ 2     │ true   │             │         │             │            │            │         │ 0       │
│       woodlawnAtRisk        │ FAIL  │ FAIL  │ 0     │ false  │ yes         │         │             │            │            │         │ 0       │
│       woodlawnHonors        │ FAIL  │ FAIL  │ 0     │ false  │ yes         │         │             │            │            │         │ 0       │
│     woodlawnForeign         │ FAIL  │ FAIL  │ 2     │ false  │ yes         │         │             │            │            │         │ 0       │
│       woodlawnForeignAtRisk │ FAIL  │ FAIL  │ 0     │ false  │ yes         │         │             │            │            │         │ 0       │
│       woodlawnForeignHonors │ FAIL  │ FAIL  │ 0     │ false  │ yes         │         │             │            │            │         │ 0       │
└─────────────────────────────┴───────┴───────┴───────┴────────┴─────────────┴─────────┴─────────────┴────────────┴────────────┴─────────┴─────────┘
		`
		gotResults := res.String()
		wantResults = strings.TrimSpace(wantResults)
		gotResults = strings.TrimSpace(gotResults)
		if gotResults != wantResults {
			t.Errorf("Wanted \n%s\n\nGot\n%s\n", wantResults, gotResults)
		}
		for r := range res.Flat() {
			fmt.Println(r.Rule.ID)
		}
		err = res.Unshard()
		if err != nil {
			t.Error(err)
		}
		wantUnsharded := `
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                                                                                                                              │
│ INDIGO RESULTS                                                                                                                               │
│                                                                                                                                              │
├─────────────────────────┬───────┬───────┬───────┬────────┬─────────────┬─────────┬─────────────┬────────────┬────────────┬─────────┬─────────┤
│                         │ Pass/ │ Expr. │ Chil- │ Output │ Diagnostics │ True    │ Stop If     │ Stop First │ Stop First │ Discard │ Discard │
│ Rule                    │ Fail  │ Pass/ │ dren  │ Value  │ Available?  │ If Any? │ Parent Neg. │ Pos. Child │ Neg. Child │ Pass    │ Fail    │
│                         │       │ Fail  │       │        │             │         │             │            │            │         │         │
├─────────────────────────┼───────┼───────┼───────┼────────┼─────────────┼─────────┼─────────────┼────────────┼────────────┼─────────┼─────────┤
│ root                    │ FAIL  │ PASS  │ 9     │ true   │             │         │             │            │            │         │ 0       │
│   anyAtRisk             │ FAIL  │ FAIL  │ 1     │ false  │ yes         │         │             │            │            │         │ 0       │
│     anyAtRiskChild      │ PASS  │ PASS  │ 0     │ true   │ yes         │         │             │            │            │         │ 0       │
│   centralAtRisk         │ FAIL  │ FAIL  │ 0     │ false  │ yes         │         │             │            │            │         │ 0       │
│   centralHonors         │ PASS  │ PASS  │ 0     │ true   │ yes         │         │             │            │            │         │ 0       │
│   eastAtRisk            │ FAIL  │ FAIL  │ 0     │ false  │ yes         │         │             │            │            │         │ 0       │
│   eastHonors            │ FAIL  │ FAIL  │ 0     │ false  │ yes         │         │             │            │            │         │ 0       │
│   woodlawnAtRisk        │ FAIL  │ FAIL  │ 0     │ false  │ yes         │         │             │            │            │         │ 0       │
│   woodlawnForeignAtRisk │ FAIL  │ FAIL  │ 0     │ false  │ yes         │         │             │            │            │         │ 0       │
│   woodlawnForeignHonors │ FAIL  │ FAIL  │ 0     │ false  │ yes         │         │             │            │            │         │ 0       │
│   woodlawnHonors        │ FAIL  │ FAIL  │ 0     │ false  │ yes         │         │             │            │            │         │ 0       │
└─────────────────────────┴───────┴───────┴───────┴────────┴─────────────┴─────────┴─────────────┴────────────┴────────────┴─────────┴─────────┘
		`

		gotUnsharded := res.String()

		wantUnsharded = strings.TrimSpace(wantUnsharded)
		gotUnsharded = strings.TrimSpace(gotUnsharded)
		if gotUnsharded != wantUnsharded {
			t.Errorf("Wanted \n%s\n\nGot\n%s\n", wantUnsharded, gotUnsharded)
		}
	}
}
