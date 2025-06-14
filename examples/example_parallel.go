package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
)

func main() {
	ctx := context.Background()
	engine := indigo.NewEngine(cel.NewEvaluator())

	// Create a parent rule with many child rules
	parent := indigo.NewRule("parent", "true")

	// Add 100 child rules
	for i := 0; i < 10000; i++ {
		child := indigo.NewRule(fmt.Sprintf("child_%d", i), "true")
		parent.Rules[child.ID] = child
	}

	// Compile the rules
	err := engine.Compile(parent)
	if err != nil {
		panic(err)
	}

	data := map[string]interface{}{}

	// Test sequential evaluation
	start := time.Now()
	result1, err := engine.Eval(ctx, parent, data)
	if err != nil {
		panic(err)
	}
	sequentialTime := time.Since(start)

	// Test parallel evaluation with batch size 10
	start = time.Now()
	result2, err := engine.Eval(ctx, parent, data, indigo.EnableParallel(1000))
	if err != nil {
		panic(err)
	}
	parallelTime := time.Since(start)

	fmt.Printf("Sequential evaluation: %v\n", sequentialTime)
	fmt.Printf("Parallel evaluation:   %v\n", parallelTime)
	fmt.Printf("Speedup factor:        %.2fx\n", float64(sequentialTime)/float64(parallelTime))

	fmt.Printf("Results match: %v\n", result1.Pass == result2.Pass && len(result1.Results) == len(result2.Results))
}
