package indigo

// ExpressionEvaluator is the interface that wraps the Evaluate method.
// Evaluate tests the rule expression against the data.
// Returns the result of the evaluation and a string containing diagnostic information.
// Diagnostic information is only returned if explicitly requested.
// Evaluate should check the result against the expected resultType and return an error if the
// result does not match.
type ExpressionEvaluator interface {
	Evaluate(data map[string]any, expr string, s Schema,
		self any, evalData any, resultType Type, returnDiagnostics bool) (any, *Diagnostics, error)
}

// ExpressionCompiler is the interface that wraps the Compile method.
// Compile pre-processes the expression, returning a compiled version.
// The Indigo Compiler will store the compiled version, later providing it back to the
// evaluator.
//
// collectDiagnostics instructs the compiler to generate additional information
// to help provide diagnostic information on the evaluation later.
// dryRun performs the compilation, but doesn't store the results, mainly
// for the purpose of checking rule correctness.
type ExpressionCompiler interface {
	Compile(expr string, s Schema, resultType Type, collectDiagnostics, dryRun bool) (any, error)
}

// ExpressionCompilerEvaluator is the interface that groups the ExpressionCompiler
// and ExpressionEvaluator interfaces for back-end evaluators that require a compile and an evaluate step.
type ExpressionCompilerEvaluator interface {
	ExpressionCompiler
	ExpressionEvaluator
}
