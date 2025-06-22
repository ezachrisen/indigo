# Analysis of Parallel Rule Evaluation in engine.go

## 1. Introduction

This report details the analysis of the parallel rule evaluation mechanism within `engine.go` of the Indigo package. The primary focus was to identify and address potential goroutine leaks and ensure robust handling of early exit signals (due to errors or context cancellation).

## 2. Methodology

The `evalChildren` function and its helper goroutines (`hurlChunks`, `eatChunks`, and the worker goroutines spawned by `eatChunks`) were reviewed. The core concern was the lifecycle management of these goroutines, especially when the main `evalChildren` function exits prematurely.

Testing involved:
- Creating a `trackingMockEvaluator` to simulate rule evaluations with controllable delays, errors, and panics.
- Using an atomic counter (`activeEvals`) within the mock evaluator to track the number of currently active (i.e., not yet returned from `Evaluate`) evaluation calls.
- Writing test cases in `engine_test.go` to trigger scenarios that could lead to leaks:
    - Early exit due to an error in one rule.
    - Early exit due to user-initiated context cancellation.
    - Early exit due to a panic in one rule.
- Verifying that `activeEvals` returns to zero after `engine.Eval()` completes and a sufficient grace period, indicating that no evaluation goroutines are stuck.

No external libraries like `goleak` were used, nor `sync.WaitGroup` or `errgroup`, per constraints.

## 3. Findings: Potential Goroutine Leaks

The original implementation had potential for goroutine leaks in the following scenarios:

### 3.1. Worker Goroutine Leaks (spawned by `eatChunks`)

When `evalChildren` initiates parallel processing, `eatChunks` spawns a new goroutine for each chunk of rules to call `evalRuleSlice`. These worker goroutines send their results or errors back to `evalChildren` via `resultsCh` or `errCh`.

**Issue**: If `evalChildren` stopped reading from `resultsCh` or `errCh` (because it itself was returning due to an error from another worker, or because the user's `ctx` was cancelled), any in-flight worker goroutines attempting to send to these unbuffered (or full buffered) channels would block indefinitely. The original code used `select` with the *user's context* (`ctx.Done()`) for these sends. This is insufficient because `evalChildren` might decide to terminate its internal pipeline (and stop listening) *before* the user's context is cancelled (e.g., if one worker returns an error, `evalChildren` returns, but other workers for other chunks might still be processing and then try to send).

This applies to:
- Sending results via `resultsCh`.
- Sending errors via `errCh`.
- Sending panic-recovered errors via `errCh` from the `defer` block.

### 3.2. `hurlChunks` Goroutine Leak

The `hurlChunks` goroutine is responsible for sending rule chunks to `eatChunks` via `chunkCh`.

**Issue**: `hurlChunks` originally only listened to the *user's context* (`ctx`) for cancellation. If `evalChildren` decided to terminate early (e.g., due to an error from a worker), it would cancel its `internalCtx`. `eatChunks` would stop reading from `chunkCh`. If the user's `ctx` was not yet done, `hurlChunks` could block indefinitely trying to send to a full `chunkCh`, leading to a leak.

## 4. Proof of Issues (Tests)

The following tests were designed to fail with the original code due to goroutine leaks (manifesting as `activeEvals > 0` after `Eval` returns):

-   **`TestParallelEvalLeakOnEarlyError`**: Simulates a scenario where one rule evaluation errors out quickly. Other rules have delays. Without the fix, goroutines for the delayed rules would leak.
-   **`TestParallelEvalLeakOnContextCancel`**: Simulates user context cancellation while slow rules are being processed. Without the fix, goroutines for these rules would leak.
-   **`TestParallelEvalPanicHandling`**: Simulates a panic in one rule. Without the fix for the panic recovery path, goroutines could leak, and other worker goroutines might also leak if the main processing stops.

## 5. Fixes Implemented

To address these issues, the internal context (`internalCtx`) created within `evalChildren` (and cancelled via `defer cancelInternal()`) is now consistently used to signal termination to all goroutines participating in that specific `evalChildren` invocation's pipeline.

1.  **`hurlChunks` Modification**:
    *   The `hurlChunks` function now accepts `internalCtx` instead of the user's `ctx`.
    *   Its `select` statement for sending chunks now also checks `internalCtx.Done()`. This ensures `hurlChunks` stops trying to send chunks if `evalChildren` is shutting down its pipeline. The call in `evalChildren` was updated to pass `internalCtx`.

2.  **`eatChunks` Worker Goroutine Modification**:
    *   The `eatChunks` function was updated to accept both `internalCtx` (for its own lifecycle and for its spawned workers' send operations) and `userCtx` (to be passed to `evalRuleSlice` for the actual rule evaluation logic).
    *   Inside the anonymous goroutine spawned by `eatChunks` for each chunk:
        *   When sending a result to `resultsCh`, the `select` statement now includes a case for `<-internalCtx.Done()`.
        *   When sending an error to `errCh`, the `select` statement now includes a case for `<-internalCtx.Done()`.
        *   In the `defer` function that recovers from panics, when sending the `panicErr` to `errCh`, the `select` statement now includes a case for `<-internalCtx.Done()`.
    *   This ensures that if `evalChildren` has stopped listening (signaled by `internalCtx` being done), these worker goroutines will not block on send operations and will terminate correctly. The actual rule evaluation (`evalRuleSlice`) still respects the `userCtx`.

## 6. Verification

With the implemented fixes:
-   All goroutines spawned as part of the `evalChildren` pipeline (`hurlChunks`, `eatChunks`, and its worker goroutines) now correctly terminate when `internalCtx` is cancelled.
-   The tests `TestParallelEvalLeakOnEarlyError`, `TestParallelEvalLeakOnContextCancel`, and `TestParallelEvalPanicHandling` now pass, with the `activeEvals` counter returning to zero, indicating no leaks of the mock evaluation goroutines.

## 7. Conclusion

The implemented changes significantly improve the robustness of the parallel rule evaluation mechanism in `engine.go` by preventing goroutine leaks in scenarios involving early exits due to errors, panics, or context cancellation. The use of an internal context specific to each `evalChildren` call ensures that its managed goroutine pipeline can be reliably shut down.
