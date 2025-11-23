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
// The client can submit mutations to the Vault rule via the [Mutate]
// method.
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

func (v *Vault) LastUpdate() time.Time {
	return *v.lastUpdate.Load()
}

// vaultMutation defines a single change to the rule tree
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

	// A time stamp to record as the last time the vault was updated
	lastUpdate time.Time

	op mutationOp
}

type mutationOp int

const (
	add mutationOp = iota
	update
	remove
	move
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

// Remove deletes the rule with the id
func Remove(id string) vaultMutation {
	return vaultMutation{
		id: id,
		op: remove,
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
func (v *Vault) Mutate(mutations ...vaultMutation) error {
	root := v.Rule()
	mut, err := v.preProcessMoves(root, mutations)
	if err != nil {
		return fmt.Errorf("preprocessing moves: %w", err)
	}
	rulesTouched := v.rulesTouched(root, mutations)
	for _, r := range rulesTouched {
		fmt.Println("touching ", r.ID)
	}
	return v.applyMutations(root, mut)
}

func (v *Vault) rulesTouched(r *Rule, mut []vaultMutation) []*Rule {
	for _, m := range mut {
		_ = m
	}
	return nil
}

// preProcessMoves converts a "move" mutation into a "delete" and an "add" mutation
func (v *Vault) preProcessMoves(root *Rule, mutations []vaultMutation) ([]vaultMutation, error) {
	mut := slices.Clone(mutations)

	for _, m := range mut {
		if m.op != move {
			continue
		}
		parent := root.FindParent(m.newParent)
		if parent == nil {
			return nil, fmt.Errorf("moving rule %s: from-parent not found", m.id)
		}
		mut = append(mut, Remove(m.id)) // delete from current parent
		rule, _ := root.FindRule(m.id)
		if rule == nil {
			return nil, fmt.Errorf("moving rule %s: not found", m.id)
		}
		mut = append(mut, Add(*rule, m.newParent))
	}
	mut = slices.DeleteFunc(mut, func(m vaultMutation) bool {
		return m.newParent != ""
	})
	return mut, nil
}

func (v *Vault) applyMutations(root *Rule, mutations []vaultMutation) error {
	newRoot := shallowCopy(root)
	// fmt.Println("New root=", newRoot)
	for _, m := range mutations {
		switch m.op {
		case remove:
			if err := v.delete(newRoot, m.id); err != nil {
				return fmt.Errorf("deleting rule %s: %w", m.id, err)
			}
		case timeUpdate:
			v.lastUpdate.Store(&m.lastUpdate)
		default:
			if m.parent == "" {
				p := newRoot.FindParent(m.id)
				if p == nil {
					return fmt.Errorf("parent not found for rule %s", m.id)
				}
				m.parent = p.ID
			}
			if err := v.upsert(newRoot, m); err != nil {
				return fmt.Errorf("upserting rule %s: %w", m.id, err)
			}
		}
	}
	v.root.Store(newRoot)
	return nil
}

// shallowCopy makes a shallow copy of r, allowing changes to the copy of r.Rules, and r's sortedRules
func shallowCopy(r *Rule) *Rule {
	rr := *r
	rr.Rules = maps.Clone(r.Rules)
	rr.sortedRules = slices.Clone(r.sortedRules)
	for k, c := range rr.Rules {
		rr.Rules[k] = shallowCopy(c)
	}
	return &rr
}

// delete removes the rule with m.ID. If given, m.Parent is
// used to find the parent from which to delete the rule.
func (v *Vault) delete(r *Rule, id string) error {
	parent := r.FindParent(id)
	if parent == nil {
		return fmt.Errorf("parent not found for rule %s", id)
	}
	delete(parent.Rules, id)
	parent.sortedRules = r.sortChildRules(r.EvalOptions.SortFunc, true)
	return nil
}

//
// // findParent uses the m.ID and m.Parent to find the parent of the
// // rule with m.ID
// func (v *Vault) findParent(r *Rule, m ruleMutation) (parent *Rule) {
// 	if m.parent != "" {
// 		parent, _ = r.FindRule(m.parent)
// 	}
// 	// if parent == nil {
// 	 	parent = findParent(r, nil, m.id)
// 	// }
// 	return parent
// }

// upsert either updates (replaces) or adds the rule in m.Rule.
// When adding a new rule, m.Parent is required.
func (v *Vault) upsert(r *Rule, m vaultMutation) error {
	parent, _ := r.FindRule(m.parent)
	if parent == nil {
		return fmt.Errorf("parent not found (%s)", m.parent)
	}
	err := v.engine.Compile(m.rule, v.compileOptions...)
	if err != nil {
		return fmt.Errorf("compiling upsert rule %s: %w", m.rule.ID, err)
	}

	if parent.Rules == nil {
		parent.Rules = map[string]*Rule{}
	}
	parent.Rules[m.id] = m.rule
	parent.sortedRules = r.sortChildRules(r.EvalOptions.SortFunc, true)
	return nil
}
