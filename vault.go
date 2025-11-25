package indigo

import (
	"fmt"
	"maps"
	"slices"
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
type Vault struct {
	// Current immutable root
	root atomic.Pointer[Rule]

	// the Indigo engine which will be used to compile rules before adding them
	// to the root rule
	engine Engine

	// A list of compilation options, required for when we compile the rule
	// before adding to the vault
	compileOptions []CompilationOption

	// An optional timestamp maintained by the client to keep track of when the
	// Vault was last updated. Initialized to time.Time{} on Vault creation.
	lastUpdate atomic.Pointer[time.Time]
}

// NewVault creates a new Vault with an optional initial rule tree.
// If no initial root is provided, a default rule with id "root" is created.
// The Vault will compile rules passed to it in [Mutate] using the compilation options.
func NewVault(engine Engine, initialRoot *Rule, opts ...CompilationOption) (*Vault, error) {
	v := &Vault{
		engine:         engine,
		compileOptions: opts,
	}
	if initialRoot == nil {
		initialRoot = NewRule("root", "")
	}

	err := v.engine.Compile(initialRoot, opts...)
	if err != nil {
		return nil, fmt.Errorf("compiling initial root for the vault: %w", err)
	}
	v.lastUpdate.Store(&time.Time{})
	v.root.Store(initialRoot)
	return v, nil
}

// Rule returns the current immutable root rule for evaluation or inspection.
func (v *Vault) Rule() *Rule {
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
}

// mutationOp is an enum for the types of mutations supported by the Vault
type mutationOp int

const (
	add mutationOp = iota
	update
	deleteOp
	move
	rename
	timeUpdate
)

// Add returns a mutation that adds the rule to the parent.
// The parent must exist
func Add(r Rule, parent string) vaultMutation {
	return vaultMutation{
		id:     r.ID,
		rule:   &r,
		parent: parent,
		op:     add,
	}
}

// Update returns a mutation that replaces the rule with the
// id r.ID with the new rule
func Update(r Rule) vaultMutation {
	return vaultMutation{
		id:   r.ID,
		rule: &r,
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
// mutations in sequence. Each rule is compiled before being added to the vault.
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
func (v *Vault) Mutate(mutations ...vaultMutation) error {
	r := v.Rule()
	mut, err := v.preProcessMoves(r, mutations)
	if err != nil {
		return fmt.Errorf("preprocessing moves: %w", err)
	}
	return v.applyMutations(r, mut)
}

// preProcessMoves converts a "move" mutation into a "delete" and an "add" mutation
// When we process mutations later, the "move" operation will be ignored.
func (v *Vault) preProcessMoves(root *Rule, mut []vaultMutation) ([]vaultMutation, error) {
	for _, m := range mut {
		if m.op != move {
			continue
		}
		rule, _ := root.FindRule(m.id)
		if m.newParent == m.id {
			return nil, fmt.Errorf("cannot move rule %s to itself", m.id)
		}
		if found, _ := rule.FindRule(m.newParent); found != nil {
			return nil, fmt.Errorf("cannot move rule %s to its descendant %s", m.id, m.newParent)
		}
		if rule == nil {
			return nil, fmt.Errorf("moving rule %s: not found", m.id)
		}
		mut = append(mut, Delete(m.id))            // delete from current parent
		mut = append(mut, Add(*rule, m.newParent)) // add to new parent
	}
	return mut, nil
}

// applyMutations performs the mutations against the root rule.
func (v *Vault) applyMutations(root *Rule, mutations []vaultMutation) error {
	// we keep track of which rules have already been cloned, i.e., made safe
	// for modifications
	var alreadyCopied []*Rule
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
			root, alreadyCopied, err = v.add(root, m.rule, alreadyCopied, m.parent)
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
	return &rr
}

// delete removes the rule with the id from the root rule r.
func (v *Vault) delete(r *Rule, alreadyCopied []*Rule, id string) (*Rule, []*Rule, error) {
	parent := r.FindParent(id)
	if parent == nil {
		return nil, nil, fmt.Errorf("parent not found for rule %s", id)
	}
	r, alreadyCopied = makeSafePath(r, alreadyCopied, parent.ID)

	parentInNew := r.FindParent(id)
	if parentInNew == nil {
		return nil, nil, fmt.Errorf("parent not found for rule after cloning %s", id)
	}

	delete(parentInNew.Rules, id)
	parentInNew.sortedRules = parentInNew.sortChildRules(parentInNew.EvalOptions.SortFunc, true)
	return r, alreadyCopied, nil
}

// update replaces the rule with the id newRule.ID with newRule inside the root rule r.
func (v *Vault) update(r, newRule *Rule, alreadyCopied []*Rule) (*Rule, []*Rule, error) {
	// Special case to allow replacing the root
	if newRule.ID == r.ID {
		r = newRule
		return r, alreadyCopied, nil
	}

	parent := r.FindParent(newRule.ID)
	if parent == nil {
		return nil, nil, fmt.Errorf("parent not found for rule: %s", newRule.ID)
	}
	r, alreadyCopied = makeSafePath(r, alreadyCopied, parent.ID)
	err := v.engine.Compile(newRule, v.compileOptions...)
	if err != nil {
		return nil, nil, fmt.Errorf("compiling new rule %s: %w", newRule.ID, err)
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
func (v *Vault) add(r, newRule *Rule, alreadyCopied []*Rule, parentID string) (*Rule, []*Rule, error) {
	r, alreadyCopied = makeSafePath(r, alreadyCopied, parentID)
	if newRule.ID == "" {
		return nil, nil, fmt.Errorf("rule ID cannot be empty")
	}
	if existing, _ := r.FindRule(newRule.ID); existing != nil {
		return nil, nil, fmt.Errorf("rule with ID %s already exists", newRule.ID)
	}
	err := v.engine.Compile(newRule, v.compileOptions...)
	if err != nil {
		return nil, nil, fmt.Errorf("compiling new rule %s: %w", newRule.ID, err)
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
func makeSafePath(root *Rule, alreadyCopied []*Rule, id string) (*Rule, []*Rule) {
	path := root.Path(id)
	for _, p := range path {
		if slices.ContainsFunc(alreadyCopied, func(a *Rule) bool {
			return a.ID == p.ID
		}) {
			continue
		}
		// p is the root, has no parents
		if p == root {
			root = shallowCopy(root)
			continue
		}
		root.Rules[p.ID] = shallowCopy(p)
	}
	alreadyCopied = append(alreadyCopied, path...)
	return root, alreadyCopied
}
