# cel

Package cel provides an implementation of the Indigo evaluator and compiler interfaces backed by Google's cel-go rules engine.

See [https://github.com/google/cel-go](https://github.com/google/cel-go) and [https://opensource.google/projects/cel](https://opensource.google/projects/cel) for more information
about CEL.

The rule expressions you write must conform to the CEL spec: [https://github.com/google/cel-spec](https://github.com/google/cel-spec).

## Working with Protocol Buffers

While it is possible to use CEL with "native" simple types, it is built on protocol buffers. CEL does not
support Go structs, so if you need to use native types to access fields in a struct, you must first
"flatten" the fields into plain values to pass to CEL. See the makeStudentData() function in the tests
in this package for an example of "flatting" a struct to individual data elements.

Organizing your input data using protocol buffers gives you the benefit of being able to move
data between Go code and CEL expressions without needing to translate or reorganize the data.
There a number of examples (they start with proto) that show how to use protocol buffers in Indigo.
They use the protocol buffer definitions in indigo/testdata/proto, and there's a Makefile in that directory
that shows how to generate the Go types.

## Protocol Buffer Names

When declaring the protocol buffer type in the schema, the Protoname is the proto package name followed by the type.
For example, in the indigo/testdata/proto/student.proto file, the package name is "testdata.school" and the type defined is Student.
When using this type in the schema, the declaration looks like this:

In student.proto:

```go
package testdata.school;
option go_package = "github.com/ezachrisen/indigo/testdata/school;school";
...
message Student { ... }
```

In Go code when declaring an Indigo schema:

```go
{Name: "student", Type: indigo.Proto{Protoname: "testdata.school.Student", Message: &school.Student{}}},
```

## Protocol Buffer Fields in Expressions

When refererring to fields of a protocol buffer in an expression, the field names are the proto names, NOT the generated Go names.
For example, in the student.proto file, a field called "enrollment_date" is defined.
When the Go struct is generated, this field name is now called "EnrollmentDate".
In the CEL expression, you must use the "enrollment_date" name, as in

In student.proto:

```go
message Student {
  google.protobuf.Timestamp enrollment_date = 7;
}
```

In Go code:

```go
s := school.Student{}
s.EnrollmentDate = &timestamp.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()}
```

In the CEL expression:

```go
rule.Expr = ` now - student.enrollment_date > duration("4320h")`
```

## Protocol Buffer Enums

The student protocol buffer definition includes an enum type for a student's status. When referring to enum values
in the CEL expression, use the full protocol buffer name:

In student.proto:

```go
message Student {
 status_type status = 4;
 ...
  enum status_type {
   ENROLLED=0;
   PROBATION=1;
 }
 ...
}
```

In Go Code:

```go
s:= school.Student{}
s.Status = school.Student_PROBATION
```

In the CEL expression:

```go
rule.Expr = `student.Status == testdata.school.Student.status_type.PROBATION`
```

## Protocol Buffer Timestamps

The examples demonstrate how to convert to/from the Go time.Time type and
the protocol buffer timestamp.

This package has generated types for google/protobuf/timestamp.proto:

```go
import	"google.golang.org/protobuf/types/known/timestamppb"
```

To create a new timestamp:

```go
now := timestamppb.Now()
```

To convert a Go time.Time to a proto timestamp:

```go
prototime := timestamppb.New(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC))
```

To convert from a proto time to Go time:

```go
goTime := pbtime.AsTime()
```

## Protocol Buffer Durations

This package has generated types for google/protobuf/duration.proto:

```go
import	"google.golang.org/protobuf/types/known/durationpb"
```

To convert a Go duration to a proto duration:

```go
protodur := durationpb.New(godur)
```

To convert back from a protocol buffer duration to a Go duration:

```go
goDur := protodur.AsDuration()
```

## Examples

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
		ID:     "hello_check",
		Schema: schema,
		Expr:   `message == "hello world"`,
	}

	// Step 3: Create an Indigo engine and give it an evaluator
	// In this case, CEL
	engine := indigo.NewEngine(cel.NewEvaluator())

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
	results, err := engine.Eval(context.Background(), &rule, data)
	fmt.Println(results.Pass)
}

```

 Output:

```
true
```

### NativeTimestampComparison

```golang
package main

import (
	"context"
	"fmt"
	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	schema := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "then", Type: indigo.String{}},
			{Name: "now", Type: indigo.Timestamp{}},
		},
	}

	data := map[string]interface{}{
		"then": "1972-01-01T10:00:20.021-05:00", //"2018-08-03T16:00:00-07:00",
		"now":  timestamppb.Now(),
	}

	rule := indigo.Rule{
		ID:     "time_check",
		Schema: schema,
		Expr:   `now > timestamp(then)`,
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println(results.Pass)
}

```

 Output:

```
true
```

### ProtoConstruction

Demonstrate constructing a proto message in an expression

```golang
package main

import (
	"context"
	"fmt"
	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"github.com/ezachrisen/indigo/testdata/school"
)

func main() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Protoname: "testdata.school.Student", Message: &school.Student{}}},
			{Name: "student_suspension", Type: indigo.Proto{Protoname: "testdata.school.Student.Suspension", Message: &school.Student_Suspension{}}},
			{Name: "studentSummary", Type: indigo.Proto{Protoname: "testdata.school.StudentSummary", Message: &school.StudentSummary{}}},
		},
	}

	data := map[string]interface{}{
		"student": school.Student{
			Grades: []float64{3.0, 2.9, 4.0, 2.1},
			Suspensions: []*school.Student_Suspension{
				&school.Student_Suspension{Cause: "Cheating"},
				&school.Student_Suspension{Cause: "Fighting"},
			},
		},
	}

	rule := indigo.Rule{
		ID:         "create_summary",
		Schema:     education,
		ResultType: indigo.Proto{Protoname: "testdata.school.StudentSummary", Message: &school.StudentSummary{}},
		Expr: `
			testdata.school.StudentSummary {
				gpa: student.gpa,
				risk_factor: 2.0 + 3.0,
				tenure: duration("12h")
			}`,
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}

	// The result is a fully-formed school.StudentSummary message.
	// There is no need to convert it.
	fmt.Printf("%T\n", results.Value)
	summ := results.Value.(*school.StudentSummary)
	fmt.Println(summ.RiskFactor)
}

```

 Output:

```
*school.StudentSummary
5
```

### ProtoConstructionConditional

Demonstrate using the ? : operator to conditionally construct a proto message

```golang
package main

import (
	"context"
	"fmt"
	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"github.com/ezachrisen/indigo/testdata/school"
)

func main() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Protoname: "testdata.school.Student", Message: &school.Student{}}},
			{Name: "student_suspension", Type: indigo.Proto{Protoname: "testdata.school.Student.Suspension", Message: &school.Student_Suspension{}}},
			{Name: "studentSummary", Type: indigo.Proto{Protoname: "testdata.school.StudentSummary", Message: &school.StudentSummary{}}},
		},
	}

	data := map[string]interface{}{
		"student": school.Student{
			Gpa:    3.6,
			Grades: []float64{3.0, 2.9, 4.0, 2.1},
			Suspensions: []*school.Student_Suspension{
				&school.Student_Suspension{Cause: "Cheating"},
				&school.Student_Suspension{Cause: "Fighting"},
			},
		},
	}

	rule := indigo.Rule{
		ID:         "create_summary",
		Schema:     education,
		ResultType: indigo.Proto{Protoname: "testdata.school.StudentSummary", Message: &school.StudentSummary{}},
		Expr: `
			student.gpa > 3.0 ? 
				testdata.school.StudentSummary {
					gpa: student.gpa,
					risk_factor: 0.0
				}
			:
				testdata.school.StudentSummary {
					gpa: student.gpa,
					risk_factor: 2.0 + 3.0,
					tenure: duration("12h")
				}
			`,
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}

	// The result is a fully-formed school.StudentSummary message.
	// There is no need to convert it.
	fmt.Printf("%T\n", results.Value)
	summ := results.Value.(*school.StudentSummary)
	fmt.Println(summ.RiskFactor)
}

```

 Output:

```
*school.StudentSummary
0
```

### ProtoDurationComparison

Demonstrates conversion between protobuf durations (google.protobuf.Duration) and time.Duration

```golang
package main

import (
	"context"
	"fmt"
	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"github.com/ezachrisen/indigo/testdata/school"
	"google.golang.org/protobuf/types/known/durationpb"
	"time"
)

func main() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "smry", Type: indigo.Proto{Protoname: "testdata.school.StudentSummary", Message: &school.StudentSummary{}}},
		},
	}

	godur, _ := time.ParseDuration("10h")

	data := map[string]interface{}{
		"smry": school.StudentSummary{
			Tenure: durationpb.New(godur),
		},
	}

	rule := indigo.Rule{
		ID:     "tenure_check",
		Schema: education,
		Expr:   `smry.tenure > duration("1h")`,
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println(results.Pass)
}

```

 Output:

```
true
```

### ProtoExistsOperator

Demonstrates using the CEL exists function to check for a value in a slice

```golang
package main

import (
	"context"
	"fmt"
	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"github.com/ezachrisen/indigo/testdata/school"
)

func main() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Protoname: "testdata.school.Student", Message: &school.Student{}}},
		},
	}

	data := map[string]interface{}{
		"student": school.Student{
			Grades: []float64{3.0, 2.9, 4.0, 2.1},
		},
	}

	rule := indigo.Rule{
		ID:     "grade_check",
		Schema: education,
		Expr:   `student.grades.exists(g, g < 2.0)`,
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println(results.Pass)
}

```

 Output:

```
false
```

### ProtoNestedMessages

Demonstrates using the exists macro to inspect the value of nested messages in the list

```golang
package main

import (
	"context"
	"fmt"
	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"github.com/ezachrisen/indigo/testdata/school"
)

func main() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Protoname: "testdata.school.Student", Message: &school.Student{}}},
			{Name: "student_suspension", Type: indigo.Proto{Protoname: "testdata.school.Student.Suspension", Message: &school.Student_Suspension{}}},
		},
	}

	data := map[string]interface{}{
		"student": school.Student{
			Grades: []float64{3.0, 2.9, 4.0, 2.1},
			Suspensions: []*school.Student_Suspension{
				&school.Student_Suspension{Cause: "Cheating"},
				&school.Student_Suspension{Cause: "Fighting"},
			},
		},
	}

	// Check if the student was ever suspended for fighting
	rule := indigo.Rule{
		ID:     "fighting_check",
		Schema: education,
		Expr:   `student.suspensions.exists(s, s.cause == "Fighting")`,
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	fmt.Println(results.Value)
}

```

 Output:

```
true
```

### ProtoTimestampComparison

Demonstrates conversion between protobuf timestamps (google.protobuf.Timestamp) and time.Time

```golang
package main

import (
	"context"
	"fmt"
	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"github.com/ezachrisen/indigo/testdata/school"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

func main() {

	education := indigo.Schema{
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Protoname: "testdata.school.Student", Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
		},
	}

	data := map[string]interface{}{
		"student": school.Student{
			// Make a protobuf timestamp from a time.Time
			EnrollmentDate: timestamppb.New(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
			Grades:         []float64{3.0, 2.9, 4.0, 2.1},
		},
		"now": timestamppb.Now(),
	}

	// The rule will return the earlier of the two dates (enrollment date or now)
	rule := indigo.Rule{
		ID:     "grade_check",
		Schema: education,
		Expr: `student.enrollment_date < now
?
student.enrollment_date
:
now
`,
	}

	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(&rule)
	if err != nil {
		fmt.Printf("Error adding rule %v", err)
		return
	}

	results, err := engine.Eval(context.Background(), &rule, data)
	if err != nil {
		fmt.Printf("Error evaluating: %v", err)
		return
	}
	if pbtime, ok := results.Value.(*timestamppb.Timestamp); ok {
		// Convert from a protobuf timestamp to a go time.Time
		goTime := pbtime.AsTime()
		fmt.Printf("Gotime is %v\n", goTime)
	}
}

```

 Output:

```
Gotime is 2009-11-10 23:00:00 +0000 UTC
```

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
