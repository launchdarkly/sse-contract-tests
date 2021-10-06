package testsuite

import (
	"fmt"
	"strings"
)

type Result struct {
	Tests    []TestResult
	Failures []TestResult
}

type TestResult struct {
	TestID  TestID
	Errors  []error
	Skipped bool
}

func (r Result) OK() bool {
	return len(r.Failures) == 0
}

type TestID struct {
	Path []string
}

func (t TestID) String() string {
	return strings.Join(t.Path, "/")
}

type TestFailure struct {
	ID  TestID
	Err error
}

func (f TestFailure) Error() string {
	return fmt.Sprintf("[%s]: %s", f.ID, f.Err)
}
