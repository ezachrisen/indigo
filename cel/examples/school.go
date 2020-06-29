package examples

import "time"

//go:generate gentype -v -types=Student,Suspension,StudentSummary,Grade
type Student struct {
	ID             float64
	Age            int
	GPA            float64
	Status         int
	EnrollmentDate time.Time
	Grades         []float64
	GradeBook      []Grade
	Teachers       map[string]string
	Summary        StudentSummary
}

type Grade struct {
	NumericGrade float64
	LetterGrade  string
}

type Suspension struct {
	Cause string
	Date  time.Time
}

type StudentSummary struct {
	ClassesTaken int
	RiskFactor   float64
	ABC          int
	Tenure       time.Duration
}
