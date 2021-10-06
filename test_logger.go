package main

import (
	"fmt"

	"github.com/launchdarkly/sse-contract-tests/testsuite"
)

type ConsoleTestLogger struct{}

func (c *ConsoleTestLogger) TestStarted(id testsuite.TestID) {
	fmt.Printf("[%s]\n", id)
}

func (c *ConsoleTestLogger) TestError(id testsuite.TestID, err error) {
	fmt.Printf("  %s\n", err)
}

func (c *ConsoleTestLogger) TestFinished(id testsuite.TestID, failed bool) {
	if failed {
		fmt.Printf("  [%s] FAILED\n", id)
	}
}

func (c *ConsoleTestLogger) TestSkipped(id testsuite.TestID) {
	fmt.Printf("  [%s] SKIPPED\n", id)
}
