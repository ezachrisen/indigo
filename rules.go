package rules

// --------------------------------------------------
// Rules Engine

// The Engine interface represents a rules engine capable of evaluating rules.
// against a specific rule set.
//

type Engine interface {

	// Add the rule to the engine.
	// The rule must have an ID that is unique among the root rules.
	// An existing rule with the same ID will be replaced.
	AddRule(Rule) error

	// Return the rule with the ID
	Rule(id string) (Rule, bool)

	// Evaluate a rule agains the the data.
	// Will recursively evaluate child rules up to maxLevels deep.
	// No error will be returned if MaxLevels is reached.
	// To control the depth, call EvaluateN.
	Evaluate(data map[string]interface{}, id string) (*Result, error)

	// Evaluate a rule against the data.
	// Recursively evaluate child rules, up to n levels deep.
	// Use n=1 to only evaluate the root rule, and no child rules.
	// EvaluateN is not affected by MaxLevels.
	EvaluateN(data map[string]interface{}, id string, n int) (*Result, error)

	// Calculate the expression and return the result.
	Calculate(data map[string]interface{}, expr string, schema Schema) (float64, error)
}

const (
	// MaxLevels is the maximum depth that Evaluate will recursively visit the child
	// rules when Evaluate is called.
	// Does not impact EvaluateN
	MaxLevels = 100
	SelfKey   = "self"
)

// --------------------------------------------------
// Rules
type Rule struct {
	// A rule identifer. (required)
	// No two root rules can have the same identifier.
	ID string

	// The expression to evaluate (required)
	// The expression can return a boolean (true or false), or any
	// other value the underlying expression engine can produce.
	// All values are returned in the Results.Value field.
	// Boolean values are also returned in the results as Pass = true  / false
	Expr string

	// The schema describing the data provided in the Evaluate input. (optional)
	// Some implementations of Rules require a schema.
	Schema Schema

	// A reference to the object that "owns" this rule. (optional)
	// This is useful to allow dynamic value substitution in the rule expression.
	// For example, in a rule determining if a thermostat action,
	// set Self = Thermostat, where Thermostat holds the user's temperature preference.
	// When evaluating the rules, the Thermostat object will be included in the data
	// with the key SelfKey. Rules can then reference the user's temperature preference
	// in the rule expression. This allows rule inputs to be changed without recompiling
	// the rule.
	Self interface{}

	// A set of child rules. See how to call Evaluate(N) to control which child rules
	// are evaluated.
	Rules map[string]Rule

	// A reference to a custom object. The same reference will be returned in the results,
	// so the engine user can use this reference to get back a relevant object.
	Meta interface{}
}

// Result of evaluating a rule.
type Result struct {
	// The ID of the rule that was evaluated
	RuleID string

	// Whether the rule yielded a TRUE logical value.
	// The default is FALSE
	// This is the result of evaluating THIS rule only.
	// The result will not be affected by the results of the child rules.
	// If no rule expression is supplied for a rule, the result will be TRUE.
	Pass bool

	// User-supplied reference from the rule. See the Rule struct for information.
	Meta interface{}

	// The result of the evaluation. Boolean for logical expressions.
	// Calculations or string manipulations will return the appropriate type.
	Value interface{}

	// Results of evaluating the child rules.
	Results map[string]Result
}

// --------------------------------------------------
// Schema

// Schema defines the keys (variable names) and their data types used in a
// rule expression. The same keys and types must be supplied in the data map
// when rules are evaluated.
type Schema struct {
	ID       string
	Elements []DataElement
}

// DataElement defines a named variable in a schema
type DataElement struct {
	// Short, user-friendly name of the variable. This is the name
	// that will be used in rules to refer to data passed in.
	Name string

	// One of the Type interface defined.
	Type Type

	// Optional description of the type.
	Description string
}

// --------------------------------------------------
// Data Types in a Schema

// Type of data element represented in a schema
type Type interface {
	TypeName()
}

type String struct{}
type Int struct{}
type Float struct{}
type Any struct{}
type Bool struct{}
type Duration struct{}
type Timestamp struct{}
type Proto struct {
	Protoname string
	Message   interface{}
}

// List is an array of items
type List struct {
	ValueType Type // The type of value stored in the list
}

// Map is a map of items. Maps can be nested.
type Map struct {
	KeyType   Type
	ValueType Type
}

func (t Int) TypeName()       {}
func (t Bool) TypeName()      {}
func (t String) TypeName()    {}
func (t List) TypeName()      {}
func (t Map) TypeName()       {}
func (t Any) TypeName()       {}
func (t Duration) TypeName()  {}
func (t Timestamp) TypeName() {}
func (t Float) TypeName()     {}
func (t Proto) TypeName()     {}
