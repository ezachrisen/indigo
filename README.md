
# Indigo 
Indigo is a rules engine created to enable application developers to build systems whose logic can be controlled by end-users via rules. Rules are expressions (such as "a > b") that are evaluated, and the outcomes used to direct appliation logic. Indigo does not itself provide a language for expressions, relying instead on a backend evaluator (```interface Evaluator```) to provide that. You can create your own backend evaluator, or use the default one, Google's Common Expression Language, CEL. 


### Google's Common Expression Language
Indigo supports the CEL language as an evaluation back-end. See https://github.com/google/cel-go and https://opensource.google/projects/cel for more information about CEL and the Go implementation of it. CEL is a rich expression language, providing not only boolean expression output, but also calculations and object construction. Particularly powerful is the "native" use of protocol buffers in CEL's schema, rule language and return types. 


### Usage

The example ExampleHelloWorld shows basic usage of the engine. 

``` go

func ExampleHelloWorld() {

	// Step 1: Create a schema
	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "message", Type: indigo.String{}},
		},
	}

	// Step 2: Create rules
	rule := indigo.Rule{
		ID:     "hello_check",
		Schema: schema,
		Expr:   `message == "hello world"`,
	}

	// Step 3: Create Indigo and give it an evaluator
	// In this case, CEL
	evaluator := cel.NewEvaluator()
	engine := indigo.NewEngine(evaluator)

	// Step 4: Compile the rule
	err := engine.Compile(&rule)
	if err != nil {
		fmt.Println(err)
		return
	}

	data := map[string]interface{}{
		"message": "hello world",
	}

	// Step 5: Evaluate and check the results
	results, err := engine.Evaluate(data, &rule)
	fmt.Println(results.Pass)
	// Output: true
}

```

### Sorted Evaluations
By default, rule evalution order is not specified. If you want to control the sort order, set the SortFunc evaluation option, either on an individual rule, or for a particular invocation of Evaluate.

Be aware that sorting has a cost. Below is a banchmark comparison of old (no sorting) and new (with sorting). 

```
name                         old time/op  new time/op  delta
SimpleRule-2                  877ns ± 1%  1130ns ± 4%  +28.88%  (p=0.008 n=5+5)
SimpleRuleWithDiagnostics-2  20.3µs ± 3%  20.2µs ± 1%     ~     (p=0.841 n=5+5)
RuleWithArray-2               881ns ± 1%  1122ns ± 0%  +27.32%  (p=0.008 n=5+5)
ProtoWithSelfX-2             1.35µs ± 1%  1.60µs ± 1%  +18.31%  (p=0.008 n=5+5)
ProtoWithoutSelf-2           1.20µs ± 1%  1.44µs ± 1%  +20.00%  (p=0.008 n=5+5)
ProtoCreation-2              1.96µs ± 1%  1.97µs ± 0%     ~     (p=0.222 n=5+5)
Eval2000Rules-2              5.39ms ± 9%  6.47ms ±10%  +20.10%  (p=0.016 n=5+5)
AddRule-2                    11.5ms ± 2%  11.4ms ± 1%     ~     (p=0.690 n=5+5)

```

### For More Information
While there is a lot of power in expression evaluation, Indigo organizes rules in a tree-based hierarchy, allowing precise control over what rules are evaluated and how. 

Check out the [use cases](UseCases.md) for examples of ways you can struture rules in Indigo.

See the package documentation:

[![Go Reference](https://pkg.go.dev/badge/github.com/ezachrisen/indigo.svg)](https://pkg.go.dev/github.com/ezachrisen/indigo)


