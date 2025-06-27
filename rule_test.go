package indigo_test

import (
	"strings"
	"testing"

	"github.com/ezachrisen/indigo"
)

func TestNew(t *testing.T) {
	r := indigo.NewRule("blah", "")

	if r.Rules == nil {
		t.Error("expected Rules to be initialized")
	}
	if r.ID != "blah" {
		t.Errorf("expected ID to be 'blah', got %q", r.ID)
	}
	if len(r.Schema.Elements) != 0 {
		t.Errorf("expected Schema.Elements length to be 0, got %d", len(r.Schema.Elements))
	}
}

func TestApplyToRuleDelete(t *testing.T) {
	// Create multi-level rule hierarchy
	parent := indigo.NewRule("parent", "true")

	// Level 1 children
	child1 := indigo.NewRule("delete_child1", "true")
	child2 := indigo.NewRule("keep_child2", "true")
	parent.Rules["delete_child1"] = child1
	parent.Rules["keep_child2"] = child2

	// Level 2 grandchildren under child1
	grandchild1 := indigo.NewRule("keep_grandchild1", "true")
	grandchild2 := indigo.NewRule("delete_grandchild2", "true")
	child1.Rules["keep_grandchild1"] = grandchild1
	child1.Rules["delete_grandchild2"] = grandchild2

	// Level 2 grandchildren under child2
	grandchild3 := indigo.NewRule("delete_grandchild3", "true")
	grandchild4 := indigo.NewRule("keep_grandchild4", "true")
	child2.Rules["delete_grandchild3"] = grandchild3
	child2.Rules["keep_grandchild4"] = grandchild4

	// Count initial rules at each level
	initialParentRules := len(parent.Rules)
	initialChild1Rules := len(child1.Rules)
	initialChild2Rules := len(child2.Rules)

	// Apply deletion function that removes rules with IDs starting with "delete_"
	err := indigo.ApplyToRule(parent, func(r *indigo.Rule) error {
		if r == parent {
			return nil // Don't delete the root
		}

		// Find parent of this rule and delete if ID starts with "delete_"
		for parentID, parentRule := range map[string]*indigo.Rule{
			"parent":        parent,
			"delete_child1": child1,
			"keep_child2":   child2,
		} {
			if parentRule.Rules != nil {
				for childID, childRule := range parentRule.Rules {
					if childRule == r && strings.HasPrefix(r.ID, "delete_") {
						delete(parentRule.Rules, childID)
						_ = parentID
						//t.Logf("Deleted rule %s from parent %s", r.ID, parentID)
						break
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		t.Fatalf("ApplyToRule failed: %v", err)
	}

	// Verify deletions occurred at each level
	if len(parent.Rules) >= initialParentRules {
		t.Errorf("Expected parent rules to be reduced, was %d, now %d", initialParentRules, len(parent.Rules))
	}

	if len(child1.Rules) >= initialChild1Rules {
		t.Errorf("Expected child1 rules to be reduced, was %d, now %d", initialChild1Rules, len(child1.Rules))
	}

	if len(child2.Rules) >= initialChild2Rules {
		t.Errorf("Expected child2 rules to be reduced, was %d, now %d", initialChild2Rules, len(child2.Rules))
	}

	// Verify specific rules were deleted/kept
	if _, exists := parent.Rules["delete_child1"]; exists {
		t.Error("Expected delete_child1 to be deleted from parent")
	}

	if _, exists := parent.Rules["keep_child2"]; !exists {
		t.Error("Expected keep_child2 to remain in parent")
	}

	if _, exists := child1.Rules["delete_grandchild2"]; exists {
		t.Error("Expected delete_grandchild2 to be deleted from child1")
	}

	if _, exists := child1.Rules["keep_grandchild1"]; !exists {
		t.Error("Expected keep_grandchild1 to remain in child1")
	}

	if _, exists := child2.Rules["delete_grandchild3"]; exists {
		t.Error("Expected delete_grandchild3 to be deleted from child2")
	}

	if _, exists := child2.Rules["keep_grandchild4"]; !exists {
		t.Error("Expected keep_grandchild4 to remain in child2")
	}
}
