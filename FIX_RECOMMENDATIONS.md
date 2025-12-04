# Race Condition Fixes for Indigo Vault

## Critical Issue Summary
The Indigo vault system has critical race conditions when:
1. Parallel evaluations are in-flight
2. Vault mutations occur simultaneously

This is because Rules maps are mutated without synchronization while parallel goroutines may be reading them.

## Architecture Overview of the Problem

The design uses:
- **Atomic.Pointer[Rule]**: For lock-free reads of the root rule
- **Mutex**: Only for write coordination (mutations)
- **Copy-on-Write**: Shallow copying of rules during mutations

**The Problem**: Readers get a snapshot of the root via atomic load, but the snapshot contains references to mutable Rule objects. While shallow-copying the path, the underlying Rule.Rules map can still be modified.

## Detailed Fixes

### Fix 1: Add Read-Write Synchronization

**Problem**: Parallel evaluations and mutations have no synchronization point.

**Solution**: Add an RWMutex to the Vault struct that readers (evaluations) must acquire, while writers (mutations) already hold the exclusive lock.

**Code Changes**:

```go
// In vault.go, add to Vault struct:
type Vault struct {
    // ... existing fields ...
    mu sync.Mutex          // Existing - writers only
    evalMu sync.RWMutex    // NEW - readers acquire RLock
    root atomic.Pointer[Rule]
    // ... rest of fields ...
}

// In Eval method, acquire read lock:
func (v *Vault) ImmutableRule() *Rule {
    v.evalMu.RLock()
    defer v.evalMu.RUnlock()
    return v.root.Load()
}

// In Mutate method, upgrade to exclusive write lock:
func (v *Vault) Mutate(mutations ...vaultMutation) error {
    v.evalMu.Lock()         // CHANGE: Use evalMu instead of mu
    defer v.evalMu.Unlock()
    v.mu.Lock()
    defer v.mu.Unlock()
    
    // ... existing mutation code ...
}
```

**Tradeoff**: This introduces lock contention on all evaluations. For high-frequency evaluations, this may be too slow.

---

### Fix 2: Defensive Copy with Validation (Recommended Quick Fix)

**Problem**: Rules can become invalid (Program = nil) due to mutations.

**Solution**: Validate rules before evaluation and catch race conditions early.

**Code Changes in engine.go**:

```go
func (e *DefaultEngine) Eval(ctx context.Context, r *Rule,
    d map[string]any, opts ...EvalOption,
) (*Result, error) {
    if err := validateEvalArguments(r, e, d); err != nil {
        return nil, err
    }
    
    // NEW: Validate rule tree is not being mutated
    if r.Rules != nil && len(r.Rules) > 0 && r.Expr != "" && r.Program == nil {
        return nil, fmt.Errorf("rule %s: detected concurrent mutation - Program field is nil", r.ID)
    }
    
    // ... rest of function ...
}
```

**In rule.go - sortChildRules defensive code**:

```go
func (r *Rule) sortChildRules(fn func(rules []*Rule, i, j int) bool, force bool) []*Rule {
    // Defensive: take snapshot of keys first
    if r.Rules == nil || len(r.Rules) == 0 {
        return []*Rule{}
    }
    
    keys := make([]*Rule, 0, len(r.Rules))
    for k := range r.Rules {
        if r.Rules[k] != nil {
            keys = append(keys, r.Rules[k])
        }
    }
    
    if fn != nil && len(keys) > 0 && force {
        sort.Slice(keys, func(i, j int) bool {
            if i >= len(keys) || j >= len(keys) {
                return false // Safety check
            }
            return fn(keys, i, j)
        })
    }
    return keys
}
```

---

### Fix 3: Snapshot Isolation for Parallel Evaluation (Best Long-term)

**Problem**: Mutations invalidate in-flight evaluations.

**Solution**: Use generation counter to detect when a rule tree has been mutated.

**Code Changes**:

```go
// In Rule struct (rule.go):
type Rule struct {
    // ... existing fields ...
    generation uint64  // NEW - incremented on mutation
    // ... rest of fields ...
}

// In Vault struct (vault.go):
type Vault struct {
    // ... existing fields ...
    generation atomic.Uint64  // NEW
    // ... rest of fields ...
}

// In Vault.Mutate():
func (v *Vault) Mutate(mutations ...vaultMutation) error {
    v.mu.Lock()
    defer v.mu.Unlock()
    
    r := v.ImmutableRule()
    // ... apply mutations ...
    
    // NEW: Increment generation after successful mutation
    v.generation.Add(1)
    v.root.Store(newRoot)
    return nil
}

// In Eval - detect if rule is stale:
func (e *DefaultEngine) Eval(ctx context.Context, r *Rule,
    d map[string]any, opts ...EvalOption,
) (*Result, error) {
    // Validate rule hasn't been garbage collected/invalidated
    if r.ID != "" && r.Program == nil && r.Expr != "" {
        return nil, fmt.Errorf("rule %s: stale snapshot - vault was mutated during evaluation", r.ID)
    }
    
    // ... rest of function ...
}
```

---

### Fix 4: Safe Rules Map Operations

**Problem**: Direct map access like `parent.Rules[id] = rule` is not safe with concurrent reads.

**Solution**: Use synchronized map operations or take defensive snapshots.

**Code Changes in vault.go**:

```go
// NEW: Safe map update function
func updateRulesMap(rules map[string]*Rule, id string, rule *Rule) map[string]*Rule {
    // Create new map to avoid race
    newRules := make(map[string]*Rule, len(rules)+1)
    for k, v := range rules {
        if k != id {
            newRules[k] = v
        }
    }
    newRules[id] = rule
    return newRules
}

// Update vault.go update() method to use safe map:
func (v *Vault) update(r, newRule *Rule, alreadyCopied map[*Rule]any) (*Root, map[*Rule]any, error) {
    // ... setup code ...
    
    // Change this line:
    // parentInNew.Rules[newRule.ID] = newRule
    // To this:
    parentInNew.Rules = updateRulesMap(parentInNew.Rules, newRule.ID, newRule)
    
    // ... rest of function ...
}
```

---

### Fix 5: Prevent destinationShard Race

**Problem**: `destinationShard()` walks the tree while mutations may be happening.

**Solution**: Take a snapshot or use generation-based validation.

**Code Changes in vault.go**:

```go
func destinationShard(r, rr *Rule) (*Rule, error) {
    // Snapshot the sorted rules to prevent concurrent modification
    sortedRulesSnapshot := r.sortChildRules(r.EvalOptions.SortFunc, false)
    if sortedRulesSnapshot == nil {
        sortedRulesSnapshot = []*Rule{}
    }
    
    var toReturn *Rule
    shardCount := 0
    if r.shard {
        shardCount++
        ok, err := matchMeta(r, rr)
        if err != nil {
            return nil, err
        }
        if ok {
            toReturn = r
        }
    }

shardLoop:
    for _, shard := range sortedRulesSnapshot {  // Use snapshot, not r.sortedRules
        if !shard.shard {
            continue
        }
        shardCount++
        ok, err := matchMeta(shard, rr)
        if err != nil {
            return nil, err
        }
        if ok {
            toReturn = shard
            break shardLoop
        }
    }
    
    // ... rest of function unchanged ...
}
```

---

## Implementation Priority

### Phase 1 (Emergency - Do First)
1. **Fix 2**: Add defensive validations in sortChildRules
2. Add nil checks throughout the evaluation path
3. Catch "missing program" errors early with clear messages

### Phase 2 (Short-term - Do Next)
1. **Fix 5**: Snapshot rules in destinationShard
2. Add generation counter to Vault
3. Validate snapshots haven't been invalidated

### Phase 3 (Long-term - Refactor)
1. **Fix 1**: Add RWMutex for proper synchronization
2. Consider immutable persistent data structures
3. Refactor to use read-write locks throughout

## Testing Strategy

### Continue Using Hellish Test
Keep `vault_shards_parallel_hellish_test.go` running with `-race` flag:
```bash
go test -run TestHellish -race -v
```

### Add Regression Tests
```go
func TestConcurrentMutationEvaluation(t *testing.T) {
    // Ensure concurrent mutation + evaluation doesn't corrupt state
}

func TestSnapshotImmutability(t *testing.T) {
    // Ensure snapshots remain valid during mutations
}

func TestGenerationCounter(t *testing.T) {
    // Verify generation counter detects mutations
}
```

---

## Performance Implications

### Current (No Fix)
- **Read**: O(1) atomic load - very fast but unsafe
- **Write**: O(n) mutex-protected mutation
- **Race Condition**: Possible data corruption

### After Fix 1 (RWMutex)
- **Read**: O(1) + RWLock acquisition - slightly slower but safe
- **Write**: O(n) + exclusive lock - same
- **Contention**: High on read-heavy workloads

### After Fix 2 (Defensive Code)
- **Read**: Same as current + validation checks
- **Write**: Same as current
- **Safety**: Detects races but doesn't prevent them

### After Fix 3 (Generation Counter)
- **Read**: O(1) + generation check - minimal overhead
- **Write**: O(n) + atomic.Add(1)
- **Safety**: Detects stale snapshots efficiently

---

## Recommended Approach

1. **Immediate** (Fix 2): Add defensive validations to catch bugs
2. **Short-term** (Fix 5 + Fix 3): Use generation counters to detect stale evaluations
3. **Long-term** (Fix 1): Consider adding read-write synchronization if needed

This approach balances safety with performance, catching race conditions without adding severe lock contention for read-heavy workloads.

---

## Documentation Update Needed

Update comments in Vault struct to clarify:
```go
// Vault provides lock-free, hot-reloadable, hierarchical rule management
// with full support for add, update, delete, and move operations.
//
// CONCURRENCY MODEL:
// - ImmutableRule() returns an atomic snapshot (lock-free reads)
// - Mutate() is protected by a mutex (exclusive writes)
// - Concurrent mutations and evaluations MAY cause race conditions if:
//   * Parallel evaluation is active AND mutations are in-flight
// RECOMMENDATION: For high-concurrency scenarios, either:
//   * Batch mutations and wait for evaluations to complete
//   * Use external synchronization (e.g., read-write lock around Eval + Mutate)
//   * Implement generation counters to detect stale snapshots
```
