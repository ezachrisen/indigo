package indigo

import (
	"fmt"
	"maps"
	"sync"
	"sync/atomic"
	"time"
)

// Vault provides lock-free, hot-reloadable, hierarchical rule management
// with full support for add, update, delete, and move operations.
//
// It provides safe access to an immutable rule which can be retrieved with the [Rule] function
// for evaluation or inspection.
//
// The client can submit mutations to the Vault via the [Mutate]
// method.
//
// Clients do not see the updated rules until all mutations submitted to [Mutate]
// succeed; if one fails no updates are applied.
//
// To use the Vault, your rules must have globally unique IDs.
//
// Clients should NOT modify rules outside the vault. Any rule added to the vault
// or retrieved from it should be considered read-only.
type Vault struct {
	// The mutex is used to ensure multiple writers (calling Mutate) do not interfere
	// with each other; the mutex is not used for readers, who always retrieve
	// a clean Rule without waiting for a read lock with the [Rule] method.
	mu sync.Mutex

	// Current immutable root
	root atomic.Pointer[Rule]

	// A list of compilation options, required for when we compile the rule
	// before adding to the vault
	compileOptions []CompilationOption

	// An optional timestamp maintained by the client to keep track of when the
	// Vault was last updated. Initialized to time.Time{} on Vault creation.
	lastUpdate atomic.Pointer[time.Time]
}

// NewVault creates a new Vault with an nitial rule tree.
//
// The Vault will call BuildShards on initialRoot if you provide it.
// When you apply mutations to the rule in the vault, the vault will automatically apply sharding.
func NewVault(root *Rule, opts ...CompilationOption) (*Vault, error) {
	v := &Vault{
		compileOptions: opts,
	}

	if root == nil {
		return nil, fmt.Errorf("missing root rule")
	}

	err := root.BuildShards()
	if err != nil {
		return nil, fmt.Errorf("building shards on root: %w", err)
	}

	v.lastUpdate.Store(&time.Time{})
	v.root.Store(root)
	return v, nil
}

// Rule returns the current immutable root rule for evaluation or inspection.
func (v *Vault) ImmutableRule() *Rule {
	return v.root.Load()
}

// LastUpdate returns the last time the Vault was updated via a LastUpdate mutation
func (v *Vault) LastUpdate() time.Time {
	return *v.lastUpdate.Load()
}

// Mutation defines a single change to the the vault
// Use the Add, Update and Delete functions to create a Mutation.
type Mutation struct {
	// Required; gobally unique id for the rule being changed/added
	id string

	// Optional; rule is the new rule that will replace an existing rule or
	// be added to the parent. If rule is nil, the rule with ID will
	// be deleted
	rule *Rule

	// Optional for updates, moves and deletes, required for adds
	parent string

	// Required when moving a rule from one parent to another
	newParent string

	// A time stamp to record as the last time the vault was updated
	// Will update the Vault's last update time
	lastUpdate time.Time

	// What kind of operation this is
	op mutationOp

	// If this is set, when a rule is used in an add operation,
	// we will not try to find the right shard for the rule,
	// rather it will be placed in the newParent rule
	doNotShard bool
}

// mutationOp is an enum for the types of mutations supported by the Vault
type mutationOp int

const (
	add mutationOp = iota
	update
	deleteOp
	move
	timeUpdate
	noOp // does nothing
)

// Add returns a mutation that adds the rule to the parent.
// The parent must exist.
// If the Vault rule is sharded the rule will be placed in the
// parent rule determined by the sharding rules instead of the parent.
func Add(r *Rule, parent string) Mutation {
	if r == nil {
		return Mutation{op: noOp}
	}
	return Mutation{
		id:     r.ID,
		rule:   r,
		parent: parent,
		op:     add,
	}
}

// Add returns a mutation that adds the rule to the parent,
// bypassing any sharding specifications.
func addDoNotShard(r *Rule, parent string) Mutation {
	return Mutation{
		id:         r.ID,
		rule:       r,
		parent:     parent,
		op:         add,
		doNotShard: true,
	}
}

// Update returns a mutation that replaces the rule with the
// id r.ID with the new rule. Keep in mind that this not only updates
// the rule's fields, such as expression or meta, but also all of its
// children.
func Update(r *Rule) Mutation {
	if r == nil {
		return Mutation{op: noOp}
	}
	return Mutation{
		id:   r.ID,
		rule: r,
		op:   update,
	}
}

// Delete deletes the rule with the id
func Delete(id string) Mutation {
	return Mutation{
		id: id,
		op: deleteOp,
	}
}

// Move moves the rule with the id to the newParent.
// The newParent must exist
func Move(id string, newParent string) Mutation {
	return Mutation{
		id:        id,
		newParent: newParent,
		op:        move,
	}
}

// LastUpdate updates the LastUpdate timestamp in the Vault
func LastUpdate(t time.Time) Mutation {
	return Mutation{
		lastUpdate: t,
		op:         timeUpdate,
	}
}

// Mutate makes the changes to the rule stored in the Vault, applying the
// mutations in sequence.
// At the end of all mutations, the resulting root rule becomes the new
// active rule in the vault, and can be retrieved with the [Rule] function.
//
// Clients do not see the updated rules until all mutations submitted to [Mutate]
// succeed; if one fails no updates are applied.
//
// This ensures that clients see a consistent rule set without partial updates.
//
// Mutations incur memory cost equal to the total size of the ancestor rules of
// the rule being mutated; only children who are affected are copied.
//
// In this examople, if grandhchild_2 is modified, the root and the child_1
// rules will be copied, and their child rule maps cloned (though the rules in the maps,
// such as child_2 and grandchild_2, grandchild_3 will not be cloned.)
//
//	root                             <-- Cloned
//	├── child_1                      <-- Cloned
//	│   ├── grandchild_1
//	│   └── grandchild_2             <-- Inserted into the cloned child_1 Rules map
//	└── child_2
//	    └── grandchild_3
//
// Moves incur the cost in both the origin and destination ancestors.
//
// To clear a vault, replace the root rule with a new, empty rule.
//
// Vaults support sharding, and will automatically place mutated rules in the correct
// shard.
func (v *Vault) Mutate(mutations ...Mutation) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	r := v.ImmutableRule()
	mut, err := v.preProcessMoves(r, mutations)
	if err != nil {
		return fmt.Errorf("preprocessing moves: %w", err)
	}
	mut, err = v.preProcessShardChanges(r, mut)
	if err != nil {
		return fmt.Errorf("preprocessing shard changes: %w", err)
	}
	return v.applyMutations(r, mut)
}

// preProcessShardChanges determines if an update to a rule will cause it to move
// to a different shard. If so, the update will be replaced with delete and add operations.
func (v *Vault) preProcessShardChanges(root *Rule, mut []Mutation) ([]Mutation, error) {
	u := make([]Mutation, 0, len(mut))
	for _, m := range mut {
		if m.op != update {
			u = append(u, m)
			continue
		}
		currentParent := root.FindParent(m.id)
		if currentParent == nil {
			return nil, fmt.Errorf("parent not found for %s", m.id)
		}

		targetParent := destinationShard(root, m.rule)
		// if targetParent != nil {
		// 	fmt.Println("Target parent for ", m.id, " is ", targetParent.ID)
		// }
		switch {
		case targetParent != nil && currentParent.ID != targetParent.ID:
			u = append(u, Delete(m.id))                 // delete from current parent
			u = append(u, Add(m.rule, targetParent.ID)) // add to new parent
		default:
			u = append(u, m)
		}
	}
	return u, nil
}

// preProcessMoves converts a "move" mutation into a "delete" and an "add" mutation
// When we process mutations later, the "move" operation will be ignored.
func (v *Vault) preProcessMoves(root *Rule, mut []Mutation) ([]Mutation, error) {
	for _, m := range mut {
		if m.op != move {
			continue
		}
		if m.newParent == m.id {
			return nil, fmt.Errorf("cannot move rule %s to itself", m.id)
		}
		rule, _ := root.FindRule(m.id)
		if rule == nil {
			return nil, fmt.Errorf("moving rule %s: not found", m.id)
		}
		if found, _ := rule.FindRule(m.newParent); found != nil {
			return nil, fmt.Errorf("cannot move rule %s to its descendant %s", m.id, m.newParent)
		}
		p, _ := root.FindRule(m.newParent)
		if p == nil {
			return nil, fmt.Errorf("destination parent %s not found", m.newParent)
		}
		if p.shard {
			return nil, fmt.Errorf("attempt to place rule in shard %s", p.ID)
		}
		mut = append(mut, Delete(m.id))                     // delete from current parent
		mut = append(mut, addDoNotShard(rule, m.newParent)) // add to new parent
	}
	return mut, nil
}

// destinationShard returns either the parent (r), or a shard within r where
// the rule rr should be placed
func destinationShard(root, rr *Rule) *Rule {
	var toReturn *Rule
	if matchMeta(root, rr) {
		toReturn = root
	}

	for _, shard := range root.sortedRules {
		if matchMeta(shard, rr) {
			toReturn = shard
			break
		}
	}

	if toReturn == nil {
		return nil
	}

	if !toReturn.shard {
		return toReturn
	}
	for _, c := range toReturn.sortedRules {
		if sh := destinationShard(c, rr); sh != nil {
			return sh
		}
	}
	return toReturn
}

// applyMutations performs the mutations against the root rule.
func (v *Vault) applyMutations(root *Rule, mutations []Mutation) error {
	// we keep track of which rules have already been cloned, i.e., made safe
	// for modifications.
	// Map is Original Pointer -> New Copy Pointer
	alreadyCopied := make(map[*Rule]*Rule)
	var err error
	for _, m := range mutations {
		switch m.op {
		case deleteOp:
			root, alreadyCopied, err = v.delete(root, alreadyCopied, m.id)
			if err != nil {
				return fmt.Errorf("deleting rule %s: %w", m.id, err)
			}

		case timeUpdate:
			v.lastUpdate.Store(&m.lastUpdate)

		case add:
			root, alreadyCopied, err = v.add(root, m.rule, alreadyCopied, m.parent, m.doNotShard)
			if err != nil {
				return fmt.Errorf("adding rule %s: %w", m.rule.ID, err)
			}

		case update:
			root, alreadyCopied, err = v.update(root, m.rule, alreadyCopied)
			if err != nil {
				return fmt.Errorf("updating rule %s: %w", m.rule.ID, err)
			}

		case move:
			// we've already handled the move operation in [preprocessMoves]
			continue
		case noOp:
			continue
		}
	}
	v.root.Store(root)
	return nil
}

// shallowCopy makes a shallow copy of r, so that we can modify its Rules map.
func shallowCopy(r *Rule) *Rule {
	rr := *r
	rr.Rules = maps.Clone(r.Rules)
	return &rr
}

// delete removes the rule with the id from the root rule r.
func (v *Vault) delete(r *Rule, alreadyCopied map[*Rule]*Rule, id string) (*Rule, map[*Rule]*Rule, error) {
	parent := r.FindParent(id)
	if parent == nil {
		return nil, nil, fmt.Errorf("parent not found for rule %s", id)
	}
	var err error
	r, alreadyCopied, err = makeSafePath(r, alreadyCopied, parent.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("making safe path: %w", err)
	}

	parentInNew := r.FindParent(id)
	if parentInNew == nil {
		return nil, nil, fmt.Errorf("parent not found for rule after cloning %s", id)
	}

	delete(parentInNew.Rules, id)
	parentInNew.sortedRules = parentInNew.sortChildRules(parentInNew.EvalOptions.SortFunc, true)
	return r, alreadyCopied, nil
}

// update replaces the rule with the id newRule.ID with newRule inside the root rule r.
// If the updated rule's expression matches different shard criteria, it will be moved
// to the appropriate shard automatically.
func (v *Vault) update(r, newRule *Rule, alreadyCopied map[*Rule]*Rule) (*Rule, map[*Rule]*Rule, error) {
	// Special case to allow replacing the root
	if newRule.ID == r.ID {
		r = newRule
		return r, alreadyCopied, nil
	}

	parent := r.FindParent(newRule.ID)
	if parent == nil {
		return nil, nil, fmt.Errorf("parent not found for rule: %s", newRule.ID)
	}

	var err error
	r, alreadyCopied, err = makeSafePath(r, alreadyCopied, parent.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("making safe path: %w", err)
	}
	// we have to find the parent again because it will have been cloned in makeSafePath
	parentInNew, _ := r.FindRule(parent.ID)
	if parentInNew == nil {
		return nil, nil, fmt.Errorf("parent not found for rule after cloning: %s", newRule.ID)
	}
	if parentInNew.Rules == nil {
		return nil, nil, fmt.Errorf("inconsistent state; parent whose child is being updated has no children")
	}

	parentInNew.Rules[newRule.ID] = newRule
	// This step is handled automatically when we compile parent, but we do not want to
	// recompile parent, so we do it manually here
	parentInNew.sortedRules = parentInNew.sortChildRules(parentInNew.EvalOptions.SortFunc, true)
	return r, alreadyCopied, nil
}

// add adds the newRule to the parent rule with parentID, somewhere inside the root rule r
func (v *Vault) add(r, newRule *Rule, alreadyCopied map[*Rule]*Rule, parentID string, doNotShard bool) (*Rule, map[*Rule]*Rule, error) {
	if newRule == nil || r == nil {
		return nil, nil, fmt.Errorf("rule inconsistency: nil")
	}
	if !doNotShard {
		if target := destinationShard(r, newRule); target != nil {
			parentID = target.ID
		}
	}
	var err error
	r, alreadyCopied, err = makeSafePath(r, alreadyCopied, parentID)
	if err != nil {
		return nil, nil, fmt.Errorf("making safe path: %w", err)
	}

	if newRule.ID == "" {
		return nil, nil, fmt.Errorf("rule ID cannot be empty")
	}

	if existing, _ := r.FindRule(newRule.ID); existing != nil {
		return nil, nil, fmt.Errorf("rule with ID %s already exists", newRule.ID)
	}

	parent, _ := r.FindRule(parentID)
	if parent == nil {
		return nil, nil, fmt.Errorf("parent not found for rule after cloning: %s", newRule.ID)
	}
	if parent.Rules == nil {
		parent.Rules = map[string]*Rule{}
	}
	parent.Rules[newRule.ID] = newRule
	// This step is handled automatically when we compile parent, but we do not want to
	// recompile parent, so we do it manually here
	parent.sortedRules = parent.sortChildRules(parent.EvalOptions.SortFunc, true)
	return r, alreadyCopied, nil
}

// makeSafePath makes shallow copies of rules between the root and the rule with the id,
// so that updates can be made to those rules. If a rule has already been copied, cloning
// is skipped.
//
// The alreadyCopied map has the key of the original Rule, and the value is the copied version of the rule.
func makeSafePath(root *Rule, alreadyCopied map[*Rule]*Rule, id string) (*Rule, map[*Rule]*Rule, error) {
	path := root.Path(id) // root, A, B
	if len(path) == 0 {
		return root, alreadyCopied, nil
	}

	var currentSafeParent *Rule

	// Handle Root
	rootOrig := path[0] // root
	if safeRoot, ok := alreadyCopied[rootOrig]; ok {
		currentSafeParent = safeRoot
	} else {
		currentSafeParent = shallowCopy(rootOrig)   // root'
		alreadyCopied[rootOrig] = currentSafeParent // root = root'
	}

	newRoot := currentSafeParent // root'

	// Traverse the rest
	for i := 0; i < len(path)-1; i++ {
		childOrig := path[i+1] // A  // B
		var childSafe *Rule

		if safe, ok := alreadyCopied[childOrig]; ok {
			childSafe = safe
		} else {
			childSafe = shallowCopy(childOrig)   // A' // B'
			alreadyCopied[childOrig] = childSafe // A = A', B = B'
			// IMPORTANT: We must add the child copy to the parent copy
			currentSafeParent.Add(childSafe) // root'.Add(A'), A'.Add
		}
		currentSafeParent = childSafe // A'
	}

	return newRoot, alreadyCopied, nil
}
