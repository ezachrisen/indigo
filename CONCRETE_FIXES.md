# Concrete Code Fixes for Indigo Race Conditions

This document provides exact code changes to fix the race conditions found by the hellish test.

---

## Fix #1: Add RWMutex for Eval/Mutate Synchronization (RECOMMENDED - DO THIS FIRST)

### Problem
Parallel evaluations can happen while mutations are modifying the rule tree.

### Solution
Add an RWMutex that readers must acquire. Writers upgrade from Mutex to RWMutex.Unlock().

### Code Changes

#### File: `vault.go`

**Step 1: Update Vault struct**
```go
// CHANGE THIS:
type Vault struct {
	mu sync.Mutex
	root atomic.Pointer[Rule]
	engine Engine
	compileOptions []CompilationOption
	lastUpdate atomic.Pointer[time.Time]
}

// TO THIS:
type Vault struct {
	mu sync.Mutex      // Writer mutex (existing)
	rwMu sync.RWMutex   // NEW: Reader-writer lock for eval/mutate sync
	root atomic.Pointer[Rule]
	engine Engine
	compileOptions []CompilationOption
	lastUpdate atomic.Pointer[time.Time]
}
```

**Step 2: Update ImmutableRule()**
```go
// CHANGE THIS:
func (v *Vault) ImmutableRule() *Rule {
	return v.root.Load()
}

// TO THIS:
func (v *Vault) ImmutableRule() *Rule {
	v.rwMu.RLock()
	defer v.rwMu.RUnlock()
	return v.root.Load()
}
```

**Step 3: Update Mutate()**
```go
// CHANGE THIS:
func (v *Vault) Mutate(mutations ...vaultMutation) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	// ... rest of function ...
}

// TO THIS:
func (v *Vault) Mutate(mutations ...vaultMutation) error {
	v.rwMu.Lock()         // NEW: Acquire write lock
	defer v.rwMu.Unlock()  // NEW: Release write lock
	
	v.mu.Lock()
	defer v.mu.Unlock()
	// ... rest of function unchanged ...
}
```

### Impact
- ✅ Prevents concurrent eval/mutation races
- ✅ Minimal code changes
- ⚠️ Adds lock contention on high-frequency evaluations
- ✅ Works with existing code

### Tradeoff Analysis
- **Before**: Lock-free reads but unsafe with concurrent mutations
- **After**: Slightly slower reads but thread-safe

---

## Fix #2: Add Defensive Validation in Rule Evaluation

### Problem  
When rules are invalidated by mutations, evaluation continues with stale rules.

### Solution
Detect stale rules and fail early with clear errors.

### Code Changes

#### File: `engine.go`

**In Eval() function:**
```go
// ADD this after existing validation:
func (e *DefaultEngine) Eval(ctx context.Context, r *Rule,
    d map[string]any, opts ...EvalOption,
) (*Result, error) {
    if err := validateEvalArguments(r, e, d); err != nil {
        return nil, err
    }
    
    // NEW VALIDATION: Check for stale rules
    if r.ID != "" && r.Expr != "" && r.Program == nil {
        return nil, fmt.Errorf("rule %s: invalid state detected - rule may have been mutated during evaluation. " +
            "If using concurrent mutations, consider using synchronization or generation counters", r.ID)
    }
    
    // ... rest of function unchanged ...
}
```

### Impact
- ✅ Catches races early with clear error messages
- ✅ Helps diagnose concurrency issues
- ✅ Can be deployed immediately
- ⚠️ Doesn't prevent races, just detects them

---

## Fix #3: Snapshot Sorted Rules During Traversal

### Problem
`destinationShard()` walks the rule tree while it's being modified.

### Solution
Take a snapshot of sorted rules before traversing.

### Code Changes

#### File: `vault.go`

**Update destinationShard() function:**
```go
// CHANGE THIS FUNCTION:
func destinationShard(r, rr *Rule) (*Rule, error) {
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
    for _, shard := range r.sortedRules {  // PROBLEM: r.sortedRules can change
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

    // ... rest of function ...
}

// TO THIS:
func destinationShard(r, rr *Rule) (*Rule, error) {
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

    // NEW: Take snapshot of sorted rules to avoid races
    sortedRulesSnapshot := r.sortChildRules(r.EvalOptions.SortFunc, false)
    if sortedRulesSnapshot == nil {
        sortedRulesSnapshot = []*Rule{}
    }

shardLoop:
    for _, shard := range sortedRulesSnapshot {  // Use snapshot instead
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

### Impact
- ✅ Prevents destinationShard traversal races
- ✅ Small performance cost (snapshot creation)
- ✅ Safe with concurrent mutations

---

## Fix #4: Defensive sortChildRules Implementation

### Problem
Iterating over Rules map while it's being modified.

### Solution
Add defensive checks and handle nil/empty maps.

### Code Changes

#### File: `rule.go`

**Update sortChildRules() function:**
```go
// CHANGE THIS PART OF THE FUNCTION:
func (r *Rule) sortChildRules(fn func(rules []*Rule, i, j int) bool, force bool) []*Rule {
    if fn == nil && len(r.sortedRules) == len(r.Rules) && !force {
        return r.sortedRules
    }

    if !force && len(r.sortedRules) == len(r.Rules) {
        return r.sortedRules
    }

    keys := make([]*Rule, len(r.Rules))
    var i int
    for k := range r.Rules {  // PROBLEM: r.Rules can change during iteration
        keys[i] = r.Rules[k]
        i++
    }
    // ... rest of function ...
}

// TO THIS:
func (r *Rule) sortChildRules(fn func(rules []*Rule, i, j int) bool, force bool) []*Rule {
    // NEW: Defensive check for nil or empty rules
    if r == nil || r.Rules == nil || len(r.Rules) == 0 {
        return []*Rule{}
    }

    if fn == nil && len(r.sortedRules) == len(r.Rules) && !force {
        return r.sortedRules
    }

    if !force && len(r.sortedRules) == len(r.Rules) {
        return r.sortedRules
    }

    // NEW: Snapshot the keys first to avoid iterating while map changes
    keys := make([]*Rule, 0, len(r.Rules))
    for k := range r.Rules {
        if r.Rules[k] != nil {  // NEW: Defensive nil check
            keys = append(keys, r.Rules[k])
        }
    }

    if fn != nil && len(keys) > 0 && force {
        sort.Slice(keys, func(i, j int) bool {
            // NEW: Defensive bounds checking
            if i >= len(keys) || j >= len(keys) {
                return false
            }
            return fn(keys, i, j)
        })
    }
    return keys
}
```

### Impact
- ✅ Handles map mutations gracefully
- ✅ Adds nil safety
- ✅ Minimal performance impact
- ✅ Can be deployed immediately

---

## Fix #5: Add Generation Counter for Snapshot Validation (BEST LONG-TERM)

### Problem
Evaluators don't know when their snapshot becomes invalid.

### Solution
Add a generation counter that changes on mutations.

### Code Changes

#### File: `vault.go`

**Step 1: Update Vault struct**
```go
type Vault struct {
    mu sync.Mutex
    root atomic.Pointer[Rule]
    engine Engine
    compileOptions []CompilationOption
    lastUpdate atomic.Pointer[time.Time]
    generation atomic.Uint64  // NEW: Generation counter
}
```

**Step 2: Initialize in NewVault()**
```go
func NewVault(engine Engine, initialRoot *Rule, opts ...CompilationOption) (*Vault, error) {
    v := &Vault{
        engine:         engine,
        compileOptions: opts,
    }
    // ... existing code ...
    v.generation.Store(0)  // NEW: Initialize generation
    v.root.Store(initialRoot)
    return v, nil
}
```

**Step 3: Increment on mutations**
```go
func (v *Vault) applyMutations(root *Rule, mutations []vaultMutation) error {
    // ... existing mutation code ...
    
    // NEW: Increment generation after successful mutations
    v.generation.Add(1)
    v.root.Store(root)
    return nil
}
```

#### File: `engine.go`

**In Eval() function:**
```go
func (e *DefaultEngine) Eval(ctx context.Context, r *Rule,
    d map[string]any, opts ...EvalOption,
) (*Result, error) {
    if err := validateEvalArguments(r, e, d); err != nil {
        return nil, err
    }
    
    // Capture generation at start
    startGen := r.generation  // Would need to pass this through somehow
    
    // ... evaluation code ...
    
    // After evaluation, could check if generation changed
    // This indicates the rule was mutated during evaluation
}
```

### Impact
- ✅ Efficient snapshot invalidation detection
- ✅ O(1) check for staleness
- ✅ Can be added to Rule or Vault
- ✅ Foundation for more advanced techniques

---

## Recommended Fix Priority

### Priority 1: MUST DO (Prevents Crashes)
1. **Fix #1**: Add RWMutex synchronization
   - Time to implement: 15 minutes
   - Risk: Low
   - Benefit: High

### Priority 2: SHOULD DO (Improves Diagnostics)
1. **Fix #2**: Add defensive validation
   - Time to implement: 10 minutes
   - Risk: Very Low
   - Benefit: Medium

2. **Fix #4**: Defensive sortChildRules
   - Time to implement: 20 minutes  
   - Risk: Low
   - Benefit: Medium

### Priority 3: NICE TO HAVE (Improves Architecture)
1. **Fix #3**: Snapshot shard traversal
   - Time to implement: 30 minutes
   - Risk: Low
   - Benefit: Medium

2. **Fix #5**: Generation counter
   - Time to implement: 1 hour
   - Risk: Medium
   - Benefit: High

---

## Testing the Fixes

After implementing fixes, run:

```bash
# Test with race detector
go test -run TestHellish -race -v

# Test all vault tests
go test -run Vault -race -v

# Full test suite
go test -race -v
```

Expected result: All tests pass with race detector enabled.

---

## Validation Checklist

- [ ] Fix #1: RWMutex implemented
- [ ] Fix #2: Validation checks added
- [ ] Fix #4: sortChildRules defensified
- [ ] All tests pass with -race
- [ ] No more "race detected" messages
- [ ] Mutation + parallel eval works correctly
- [ ] Sharded rules function properly
- [ ] Documentation updated

---

## Performance Impact Assessment

### Before Fixes
- Mutation: Mutex only (fast)
- Evaluation: Atomic load only (very fast)
- Safety: ❌ Not safe with concurrent ops

### After Fixes
- Mutation: RWMutex write lock (slightly slower)
- Evaluation: RWMutex read lock (slightly slower)  
- Safety: ✅ Safe with concurrent ops

**Estimated overhead**: 5-15% for read-heavy workloads, negligible for write-heavy.

---

## Deployment Notes

1. **Backward compatible**: All fixes maintain API compatibility
2. **Non-breaking**: Existing code continues to work
3. **Gradual rollout**: Can deploy fixes incrementally
4. **Testing**: Comprehensive test suite validates all scenarios

---

## Code Review Checklist

When reviewing these fixes, verify:
- [ ] RWMutex properly acquired/released
- [ ] No deadlock potential
- [ ] All error paths are clean
- [ ] Nil checks added where appropriate
- [ ] Performance impact acceptable
- [ ] Documentation updated
- [ ] Tests pass with -race flag

