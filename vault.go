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
	deleteOp
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
	r := v.Rule()
	mut, err := v.preProcessMoves(r, mutations)
	if err != nil {
		return fmt.Errorf("preprocessing moves: %w", err)
	}
	// rulesTouched := v.rulesTouched(r, mutations)
	// fmt.Println("Tocuched: ", rulesTouched)
	// for _, r := range rulesTouched {
	// 	fmt.Println("touching ", r.ID)
	// }
	return v.applyMutations(r, mut)
}

// func (v *Vault) rulesTouched(r *Rule, mut []vaultMutation) []*Rule {
// 	var rs []*Rule
// 	fmt.Println("R = ", r)
// 	for _, m := range mut {
// 		_, ancestors := r.FindRule(m.id)
// 		p := r.FindParent(m.id)
// 		fmt.Println("AS: ", ancestors, m.id, p)
// 		rs = append(rs, ancestors...)
// 	}
// 	return rs
// }

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
		mut = append(mut, Delete(m.id)) // delete from current parent
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
		default:
			return fmt.Errorf("unsupported operation: %d", m.op)
			// default:
			// 	// root, alreadyCopied = copyAncestorsAndMe(root, alreadyCopied, m.parent)
			// 	if err := v.add(root, m.rule, m.parent); err != nil {
			// 		return fmt.Errorf("adding rule %s: %w", m.rule.ID, err)
			// 	}
			// 	if m.parent == "" {
			// 		p := root.FindParent(m.id)
			// 		if p == nil {
			// 			return fmt.Errorf("parent not found for rule %s", m.id)
			// 		}
			// 		m.parent = p.ID
			// 	}
			// 	// root = copyAncestors(root, m.id)
			// 	if err := v.upsert(root, m); err != nil {
			// 		return fmt.Errorf("upserting rule %s: %w", m.id, err)
			// 	}
		}
	}
	v.root.Store(root)
	return nil
}

// shallowCopy makes a shallow copy of r
func shallowCopy(r *Rule) *Rule {
	rr := *r
	rr.Rules = maps.Clone(r.Rules)
	return &rr
}

// // shallowCopy makes a shallow copy of r, allowing changes to the copy of r.Rules, and r's sortedRules
// func shallowCopy(r *Rule) *Rule {
// 	rr := *r
// 	rr.Rules = maps.Clone(r.Rules)
// 	rr.sortedRules = slices.Clone(r.sortedRules)
// 	for k, c := range rr.Rules {
// 		rr.Rules[k] = shallowCopy(c)
// 	}
// 	return &rr
// }

// delete removes the rule with m.ID. If given, m.Parent is
// used to find the parent from which to delete the rule.
func (v *Vault) delete(r *Rule, alreadyCopied []*Rule, id string) (*Rule, []*Rule, error) {
	parent := r.FindParent(id)
	if parent == nil {
		return nil, nil, fmt.Errorf("parent not found for rule %s", id)
	}
	r, alreadyCopied = copyAncestorsAndMe(r, alreadyCopied, parent.ID)

	parentInNew := r.FindParent(id)
	if parentInNew == nil {
		return nil, nil, fmt.Errorf("parent not found for rule after cloning %s", id)
	}

	delete(parentInNew.Rules, id)
	parentInNew.sortedRules = parentInNew.sortChildRules(parentInNew.EvalOptions.SortFunc, true)
	return r, alreadyCopied, nil
}

func (v *Vault) update(r, newRule *Rule, alreadyCopied []*Rule) (*Rule, []*Rule, error) {
	parent, _ := r.FindRule(newRule.ID)
	if parent == nil {
		return nil, nil, fmt.Errorf("parent not found for rule: %s", newRule.ID)
	}
	r, alreadyCopied = copyAncestorsAndMe(r, alreadyCopied, parent.ID)
	err := v.engine.Compile(newRule, v.compileOptions...)
	if err != nil {
		return nil, nil, fmt.Errorf("compiling new rule %s: %w", newRule.ID, err)
	}

	parentInNew := r.FindParent(newRule.ID)
	if parentInNew == nil {
		return nil, nil, fmt.Errorf("parent not found for rule after cloning: %s", newRule.ID)
	}
	if parentInNew.Rules == nil {
		parentInNew.Rules = map[string]*Rule{}
	}
	parentInNew.Rules[newRule.ID] = newRule
	parentInNew.sortedRules = parentInNew.sortChildRules(parentInNew.EvalOptions.SortFunc, true)
	return r, alreadyCopied, nil
}

// upsert either updates (replaces) or adds the rule in m.Rule.
// When adding a new rule, m.Parent is required.
func (v *Vault) add(r, newRule *Rule, alreadyCopied []*Rule, parentID string) (*Rule, []*Rule, error) {
	parent, _ := r.FindRule(parentID)
	if parent == nil {
		return nil, nil, fmt.Errorf("parent not found: %s", parentID)
	}

	r, alreadyCopied = copyAncestorsAndMe(r, alreadyCopied, parent.ID)
	err := v.engine.Compile(newRule, v.compileOptions...)
	if err != nil {
		return nil, nil, fmt.Errorf("compiling new rule %s: %w", newRule.ID, err)
	}

	parentInNew, _ := r.FindRule(parentID)
	if parentInNew == nil {
		return nil, nil, fmt.Errorf("parent not found for rule after cloning: %s", newRule.ID)
	}
	if parentInNew.Rules == nil {
		parentInNew.Rules = map[string]*Rule{}
	}
	parentInNew.Rules[newRule.ID] = newRule
	parentInNew.sortedRules = parentInNew.sortChildRules(parentInNew.EvalOptions.SortFunc, true)
	return r, alreadyCopied, nil
}

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

func copyAncestorsAndMe(r *Rule, alreadyCopied []*Rule, id string) (*Rule, []*Rule) {
	path := r.Path(id)
	for _, p := range path {
		if slices.ContainsFunc(alreadyCopied, func(a *Rule) bool {
			return a.ID == p.ID
		}) {
			continue
		}
		// p is the root, has no parents
		if p == r {
			r = shallowCopy(r)
			continue
		}
		r.Rules[p.ID] = shallowCopy(p)
	}
	return r, path
}
