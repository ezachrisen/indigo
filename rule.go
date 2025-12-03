package indigo

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// A Rule defines logic that can be evaluated by an Evaluator.
// The logic for evaluation is specified by an expression.
// A rule can have child rules. Rule options specify to the Evaluator
// how child rules should be handled. Child rules can in turn have children,
// enabling you to create a hierarchy of rules.
//
// # Example Rule Structures
//
// A hierchy of parent/child rules, combined with evaluation options
// give many different ways of using the rules engine.
//
//	Rule with expression, no child rules:
//	 Parent rule expression is evaluated and result returned.
//
//	Rule with expression and child rules:
//	 No options specified
//	 - Parent rule xpression is evaluated, and so are all the child rules.
//	 - All children and their evaluation results are returned
//
//	Rule with expression and child rules
//	Option set: StopIfParentNegative
//	- Parent rule expression is evaluated
//	- If the parent rule is a boolean, and it returns FALSE,
//	  the children are NOT evaluated
//	- If the parent rule returns TRUE, or if it's not a
//	  boolean, all the children and their resulsts are returned
type Rule struct {
	// A rule identifier. (required)
	ID string `json:"id"`

	// The expression to evaluate (optional)
	// The expression can return a boolean (true or false), or any
	// other value the underlying expression engine can produce.
	// All values are returned in the Results.Value field.
	// Boolean values are also returned in the results as Pass = true  / false
	// If the expression is blank, the result will be true.
	Expr string `json:"expr"`

	// The output type of the expression. Evaluators with the ability to check
	// whether an expression produces the desired output should return an error
	// if the expression does not.
	// If no type is provided, evaluation and compilation will default to Bool
	ResultType Type `json:"result_type,omitempty"`

	// The schema describing the data provided in the Evaluate input. (optional)
	// Some implementations of Evaluator require a schema.
	Schema Schema `json:"schema,omitempty"`

	// A set of child rules.
	Rules map[string]*Rule `json:"rules,omitempty"`

	// Reference to intermediate compilation / evaluation data.
	Program any `json:"-"`

	// A reference to any object.
	// Not used by the rules engine.
	Meta any `json:"-"`

	// A reference to an object whose values can be used in the rule expression.
	// Add the corresponding object in the data with the reserved key name selfKey
	// (see constants).
	// Child rules do not inherit the self value.
	Self any `json:"-"`

	// Options determining how the child rules should be handled.
	EvalOptions EvalOptions `json:"eval_options"`

	// Shards is a list of rules that define a sharding strategy for the rule's children.
	// If a rule has many rules, and some of them share a common characteristic, such as applying to
	// cars registered in specific states, or to students in a particular status, the child rules can be
	// grouped into a set of "shards", where all the children share the grouping criteria. For example,
	// if you have 1000 rules that apply only to students in the "Admitted"  status and 600 rules that apply only to
	// students in the "Enrolled" status, you can create two shards, one for each of the statuses. By calling
	// BuildShards on the rule, the child rules will be re-arranged into shards using the rules you provided.
	// See the shard tests and the [BuildShards] function for more information.
	Shards []*Rule

	// sortedRules contains a list of child rules, sorted by the
	// EvalOptions.SortFunc. During rule evaluation, the rules are evaluated in
	// the order they appear in this list. The sorted list is calculated at
	// compile time. If SortFunc is not specified, the evaluation order is
	// unspecified.
	sortedRules []*Rule

	shard bool
}

const (
	// If the rule includes a Self object, it will be made available in the input
	// data with this key name.
	selfKey = "self"
)

// NewRule initializes a rule with the ID and rule expression.
// The ID and expression can be empty.
func NewRule(id string, expr string) *Rule {
	return &Rule{
		ID:    id,
		Rules: map[string]*Rule{},
		Expr:  expr,
	}
}

// Add is a convenience function to add rr to r's Rules list
func (r *Rule) Add(rr *Rule) error {
	if rr == nil {
		return fmt.Errorf("attempt to add nil rule")
	}
	if r.Rules == nil {
		r.Rules = map[string]*Rule{}
	}
	r.Rules[rr.ID] = rr
	return nil
}

// defaultRuleID is the ID for the default shard rule. When define a set of shards
// we need to account for any rules that do not meet the any shard definition. Such rules are
// collected in the "default" shard rule.
const defaultRuleID = "default"

// BuildShards uses r's shard rules (if any) to sort child rules into their respective shards.
//
// Shard definitions can be recursive, i.e., rule A can have shards X and Y, and X can be further
// subdivided into X1 and X2.
//
// Recursively applies shard rules on r's children.
//
// Since rules stored in a Vault should not be modified outside the vault, DO NOT call BuildShards on
// a rule inside the vault. You may of course call BuildShards before adding a rule to the vault, but
// the Vault will automatically call BuildShards on the rule you pass to NewVault.
//
// When you apply mutations to the rule in the vault, the vault will automatically apply sharding
// specifications and place rules in the right shard.
func (r *Rule) BuildShards() error {
	var detached map[string]*Rule
	for _, sh := range r.Shards {
		if sh == nil {
			return fmt.Errorf("nil shard in shard set")
		}
		if sh.ID == defaultRuleID {
			return fmt.Errorf("reserved shard ID %s used (indigo will automatically add a default shard)", defaultRuleID)
		}
		sh.shard = true
		// this will ensure that rules in the shard are only evaluated if the shard rule itself is true
		sh.EvalOptions.StopIfParentNegative = true
		if detached == nil {
			// We only want to do this if there are shards in r.
			detached = maps.Clone(r.Rules)
			r.Rules = map[string]*Rule{}
		}
		r.Rules[sh.ID] = sh
	}
	if len(r.Shards) > 0 {
		def := NewRule(defaultRuleID, "")
		def.shard = true
		r.Rules[def.ID] = def
		r.EvalOptions.SortFunc = func(rules []*Rule, i, j int) bool {
			if rules[i].ID == defaultRuleID {
				return false
			}
			if rules[j].ID == defaultRuleID {
				return true
			}
			return rules[i].ID < rules[j].ID
		}
		r.sortedRules = r.sortChildRules(r.EvalOptions.SortFunc, true)
	}
	for _, child := range detached {
		target, err := r.targetParent(child)
		if err != nil {
			return fmt.Errorf("finding shard for %s: %w", r.ID, err)
		}
		target.Rules[child.ID] = child
	}

	r.sortedRules = r.sortChildRules(r.EvalOptions.SortFunc, true)
	for _, newChild := range r.Rules {
		// //fmt.Println("child ", newChild.ID, r.ID)
		err := newChild.BuildShards()
		if err != nil {
			return err
		}
	}
	return nil
}

// targetParent determines whether rr should be a direct child of r,
// or placed in a shard (if any exist)
func (r *Rule) targetParent(rr *Rule) (*Rule, error) {
	shardCount := 0
	for _, shard := range r.sortedRules {
		if !shard.shard {
			continue
		}
		shardCount++
		switch f := shard.Meta.(type) {
		case func(*Rule) bool:
			if f(rr) {
				return shard, nil
			}
		default:
			if shard.ID != defaultRuleID {
				return nil, fmt.Errorf("unsupported meta type for shard %s: %t", shard.ID, shard.Meta)
			}
			return shard, nil
		}
	}
	// We're in a sharding situation, and no shard matched rr (including the default shard)
	if shardCount > 0 {
		return nil, fmt.Errorf("no shard matched %s, not even default", rr.ID)
	}
	return r, nil
}

func matchMeta(shard, r *Rule) (bool, error) {
	switch f := shard.Meta.(type) {
	case func(*Rule) bool:
		if f(r) {
			return true, nil
		}
	default:
		if shard.ID != defaultRuleID {
			return false, fmt.Errorf("unsupported meta type for shard %s: %t", shard.ID, shard.Meta)
		}
	}
	return false, nil
}

// FindRule returns the rule with the id in the rule or any of its
// children recursively, and a list of the parent rules in order, starting
// with the root of the rule tree and ending with the immediate parent of
// the rule with the id.
func (r *Rule) FindRule(id string) (rule *Rule, ancestors []*Rule) {
	if r == nil {
		return nil, nil
	}
	if r.ID == id {
		return r, nil
	}
	for _, child := range r.Rules {
		if found, p := child.FindRule(id); found != nil {
			// Prepend current root to the parent chain
			ancestors = append([]*Rule{r}, p...)
			return found, ancestors
		}
	}
	return nil, nil
}

// Path returns all of the ancestors of the rule with the ID,
// starting with the root of the rule tree.
func (r *Rule) Path(id string) []*Rule {
	me, ancestors := r.FindRule(id)
	if me == nil {
		return nil
	}
	return append(ancestors, me)
}

// FindParent returns the parent of the rule with the id
func (r *Rule) FindParent(id string) *Rule {
	_, ancestors := r.FindRule(id)

	if len(ancestors) < 1 {
		return nil
	}
	return ancestors[len(ancestors)-1]
}

// ApplyToRule applies the function f to the rule r and its children recursively.
func ApplyToRule(r *Rule, f func(r *Rule) error) error {
	err := f(r)
	if err != nil {
		return err
	}
	for _, c := range r.Rules {
		err := ApplyToRule(c, f)
		if err != nil {
			return err
		}
	}
	return nil
}

// String returns a list of all the rules in hierarchy, with
// child rules sorted in evaluation order.
func (r *Rule) String() string {
	tw := table.NewWriter()
	tw.SetTitle("\nINDIGO RULES\n")
	tw.AppendHeader(table.Row{"\nRule", "\nSchema", "\nExpression", "Result\nType", "\nMeta"})

	maxWidthOfExpressionColumn := 40
	rows, maxExprLength := r.rulesToRows(0)
	for _, r := range rows {
		tw.AppendRow(r)
	}

	tw.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1},
		{Number: 2},
		{Number: 3, WidthMax: maxWidthOfExpressionColumn},
		{Number: 4},
		{Number: 5},
	})

	style := table.StyleLight
	style.Format.Header = text.FormatDefault
	// Only add the row separator if the expression is wide enough to wrap.
	if maxExprLength > maxWidthOfExpressionColumn {
		style.Options.SeparateRows = true
	}
	tw.SetStyle(style)
	return tw.Render()
}

func (r *Rule) rulesToRows(n int) ([]table.Row, int) {
	rows := []table.Row{}
	indent := strings.Repeat("  ", n)

	row := table.Row{
		fmt.Sprintf("%s%s", indent, r.ID),
		r.Schema.ID,
		r.Expr,
		fmt.Sprintf("%v", r.ResultType),
		fmt.Sprintf("%T", r.Meta),
	}
	rows = append(rows, row)
	maxExprLength := len(r.Expr)

	for _, c := range r.Rules {
		cr, maxLen := c.rulesToRows(n + 1)
		if maxLen > maxExprLength {
			maxExprLength = maxLen
		}
		rows = append(rows, cr...)
	}
	return rows, maxExprLength
}

// sortChildRules returns a list of rules, ordered by the function.
// With a nil function, returns either a cached list of rules (whose sort order may
// have been set by a previous sort operation), or a list of rules whose order
// is not defined.
func (r *Rule) sortChildRules(fn func(rules []*Rule, i, j int) bool, force bool) []*Rule {
	if fn == nil && len(r.sortedRules) == len(r.Rules) && !force {
		return r.sortedRules
	}

	if !force && len(r.sortedRules) == len(r.Rules) {
		return r.sortedRules
	}

	keys := make([]*Rule, len(r.Rules))
	var i int
	for k := range r.Rules {
		keys[i] = r.Rules[k]
		i++
	}

	if fn != nil && len(keys) > 0 && force {
		sort.Slice(keys, func(i, j int) bool {
			return fn(keys, i, j)
		})
	}
	return keys
}

// SortRulesAlpha will sort rules alphabetically by their rule ID
func SortRulesAlpha(rules []*Rule, i, j int) bool {
	return rules[i].ID < rules[j].ID
}

// SortRulesAlphaDesc will sort rules alphabetically (descending) by their rule ID
func SortRulesAlphaDesc(rules []*Rule, i, j int) bool {
	return rules[i].ID > rules[j].ID
}

// Tree returns a tree representation of the rule hierarchy showing only rule IDs.
// The tree uses box-drawing characters to visualize parent-child relationships.
// Recursion is limited to a maximum depth of 20 levels.
//
// Example output:
//
//	root
//	├── child_1
//	│   ├── grandchild_1
//	│   └── grandchild_2
//	└── child_2
//	    └── grandchild_3
func (r *Rule) Tree() string {
	if r == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(r.ID)
	if r.shard {
		sb.WriteString(" (*)")
	}
	sb.WriteString("\n")
	r.buildTree(&sb, "", 0)
	return sb.String()
}

// buildTree recursively builds the tree representation with proper indentation
// and tree characters (├──, └──, │).
// depth limits recursion to a maximum of 20 levels.
func (r *Rule) buildTree(sb *strings.Builder, prefix string, depth int) {
	// Stop if we've reached the maximum depth
	if depth >= 20 {
		return
	}
	i := 0
	sorted := r.sortChildRules(SortRulesAlpha, true)
	for _, child := range sorted {
		isLast := i == len(sorted)-1
		// Determine the tree characters to use
		var connector, childPrefix string
		if isLast {
			connector = "└── "
			childPrefix = "    "
		} else {
			connector = "├── "
			childPrefix = "│   "
		}

		// Write the current child
		sb.WriteString(prefix)
		sb.WriteString(connector)
		sb.WriteString(child.ID)
		if child.shard {
			sb.WriteString(" (*)")
		}
		sb.WriteString("\n")
		// Recursively process this child's children
		child.buildTree(sb, prefix+childPrefix, depth+1)
		i++
	}
}
