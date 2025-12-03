package indigo_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"github.com/google/go-cmp/cmp"
)

// This test sets up a rule hierarchy with shard specifications on the root
// and a child rule. It then calls BuildShards, and verifies that the new
// rule structure matches the shard specification. We then evaluate the rules
// and verify that the results have the correct structure, and that the functions
// used to "unshard" results give the expected results.
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

		//--------------------------------------------------------------------------------
		// SETUP

		// Normal rules
		root := indigo.NewRule("root", "")

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

		// Let's define some shards
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
		//
		//	Attach the shards to the rule
		root.Shards = []*indigo.Rule{centralShard, woodlawnShard, eastShard}

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

		//--------------------------------------------------------------------------------
		// Apply the shards to the rule

		err := root.BuildShards()
		if err != nil {
			t.Fatal(err)
		}
		debugLogf(t, "After sharding:\n%s\n", root)
		gotTree := root.Tree()

		// After building the shards, root should look like this:
		wantTree := `
root
├── central (*)
│   ├── centralAtRisk
│   └── centralHonors
├── default (*)
│   └── anyAtRisk
│       └── anyAtRiskChild
├── east (*)
│   ├── eastAtRisk
│   └── eastHonors
└── woodlawn (*)
    ├── default (*)
    │   ├── woodlawnAtRisk
    │   └── woodlawnHonors
    └── woodlawnForeign (*)
        ├── woodlawnForeignAtRisk
        └── woodlawnForeignHonors
	`
		compareStrings(wantTree, gotTree, t)

		//--------------------------------------------------------------------------------
		// Evaluate the rule

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

		//--------------------------------------------------------------------------------
		// Check the results

		// Note that all shard rules under root are evaluated.
		// The central rules are evaluated because the student data we used was for a Central high school student.
		// The default shard and its children are also evaluated; the default shard is ALWAYS evaluated.
		// The east and woodlawn child rules are not evaluated since the shard east and woodlawn shard rules prevent it.
		// Viewing the results like this exposes the shards in the results.
		wantResults := `
┌───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                                                                                                                           │
│ INDIGO RESULTS                                                                                                                            │
│                                                                                                                                           │
├──────────────────────┬───────┬───────┬───────┬────────┬─────────────┬─────────┬─────────────┬────────────┬────────────┬─────────┬─────────┤
│                      │ Pass/ │ Expr. │ Chil- │ Output │ Diagnostics │ True    │ Stop If     │ Stop First │ Stop First │ Discard │ Discard │
│ Rule                 │ Fail  │ Pass/ │ dren  │ Value  │ Available?  │ If Any? │ Parent Neg. │ Pos. Child │ Neg. Child │ Pass    │ Fail    │
│                      │       │ Fail  │       │        │             │         │             │            │            │         │         │
├──────────────────────┼───────┼───────┼───────┼────────┼─────────────┼─────────┼─────────────┼────────────┼────────────┼─────────┼─────────┤
│ root                 │ FAIL  │ PASS  │ 4     │ true   │             │         │             │            │            │         │ 0       │
│   central            │ FAIL  │ PASS  │ 2     │ true   │ yes         │         │ yes         │            │            │         │ 0       │
│     centralAtRisk    │ FAIL  │ FAIL  │ 0     │ false  │ yes         │         │             │            │            │         │ 0       │
│     centralHonors    │ PASS  │ PASS  │ 0     │ true   │ yes         │         │             │            │            │         │ 0       │
│   default            │ FAIL  │ PASS  │ 1     │ true   │             │         │             │            │            │         │ 0       │
│     anyAtRisk        │ FAIL  │ FAIL  │ 1     │ false  │ yes         │         │             │            │            │         │ 0       │
│       anyAtRiskChild │ PASS  │ PASS  │ 0     │ true   │ yes         │         │             │            │            │         │ 0       │
│   east               │ FAIL  │ FAIL  │ 0     │ false  │ yes         │         │ yes         │            │            │         │ 0       │
│   woodlawn           │ FAIL  │ FAIL  │ 0     │ false  │ yes         │         │ yes         │            │            │         │ 0       │
└──────────────────────┴───────┴───────┴───────┴────────┴─────────────┴─────────┴─────────────┴────────────┴────────────┴─────────┴─────────┘
		`
		gotResults := res.String()
		compareStrings(wantResults, gotResults, t)

		// We can also remove the shard rules from the results, leaving the original structure. This preserves any parent/child relationships
		// in the original rule, but omits the shard rules.
		err = res.Unshard()
		if err != nil {
			t.Error(err)
		}
		wantUnsharded := `
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                                                                                                                         │
│ INDIGO RESULTS                                                                                                                          │
│                                                                                                                                         │
├────────────────────┬───────┬───────┬───────┬────────┬─────────────┬─────────┬─────────────┬────────────┬────────────┬─────────┬─────────┤
│                    │ Pass/ │ Expr. │ Chil- │ Output │ Diagnostics │ True    │ Stop If     │ Stop First │ Stop First │ Discard │ Discard │
│ Rule               │ Fail  │ Pass/ │ dren  │ Value  │ Available?  │ If Any? │ Parent Neg. │ Pos. Child │ Neg. Child │ Pass    │ Fail    │
│                    │       │ Fail  │       │        │             │         │             │            │            │         │         │
├────────────────────┼───────┼───────┼───────┼────────┼─────────────┼─────────┼─────────────┼────────────┼────────────┼─────────┼─────────┤
│ root               │ FAIL  │ PASS  │ 3     │ true   │             │         │             │            │            │         │ 0       │
│   anyAtRisk        │ FAIL  │ FAIL  │ 1     │ false  │ yes         │         │             │            │            │         │ 0       │
│     anyAtRiskChild │ PASS  │ PASS  │ 0     │ true   │ yes         │         │             │            │            │         │ 0       │
│   centralAtRisk    │ FAIL  │ FAIL  │ 0     │ false  │ yes         │         │             │            │            │         │ 0       │
│   centralHonors    │ PASS  │ PASS  │ 0     │ true   │ yes         │         │             │            │            │         │ 0       │
└────────────────────┴───────┴───────┴───────┴────────┴─────────────┴─────────┴─────────────┴────────────┴────────────┴─────────┴─────────┘
`
		gotUnsharded := res.String()
		compareStrings(wantUnsharded, gotUnsharded, t)

		// We can also view the results in in a "flat" way, where all returned rules are available via an iterator, but shard rules are omitted from the results
		wantFlat := `
root
centralHonors
centralAtRisk
anyAtRisk
anyAtRiskChild
		`
		flat := []string{}
		for r := range res.Flat() {
			flat = append(flat, r.Rule.ID)
		}
		gotFlat := strings.Join(flat, "\n")
		compareStrings(wantFlat, gotFlat, t)
	}
}

// In this test we create a rule hierarchy and place it in a Vault.
// We verify that the Vault correctly applied the shard specification, and that the
// rule hierarchy of the rule in the vault is correct.
func TestVaultShards(t *testing.T) {
	schema := &indigo.Schema{
		ID: "x",
		Elements: []indigo.DataElement{
			{Name: "school", Type: indigo.String{}},
			{Name: "nationality", Type: indigo.String{}},
			{Name: "class", Type: indigo.Int{}},
			{Name: "gpa", Type: indigo.Float{}},
		},
	}

	//--------------------------------------------------------------------------------
	// SETUP

	// Normal rules
	root := indigo.NewRule("root", "")

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

	// Let's define some shards
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
	//
	//	Attach the shards to the rule
	root.Shards = []*indigo.Rule{centralShard, woodlawnShard, eastShard}

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

	//--------------------------------------------------------------------------------
	// Create the vault with the root rule and the shards.

	e := indigo.NewEngine(cel.NewEvaluator(cel.FixedSchema(schema)))
	v, err := indigo.NewVault(e, root)
	if err != nil {
		t.Fatal(err)
	}
	debugLogf(t, "After sharding:\n%s\n", v.Rule())
	gotTree := v.Rule().Tree()

	// After building the shards, root should look like this:
	wantTree := `
root
├── central (*)
│   ├── centralAtRisk
│   └── centralHonors
├── default (*)
│   └── anyAtRisk
│       └── anyAtRiskChild
├── east (*)
│   ├── eastAtRisk
│   └── eastHonors
└── woodlawn (*)
    ├── default (*)
    │   ├── woodlawnAtRisk
    │   └── woodlawnHonors
    └── woodlawnForeign (*)
        ├── woodlawnForeignAtRisk
        └── woodlawnForeignHonors
	`
	compareStrings(wantTree, gotTree, t)

	//--------------------------------------------------------------------------------
	// Add a new rule, make sure it ends up in the right shard
	// Based on the sharding specification, it shold be placed into "woodlawnForeign"

	woodlawnForeignJustOK := indigo.NewRule("woodlawnForeignJustOK", `school =="woodlawn" && class == 2026 && gpa > 2.0 && gpa < 3.7 && nationality != "US"`)

	err = v.Mutate(indigo.Add(woodlawnForeignJustOK, "root")) // <-- we can just add it to the root and let sharding place it
	if err != nil {
		t.Fatal(err)
	}
	wantTree = `
  root
├── central (*)
│   ├── centralAtRisk
│   └── centralHonors
├── default (*)
│   └── anyAtRisk
│       └── anyAtRiskChild
├── east (*)
│   ├── eastAtRisk
│   └── eastHonors
└── woodlawn (*)
    ├── default (*)
    │   ├── woodlawnAtRisk
    │   └── woodlawnHonors
    └── woodlawnForeign (*)
        ├── woodlawnForeignAtRisk
        ├── woodlawnForeignHonors
        └── woodlawnForeignJustOK
  `
	gotTree = v.Rule().Tree()

	compareStrings(wantTree, gotTree, t)
}

func compareStrings(want, got string, t *testing.T) {
	t.Helper()
	want = strings.TrimSpace(want)
	got = strings.TrimSpace(got)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
		t.Errorf("Wanted\n%s\n\nGot\n%s\n", want, got)
	}
}
