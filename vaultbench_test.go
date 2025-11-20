// vault_bench_test.go
package indigo_test

import (
	"testing"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
)

// buildTree creates a balanced tree with total node count ≈ totalNodes
func buildTree(totalNodes int) *indigo.Rule {
	if totalNodes < 1 {
		totalNodes = 1
	}

	root := indigo.NewRule("root", "true")
	ids := make(chan string, totalNodes)
	for i := 0; i < totalNodes; i++ {
		ids <- string(rune('A'+(i%26))) + string(rune('0'+(i/26%10))) + string(rune('0'+(i/676%10)))
	}
	close(ids)

	queue := []*indigo.Rule{root}
	currentID := <-ids
	for len(queue) > 0 && currentID != "" {
		parent := queue[0]
		queue = queue[1:]

		// Add up to 5 children per node (tunable branching factor)
		for i := 0; i < 5 && currentID != ""; i++ {
			child := indigo.NewRule(currentID, "true")
			if parent.Rules == nil {
				parent.Rules = make(map[string]*indigo.Rule)
			}
			parent.Rules[currentID] = child
			queue = append(queue, child)

			// Try to get next ID
			select {
			case id, ok := <-ids:
				if !ok {
					currentID = ""
				} else {
					currentID = id
				}
			default:
				currentID = ""
			}
		}
	}

	return root
}

func BenchmarkVault_Mutate_LargeTree_10k(b *testing.B) {
	eng := indigo.NewEngine(cel.NewEvaluator())

	// Build a ~10,000+ node tree once
	big := buildTree(12_000)
	vault, err := indigo.NewVault(eng, big)
	if err != nil {
		b.Fatal(err)
	}

	// Reset timer after setup
	b.ResetTimer()
	b.ReportAllocs() // This is crucial — shows allocs/op and MB/s

	updateRule := indigo.Rule{
		ID:   "A00", // This rule definitely exists near the top
		Expr: "1 + 1 == 2",
	}

	for i := 0; i < b.N; i++ {
		// This is the hot path we're measuring
		err := vault.Mutate(indigo.Update(updateRule))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVault_Mutate_LargeTree_10k_Move(b *testing.B) {
	eng := indigo.NewEngine(cel.NewEvaluator())
	big := buildTree(12_000)
	vault, err := indigo.NewVault(eng, big)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Move a leaf node from one parent to another
		err := vault.Mutate(indigo.Move("Z99", "A11")) // assume these exist
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVault_Mutate_LargeTree_10k_Noop(b *testing.B) {
	eng := indigo.NewEngine(cel.NewEvaluator())
	big := buildTree(12_000)
	vault, err := indigo.NewVault(eng, big)
	if err != nil {
		b.Fatal(err)
	}

	now := time.Now()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Pure timestamp update — still triggers full copy!
		vault.Mutate(indigo.LastUpdate(now))
	}
}

// Optional: Add a baseline with smaller tree to compare scaling
func BenchmarkVault_Mutate_SmallTree(b *testing.B) {
	eng := indigo.NewEngine(cel.NewEvaluator())
	small := indigo.NewRule("root", "")
	vault, _ := indigo.NewVault(eng, small)

	update := indigo.Rule{ID: "test", Expr: "true"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		vault.Mutate(indigo.Update(update))
	}
}
