package indigo_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
)

func TestShard(t *testing.T) {
	schema := &indigo.Schema{
		ID: "x",
		Elements: []indigo.DataElement{
			{Name: "school", Type: indigo.String{}},
			{Name: "nationality", Type: indigo.String{}},
			{Name: "class", Type: indigo.Int{}},
			{Name: "gpa", Type: indigo.Float{}},
		},
	}

	root := indigo.NewRule("root", "")

	centralHSHonors := indigo.NewRule("centralHonors", `school =="Central" && class == 2026 && gpa > 3.5`)
	centralAtRisk := indigo.NewRule("centralAtRisk", `school =="Central" && class == 2026 && gpa < 2.5`)
	root.Add(centralHSHonors)
	root.Add(centralAtRisk)

	woodlawnHSHonors := indigo.NewRule("woodlawnHonors", `school =="woodlawn" && class == 2026 && gpa > 3.7`)
	woodlawnAtRisk := indigo.NewRule("woodlawnAtRisk", `school =="woodlawn" && class == 2026 && gpa < 2.0`)
	woodlawnForeignHonors := indigo.NewRule("woodlawnForeignHonors", `school =="woodlawn" && class == 2026 && gpa > 3.7 && nationality != "US"`)
	woodlawnForeignAtRisk := indigo.NewRule("woodlawnHonors", `school =="woodlawn" && class == 2026 && gpa < 2.0 && nationality != "US"`)

	root.Add(woodlawnHSHonors)
	root.Add(woodlawnAtRisk)
	root.Add(woodlawnForeignHonors)
	root.Add(woodlawnForeignAtRisk)

	eastHSHonors := indigo.NewRule("eastHonors", `school =="east" && class == 2026 && gpa > 3.3`)
	eastAtRisk := indigo.NewRule("eastAtRisk", `school =="east" && class == 2026 && gpa < 2.2`)
	root.Add(eastHSHonors)
	root.Add(eastAtRisk)

	e := indigo.NewEngine(cel.NewEvaluator(cel.FixedSchema(schema)))
	err := e.Compile(root, indigo.CollectDiagnostics(true))
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
	for _, rr := range res.Results {
		fmt.Println(rr.Rule.ID)
	}
	if _, ok := res.Results["centralHonors"]; !ok {
		t.Error("expected centralHonors")
	}

	ev := evaluated(res)
	if _, ok := ev["woodlawnHonors"]; ok {
		t.Error("woodlawn honors should not have been evaluated for a central student")
	}
}

func TestShard2(t *testing.T) {
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
	for range 100 {

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
		root.Add(generic)
		debugLogf(t, "Before sharding:\n%s\n", root)
		err := root.BuildShards()
		if err != nil {
			t.Fatal(err)
		}
		wantTree := `root
├── central
│   ├── centralAtRisk
│   └── centralHonors
├── default
│   └── anyAtRisk
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
		gotTree := root.Tree()

		wantTree = strings.TrimSpace(wantTree)
		gotTree = strings.TrimSpace(gotTree)
		if gotTree != wantTree {
			t.Errorf("Wanted \n%s\n\nGot\n%s\n", wantTree, gotTree)
		}
		// }
		debugLogf(t, "After sharding:\n%s\n", root)
		e := indigo.NewEngine(cel.NewEvaluator(cel.FixedSchema(schema)))
		err = e.Compile(root, indigo.CollectDiagnostics(true))
		if err != nil {
			t.Fatal(err)
		}
	}
	// t.FailNow()
	// d := map[string]any{
	// 	"school":      "Central",
	// 	"class":       2026,
	// 	"gpa":         3.7,
	// 	"nationality": "US",
	// }
	//
	// res, err := e.Eval(context.Background(), root, d, indigo.ReturnDiagnostics(true))
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// if _, ok := res.Results["centralHonors"]; !ok {
	// 	t.Error("expected centralHonors")
	// }
	//
	// ev := evaluated(res)
	// if _, ok := ev["woodlawnHonors"]; ok {
	// 	t.Error("woodlawn honors should not have been evaluated for a central student")
	// }
}
