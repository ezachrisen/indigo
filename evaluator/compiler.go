package evaluator

import "github.com/ezachrisen/indigo/schema"

// Compiler defines the protocol for pre-processing a rule and providing
// feedback on rule correctness.
// A rule evaluator may satisfy this interface, but Indigo does not require
// it.
type Compiler interface {
	// Compile pre-processes the rule, returning a compiled version of the rule.
	// The rule will store the compiled version, later providing it back to the
	// evaluator.
	// collectDiagnostics instructs the compiler to generate additional information
	// to help provide diagnostic information on the evaluation later.
	// dryRun performs the compilation, but doesn't store the results, mainly
	// for the purpose of checking rule correctness.
	Compile(expr string, s schema.Schema, resultType schema.Type, collectDiagnostics, dryRun bool) (interface{}, error)
}
