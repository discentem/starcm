package base

type Result struct {
	Output      *string
	Success     bool
	Changed     bool
	Diff        *string
	Comment     string
	AfterResult *AfterResult
}

type AfterResult struct {
	Result
}
