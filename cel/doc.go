// Package cel provides an implementation of the Indigo evaluator and compiler interfaces backed by Google's cel-go rules engine.
//
// See https://github.com/google/cel-go and https://opensource.google/projects/cel for more information
// about CEL.
//
// The rule expressions you write must conform to the CEL spec: https://github.com/google/cel-spec.
//
// Working with Protocol Buffers
//
// While it is possible to use CEL with "native" simple types, it is built on protocol buffers. CEL does not
// support Go structs, so if you need to use native types to access fields in a struct, you must first
// "flatten" the fields into plain values to pass to CEL. See the makeStudentData() function in the tests
// in this package for an example of "flatting" a struct to individual data elements.
//
// Organizing your input data using protocol buffers gives you the benefit of being able to move
// data between Go code and CEL expressions without needing to translate or reorganize the data.
// There a number of examples (they start with proto) that show how to use protocol buffers in Indigo.
// They use the protocol buffer definitions in indigo/testdata/proto, and there's a Makefile in that directory
// that shows how to generate the Go types.
//
// Protocol Buffer Names
//
// When declaring the protocol buffer type in the schema, the Protoname is the proto package name followed by the type.
// For example, in the indigo/testdata/proto/student.proto file, the package name is "testdata.school" and the type defined is Student.
// When using this type in the schema, the declaration looks like this:
//
// In student.proto:
//
//     package testdata.school;
//     option go_package = "github.com/ezachrisen/indigo/testdata/school;school";
//     ...
//     message Student { ... }
//
// In Go code when declaring an Indigo schema:
//
//     {Name: "student", Type: indigo.Proto{Protoname: "testdata.school.Student", Message: &school.Student{}}},
//
// Protocol Buffer Fields in Expressions
//
// When refererring to fields of a protocol buffer in an expression, the field names are the proto names, NOT the generated Go names.
// For example, in the student.proto file, a field called "enrollment_date" is defined.
// When the Go struct is generated, this field name is now called "EnrollmentDate".
// In the CEL expression, you must use the "enrollment_date" name, as in
//
// In student.proto:
//
//     message Student {
//       google.protobuf.Timestamp enrollment_date = 7;
//     }
//
// In Go code:
//
//     s := school.Student{}
//     s.EnrollmentDate = &timestamp.Timestamp{Seconds: time.Date(2010, 5, 1, 12, 12, 59, 0, time.FixedZone("UTC-8", -8*60*60)).Unix()}
//
// In the CEL expression:
//
//     rule.Expr = ` now - student.enrollment_date > duration("4320h")`
//
//
// Protocol Buffer Enums
//
// The student protocol buffer definition includes an enum type for a student's status. When referring to enum values
// in the CEL expression, use the full protocol buffer name:
//
// In student.proto:
//
//  message Student {
//   status_type status = 4;
//   ...
//    enum status_type {
//     ENROLLED=0;
//     PROBATION=1;
//   }
//   ...
//  }
//
// In Go Code:
//
//  s:= school.Student{}
//  s.Status = school.Student_PROBATION
//
// In the CEL expression:
//
//  rule.Expr = `student.Status == testdata.school.Student.status_type.PROBATION`
//
// Protocol Buffer Timestamps
//
// The examples demonstrate how to convert to/from the Go time.Time type and
// the protocol buffer timestamp.
//
// This package has generated types for google/protobuf/timestamp.proto:
//
//  import	"google.golang.org/protobuf/types/known/timestamppb"
//
// To create a new timestamp:
//
//  now := timestamppb.Now()
//
// To convert a Go time.Time to a proto timestamp:
//
//  prototime := timestamppb.New(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC))
//
// To convert from a proto time to Go time:
//
//  goTime := pbtime.AsTime()
//
// Protocol Buffer Durations
//
// This package has generated types for google/protobuf/duration.proto:
//
//  import	"google.golang.org/protobuf/types/known/durationpb"
//
// To convert a Go duration to a proto duration:
//
//  protodur := durationpb.New(godur)
//
// To convert back from a protocol buffer duration to a Go duration:
//
//  goDur := protodur.AsDuration()
//
package cel
