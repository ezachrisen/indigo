
# Rules
Rules provides a generic interface to a rules engine for the purpose of evaluating rules against data provided at runtime. 

### CEL
The interaface is implemented for cel-go, Google's Go implementation of the Common Expression Language. 
See https://github.com/google/cel-go and https://opensource.google/projects/cel.


### Usage

There are 3 steps to using the rules engine.

1. Define the schema
A schema describes types of data that the engine is expecting. See CEL's documentation on parsing and checking. 

``` go

func main() {

	engine := cel.NewEngine()

	schema := rules.Schema{
		ID: "my schema",
		Elements: []rules.DataElement{
			{Name: "objectType", Type: rules.String{}},
			{Name: "state", Type: rules.String{}},
			{Name: "grades", Type: rules.List{ValueType: rules.Any{}}},
			{Name: "claims", Type: rules.Map{KeyType: rules.String{}, ValueType: rules.Any{}}},
		},
	}

```

2. Define the rules
Rules are expressions that evaluate to true or false against a given input data set. Rules are grouped into sets that are evaluated together. A set should represent a single topic, such as "Access rules" or "Email notification rules". 

The rules are added to the engine. In the case of the CEL engine, it parses and checks the rule set against the schema.

A rules engine can contain multiple sets of rules. 

``` go
	ruleSet := rules.RuleSet{
		ID:     "testset",
		Schema: schema,
		Rules: []rules.Rule{
			{ID: "1", Expression: `objectType == "car" && "admin" in claims.roles && "A" in grades`},
			{ID: "2", Expression: `objectType == "xya" || ( "admin" in claims.roles && "A" in grades)`},
		},
	}

	err := engine.AddRuleSet(&ruleSet)

```


3. Evaluate input against the rules and take action based on the results. 
After the schema and the rules have been set up, the engine can process a wide range of inputs against the rules. 

``` go

	results, err := engine.EvaluateAll(data, "testset")

```

The results contains a list of the rules evaluated and a true/false condition.


### Benchmarks

Simple: evaluating scalar comparisons with and/or
With lists: evaluate data lookups in maps and lists

```
BenchmarkCELSimple-8      	 4129249	       288 ns/op
BenchmarkCELWithLists-8   	  918675	      1320 ns/op
```


### Todo
Implement diagnostic evaluation to aid in troubleshooting.




