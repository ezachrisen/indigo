package examples

//go:generate gentype -types=Student,Suspension,StudentSummary
type Student struct {
	ID     float64
	Age    int
	GPA    float64
	Status int
	// EnrollmentDate time.Time
	// Grades []float64
}

type Suspension struct {
	Cause string
	//	Date time.Time
}

type StudentSummary struct {
	GPA        float64
	RiskFactor float64
	//	Tenure time.Duration
}
