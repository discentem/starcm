package base

type Result struct {
	Output  *string
	Error   error
	Success bool
	Changed bool
	Diff    *string
	Comment string
}
