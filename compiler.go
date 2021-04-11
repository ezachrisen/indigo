package indigo

type Compiler interface {
	Compile(r *Rule, collectDiagnostics, dryRun bool) (interface{}, error)
}
