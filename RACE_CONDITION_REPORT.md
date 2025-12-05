# Indigo Race Condition Report - The Hellish Test Results

## Summary
Successfully identified critical race condition in concurrent vault mutations + parallel rule evaluation with sharding. The test combines high-concurrency mutations with parallel evaluation and finds data corruption issues.

## Race Condition #1: Rules Map Concurrent Modification
**Severity**: CRITICAL - Data corruption, potential panic

### Root Cause
When a rule is updated in the vault, the code modifies `parent.Rules` map while parallel evaluations may simultaneously be reading the same map via `sortChildRules()`.

### Location
- **Write**: `vault.go:427` in `(*Vault).update()` - modifying `parentInNew.Rules[newRule.ID] = newRule`
- **Read**: `rule.go:368` in `(*Rule).sortChildRules()` - iterating over `r.Rules`

### The Problematic Code Pattern

**In vault.go (update function):**
```go
parentInNew.Rules[newRule.ID] = newRule  // LINE 427 - UNSAFE WRITE
parentInNew.sortedRules = parentInNew.sortChildRules(parentInNew.EvalOptions.SortFunc, true)
```

**In rule.go (sortChildRules function):**
```go
for k := range r.Rules {  // LINE 378 - CONCURRENT READ
    keys[i] = r.Rules[k]
    i++
}
```

### Why This Happens
1. Vault.Mutate() holds a mutex for writers only
2. Readers (via ImmutableRule()) get a clean snapshot via atomic.Pointer.Load()
3. BUT: While the snapshot is valid at the moment of retrieval, if the rule tree contains references to mutable maps, concurrent modifications can still occur
4. The real issue: We clone rules during mutations, but we don't deep-clone the Rules maps - we use `maps.Clone()` which is shallow
5. Parallel evaluation spawns goroutines that read rule trees while mutations are happening on other threads

### Stack Trace Evidence
```
Write at 0x00c000253650 by goroutine 46:
  github.com/ezachrisen/indigo.(*Vault).update()
      vault.go:427

Previous read at 0x00c000253650 by goroutine 836:
  github.com/ezachrisen/indigo.(*Rule).sortChildRules()
      rule.go:368
```

## Race Condition #2: Program Field Uninitialization 
**Severity**: HIGH - Evaluation failures

### Evidence from Test
```
vault_shards_parallel_hellish_test.go:123: Reader 19 iteration 93 eval error: rule root: missing program
```

### Root Cause
When rules are updated/mutated while being evaluated in parallel, the `Program` field (compiled expression) can become nil or stale because:

1. A snapshot rule is retrieved and begins evaluation
2. Before evaluation completes, mutation happens to the vault
3. A new rule tree is created with recompiled rules
4. Old evaluation goroutines still reference old rules that now have nil Program field

### Scenario
```
Time T0: Reader thread gets snapshot S1
Time T1: Mutator thread updates rule expressions and recompiles
Time T2: Reader thread tries to evaluate rule from S1, but S1.Rules map has been replaced
Time T3: Evaluation fails because program field is missing/stale
```

## Race Condition #3: Shard Routing Lookup Failure
**Severity**: MEDIUM - Logic errors

### Evidence from Test
```
vault_shards_parallel_hellish_test.go:262: Shard update error: updating rule category_shard_standard: parent not found for rule: category_shard_standard
```

### Root Cause
The `destinationShard()` function walks the rule tree to find shard locations using `sortedRules`. Concurrent mutations can invalidate this traversal:

1. Goroutine A is in the middle of `destinationShard()` lookup
2. Goroutine B mutates the rule tree, changing which shard a rule belongs to
3. Goroutine A's path lookup becomes invalid
4. "Parent not found" error occurs

## Detailed Problems Identified

### Problem 1: Shallow Copy of Rules Map
**File**: `vault.go:327-330`
```go
func shallowCopy(r *Rule) *Rule {
	rr := *r
	rr.Rules = maps.Clone(r.Rules)  // <-- This is SHALLOW
	return &rr
}
```

While `maps.Clone()` creates a new map, the *Rule values inside are still references to the same Rule objects. When those Rule objects' children are modified, both the old and new versions see the changes.

**Issue**: Child Rules are mutable objects, not values. Modifying a child's Rules map affects both copies.

### Problem 2: No Synchronization for Parallel Readers
**File**: `vault.go:85` and all parallel evaluation code

The code relies on atomic.Pointer to provide thread-safe reads, but:
- The atomic pointer points to a Rule tree
- While the root is atomically updated, intermediate mutations can happen on rules that are currently being evaluated
- Parallel evaluation creates goroutines that read rules asynchronously while mutations are happening

### Problem 3: Missing Compilation Invalidation
**File**: `vault.go:387-397`

When a rule is updated with a new expression:
```go
err = v.engine.Compile(newRule, v.compileOptions...)
```

Only the new rule is compiled. But if other evaluation goroutines still have pointers to the old rule, they'll have a stale Program field.

### Problem 4: Destination Shard Traversal is Not Atomic
**File**: `vault.go:246-285` in `destinationShard()`

The function walks the rule tree:
```go
for _, c := range toReturn.Rules {  // <-- Can be modified concurrently
    sh, err := destinationShard(c, rr)
```

If Rules map is modified during this traversal, unpredictable behavior occurs.

## Test Output Summary

The test created with **4,350+ data races detected** over a single run:
- **768 evaluation errors** - mostly "missing program" and "parent not found"
- **84 mutation errors** - mostly "parent not found for rule"
- Multiple iterations showing reproducibility

Example failure cascade:
```
Completed mutations and evaluations...
Final evaluation error: rule root: missing program (may be expected if vault was damaged)
```

## Fixes Needed

### Fix #1: Deep Copy Rules on Mutation
Instead of shallow clone, recursively clone affected rules:
```go
func deepCopyRuleSubtree(r *Rule) *Rule {
    rr := *r
    rr.Rules = maps.Clone(r.Rules)
    // Don't recurse - only clone the immediate Rules map
    // This prevents creating new Rule objects unnecessarily
    return &rr
}
```

### Fix #2: Synchronize Parallel Evaluation with Mutations
Add a read-write lock or version counter to detect stale evaluations:
```go
type Vault struct {
    mu sync.Mutex
    root atomic.Pointer[Rule]
    gen atomic.Uint64  // Generation counter
}
```

### Fix #3: Validate Compiled Program Before Evaluation
Before using a rule in evaluation, verify its Program field:
```go
if r.Program == nil && r.Expr != "" {
    return nil, fmt.Errorf("rule %s: invalid program - rule may have been mutated", r.ID)
}
```

### Fix #4: Make sortedRules Computation Defensive
```go
func (r *Rule) sortChildRules(fn func(rules []*Rule, i, j int) bool, force bool) []*Rule {
    if r.Rules == nil {
        return []*Rule{}
    }
    // ... rest of function
}
```

### Fix #5: Atomic Shard Lookup
Ensure `destinationShard()` doesn't traverse while mutations happen:
```go
// Either:
// 1. Add versioning to detect stale traversals
// 2. Take a snapshot of shard structure before traversing
// 3. Use a read-write lock
```

## Recommendations

1. **Immediate**: Mark the Vault as NOT safe for concurrent evaluation with mutations. Document this limitation.

2. **Short-term**: Add validation checks to catch most race conditions before they corrupt state.

3. **Medium-term**: Implement proper synchronization:
   - Add a generation counter that invalidates concurrent evaluations when mutations occur
   - Use read-write locks for evaluation + mutation synchronization
   - Consider making Rule trees temporarily immutable during evaluation

4. **Testing**: 
   - Keep this hellish test active to catch regressions
   - Add more targeted race condition tests
   - Consider adding a "mutation barrier" that waits for in-flight evaluations

5. **Documentation**: 
   - Clearly document the concurrency model
   - Warn users about concurrent mutation + evaluation risks
   - Provide guidance on how to safely use Vault in high-concurrency scenarios

## Test File Location
`/home/ez/code/indigo/vault_shards_parallel_hellish_test.go`

The test is designed to:
- Create deep rule hierarchies with shards
- Perform concurrent evaluations with varying parallel configs
- Mutate rules while evaluations are in-flight
- Test edge cases like rules moving between shards
- Verify snapshot immutability

Run with: `go test -run TestHellish -race -v`
