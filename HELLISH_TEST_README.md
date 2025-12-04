# The Hellish Test - Race Condition Detection for Indigo Vault

## What is the Hellish Test?

A comprehensive stress test designed to expose race conditions in the Indigo vault system when combining:
- **Concurrent parallel rule evaluation**
- **Concurrent vault mutations**
- **Sharded rule hierarchies**
- **Deep rule nesting**
- **High concurrency (20+ goroutines)**

## Quick Start

### Run the Main Test
```bash
cd /home/ez/code/indigo
go test -run TestHellishShardedVaultParallelStress -race -timeout 120s -v
```

### Run All Hellish Tests
```bash
go test -run TestHellish -race -timeout 120s -v
```

### Run Targeted Tests
```bash
go test -run "TestRaceCondition|TestParallelEval" -race -timeout 60s -v
```

## Files Included

### Test Files
- **vault_shards_parallel_hellish_test.go** (600+ lines)
  - Main chaos test: TestHellishShardedVaultParallelStress()
  - Sharding edge cases: TestExtremeShardingEdgeCases()
  - Concurrent updates: TestConcurrentUpdateWithParallelEval()
  - Deep nesting: TestMakeSafePathRaceConditions()
  - Parallelism limits: TestParallelEvalEdgeCasesWithShards()

- **vault_race_targeted_test.go** (350+ lines)
  - Maps mutation: TestRaceConditionRulesMapMutation()
  - Eval during mutation: TestParallelEvalDuringMutation()
  - Shard routing: TestDestinationShardRaceCondition()

### Documentation Files
- **RACE_CONDITION_REPORT.md** - Detailed analysis of race conditions found
- **FIX_RECOMMENDATIONS.md** - Architecture and fix strategies
- **CONCRETE_FIXES.md** - Exact code changes to implement fixes
- **TEST_RESULTS_SUMMARY.md** - Summary of test execution results
- **HELLISH_TEST_README.md** - This file

## Race Conditions Found

### 1. Rules Map Concurrent Modification (CRITICAL)
```
Location: vault.go:427 (write) vs rule.go:368 (read)
Impact: Data corruption, panic, unpredictable behavior
Status: CONFIRMED by race detector
```

The vault modifies the Rules map while parallel evaluators read it via sortChildRules().

### 2. Program Field Invalidation (HIGH)
```
Impact: Evaluation failures with "missing program" errors
Status: CONFIRMED - 768+ errors in single test run
```

Mutations invalidate the compiled Program field while evaluations are in-flight.

### 3. Shard Routing Lookup Failure (MEDIUM)
```
Impact: Parent lookup errors when moving between shards
Status: CONFIRMED - 84+ mutation errors in single test run
```

The destinationShard() tree traversal becomes invalid during mutations.

## Test Architecture

### Reader Goroutines (20)
- Continuously retrieve immutable rules via ImmutableRule()
- Parallel evaluate with varying batch/parallelism settings
- Stress the read path of the vault

### Writer Goroutines (5)
- Continuous add/update/delete/move operations
- Rapid mutations on sharded rules
- Time updates to stress last-update field

### Special Goroutines
- Context cancellation stress
- Snapshot immutability verification
- Shard-specific update testing

## Expected Behavior

### Without Fixes
```
FAIL - race detector output:
  WARNING: DATA RACE
  Write at 0x... by goroutine 46
  Previous read at 0x... by goroutine 836
  
Test failures:
  1953 evaluation errors
  84 mutation errors
  Final rule invalid (missing program)
```

### With Fixes (After Applying Recommendations)
```
PASS - all tests complete successfully
Race detector: No warnings
Evaluation errors: 0
Mutation errors: 0
Final rule: Valid and consistent
```

## How the Test Works

```
Timeline of a typical race condition:

T1: Reader thread acquires snapshot of root rule via atomic.Load()
    Snapshot contains references to child rules

T2: Mutator thread acquires vault mutex and begins mutation
    Mutator modifies parent.Rules[id] = newRule

T3: Parallel evaluator spawns goroutines to evaluate children
    Goroutine calls rule.sortChildRules()

T4: sortChildRules() iterates over r.Rules map
    At same time, mutator writes to same map

T5: RACE CONDITION: Concurrent read/write detected

T6: Race detector alerts - test fails
```

## Test Execution Statistics

### Test: TestHellishShardedVaultParallelStress
- Duration: ~1.7 seconds
- Concurrent readers: 20
- Concurrent writers: 5
- Concurrent special goroutines: 5
- Total operations: 2000+
- Races detected: Multiple

### Test: TestHellishShardedVaultParallelStress (with fixes)
- Expected duration: ~2.0 seconds (slightly slower with sync)
- Expected races detected: 0
- Expected errors: 0

## Key Insights

### Problem #1: Lock-Free Reads Aren't Safe Enough
The atomic.Pointer design works great for simple cases, but breaks when:
- The rule tree is being modified
- Parallel evaluation extends evaluation windows
- Deep nesting increases complexity

### Problem #2: Shallow Copy Semantics
Using maps.Clone() creates a new map but points to old Rule objects.
When those objects' children are modified, both copies are affected.

### Problem #3: No Eval/Mutation Synchronization
The mutex only protects writers from each other, not readers from writers.
When evaluations are in-flight, mutations proceed freely.

## How to Use This for Development

### Before Committing
```bash
# Always run with race detector
go test -race -v

# Specifically test vault concurrency
go test -run Vault -race -v
```

### When Debugging Failures
```bash
# Run the specific failing test with detailed output
go test -run TestHellishShardedVaultParallelStress -race -v 2>&1 | tee debug.log

# Look for "WARNING: DATA RACE" in output
# Check the stack traces to identify which operations collided
```

### When Implementing Fixes
```bash
# After each change, re-run
go test -run TestHellish -race

# Must achieve: No race detector warnings + All tests pass
# Watch for: Performance degradation, new race conditions
```

## Recommended Reading Order

1. **START HERE**: This file (you are here)
2. **THEN READ**: TEST_RESULTS_SUMMARY.md - Overview of findings
3. **DEEP DIVE**: RACE_CONDITION_REPORT.md - Technical analysis
4. **IMPLEMENT**: FIX_RECOMMENDATIONS.md - Architecture options
5. **EXECUTE**: CONCRETE_FIXES.md - Exact code changes

## Common Questions

### Q: Why does the test sometimes not detect races?
A: Race conditions are probabilistic. Longer timeouts (60+ seconds) improve detection rate.

### Q: Why is this test called "hellish"?
A: It combines multiple sources of non-determinism:
- Concurrent reads/writes
- Goroutine scheduling variability
- Deep recursion in sharding
- Parallel evaluation with thread pools
- Random data patterns
This creates a "nightmare scenario" for testing.

### Q: Can I disable the hellish test?
A: Not recommended. It's one of the few tests that catches these subtle race conditions.

### Q: What's the performance impact of fixes?
A: Estimated 5-15% slowdown for read-heavy workloads due to RWMutex overhead.

## Integration with CI/CD

### Recommended CI Configuration
```yaml
test:
  script:
    - go test -race -v ./...
  timeout: 300
  allow_failure: false  # Fail if ANY race detected
```

### Monitoring
- Alert if test execution time increases >50%
- Alert if race detector warnings appear
- Track mutation vs read performance ratio

## Maintenance Notes

### Keep This Test Active
- Don't skip in short mode (-short flag)
- Keep -race flag enabled
- Review race detector output regularly

### Update When
- Adding new mutations (add, update, delete, move)
- Modifying sharding logic
- Changing evaluation parallelism
- Refactoring vault mutex usage

### Regression Prevention
- Any PR touching Vault must pass this test
- Any PR touching parallel eval must pass this test
- Any PR touching sharding must pass this test

## Contact / Questions

If the test fails or you have questions:
1. Check RACE_CONDITION_REPORT.md for analysis
2. Review FIX_RECOMMENDATIONS.md for solutions
3. Look at race detector output - usually identifies exact location
4. Run CONCRETE_FIXES.md to implement fix

## Summary

This test represents the **absolute worst-case scenario** for the Indigo vault:
- Maximum concurrency
- Maximum mutation frequency
- Maximum complexity (shards + deep nesting + parallelism)
- Intentional race creation

By ensuring this test passes with the race detector enabled, you ensure the vault is bulletproof for production use.

**Status**: 🔴 CURRENTLY FAILING (Race conditions detected)  
**Goal**: 🟢 ALL PASSING (No race detector warnings)

Good luck! 🚀
