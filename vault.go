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

// vaultMutation defines a single change to the the vault
type vaultMutation struct {
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

	// Required when changing the ID of a rule
	newID string

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
)

// Add returns a mutation that adds the rule to the parent.
// The parent must exist.
// If the Vault rule is sharded the rule will be placed in the
// parent rule determined by the sharding rules instead of the parent.
func Add(r *Rule, parent string) vaultMutation {
	return vaultMutation{
		id:     r.ID,
		rule:   r,
		parent: parent,
		op:     add,
	}
}

// Add returns a mutation that adds the rule to the parent,
// bypassing any sharding specifications.
func addDoNotShard(r *Rule, parent string) vaultMutation {
	return vaultMutation{
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
func Update(r *Rule) vaultMutation {
	return vaultMutation{
		id:   r.ID,
		rule: r,
		op:   update,
	}
}

// Delete deletes the rule with the id
func Delete(id string) vaultMutation {
	return vaultMutation{
		id: id,
		op: deleteOp,
	}
}

// Move moves the rule with the id to the newParent.
// The newParent must exist
func Move(id string, newParent string) vaultMutation {
	return vaultMutation{
		id:        id,
		newParent: newParent,
		op:        move,
	}
}

// LastUpdate updates the LastUpdate timestamp in the Vault
func LastUpdate(t time.Time) vaultMutation {
	return vaultMutation{
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
func (v *Vault) Mutate(mutations ...vaultMutation) error {
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
func (v *Vault) preProcessShardChanges(root *Rule, mut []vaultMutation) ([]vaultMutation, error) {
	u := make([]vaultMutation, 0, len(mut))
	for _, m := range mut {
		if m.op != update {
			u = append(u, m)
			continue
		}
		currentParent := root.FindParent(m.id)
		targetParent, err := destinationParent(root, m.rule)
		if err != nil {
			return nil, fmt.Errorf("finding destination parent for %s: %w", m.id, err)
		}

		if currentParent == nil {
			return nil, fmt.Errorf("parent not found for %s", m.id)
		}

		if targetParent == nil {
			return nil, fmt.Errorf("no target parent found for %s", m.id)
		}
		switch {
		case currentParent.ID != targetParent.ID:
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
func (v *Vault) preProcessMoves(root *Rule, mut []vaultMutation) ([]vaultMutation, error) {
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

// destinationParent returns either the parent (r), or a shard within r where
// the rule rr should be placed
func destinationParent(r, rr *Rule) (*Rule, error) {
	toReturn := r.FindParent(rr.ID)
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
	for _, shard := range r.sortedRules {
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

	// We're in a sharding situation, and we found a matching shard,
	// check if there is a more specific shard in that shard's children
	if shardCount > 0 && toReturn != nil {
		for _, c := range toReturn.Rules {
			sh, err := destinationParent(c, rr)
			if err != nil {
				return nil, err
			}
			if sh != nil {
				return sh, nil
			}
		}
	}
	return toReturn, nil
}

// applyMutations performs the mutations against the root rule.
func (v *Vault) applyMutations(root *Rule, mutations []vaultMutation) error {
	// we keep track of which rules have already been cloned, i.e., made safe
	// for modifications
	alreadyCopied := make(map[*Rule]any)
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
		}
	}
	v.root.Store(root)
	return nil
}

// shallowCopy makes a shallow copy of r, so that we can modify its Rules map.
func shallowCopy(r *Rule) *Rule {
	rr := *r
	rr.Rules = maps.Clone(r.Rules)
	// fmt.Printf("r origran: %T\n", r.Program)
	//
	// // rr.Program = r.Program
	// fmt.Printf("rr origran: %T\n", rr.Program)
	return &rr
}

// delete removes the rule with the id from the root rule r.
func (v *Vault) delete(r *Rule, alreadyCopied map[*Rule]any, id string) (*Rule, map[*Rule]any, error) {
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
func (v *Vault) update(r, newRule *Rule, alreadyCopied map[*Rule]any) (*Rule, map[*Rule]any, error) {
	// Special case to allow replacing the root
	if newRule.ID == r.ID {
		r = newRule
		return r, alreadyCopied, nil
	}

	parent := r.FindParent(newRule.ID)
	if parent == nil {
		return nil, nil, fmt.Errorf("parent not found for rule: %s", newRule.ID)
	}
	//
	// var err error
	//
	// // Check if the updated rule should be in a different shard
	// destinationShardRule, err := destinationParent(r, newRule)
	// if err != nil {
	// 	return nil, nil, fmt.Errorf("checking destination shard: %w", err)
	// }
	// if destinationShardRule == nil {
	// 	return nil, nil, fmt.Errorf("no destination parent found for %s", newRule.ID)
	// }
	//
	// // If destination shard is different from current parent, move the rule to the correct shard
	// if destinationShardRule != nil && destinationShardRule.ID != parent.ID {
	// 	// Delete from current parent
	// 	r, alreadyCopied, err = v.delete(r, alreadyCopied, newRule.ID)
	// 	if err != nil {
	// 		return nil, nil, fmt.Errorf("deleting rule %s for shard movement: %w", newRule.ID, err)
	// 	}
	// 	// Add to destination shard
	// 	r, alreadyCopied, err = v.add(r, newRule, alreadyCopied, destinationShardRule.ID)
	// 	if err != nil {
	// 		return nil, nil, fmt.Errorf("adding rule %s to destination shard: %w", newRule.ID, err)
	// 	}
	// 	return r, alreadyCopied, nil
	// }
	//
	// Normal update: rule stays in same parent
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
func (v *Vault) add(r, newRule *Rule, alreadyCopied map[*Rule]any, parentID string, doNotShard bool) (*Rule, map[*Rule]any, error) {
	if newRule == nil || r == nil {
		return nil, nil, fmt.Errorf("rule inconsistency: nil")
	}
	if !doNotShard {
		target, err := destinationParent(r, newRule)
		if err != nil {
			return nil, nil, err
		}
		if target != nil {
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
	// r.Rules[parent.ID] = parent
	return r, alreadyCopied, nil
}

// makeSafePath makes shallow copies of rules between the root and the rule with the id,
// so that updates can be made to those rules. If a rule has already been copied, cloning
// is skipped.
func makeSafePath(root *Rule, alreadyCopied map[*Rule]any, id string) (*Rule, map[*Rule]any, error) {
	path := root.Path(id)
	for i := range path {
		var parent, child, updated *Rule
		var err error
		parent = path[i]
		if i < len(path)-1 {
			child = path[i+1]
		} else {
			updated, alreadyCopied, err = makeSafe(parent, nil, alreadyCopied)
			if err != nil {
				return nil, nil, err
			}
		}
		updated, alreadyCopied, err = makeSafe(parent, child, alreadyCopied)
		if err != nil {
			return nil, nil, err
		}
		if parent == root {
			root = updated
		}
	}
	return root, alreadyCopied, nil
}

// makeSafe makes the parent safe, then the child, and adds the child
// to the (now) safe parent. If the only the parent needs to be made safe,
// such as in the case of adding a new rule to the Vault, pass child as nil.
func makeSafe(parent, child *Rule, alreadyCopied map[*Rule]any) (*Rule, map[*Rule]any, error) {
	if parent != nil {
		if _, ok := alreadyCopied[parent]; !ok {
			parent = shallowCopy(parent)
			alreadyCopied[parent] = nil
		}
	}
	if child != nil {
		if parent == nil {
			return nil, nil, fmt.Errorf("orphaned rule: %s", child.ID)
		}
		if _, ok := alreadyCopied[child]; !ok {
			childCopy := shallowCopy(child)
			parent.Add(childCopy)
			alreadyCopied[child] = nil
		}
	}
	return parent, alreadyCopied, nil
}
