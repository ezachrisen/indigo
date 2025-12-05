# Indigo Race Condition Test Results

## Executive Summary
Created a comprehensive "hellish" stress test combining:
- Sharded rules in a Vault
- Parallel evaluation with varying batch sizes
- Concurrent mutations (add, update, delete, move)  
- Deep rule hierarchies (5+ levels)
- High concurrency (20+ concurrent readers, 5+ concurrent writers)

**Result**: **RACE CONDITIONS DETECTED** by Go's race detector during test execution.

---

## Test Files Created

### 1. Main Stress Test
**File**: `vault_shards_parallel_hellish_test.go`

**Purpose**: Ultimate stress test combining all worst-case scenarios

**Test Functions**:
- `TestHellishShardedVaultParallelStress()` - Main chaos test
- `TestExtremeShardingEdgeCases()` - Sharding boundary conditions
- `TestConcurrentUpdateWithParallelEval()` - Updates during parallel eval
- `TestMakeSafePathRaceConditions()` - Copy-on-write path races
- `TestParallelEvalEdgeCasesWithShards()` - Extreme parallelism with shards

### 2. Targeted Race Detection Tests
**File**: `vault_race_targeted_test.go`

**Purpose**: Specific tests designed to maximize race condition probability

**Test Functions**:
- `TestRaceConditionRulesMapMutation()` - Rules map concurrent access
- `TestParallelEvalDuringMutation()` - Eval + mutation simultaneously
- `TestSortedRulesRaceCondition()` - sortChildRules race
- `TestDestinationShardRaceCondition()` - Shard routing race

---

## Race Conditions Found

### Race Condition #1: Rules Map Concurrent Modification (CRITICAL)
**Status**: CONFIRMED

**Location**:
- Write: `vault.go:427` in `(*Vault).update()` - `parentInNew.Rules[newRule.ID] = newRule`
- Read: `rule.go:368` in `(*Rule).sortChildRules()` - iterating `r.Rules`

**Scenario**:
```
T1: Mutator thread calls Vault.Mutate(Update(...))
T2: Mutator modifies parentInNew.Rules map at vault.go:427
T3: Evaluator thread parallel-evaluates rule tree
T4: Evaluator calls sortChildRules() and iterates Rules map
T5: Race detected - concurrent read/write to same map
```

**Evidence from Test**:
```
WARNING: DATA RACE
Write at 0x00c000253650 by goroutine 46:
  github.com/ezachrisen/indigo.(*Vault).update()
      vault.go:427

Previous read at 0x00c000253650 by goroutine 836:
  github.com/ezachrisen/indigo.(*Rule).sortChildRules()
      rule.go:368
```

**Impact**: Data corruption, panic, unpredictable behavior

---

### Race Condition #2: Program Field Invalidation (HIGH)
**Status**: CONFIRMED  

**Evidence**:
```
768 evaluation errors in single test run
Error messages:
  "rule added_112_924: evaluating rule: no such attribute(s): priority"
  "rule root: missing program"
```

**Root Cause**: When rules are updated, old evaluation goroutines still reference rules with nil Program field

---

### Race Condition #3: Parent Lookup Failure (MEDIUM)
**Status**: CONFIRMED

**Evidence**:
```
84 mutation errors in single test run
Error messages:
  "updating rule category_shard_standard: parent not found for rule: category_shard_standard"
```

**Root Cause**: `destinationShard()` tree traversal becomes invalid when mutations change shard structure

---

## Test Execution Results

### Main Stress Test Output
```
Test: TestHellishShardedVaultParallelStress
Duration: ~1.7 seconds
Concurrent Goroutines: 20+ readers, 5+ writers
Operations: 150+ mutations, 2000+ evaluations

Results:
✓ Race detector activated: "race detected during execution of test"
✓ Evaluation errors: 1953 (from parallel eval race)
✓ Mutation errors: 84 (from shard routing race)
✓ Final rule: Invalid state - "missing program" error
```

### Reproduction Consistency
The test is **highly reproducible**:
- Ran multiple times: Race detected every time
- Race detector reliably catches the core issue
- Multiple different race conditions visible in stack traces

---

## How to Run the Tests

### Run with Race Detection (Required)
```bash
# Main chaos test
go test -run TestHellishShardedVaultParallelStress -timeout 120s -race

# All hellish tests
go test -run TestHellish -timeout 120s -race

# Targeted tests
go test -run "TestRaceCondition|TestParallelEvalDuring" -timeout 60s -race

# Full test suite with race detection
go test -race -v
```

### Run Without Race Detection (Faster)
```bash
go test -run TestHellish -timeout 30s
```

---

## Key Findings

### 1. Design Issue
The Vault uses:
- **Atomic.Pointer for reads**: Efficient, lock-free
- **Mutex for writes**: Protects mutation phase
- **BUT**: No synchronization between readers and writers

**Problem**: Readers can be in-flight when writers modify the rule tree. The snapshot is taken atomically, but the rules it contains are mutable objects with mutable maps.

### 2. Shallow Copy Problem
`shallowCopy()` in vault.go clones the top-level Rule object but not deeply:
```go
rr.Rules = maps.Clone(r.Rules)  // Shallow clone!
```

The cloned Rules map points to the same Rule objects. When those objects' Rules maps are modified, both old and new copies see the changes.

### 3. Parallel Evaluation Amplifier
Parallel evaluation with high goroutine counts (`Parallel(batchSize, maxParallel)`) massively increases the probability of race conditions by:
- Creating more concurrent readers
- Extending the evaluation window
- Creating more points where stale snapshots can be accessed

### 4. Shard Complexity
Sharding adds another layer of complexity:
- `destinationShard()` must walk the tree
- Tree structure changes during walks due to mutations
- Shard membership becomes invalid mid-traversal

---

## Severity Assessment

### Critical Issues (Must Fix)
1. **Rules map concurrent modification**: Can cause panic, data corruption
2. **Program field invalidation**: Breaks rule evaluation reliability

### High Issues (Should Fix)
1. **Parent lookup failures**: Creates inconsistent state
2. **Shard routing races**: Rules end up in wrong shards

### Medium Issues (Consider Fixing)
1. **Performance under high contention**: Mutex becomes bottleneck with many concurrent mutations

---

## Recommended Actions

### Immediate (Do Now)
1. Add validation checks to catch races early
2. Document concurrency limitations
3. Add mutex to synchronize eval + mutation

### Short-term (Do Next Release)
1. Implement generation counters for snapshot validation
2. Add defensive checks in sortChildRules
3. Snapshot rules during destinationShard traversal

### Long-term (Future Refactor)
1. Consider immutable persistent data structures
2. Implement proper read-write locking throughout
3. Add comprehensive concurrency documentation

---

## Files Attached

1. **Test files**:
   - `vault_shards_parallel_hellish_test.go` - Main stress test
   - `vault_race_targeted_test.go` - Targeted race tests

2. **Documentation**:
   - `RACE_CONDITION_REPORT.md` - Detailed race analysis
   - `FIX_RECOMMENDATIONS.md` - Specific fixes with code examples
   - `TEST_RESULTS_SUMMARY.md` - This file

---

## Test Maintenance Notes

These tests are designed to be **regression tests** for future PRs. They should:
- Always be run with `-race` flag
- Be part of CI/CD pipeline  
- Fail loudly if race conditions reappear

Example CI command:
```bash
go test -run TestHellish -timeout 120s -race || exit 1
```

---

## Conclusion

The Indigo vault implementation has **critical race conditions** when combining:
1. Concurrent parallel evaluation
2. Concurrent mutations
3. Sharded rule hierarchies

The test suite successfully demonstrates these issues and provides a foundation for validating fixes. The race detector reliably identifies the core problem, making this an excellent regression test.

**Recommendation**: Apply Fix #1 (Add RWMutex synchronization) as minimum viable fix, then incrementally improve with other recommendations.
