# indigo

Package indigo provides a rules engine.

Indigo is a rules engine created to enable application developers to build systems whose logic
can be controlled by end-users via rules.
Rules are expressions (such as "a > b") that are evaluated, and the outcomes used to direct appliation logic.

Indigo does not specify a language for rules, relying instead on a rule evaluator to perform the work.
The default rule evaluator (in the cel package) is the Common Expression Language from Google
([https://github.com/google/cel-go](https://github.com/google/cel-go)).

## Compilation and Evaluation

Indigo provides methods to compile and evaluate rules. The compilation step gives
the evaluator a chance to pre-process the rule, provide feedback on rule correctness, and store an intermediate form
of the rule for evaluation efficiency. The evaluation evaluates the rule against
input data and provides the output.

## Basic Structure

Indigo organizes rules in hierarchies. A parent rule can have 0 or many child
rules. You do not have to organize rules in a complex tree; a single parent with 1,000s of child rules is OK.
There are 3 main reasons for using a tree to organize rules:

```go
1. Allow atomic rule updates (see separate section)
2. Use options on the parent rule to control if child rules are evaluated
   (in effect, child rules "inherit" the parent rule's condition)
3. Use options on the parent rule to control which child rules are returned as
   results (such as returning true or false results, or both)
4. Logically separate disparate groups of rules
```

## Rule Ownership

The calling application is responsible for managing the lifecycle of rules, including ensuring
concurrency safety. Some things to keep in mind:

```go
1. You must not allow changes to a rule during compilation.
2. You may not modify the rule after compilation and before evaluation.
3. You must not allow changes to a rule during evaluation.
4. You should not modify a rule after it's been evaluated and before the results have been consumed.
5. A rule must not be a child rule of more than one parent.
```

## Updating Rules

To add or remove rules, you do so by modifying the parent rule's map of Rules

```go
delete(parent.Rules, "child-id-to-delete")
```

and

```go
myNewRule.Compile(myCompiler)
parent.Rules["my-new-rule"] = myNewRule
```

It is not recommended to update a rule IN PLACE, unless you
manage the rule lifecycle beyond evaluation and use of the rule in interpreting
the results. Users of your result should expect that the definition of the rule stays constant.
Instead, we recommend creating a new rule with a new version number in the ID to separate updates.

## Structuring Rule Hierarchies for Updates

The ability to organize rules in a hierarchy is useful to ensure that rule updates are atomic and consistent.

You should structure the hierarchy so that a rule and its children can be seen as a
"transaction" as far as updates are concerned.

In this example, where Indigo is being used to enforce firewall rules, being able
to update ALL firewall rules as a group, rather than one by one (where one update may fail)
is important.

```go
Firewall Rules (parent)
  "Deny all traffic" (child 1)
  "Allow traffic from known_IPs" (child 2)
```

If the user changes child 1 to be "Allow all traffic" and changes child 2 to "Deny all traffic,
except for known_IPs",
there's a risk that child 1 is changed first, without the child 2 change happening. This would leave us with this:

```go
Firewall Rules (parent)
  "Allow all traffic" (child 1)
  "Allow traffic from known_IPs" (child 2)
```

This is clearly bad!

Instead of accepting a change to child 1 and child 2 separately, ONLY accept a change to your rule hierarchy for the
Firewall Rules parent. That way the update succeeds or fails as a "transaction".

If Firewall Rules is itself a child of a larger set of parent rules, it's recommended to compile the Firewall Rules
parent and
children BEFORE adding it to its eventual parent. That way you ensure that if compilation of Firewall Rules fails,
the "production" firewall rules are still intact.

## Sub Packages

* [cel](./cel): Package cel provides an implementation of the Indigo evaluator and compiler interfaces backed by Google's cel-go rules engine.

## Examples

Example showing basic use of the Indigo rules engine
with the CEL evaluator

```golang
package main

import (
	"context"
	"fmt"
	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
)

func main() {

	// Step 1: Create a schema
	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "message", Type: indigo.String{}},
		},
	}

	// Step 2: Create rules
	rule := indigo.Rule{
		ID:         "hello_check",
		Schema:     schema,
		Expr:       `message == "hello world"`,
		ResultType: indigo.Bool{},
	}

	// Step 3: Create an Engine with a CEL evaluator
	engine := indigo.NewEngine(cel.NewEvaluator())

	// Step 4: Compile the rule
	err := engine.Compile(&rule)
	if err != nil {
		fmt.Println(err)
		return
	}

	// The data we wish to evaluate the rule on
	data := map[string]interface{}{
		"message": "hello world",
	}

	// Step 5: Evaluate and check the results
	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(results.Pass)
	}
}

```

 Output:

```
true
```

### ApplyToRule

Demonstrate applying a function to a rule

```golang

r := makeRule()

err := indigo.ApplyToRule(r, func(r *indigo.Rule) error {
    fmt.Printf("%s ", r.ID)
    return nil
})

if err != nil {
    fmt.Println("Failure!")
}

fmt.Printf("\n")
// Output unordered: rule1 B b1 b2 b3 b4 b4-1 b4-2 E e1 e2 e3 D d1 d2 d3

```

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
