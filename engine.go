package indigo

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
)

// Compiler is the interface that wraps the Compile method.
// Compile pre-processes the rule recursively using an ExpressionCompiler, which
// is applied to each rule.
type Compiler interface {
	Compile(r *Rule, opts ...CompilationOption) error
}

// Evaluator is the interface that wraps the Evaluate method.
// Evaluate tests the rule recursively against the input data using an ExpressionEvaluator,
// which is applied to each rule.
type Evaluator interface {
	Eval(ctx context.Context, r *Rule, d map[string]any, opts ...EvalOption) (*Result, error)
}

// Engine is the interface that groups the Compiler and Evaluator interfaces.
// An Engine is used to compile and evaluate rules.
type Engine interface {
	Compiler
	Evaluator
}

// DefaultEngine provides an implementation of the Indigo Engine interface
// to evaluate rules locally.
type DefaultEngine struct {
	e ExpressionCompilerEvaluator
}

// NewEngine initializes and returns a DefaultEngine.
func NewEngine(e ExpressionCompilerEvaluator) *DefaultEngine {
	return &DefaultEngine{
		e: e,
	}
}

// Eval evaluates the expression of the rule and its children. It uses the evaluation
// options of each rule to determine what to do with the results, and whether to proceed
// evaluating. Options passed to this function will override the options set on the rules.
// Eval uses the Evaluator provided to the engine to perform the expression evaluation.
func (e *DefaultEngine) Eval(ctx context.Context, r *Rule,
	d map[string]any, opts ...EvalOption) (*Result, error) {

	if err := validateEvalArguments(r, e, d); err != nil {
		return nil, err
	}

	o := r.EvalOptions
	applyEvaluatorOptions(&o, opts...)

	d = setSelfKey(r, d, o)

	// Check for incompatible options: sortFunc and parallel cannot be used together
	if o.SortFunc != nil && o.parallel.batchSize > 1 {
		return nil, fmt.Errorf("rule %s: sortFunc and parallel options are incompatible - parallel processing cannot guarantee evaluation order", r.ID)
	}

	// Evaluate the rule's expression using the engine's ExpressionEvaluator
	val, diagnostics, err := e.e.Evaluate(d, r.Expr, r.Schema, r.Self, r.Program,
		defaultResultType(r), o.ReturnDiagnostics)
	if err != nil {
		return nil, fmt.Errorf("rule %s: %w", r.ID, err)
	}

	u := &Result{
		Rule:           r,
		ExpressionPass: true, // default boolean result
		Value:          val,
		Diagnostics:    diagnostics,
		EvalOptions:    o,
		EvalCount:      1,
	}

	// If the evaluation returned a boolean, set the Result's value,
	// otherwise keep the default, true
	if pass, ok := val.(bool); ok {
		u.ExpressionPass = pass
	}
	// By default, the rule's pass/fail is determined by the pass/fail of the
	// expression. If the rule has child rules, we'll iterate through them next
	// and change the rule's pass/fail (but not expresion pass/fail) if any child
	// rules are negative.
	u.Pass = u.ExpressionPass

	// We've been asked not to evaluate child rules if this rule failed.
	if o.StopIfParentNegative && !u.ExpressionPass {
		return u, nil
	}

	eu, err := e.evalChildren(ctx, r.sortChildRules(o.SortFunc, o.overrideSort), d, o, opts...)
	if err != nil {
		return nil, err
	}
	u.Results = eu.results
	u.RulesEvaluated = eu.evaluated
	u.EvalCount += eu.evalCount
	u.EvalParallelCount += eu.evalParallel

	// Based on the results of the child rules, determine the result of the parent rule
	switch r.EvalOptions.TrueIfAny {
	case true:
		if u.ExpressionPass {
			// If none of the child rules passed AND the parent's expression passed, the rule
			// shouldn't pass
			hasChildren := len(r.Rules) > 0
			if hasChildren && eu.passCount == 0 {
				u.Pass = false
			}
		}
	case false:
		// If one or more of child rules failed, we will fail also, regardless of the parent rule's result
		if eu.failCount > 0 {
			u.Pass = false
		}
	}

	return u, nil
}

func (e *DefaultEngine) evalChildren(ctx context.Context, rules []*Rule, d map[string]any, o EvalOptions, opts ...EvalOption) (r evalResult, err error) {
	if len(rules) == 0 {
		return
	}

	if o.parallel.batchSize <= 1 || len(rules) < o.parallel.minSize {
		return e.evalRuleSlice(ctx, rules, d, o, opts...)
	}

	if o.parallel.batchSize > math.MaxInt/2 || o.parallel.maxGoroutines > math.MaxInt/2 {
		return r, errors.New("batch size or parallel out of range")
	}

	r.results = make(map[string]*Result, len(rules))

	chunkCh := make(chan chunk)
	// Results will be sent back to the main goroutine via this channel.
	resultsCh := make(chan evalResult)
	// Errors will be sent back to the main goroutine via this channel.
	errCh := make(chan error)

	// A chunk is a range of rules (a batch) to evaluate together
	chunks := makeChunks(0, len(rules), o.parallel.batchSize)

	// Create internal context for coordinating goroutine cleanup
	// This allows us to signal all goroutines to stop when we're done,
	// separate from the user's context which might have different semantics
	// We do not want to derive this internal context from the user's context,
	// because of locking contention on the user's context in high concurrency situations.
	// Rather, we link cancellation of the internal context to cancelation of the request context
	// by canceling the internal context when this function returns, which it will if
	// the request context is cancelled (in the for/select loop below).
	internalCtx, cancelInternal := context.WithCancel(context.Background())
	defer cancelInternal() // Ensure goroutines are cleaned up when function exits

	// Start producer goroutine: sends chunks to workers on chunkCh
	// This goroutine will close chunkCh when all chunks are sent
	go sendChunks(internalCtx, chunks, chunkCh)

	// Start consumer goroutine: processes chunks and sends results
	// This goroutine reads from chunkCh and evaluates each chunk
	for i := 0; i < o.parallel.maxGoroutines; i++ {
		go func() {
			e.processChunks(internalCtx, chunkCh, rules, d, resultsCh, errCh, o, opts...)
		}()
	}

	// Collect results from all chunks
	// We know exactly how many results to expect (one per chunk)
	numChunks := len(chunks)
	for range numChunks {
		// Use select to handle multiple channel operations non-blockingly
		select {
		case <-ctx.Done():
			// User cancelled or timeout occurred - stop immediately
			// The defer cancelInternal() will clean up our goroutines
			return r, ctx.Err()
		case res := <-resultsCh:
			// Got results from a chunk - add them to our matches
			// Note: append is safe here because we're the only goroutine modifying matches
			r.evaluated = append(r.evaluated, res.evaluated...)
			r.failCount += res.failCount
			r.passCount += res.passCount
			r.evalCount += res.evalCount
			r.evalParallel += res.evalCount // we're in parallel mode, so we count the number of rules evaluated
			maps.Copy(r.results, res.results)
		case err := <-errCh:
			// A worker encountered an error - stop processing and return it
			return r, err
		}
	}

	// All chunks processed successfully
	return

}

// evalResult is a convenience type to let us pass multiple values on a channel
type evalResult struct {
	passCount         int
	failCount         int
	evalCount         int
	evalParallel      int
	evalParallelLocal int
	results           map[string]*Result
	evaluated         []*Rule
}

// makeChunks divides a range [start, len) into batches of the specified size.
// It returns a slice of chunks, where each chunk represents a contiguous range
// with start and end positions. The last chunk may be smaller than batchSize
// if the remaining range is not evenly divisible.
//
// This function uses recursion to build the chunk list, which is an interesting
// approach but not necessarily the most efficient for large datasets. An iterative
// approach might be more suitable for production code, but recursion demonstrates
// functional programming concepts in Go.
//
// Parameters:
//   - start: the starting position of the range (inclusive)
//   - len: the end position of the range (exclusive) - this is the total length
//   - batchSize: the maximum size of each chunk
//
// Returns a slice of chunks covering the entire range [start, len).
//
// Example: makeChunks(0, 10, 3) returns chunks:
//   - {0, 3} covers indices 0, 1, 2
//   - {3, 6} covers indices 3, 4, 5
//   - {6, 9} covers indices 6, 7, 8
//   - {9, 10} covers index 9
func makeChunks(start, len, batchSize int) (chunks []chunk) {
	// Create a chunk starting at 'start' with size 'batchSize'
	c := chunk{start, start + batchSize}

	// Check if this chunk extends beyond our range
	if c.end > len {
		// Last chunk - trim it to fit exactly
		c.end = len
		chunks = append(chunks, c)
		return // Base case: we've covered the entire range
	}

	// This chunk fits completely within our range
	chunks = append(chunks, c)

	// Recursively create chunks for the remaining range
	// Note: This is tail recursion - the recursive call is the last operation
	chunks = append(chunks, makeChunks(c.end, len, batchSize)...)
	return
}

// sendChunks is the producer goroutine that sends work chunks to the consumer.
// It runs in a separate goroutine and sends all chunks to chunkCh for processing.
//
// This function implements the "producer" part of a producer-consumer pattern.
// Key patterns demonstrated:
//   - defer close() to ensure the channel is closed when done
//   - Select with context cancellation for graceful shutdown
//   - Send-only channel parameter (chan<- chunk) for type safety
//
// Parameters:
//   - ctx: Context for cancellation - if cancelled, stops sending chunks
//   - chunks: Slice of all work chunks to send
//   - chunkCh: Send-only channel where chunks are sent for processing
//
// Important: This function MUST close chunkCh when done to signal the consumer
// that no more work is coming. We use defer to ensure it's closed even if
// the context is cancelled partway through.
func sendChunks(ctx context.Context, chunks []chunk, chunkCh chan<- chunk) {
	// defer close(chunkCh) ensures the channel is closed no matter how this function exits
	// This is CRITICAL - the consumer (eatChunks) waits for the channel to be closed
	// to know when all work has been sent. Without this, eatChunks would wait forever!
	defer close(chunkCh)

	// Send each chunk to the worker
	for _, c := range chunks {
		// Use select to handle both sending and cancellation
		select {
		case chunkCh <- c:
			// Successfully sent the chunk - continue to next one
			// Note: this might block if the channel buffer is full,
			// which provides natural backpressure (flow control)
		case <-ctx.Done():
			// Context was cancelled - stop sending chunks immediately
			// The defer close(chunkCh) will still run, properly signaling the consumer
			return
		}
	}
	// All chunks sent successfully
	// The defer close(chunkCh) will execute here, signaling completion
}

// processChunks is the consumer goroutine that processes work chunks.
// It runs in a separate goroutine and continuously reads chunks from chunkCh,
// evaluates them, and sends results back through resultsCh or errCh.
//
// This function implements the "worker" part of a producer-consumer pattern.
// It demonstrates several important Go concurrency patterns:
//   - Channel communication for receiving work and sending results
//   - Select statements for handling multiple channels and cancellation
//   - Proper goroutine termination when channels are closed or context is cancelled
//
// Parameters:
//   - ctx: Context for cancellation - when cancelled, this goroutine exits
//   - chunkCh: Receive-only channel for getting work chunks (note the <-chan syntax)
//   - rules: The full slice of rules (chunks contain indices into this slice)
//   - resultsCh: Send-only channel for sending successful results (note the chan<- syntax)
//   - errCh: Send-only channel for sending errors when evaluation fails
//
// Channel directions (important Go concept):
//   - <-chan chunk: Can only receive from this channel
//   - chan<- result: Can only send to this channel
//   - This provides compile-time safety and documents the intended data flow
func (e *DefaultEngine) processChunks(ctx context.Context, chunkCh <-chan chunk, rules []*Rule, d map[string]any, resultsCh chan<- evalResult, errCh chan<- error, o EvalOptions, opts ...EvalOption) {
	// Recover from any panics that occur during evaluation
	// This prevents a panic in one worker goroutine from crashing the entire evaluation
	defer func() {
		if recovered := recover(); recovered != nil {
			// Convert the panic to an error and send it back to the main goroutine
			panicErr := fmt.Errorf("panic during parallel rule evaluation: %v", recovered)
			select {
			case errCh <- panicErr:
				// Successfully sent the panic error
			case <-ctx.Done():
				// Context cancelled while sending panic error - just exit
				// The main goroutine will see the cancellation
			}
		}
	}()

	// Process chunks as they are sent to us
	for chunk := range chunkCh {
		r, err := e.evalRuleSlice(ctx, rules[chunk.start:chunk.end], d, o, opts...)
		if err != nil {
			// An error occurred during evaluation
			// We need to send it back to the main goroutine, but we also
			// need to handle the case where the context gets cancelled
			// while we're trying to send the error
			select {
			case errCh <- err:
				// Successfully sent the error
			case <-ctx.Done():
				// Context cancelled while sending error - just exit
				// The main goroutine will see the cancellation
			}
			return // Exit after sending error (fail-fast behavior)
		}
		// Successfully processed the chunk - send results back
		// Again, we need to handle potential cancellation during the send
		select {
		case resultsCh <- r:
		// Successfully sent results - continue to next chunk
		case <-ctx.Done():
			// Context cancelled while sending results - exit
			return
		}

	}
}

// chunk represents a range of indices in the rules slice that should be processed together.
// This allows us to divide the work into batches for parallel processing.
//
// Fields:
//   - start: Starting index (inclusive) - first rule to include in this chunk
//   - end: Ending index (exclusive) - first rule NOT to include in this chunk
//
// Example: chunk{start: 5, end: 8} includes rules at indices 5, 6, and 7
// This follows Go's slice convention where ranges are [start, end)
type chunk struct {
	start, end int
}

func (e *DefaultEngine) evalRuleSlice(ctx context.Context, rules []*Rule, d map[string]any,
	o EvalOptions, opts ...EvalOption) (r evalResult, err error) {

	r.results = make(map[string]*Result, len(rules))

	for _, cr := range rules {
		select {
		case <-ctx.Done():
			return r, ctx.Err()
		default:
			if o.ReturnDiagnostics {
				r.evaluated = append(r.evaluated, cr)
			}
			var result *Result
			result, err = e.Eval(ctx, cr, d, opts...)
			if err != nil {
				return r, err
			}
			r.evalCount += result.EvalCount
			r.evalParallel += result.EvalParallelCount
			// If the child rule failed, either due to its own expression evaluation
			// or its children, we have encountered a failure, and we'll count it
			// The reason to keep this count, rather than look at the child results,
			// is that we may be discarding passes or failures.
			switch result.Pass {
			case true:
				r.passCount++
			case false:
				r.failCount++
			}

			// Decide if we should return the child rule's result or not
			switch result.Pass {
			case true:
				if !o.DiscardPass {
					r.results[cr.ID] = result
				}
			case false:
				switch o.DiscardFail {
				case KeepAll:
					r.results[cr.ID] = result
				case Discard:
				case DiscardOnlyIfExpressionFailed:
					if result.ExpressionPass {
						r.results[cr.ID] = result
					}
				}
			}

			if o.StopFirstPositiveChild && result.Pass {
				return
			}

			if o.StopFirstNegativeChild && !result.Pass {
				return
			}
		}
	}
	return

}

// Compile uses the Evaluator's compile method to check the rule and its children,
// returning any validation errors. Stores a compiled version of the rule in the
// rule.Program field (if the compiler returns a program).
func (e *DefaultEngine) Compile(r *Rule, opts ...CompilationOption) error {
	if err := validateCompileArguments(r, e); err != nil {
		return err
	}

	o := compileOptions{}
	applyCompilerOptions(&o, opts...)

	resultType := r.ResultType
	if resultType == nil {
		resultType = Bool{}
	}

	prg, err := e.e.Compile(r.Expr, r.Schema, resultType, o.collectDiagnostics, o.dryRun)
	if err != nil {
		return fmt.Errorf("rule %s: %w", r.ID, err)
	}

	if !o.dryRun {
		r.Program = prg
	}

	for _, cr := range r.Rules {
		err := e.Compile(cr, opts...)
		if err != nil {
			return err
		}
	}

	r.sortedRules = r.sortChildRules(r.EvalOptions.SortFunc, true)

	return nil
}

type compileOptions struct {
	dryRun             bool
	collectDiagnostics bool
}

// CompilationOption is a functional option to specify compilation behavior.
type CompilationOption func(f *compileOptions)

// DryRun specifies to perform all compilation steps, but do not save the results.
// This is to allow a client to check all rules in a rule tree before
// committing the actual compilation results to the rule.
func DryRun(b bool) CompilationOption {
	return func(f *compileOptions) {
		f.dryRun = b
	}
}

// CollectDiagnostics instructs the engine and its evaluator to save any
// intermediate results of compilation in order to provide good diagnostic
// information after evaluation. Not all evaluators need to have this option set.
func CollectDiagnostics(b bool) CompilationOption {
	return func(f *compileOptions) {
		f.collectDiagnostics = b
	}
}

// Given an array of EngineOption functions, apply their effect
// on the engineOptions struct.
func applyCompilerOptions(o *compileOptions, opts ...CompilationOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// EvalOptions determines how the engine should treat the results of evaluating a rule.
type EvalOptions struct {

	// TrueIfAny makes a parent rule Pass = true if any of its child rules are true.
	// The default behavior is that a rule is only true if all of its child rules are true, and
	// the parent rule itself is true.
	// Setting TrueIfAny changes this behvior so that the parent rule is true if at least one of its child rules
	// are true, and the parent rule itself is true.
	TrueIfAny bool `json:"true_if_any"`

	// StopIfParentNegative prevents the evaluation of child rules if the parent's expression is false.
	// Use case: apply a "global" rule to all the child rules.
	StopIfParentNegative bool `json:"stop_if_parent_negative"`

	// Stops the evaluation of child rules when the first positive child is encountered.
	// Results will be partial. Only the child rules that were evaluated will be in the results.
	// Use case: role-based access; allow action if any child rule (permission rule) allows it.
	StopFirstPositiveChild bool `json:"stop_first_positive_child"`

	// Stops the evaluation of child rules when the first negative child is encountered.
	// Results will be partial. Only the child rules that were evaluated will be in the results.
	// Use case: you require ALL child rules to be satisfied.
	StopFirstNegativeChild bool `json:"stop_first_negative_child"`

	// Do not return rules that passed
	// Default: all rules are returned
	DiscardPass bool `json:"discard_pass"`

	// Decide what to do to rules that failed
	// Default: all rules are returned
	DiscardFail FailAction

	// Include diagnostic information with the results.
	// To enable this option, you must first turn on diagnostic
	// collection at the engine level with the CollectDiagnostics EngineOption.
	ReturnDiagnostics bool `json:"return_diagnostics"`

	// Specify the function used to sort the child rules before evaluation.
	// Useful in scenarios where you are asking the engine to stop evaluating
	// after either the first negative or first positive child in order to
	// select a rule with some relative characteristic, such as the "highest
	// priority rule".
	//
	// See the ExampleSortFunc() for an example.
	// The function returns whether rules[i] < rules[j] for some attribute.
	// Default: No sort
	SortFunc func(rules []*Rule, i, j int) bool `json:"-"`

	// this special field is updated by the SortFunc option. It is necessary
	// because we need to know if the local rule-specific sort function
	// is being overridden by a global option.
	//  (1) Rule supplied its own sort function, overriding with global
	//  (2) Rule did not supply its own sort
	// and was overridden by a global eval option,
	overrideSort bool

	// parallel enables parallel evaluation of child rules with batching.
	//
	// - MinSize determines the minimum number of rules that should be in a list
	//   before we divide the work into batches. parallel processing has a time cost;
	//   this setting allows users to specify at what point it makes sense to batch.
	//
	// - BatchSize controls how many rules are evaluated concurrently in each batch.
	//   If BatchSize is 0, all rules are processed in a single batch, and no additional
	//   overhead cost is incurred for parallel processing. This is Indigo's default.
	//
	// - MaxGoroutines limits the maximum number of goroutines used for evaluation.
	//   This limits the number of batches that can be evaluated at the same time.
	//   If MaxGoroutines is 0, a default number is used (defaultMaxParallel).
	//
	// Unlike most EvalOption fields, this is unexported, so it cannot be set
	// on a rule directly. The reason for this is the special handling required
	// for the "self" field in parallel processing situations.
	parallel struct {
		minSize       int
		batchSize     int
		maxGoroutines int
	}
}

// FailAction is used to tell Indigo what to do with the results of
// rules that did not pass.
type FailAction int

const (
	// KeepAll means that all results, whether the rule passed or not,
	// are returned by Indigo after evaluation.
	KeepAll FailAction = iota

	// Discard means that the results of rules that failed are not
	// returned by Indigo after evaluation, though their effect on a parent
	// rule's pass/fail state is retained.
	Discard

	// DiscardOnlyIfExpressionFailed means that the result of a rule will
	// not be discarded unless it's ExpressionPass result is false.
	// Even if the rule itself has result of Pass = false, the rule will
	// be returned in the result.
	DiscardOnlyIfExpressionFailed
)

// EvalOption is a functional option for specifying how evaluations behave.
type EvalOption func(f *EvalOptions)

// ReturnDiagnostics specifies that diagnostics should be returned
// from this evaluation. You must first turn on diagnostic collectionat the
// engine level when compiling the rule.
func ReturnDiagnostics(b bool) EvalOption {
	return func(f *EvalOptions) {
		f.ReturnDiagnostics = b
	}
}

// SortFunc specifies the function used to sort child rules before evaluation.
// Sorting is only performed if the evaluation order of the child rules is important (i.e.,
// if an option such as StopFirstNegativeChild is set).
func SortFunc(x func(rules []*Rule, i, j int) bool) EvalOption {
	return func(f *EvalOptions) {
		f.SortFunc = x
		f.overrideSort = true
	}
}

// DiscardFail specifies whether to omit failed rules from the results.
func DiscardFail(k FailAction) EvalOption {
	return func(f *EvalOptions) {
		f.DiscardFail = k
	}
}

// DiscardPass specifies whether to omit passed rules from the results.
func DiscardPass(b bool) EvalOption {
	return func(f *EvalOptions) {
		f.DiscardPass = b
	}
}

// StopIfParentNegative prevents the evaluation of child rules if the
// parent rule itself is negative.
func StopIfParentNegative(b bool) EvalOption {
	return func(f *EvalOptions) {
		f.StopIfParentNegative = b
	}
}

// StopFirstNegativeChild stops the evaluation of child rules once the first
// negative child has been found.
func StopFirstNegativeChild(b bool) EvalOption {
	return func(f *EvalOptions) {
		f.StopFirstNegativeChild = b
	}
}

// StopFirstPositiveChild stops the evaluation of child rules once the first
// positive child has been found.
func StopFirstPositiveChild(b bool) EvalOption {
	return func(f *EvalOptions) {
		f.StopFirstPositiveChild = b
	}
}

// Parallel enables parallel evaluation of child rules with the specified
// batch size and maximum parallel goroutines.
func Parallel(minSize, batchSize, maxParallel int) EvalOption {
	return func(f *EvalOptions) {
		f.parallel.minSize = minSize
		f.parallel.batchSize = batchSize

		switch maxParallel {
		case 0:
			f.parallel.maxGoroutines = defaultMaxParallel
		default:
			f.parallel.maxGoroutines = maxParallel
		}
	}
}

// the default number of goroutines to use in parallel processing if the user
// has specified 0
const defaultMaxParallel = 255

// See the EvalOptions struct for documentation.
func applyEvaluatorOptions(o *EvalOptions, opts ...EvalOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// setSelfKey manages the special "self" key in rule evaluation data.
// This function makes the Rule.Self object available to expressions via the reserved "self" key.
// The implementation strategy depends on whether parallel processing is enabled:
//
// - Parallel mode: Creates a copy of the data map to avoid race conditions between goroutines
// - Sequential mode: Modifies the data map in place for better performance
//
// Parameters:
//   - r: The rule being evaluated, containing the Self object to expose
//   - d: The data map containing variables for expression evaluation
//   - o: Evaluation options that determine processing mode (parallel vs sequential)
//
// Returns: A data map with the "self" key properly set or removed
func setSelfKey(r *Rule, d map[string]any, o EvalOptions) map[string]any {
	if d == nil {
		return nil
	}

	switch o.parallel.batchSize > 0 {
	case true:
		return setSelfKeyParallelMode(r, d)
	default:
		return setSelfKeySequentialMode(r, d)
	}

}

// setSelfKeyParallelMode handles self key management for parallel rule evaluation.
// This function creates copies of the data map to prevent race conditions when multiple
// goroutines are evaluating rules concurrently. Each goroutine gets its own copy of the
// data map with the appropriate self key value.
//
// Behavior:
//   - When r.Self is nil: Removes any existing "self" key from a copy of the data map
//     to prevent conflicts. If no "self" key exists, returns the original map unchanged.
//   - When r.Self is not nil: Creates a copy of the data map and sets the "self" key
//     to the value of r.Self.
//
// Thread Safety: Always safe for concurrent use as it never modifies the input map.
// Performance: O(n) time and space complexity due to map copying, where n is map size.
func setSelfKeyParallelMode(r *Rule, d map[string]any) map[string]any {
	switch r.Self {
	case nil:
		_, ok := d[selfKey]
		if !ok {
			return d
		}
		d2 := make(map[string]any, len(d))
		maps.Copy(d2, d)
		delete(d2, selfKey)
		return d2
	default:
		d2 := make(map[string]any, len(d))
		maps.Copy(d2, d)
		d2[selfKey] = r.Self
		return d2
	}
}

// setSelfKeySequentialMode handles self key management for sequential rule evaluation.
// This function modifies the input data map in place for optimal performance when
// rules are evaluated sequentially (no concurrency concerns).
//
// Behavior:
//   - When r.Self is nil, deletes the self key
//   - When r.Self is not nil, sets the "self" key to the value of r.Self
//
// Thread Safety: NOT safe for concurrent use as it modifies the input map directly.
// Performance: O(1) time and space complexity - highly efficient in-place modification.
func setSelfKeySequentialMode(r *Rule, d map[string]any) map[string]any {
	switch r.Self {
	case nil:
		delete(d, selfKey)
	default:
		d[selfKey] = r.Self
	}
	return d
}

// validateEvalArguments checks the input parameters to engine.Eval
func validateEvalArguments(r *Rule, e *DefaultEngine, d map[string]any) error {

	switch {
	case r == nil:
		return errors.New("rule is nil")
	case e == nil:
		return errors.New("engine is nil")
	case e.e == nil:
		return errors.New("evaluator is nil")
	case d == nil:
		return errors.New("data is nil")
	default:
		return nil
	}
}

// Default the result type to boolean
// This is the result type passed to the evaluator. The evaluator may use it to
// inspect / validate the result it generates.
func defaultResultType(r *Rule) Type {

	switch r.ResultType {
	case nil:
		return Bool{}
	default:
		return r.ResultType
	}

}

// validateEvalArguments checks the input parameters to engine.Eval
func validateCompileArguments(r *Rule, e *DefaultEngine) error {

	switch {
	case r == nil:
		return errors.New("rule is nil")
	case e == nil:
		return errors.New("engine is nil")
	case e.e == nil:
		return errors.New("evaluator is nil")
	default:
		return nil
	}
}
