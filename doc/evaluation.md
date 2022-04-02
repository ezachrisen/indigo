# Expression Evaluation

We start by looking at how to evaluate individual expressions. In this chapter we will use the terms "expression" and "rule" interchangeably. Later in this guide we will consider ways to arrange rules in hierarchies for different purposes. 

All of examples use [Google's Common Expression Language](https://github.com/google/cel-go), CEL for short. 

{% note %}

**Note:** In this guide we will touch on the basics of the language, for complete coverage you should read the [CEL language spec](https://github.com/google/cel-spec/blob/master/doc/langdef.md).

{% note %}


See the ``indigo/cel/example_test.go`` file for the code for the examples.

In the code samples below we will provide the corresponding example code like this:
```
    // indigo/cel/example_test.go:Example_basic()
    ... 
    sample code
	...
```

This means that the sample code is taken from the function `Example_basic()` in the `indigo/cel/example_test.go` file. 


## Compilation and Evaluation
Indigo supports two-step processing of rules. The first step, compilation, is performed only once, when the rule is created or changed. The second step, evaluation, is performed whenever the software has new data to check against the rule. 

During compilation the rule syntax, functions and data types are checked to ensure the expression is correct. The string is then transformed into an abstract syntax tree that the evaluator will later use to perform the evaluation. The engine only performs the compilation once, but can then evaluate the rules millions of times after. 

## Input Data
In our simple rule example from the introduction, the 'x' was the input data:

```proto
x > 10
```

In order to evaluate the rule, we had to provide the value of x in our "real" world. 

In reality, rules may have many pieces of data, such as in this expression:

```proto
x > 10 && y != "blue" 
```
Indigo allows you to pass a list of data elements to evaluation. 


## Schemas
One of the strengths of CEL is that it provides type safe rule expressions, and violations of type safety is detected at compile time. This is very useful, since rule-writers get immediate feedback on the validity of their rules. 

Building on our x and y example above , we have to define that in the *schema* for rules that use ``x`` and ``y``, ``x`` is an integer and ``y`` is a string. 



```go
    // indigo/cel/example_test.go:Example_basic()
	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "x", Type: indigo.Int{}},
			{Name: "y", Type: indigo.String{}},
		},
	}
```

Every rule needs to have a schema, **but a schema can be used by many rules**. As we will see later when evaluating multiple rules, each rule must use the same or compatible schemas. 


### Data types
The data types supported by Indigo are in the `schema.go` file:

- String
- Integer
- Float 
- Bool
- Duration
- Timestamp
- Lists 
- Maps 
- Proto 

Proto is the only type that can have a "nested" structure, similar to a Go struct. Go structs are not supported. 


#### Exercises

1. In the schema in `Example_basic()`, change the declared data type of `x` to a boolean


## Boolean Scalar Expressions 
The simplest type of rule compares scalar values to each other to produce a true/false outcome. 
Our simple x and y rule is such an expression: 

```proto
x > 10 && y != "blue" 
```

Expressions support && (and) and || (or), and parentheses to group conditions. 




