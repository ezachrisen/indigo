package cel_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/ezachrisen/indigo/cel"
	"github.com/ezachrisen/indigo/testdata/school"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// createHierarchicalRules generates a deep hierarchy of rules with configurable depth and breadth.
//
// Parameters:
//   - depth: The number of levels in the hierarchy (1 = single rule, 2 = root + children, etc.)
//   - breadth: The number of child rules each parent rule should have
//   - schema: The Indigo schema to apply to all rules in the hierarchy
//
// The function creates a tree structure where each rule (except leaf rules) has exactly
// 'breadth' number of children, and the tree extends 'depth' levels deep.
//
// Example 1: depth=3, breadth=2
//
//	Root
//	├── level_3_child_0
//	│   ├── level_2_child_0
//	│   └── level_2_child_1
//	└── level_3_child_1
//	    ├── level_2_child_0
//	    └── level_2_child_1
//
// Total rules: 7 (1 root + 2 level-2 + 4 level-1)
//
// Example 2: depth=4, breadth=3
//
//	Root (level_4)
//	├── level_4_child_0 (level_3)
//	│   ├── level_3_child_0 (level_2)
//	│   │   ├── level_2_child_0 (level_1)
//	│   │   ├── level_2_child_1 (level_1)
//	│   │   └── level_2_child_2 (level_1)
//	│   ├── level_3_child_1 (level_2)
//	│   │   ├── level_2_child_0 (level_1)
//	│   │   ├── level_2_child_1 (level_1)
//	│   │   └── level_2_child_2 (level_1)
//	│   └── level_3_child_2 (level_2)
//	│       ├── level_2_child_0 (level_1)
//	│       ├── level_2_child_1 (level_1)
//	│       └── level_2_child_2 (level_1)
//	├── level_4_child_1 (level_3)
//	│   └── [... 3 children each with 3 children ...]
//	└── level_4_child_2 (level_3)
//	    └── [... 3 children each with 3 children ...]
//
// Total rules: 40 (1 root + 3 level-3 + 9 level-2 + 27 level-1)
//
// Each rule in the hierarchy gets a complex CEL expression that exercises different
// aspects of the Student proto, including GPA analysis, status checks, suspension
// logic, attribute filtering, time calculations, and list operations.
func createHierarchicalRules(depth, breadth int, schema indigo.Schema) *indigo.Rule {
	if depth <= 0 {
		return nil
	}

	// Complex expressions using various Student proto fields
	complexExprs := []string{
		// GPA and grade analysis
		`student.gpa >= 3.5 && student.grades.exists(g, g >= 3.0) && size(student.grades) >= 3`,
		// Status and enrollment checks
		`student.status == testdata.school.Student.status_type.ENROLLED && student.age >= 18 && student.credits >= 12`,
		// Suspension and behavioral analysis
		`size(student.suspensions) == 0 || student.suspensions.all(s, s.cause != "Cheating")`,
		// Attribute and housing checks
		`student.attrs.exists(k, k == "major") && has(student.on_campus) && student.on_campus.building != ""`,
		// Time-based calculations
		`now - student.enrollment_date > duration("8760h") && student.gpa > 2.0`, // 1 year
		// Complex grade calculations using sum
		`student.grades.exists(g, g == 4.0) && student.grades.all(g, g >= 2.0) && student.gpa >= 3.0`,
		// Advanced attribute filtering
		`student.attrs.exists(k, k == "home_town") && student.attrs["home_town"] in ["Chicago", "Boston", "Seattle"]`,
		// Combined logic with multiple conditions
		`(student.gpa >= 3.0 && student.status != testdata.school.Student.status_type.PROBATION) || (student.credits >= 60 && student.age >= 21)`,
		// List operations and filtering
		`student.grades.filter(g, g >= 3.0).size() >= student.grades.filter(g, g < 3.0).size()`,
		// Complex suspension logic
		`!student.suspensions.exists(s, s.cause == "Fighting") && student.suspensions.size() <= 1`,
	}

	root := indigo.NewRule(fmt.Sprintf("level_%d", depth), "")
	root.Schema = schema
	root.Expr = complexExprs[depth%len(complexExprs)]

	// Create child rules if we haven't reached the bottom
	if depth > 1 {
		for i := 0; i < breadth; i++ {
			childID := fmt.Sprintf("level_%d_child_%d", depth, i)
			childRule := createHierarchicalRules(depth-1, breadth, schema)
			if childRule != nil {
				childRule.ID = childID
				childRule.Expr = complexExprs[(depth+i)%len(complexExprs)]
				root.Rules[childID] = childRule
			}
		}
	}

	return root
}

// createComprehensiveStudentData creates a Student proto that satisfies all complex expressions
func createComprehensiveStudentData() map[string]any {
	student := &school.Student{
		Id:      123456,
		Age:     22,
		Credits: 75,
		Gpa:     3.85,
		Status:  school.Student_ENROLLED,
		Grades:  []float64{4.0, 3.8, 3.9, 4.0, 3.7, 3.6, 4.0, 3.5},
		Attrs: map[string]string{
			"major":     "Computer Science",
			"home_town": "Chicago",
			"year":      "Senior",
			"dean_list": "true",
		},
		EnrollmentDate: timestamppb.New(time.Now().Add(-2 * 365 * 24 * time.Hour)), // 2 years ago
		Suspensions:    []*school.Student_Suspension{},                             // No suspensions
		HousingAddress: &school.Student_OnCampus{
			OnCampus: &school.Student_CampusAddress{
				Building: "North Hall",
				Room:     "302A",
			},
		},
	}

	return map[string]any{
		"student": student,
		"now":     timestamppb.Now(),
	}
}

// createComprehensiveStudentDataNoProto creates a student data object without using a proto
func createComprehensiveStudentDataNoProto() map[string]any {
	return map[string]any{
		"student_gpa":             3.85,
		"student_status":          1,
		"student_credits":         75,
		"student_age":             22,
		"student_grades":          []float64{4.0, 3.8, 3.9, 4.0, 3.7, 3.6, 4.0, 3.5},
		"student_suspensions":     []string{"nothing", "cheating"},
		"student_enrollment_date": time.Now().Add(-2 * 365 * 24 * time.Hour),
		"student_attrs": map[string]string{
			"major":     "Computer Science",
			"home_town": "Chicago",
			"year":      "Senior",
			"dean_list": "true",
		},
		"now": timestamppb.Now(),
	}
}

// TestHierarchicalRulesDeep tests deep hierarchical rule evaluation with complex expressions
func TestHierarchicalRulesDeep(t *testing.T) {
	testCases := []struct {
		name        string
		depth       int
		breadth     int
		expectPass  bool
		description string
	}{
		{
			name:        "shallow_narrow",
			depth:       3,
			breadth:     2,
			expectPass:  true,
			description: "3 levels deep, 2 children per level",
		},
		{
			name:        "medium_depth",
			depth:       5,
			breadth:     3,
			expectPass:  true,
			description: "5 levels deep, 3 children per level",
		},
		{
			name:        "deep_hierarchy",
			depth:       7,
			breadth:     2,
			expectPass:  true,
			description: "7 levels deep, 2 children per level",
		},
		{
			name:        "wide_shallow",
			depth:       3,
			breadth:     5,
			expectPass:  true,
			description: "3 levels deep, 5 children per level",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create schema
			schema := indigo.Schema{
				ID: "comprehensive_student_schema",
				Elements: []indigo.DataElement{
					{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
					{Name: "now", Type: indigo.Timestamp{}},
				},
			}

			// Create hierarchical rules
			root := createHierarchicalRules(tc.depth, tc.breadth, schema)
			if root == nil {
				t.Fatal("root should not be nil")
			}

			// Create engine and compile
			engine := indigo.NewEngine(cel.NewEvaluator())
			err := engine.Compile(root, indigo.CollectDiagnostics(true))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Create test data that satisfies all conditions
			data := createComprehensiveStudentData()

			// Evaluate rules
			results, err := engine.Eval(context.Background(), root, data, indigo.ReturnDiagnostics(true), indigo.Parallel(2, 1000))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if results == nil {
				t.Fatal("results should not be nil")
			}

			// Verify results structure
			if tc.expectPass {
				if !results.ExpressionPass {
					t.Fatal("expected results.ExpressionPass to be true")
				}
			}

			// Count total rules evaluated
			totalRules := countRulesInResult(results)
			expectedMinRules := calculateExpectedRules(tc.depth, tc.breadth)
			if totalRules < expectedMinRules {
				t.Fatalf("expected at least %d rules, got %d", expectedMinRules, totalRules)
			}

			t.Logf("Test case: %s - Evaluated %d rules across %d levels with %d children per level",
				tc.description, totalRules, tc.depth, tc.breadth)
		})
	}
}

// countRulesInResult recursively counts all rules in the result hierarchy
func countRulesInResult(result *indigo.Result) int {
	count := 1 // Count this rule
	for _, childResult := range result.Results {
		count += countRulesInResult(childResult)
	}
	return count
}

// calculateExpectedRules calculates the minimum expected number of rules for a given depth and breadth
func calculateExpectedRules(depth, breadth int) int {
	if depth <= 0 {
		return 0
	}
	if depth == 1 {
		return 1
	}

	total := 1 // Root rule
	for level := 1; level < depth; level++ {
		levelRules := 1
		for i := 0; i < level; i++ {
			levelRules *= breadth
		}
		total += levelRules
	}
	return total
}

var (
	depth   = 2
	breadth = 1000
)

// BenchmarkHierarchicalRules benchmarks performance of deep hierarchical rule evaluation
func BenchmarkHierarchicalRules(b *testing.B) {
	schema := indigo.Schema{
		ID: "benchmark_student_schema",
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
		},
	}

	// Create a moderately complex hierarchy for benchmarking
	root := createHierarchicalRules(depth, breadth, schema)
	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(root)
	if err != nil {
		b.Fatalf("Failed to compile hierarchical rules: %v", err)
	}

	data := createComprehensiveStudentData()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Eval(context.Background(), root, data)
		if err != nil {
			b.Fatalf("Failed to evaluate hierarchical rules: %v", err)
		}
	}
}

// BenchmarkHierarchicalRules benchmarks performance of deep hierarchical rule evaluation
func BenchmarkHierarchicalRulesParallel(b *testing.B) {
	schema := indigo.Schema{
		ID: "benchmark_student_schema",
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
		},
	}

	// Create a moderately complex hierarchy for benchmarking
	root := createHierarchicalRules(depth, breadth, schema)
	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(root)
	if err != nil {
		b.Fatalf("Failed to compile hierarchical rules: %v", err)
	}

	data := createComprehensiveStudentData()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Eval(context.Background(), root, data, indigo.Parallel(100, 100))
		if err != nil {
			b.Fatalf("Failed to evaluate hierarchical rules: %v", err)
		}
	}
}

// BenchmarkProtoComplex tests a rule with a complex proto expression
func BenchmarkProtoComplex(b *testing.B) {
	// b.ResetTimer()

	schema := indigo.Schema{
		ID: "benchmark_student_schema",
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
		},
	}
	root := createHierarchicalRules(2, 1000, schema)
	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(root)
	if err != nil {
		b.Fatalf("Failed to compile hierarchical rules: %v", err)
	}

	data := createComprehensiveStudentData()

	for i := 0; i < b.N; i++ {
		_, err := engine.Eval(context.Background(), root, data)
		if err != nil {
			b.Fatalf("Failed to evaluate hierarchical rules: %v", err)
		}
	}
}

// BenchmarkHierarchicalNoProto tests a large hierarchy of rules that use native simple types
// rather than protobuf objects
func BenchmarkHierarchicalNoProto(b *testing.B) {
	// b.ResetTimer()

	schema := indigo.Schema{
		ID: "benchmark_student_schema",
		Elements: []indigo.DataElement{
			{Name: "student_gpa", Type: indigo.Float{}},
			{Name: "student_status", Type: indigo.Int{}},
			{Name: "student_credits", Type: indigo.Int{}},
			{Name: "student_age", Type: indigo.Int{}},
			{Name: "student_grades", Type: indigo.List{ValueType: indigo.Float{}}},
			{Name: "student_suspensions", Type: indigo.List{ValueType: indigo.String{}}},
			{Name: "student_enrollment_date", Type: indigo.Timestamp{}},
			{Name: "student_attrs", Type: indigo.Map{KeyType: indigo.String{}, ValueType: indigo.String{}}},
			{Name: "now", Type: indigo.Timestamp{}},
		},
	}
	root := createHierarchicalRules3(2, 1000, schema)
	engine := indigo.NewEngine(cel.NewEvaluator())

	err := engine.Compile(root)
	if err != nil {
		b.Fatalf("Failed to compile hierarchical rules: %v", err)
	}

	data := createComprehensiveStudentDataNoProto()

	for i := 0; i < b.N; i++ {
		_, err := engine.Eval(context.Background(), root, data)
		if err != nil {
			b.Fatalf("Failed to evaluate hierarchical rules: %v", err)
		}
	}
}

func studentSchema() indigo.Schema {
	schema := indigo.Schema{
		ID: "benchmark_student_schema",
		Elements: []indigo.DataElement{
			{Name: "student", Type: indigo.Proto{Message: &school.Student{}}},
			{Name: "now", Type: indigo.Timestamp{}},
		},
	}
	return schema
}

func createHierarchicalRules3(depth, breadth int, schema indigo.Schema) *indigo.Rule {
	if depth <= 0 {
		return nil
	}

	// Complex expressions using various Student proto fields
	complexExprs := []string{
		// GPA and grade analysis
		`student_gpa >= 3.5 && student_grades.exists(g, g >= 3.0) && size(student_grades) >= 3`,
		// Status and enrollment checks
		// Suspension and behavioral analysis
		// Attribute and housing checks
		// Time-based calculations
		`now - student_enrollment_date > duration("8760h") && student_gpa > 2.0`, // 1 year
		// Complex grade calculations using sum
		`student_grades.exists(g, g == 4.0) && student_grades.all(g, g >= 2.0) && student_gpa >= 3.0`,
		// Advanced attribute filtering
		`student_attrs.exists(k, k == "home_town") && student_attrs["home_town"] in ["Chicago", "Boston", "Seattle"]`,
		// Combined logic with multiple conditions
		`(student_gpa >= 3.0 && student_status != 5) || (student_credits >= 60 && student_age >= 21)`,
		// List operations and filtering
		`student_grades.filter(g, g >= 3.0).size() >= student_grades.filter(g, g < 3.0).size()`,
		// Complex suspension logic
	}

	root := indigo.NewRule(fmt.Sprintf("level_%d", depth), "")
	root.Schema = schema
	root.Expr = complexExprs[depth%len(complexExprs)]

	// Create child rules if we haven't reached the bottom
	if depth > 1 {
		for i := 0; i < breadth; i++ {
			childID := fmt.Sprintf("level_%d_child_%d", depth, i)
			childRule := createHierarchicalRules(depth-1, breadth, schema)
			if childRule != nil {
				childRule.ID = childID
				childRule.Expr = complexExprs[(depth+i)%len(complexExprs)]
				root.Rules[childID] = childRule
			}
		}
	}

	return root
}
