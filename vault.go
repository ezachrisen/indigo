package indigo

import (
	"fmt"
	"maps"
	"slices"
	"sync/atomic"
)

// Vault provides lock-free, hot-reloadable, hierarchical rule management
// with full support for add, update, delete, and move operations.
type Vault struct {
	root           atomic.Pointer[Rule] // current immutable root
	engine         Engine
	compileOptions []CompilationOption
}

// RuleMutation defines a single change to the rule tree
type RuleMutation struct {
	// Required; gobally unique ID for the rule being changed/added
	ID string

	// Rule is the new rule that will replace an existing rule or
	// be added to the parent. If Rule is nil, the rule with ID will
	// be deleted
	Rule *Rule

	// Optional for updates and deletes, required for adds
	Parent string

	// Required when moving a rule from one parent to another
	NewParent string
}

// NewVault creates a new Vault with an optional initial rule tree.
// If no initial root is provided, a default rule with id "root" is created.
// The Vault will compile rules passed to it in ApplyMutations, the compilation options
// are used at that time.
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
	v.root.Store(initialRoot)
	return v, nil
}

// CurrentRoot returns the current immutable root rule (for inspection/traversal)
func (v *Vault) CurrentRoot() *Rule {
	return v.root.Load()
}

// ApplyMutations makes the changes to the rule stored in the Vault.
func (v *Vault) ApplyMutations(mutations []RuleMutation) error {
	mut := slices.Clone(mutations)

	for _, m := range mut {
		if m.NewParent != "" {
			parent := v.findParent(v.CurrentRoot(), m)
			if parent == nil {
				return fmt.Errorf("moving rule %s: from-parent not found", m.ID)
			}
			mut = append(mut, RuleMutation{m.ID, nil, parent.ID, ""}) // delete from current parent
			// a move operation only needs to give us the rule ID, then the destination parent ID.
			// If we didn't receive the
			rule := m.Rule
			if rule == nil {
				rule = FindRule(v.CurrentRoot(), m.ID)
			}
			if rule == nil {
				return fmt.Errorf("moving rule %s: not found", m.ID)
			}
			mut = append(mut, RuleMutation{m.ID, rule, m.NewParent, ""}) // add to new parent
		}
	}
	mut = slices.DeleteFunc(mut, func(m RuleMutation) bool {
		return m.NewParent != ""
	})

	return v.applyMutations(mut)
}

func (v *Vault) applyMutations(mutations []RuleMutation) error {
	oldRoot := v.root.Load()
	newRoot := shallowCopy(oldRoot)
	for _, m := range mutations {
		switch m.Rule {
		case nil:
			if err := v.delete(newRoot, m); err != nil {
				return fmt.Errorf("deleting rule %s: %w", m.ID, err)
			}
		default:
			if err := v.upsert(newRoot, m); err != nil {
				return fmt.Errorf("upserting rule %s: %w", m.ID, err)
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
func (v *Vault) delete(r *Rule, m RuleMutation) error {
	parent := v.findParent(r, m)
	if parent == nil {
		return fmt.Errorf("parent not found (%s, %s)", m.Parent, m.ID)
	}
	delete(parent.Rules, m.ID)
	return nil
}

// findParent uses the m.ID and m.Parent to find the parent of the
// rule with m.ID
func (v *Vault) findParent(r *Rule, m RuleMutation) (parent *Rule) {
	if m.Parent != "" {
		parent = FindRule(r, m.Parent)
	}
	if parent == nil {
		parent = findParent(r, nil, m.ID)
	}
	return parent
}

// upsert either updates (replaces) or adds the rule in m.Rule.
// When adding a new rule, m.Parent is required.
func (v *Vault) upsert(r *Rule, m RuleMutation) error {
	parent := v.findParent(r, m)
	if parent == nil {
		return fmt.Errorf("parent not found (%s)", m.Parent)
	}
	err := v.engine.Compile(m.Rule, v.compileOptions...)
	if err != nil {
		return fmt.Errorf("compiling upsert rule %s: %w", m.Rule.ID, err)
	}

	if parent.Rules == nil {
		parent.Rules = map[string]*Rule{}
	}
	parent.Rules[m.ID] = m.Rule
	return nil
}

// // All changes become visible instantly to concurrent Eval calls.
//
//	func (v *Vault) ApplyMutations(mutations []RuleMutation) error {
//		oldRoot := v.root.Load()
//
//		desired := make(map[string]*Rule) // ID → final rule (nil = delete)
//		moves := make(map[string]string)  // ID → new parent ID
//
//		for _, m := range mutations {
//			if m.ID == "" {
//				continue
//			}
//			if m.Rule != nil && m.Rule.ID != m.ID {
//				return fmt.Errorf("invalid rule")
//			}
//
//			desired[m.ID] = m.Rule
//
//			if m.NewParentID != "" {
//				moves[m.ID] = m.NewParentID
//			}
//		}
//
//		newRoot := v.rebuildTree(oldRoot, desired, moves)
//		if newRoot == nil {
//			newRoot = &Rule{Rules: make(map[string]*Rule)}
//		}
//		fmt.Println("after rebuild:", newRoot)
//		err := v.engine.Compile(newRoot, v.compileOptions...)
//		if err != nil {
//			return err
//		}
//		v.root.Store(newRoot)
//		return nil
//	}
//

// -------------------------------------------------------------------
// Internal rebuild logic (private to Vault)
// -------------------------------------------------------------------
///
// func (v *Vault) rebuildTree(orig *Rule, desired map[string]*Rule, moves map[string]string) *Rule {
// 	if orig == nil {
// 		orig = &Rule{Rules: make(map[string]*Rule)}
// 	}
//
// 	tree := v.applyDesired(orig, desired, moves)
// 	return v.applyMoves(tree, moves)
// }
//
// func (v *Vault) applyDesired(orig *Rule, desired map[string]*Rule, moves map[string]string) *Rule {
// 	if orig == nil {
// 		return nil
// 	}
//
// 	if want, ok := desired[orig.ID]; ok {
// 		if want == nil {
// 			return nil // deleted
// 		}
// 		orig = want
// 	}
//
// 	clone := *orig
// 	clone.Rules = make(map[string]*Rule, len(orig.Rules))
//
// 	fmt.Println("before children: ", &clone)
// 	for key, child := range orig.Rules {
// 		if newChild := v.applyDesired(child, desired, moves); newChild != nil {
// 			clone.Rules[key] = newChild
// 		}
// 	}
// 	fmt.Println("After children: ", &clone)
// 	for id, rule := range desired {
// 		if rule != nil && rule.ID == id {
// 			if _, moving := moves[id]; !moving {
// 				if v.findRuleByID(orig, id) == nil {
// 					clone.Rules[id] = rule
// 				}
// 			}
// 		}
// 	}
//
// 	return &clone
// }
//
// func (v *Vault) applyMoves(root *Rule, moves map[string]string) *Rule {
// 	if len(moves) == 0 || root == nil {
// 		return root
// 	}
//
// 	toMove := make(map[string]*Rule)
// 	clean := v.detachMoved(root, moves, toMove)
//
// 	for id, newParentID := range moves {
// 		rule := toMove[id]
// 		if rule == nil {
// 			continue
// 		}
//
// 		if newParentID == "" {
// 			if clean.Rules == nil {
// 				clean.Rules = make(map[string]*Rule)
// 			}
// 			clean.Rules[id] = rule
// 			continue
// 		}
//
// 		parent := v.findRuleByID(clean, newParentID)
// 		if parent != nil {
// 			if parent.Rules == nil {
// 				parent.Rules = make(map[string]*Rule)
// 			}
// 			parent.Rules[id] = rule
// 		}
// 	}
//
// 	return clean
// }
//
// func (v *Vault) detachMoved(node *Rule, moves map[string]string, collector map[string]*Rule) *Rule {
// 	if node == nil {
// 		return nil
// 	}
//
// 	clone := *node
// 	if len(node.Rules) > 0 {
// 		clone.Rules = make(map[string]*Rule)
// 		for k, child := range node.Rules {
// 			if _, shouldMove := moves[child.ID]; shouldMove {
// 				collector[child.ID] = child
// 				continue
// 			}
// 			if newChild := v.detachMoved(child, moves, collector); newChild != nil {
// 				clone.Rules[k] = newChild
// 			}
// 		}
// 	} else {
// 		clone.Rules = make(map[string]*Rule)
// 	}
// 	return &clone
// }
