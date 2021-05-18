[![Go Reference](https://pkg.go.dev/badge/github.com/ezachrisen/indigo.svg)](https://pkg.go.dev/github.com/ezachrisen/indigo)

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


### For More Information
Indigo organizes rules in a tree-based hierarchy, allowing precise control over what rules are evaluated and how. 

Check out the [use cases](UseCases.md) for examples of ways you can struture rules in Indigo.

