# Indigo Handbook

The purpose of this document is to describe how Indigo's rules and the evaluation engine works. We encourage you to read the Indigo source code and examples as primary material, and consider this document as a companion to guide you through the concepts. 



Useful links:
[Indigo examples](../example_test.go)
[CEL examples](../cel/example_test.go)

---


[Chapter 1: Introduction](#chapter-1-introduction)

   1. [What is a rule?](#what-is-a-rule)
   1. [Why use rules?](#why-use-rules)
   1. [Expressions and rules in Indigo](#expressions-and-rules–in-indigo)

[Chapter 2: Expression Evaluation](#chapter-2-expression-evaluation)
   1. Compilation and Evaluation
   1. Schemas
   1. Data types 
   1. Boolean Scalar Expressions
   1. Operators
   1. Creating a rule
   1. Compilation
   1. Input Data
   1. Evaluation 
   1. Evaluation results 
   1. Short circuiting 

Chapter 3: Indigo Rules Engine Types
   1. The Engine type 
   1. The Rule type 



1. Lists and Map
   1. Lists
   1. Maps
   1. Functions 
   1. Macros
1. Using Protobufs in Rules
1. Non-boolean Output
1. Rule Hierarchies
1. Diagnostics
   1. Runtime diagnostics 
   1. Compilation output


---
# Chapter 1 <br/>Introduction

## What is a rule?
A rule is an expression that can be evaluated to produce an outcome. The outcome may be true or false, or it may be a number, or a string or any other value. The same can be said of any computer language code, but what makes rules different is that their expression language is "configurable" by end users of the software, allowing them to modify how the software works without re-compiling or re-deploying the software. 

Rules are data fed into the software, and the software processes the rules. 

Consider these two equivalent rules, one written in Go, the other in a simplified "rule expression language":


##### Rule in Go
```
  func qualify(x int) bool {
    if x > 10 {
      return true
    } else {
      return false
	}
  }
```

If the business logic changes, and instead of x > 10 we want x > 15, the Go program must be changed and recompiled. Of course, we could make the value a parameter, but what if instead of simply comparing x to a value, we also want to compare y to another value? We will have to continue changing the qualify function. 

##### Rule in Expression Language
```
x > 10
```

In the rule language example, we have an expression entered by the user at run-time as a string, and evaluated by the software. We are assuming that the software knows how to interpret the expression and produce a true/false result. If the logic changes, the user can change the rule at run-time without changing the software. 

## Why use rules?
Rules are not appropriate for all situations. For one thing, rules are much slower to evaluate than compiled code. Rules are also more difficult to debug, and because users are able to enter them into the software, they are less predictable than thoroughly tested, quality software. 

So why use them? The primary reason to use rules is that they allow business logic to be configured outside of the software itself. It removes business logic from the software, and makes the software into a data processing *platform*, where the rules are the data. 

A side benefit of using rules in software is that it changes the way engineers think about writing the application. Instead of focusing on the specific business logic that must be implemented, engineers instead think about how to enable *configurable*, *dynamic* business logic that users control. 


## Expressions and rules in Indigo
The Indigo rules engine goes beyond evaluating individual expressions (such as the "rule expression language" example above) and provides a structure and mechanisms to evaluate *groups* of rules arranged in specific ways for specific purposes. Indigo "outsources" the actual expression evaluation to Google's Common Expression Language, though other languages can be plugged in as well. 

---
# Chapter 2 <br/>Expression Evaluation

We start by looking at how to evaluate individual expressions. In this chapter we will use the terms "expression" and "rule" interchangeably. Later in this guide we will consider ways to arrange rules in hierarchies for different purposes. 

All of examples use [Google's Common Expression Language](https://github.com/google/cel-go), CEL for short. 


**Note:** In this guide we will touch on the basics of the language, for complete coverage you should read the [CEL language spec](https://github.com/google/cel-spec/blob/master/doc/langdef.md).




See the ``indigo/cel/example_test.go`` file for the code for the examples.

In the code samples below we will provide the corresponding example code like this:
```go
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

In Indigo, the indigo.Schema type is used to specify the schema:


```go
// indigo/cel/example_test.go:Example_basic()
schema := indigo.Schema{
	Elements: []indigo.DataElement{
		{Name: "x", Type: indigo.Int{}},
		{Name: "y", Type: indigo.String{}},
	},
}
```

The schema is a list of indigo.DataElements. Each data element has a Name, which is the **variable** name that users will use in rule expressions. 

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

### Output Schema 

Just as we specify the data types of the input to the rule, we can also specify the data type we expect as a result of rule evaluation. Unlike input schemas which can have many types, the output schema is a single type, and the default is boolean. 

During compilation CEL will verify that the expression produces the type of output you specified in your output schema. 

You specify the output type by setting the ``ResultType`` field on the rule. 

#### Exercises

Modify the example `indigo/cel/example_test.go:Example_basic()` and run ``go test`` in the ``indigo/cel`` directory. 

1. Change the declared data type of `x` to a boolean
1. Change the ResultType of the rule to a string


## Boolean Scalar Expressions 
The simplest type of rule compares scalar values to each other to produce a true/false outcome. 
Our simple x and y rule is such an expression: 

```proto
x > 10 && y != "blue" 
```

In some rule engine languages you specify two parts: "if" and "then", like this:

```proto
// pseudo-code for a fake rule engine language
if 
   x > 10 && y != "blue" 
then 
   set output = true 
```

CEL derives the value of the output from the value of the expression, in this case a boolean, so the "if/then" construct is not needed. (CEL does support the if/then functionality with the :? operators which we will cover later in the book.)

## Operators

Supported logical operators: ``&& || !``. Parentheses can be used to group expressions. 

Supported comparison operators: `` < <= >= > == != in ``

Supported math operators: `` + - * / % ``


## Creating a rule 

An Indigo rule is a Go struct:


```go
// indigo/cel/example_test.go:Example_basic()
rule := indigo.Rule{
	Schema:     schema,
	ResultType: indigo.Bool{},
	Expr:       `x > 10 && y != "blue"`,
}
```

This initializes the rule, sets its schema, the expected output (boolean) and the expression to evaluate. Indigo defaults the result type to boolean if you don't specify one. 

## Compilation 

Before compiling the rule, we need to create an instance of indigo.DefaultEngine:


```go
// indigo/cel/example_test.go:Example_basic()
engine := indigo.NewEngine(cel.NewEvaluator())
```

This creates a new engine that will use CEL as its expression evaluation language. See the chapter on Indigo rules engine types for language evaluators and how the Indigo interface types work. 

With an engine, we can now compile the rule:

```go
// indigo/cel/example_test.go:Example_basic()
err := engine.Compile(&rule)
if err != nil {
	fmt.Println(err)
	return
}
```

If there are any compilation errors, the returned error message will tell you what the error is, and where in your expression the error occurred. 

## Input Data

Now that we have a compiled rule, we can start to evaluate data against it. Let's assume we are serving an API that receives data, and it's our job to check the data against the rule and report the results. 

We prepare the data for evaluation like this: 

```go
// indigo/cel/example_test.go:Example_basic()
data := map[string]interface{}{
	"x": 11,
	"y": "red",
}
```

The data is placed in a map, where the key is the variable name you specified in the schema. If we look back at the schema definition from earlier in this chapter, you'll see that we defined an "x" and a "y" variable:

```go
// Flashback to the schema definition earlier in this chapter
schema := indigo.Schema{
	Elements: []indigo.DataElement{
		{Name: "x", Type: indigo.Int{}},
		{Name: "y", Type: indigo.String{}},
	},
}
```

A data map does not need to define values for all variables in a schema, but if a variable is used in a rule, it must be in the data map. 

## Evaluation 

We are finally ready to evaluate the input data (x=11, y=red) against the rule to determine if the rule is true or false:

```go
// indigo/cel/example_test.go:Example_basic()
results, err := engine.Eval(context.Background(), &rule, data)
```

The evaluation returns an indigo.Result struct, and an error. It is not an error if the rule returns false; an error is an unexpected issue, such as incorrect data passed. 

Eval accepts a context.Context, and will stop evaluation if the context's deadline expires or is canceled, returning no results and the context's error. 

## Evaluation Results 

The indigo.Result struct contains the output of the expression evaluation, and additional information to help calling code process the results. For now, we'll focus on the boolean Result.ExpressionPass field. This field will tell you if the output of the expression is true or false, which is what we are interested in our example. 

## Short Circuiting 

Many languages, including Go, implement and/or short-circuiting. 

For example, in this statement, the comparison ``b == "blue"`` will never be executed: 

```go
var a *int 
b := "red"

if a!=nil && b == "blue" {
...
}
```

CEL also implements short circuiting, but even allows for a!=nil to be the second clause of an && comparison. See the [table](https://github.com/google/cel-go/blob/master/README.md#partial-state) for examples. 



#### Exercises

Modify the example `indigo/cel/example_test.go:Example_basic()` and run ``go test`` in the ``indigo/cel`` directory. 

1. Comment out the input value for y
   Notice the error message, what does it mean? 

1. Change the input value of x to 7
   Why did this not give an error message? (Although we got false when we wanted true)



# Indigo Rules Engine Types

This chapter is a deeper dive into how Indigo is organized. 

## The Engine Types

Indigo's rules engine is specified by an interface called Engine, which is composed of two interfaces, a Compiler and an Evaluator. The Compiler interface specifies the Compile method, and Evaluator specifies the Eval method. 

The DefaultEngine struct type implements the Engine interface, and most users will use this type. Since it implements Engine, DefaultEngine implements both Compile and Evaluate. Alternate implementations are possible, including rule evaluators that do not support compilation, for example. 

The engine types are concerned with processing groups of rules organized in a hierarchies. The engine types **do not** concern themselves with the exact nature of the expressions being compiled or evaluated. Instead, they rely on the expression types for that. 

## The Expression Types 
There are two expression interfaces, ExpressionEvaluator, which specifies the evaluation of a single expression, and ExpressionCompiler, which specifies the compilation of a single expression. 

These two interfaces are combined in the ExpressionCompilerEvaluator. 

The indigo/cel.Evaluator implements the ExpressionCompilerEvaluator.

Users of Indigo do not need to interact directly with expression compilation or evaluation; the DefaultEngine handles this. 

## Rule ownership 

Rules are owned by the Go code that calls Indigo engine methods. The indigo.DefaultEngine does NOT store or maintain rules. It is the responsibility of the calling code to be goroutine-safe and to persist rule data for as long as it is needed in the life of the application. 

During compilation, an Engine may update a rule by setting the Program field to the compilation output of the rule. The Engine may require that data later during the evaluation phase. It is the responsibility of the calling code to ensure that the Program data is not tampered with. 

## Using a Non-CEL Evaluator

There are several Go implementations of scripting languages, such as Javscript implemented by [Otto](https://github.com/robertkrimen/otto), and Lua. These languages are good choices for rule evaluation as well. 

To illustrate how Indigo's interface types work together, here's how you could implement and use a different evaluator such as Otto:

1. In your own module, create a new struct, ``MyEvaluator``
1. Decide if the language requires compilation; if it does, implement the ExpressionCompiler interface, including converting types from Indigo types to the evaluator's types. 
1. Implement the ExpressionEvaluator interface
1. When instantiating indigo.DefaultEngine, pass ``MyEvaluator``

Now, when you call indigo.DefaultEngine.Compile or .Evaluate, it will use your evaluator with your expression language. 




