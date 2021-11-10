package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/launchdarkly/sse-contract-tests/framework"
)

type ConsoleTestLogger struct {
	DebugOutputOnFailure bool
	DebugOutputOnSuccess bool
}

func (c *ConsoleTestLogger) TestStarted(id framework.TestID) {
	fmt.Printf("[%s]\n", id)
}

func (c *ConsoleTestLogger) TestError(id framework.TestID, err error) {
	for _, line := range strings.Split(err.Error(), "\n") {
		fmt.Printf("  %s\n", line)
	}
}

func (c *ConsoleTestLogger) TestFinished(id framework.TestID, failed bool, debugOutput framework.CapturedOutput) {
	if failed {
		fmt.Printf("  FAILED: %s\n", id)
	}
	if len(debugOutput) > 0 &&
		((failed && c.DebugOutputOnFailure) || (!failed && c.DebugOutputOnSuccess)) {
		debugOutput.Dump(os.Stdout, "    DEBUG ")
	}
}

func (c *ConsoleTestLogger) TestSkipped(id framework.TestID, reason string) {
	if reason == "" {
		fmt.Printf("  SKIPPED: %s\n", id)
	} else {
		fmt.Printf("  SKIPPED: %s (%s)\n", id, reason)
	}
}
