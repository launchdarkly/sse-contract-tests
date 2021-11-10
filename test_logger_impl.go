package main

import (
	"fmt"
	"strings"

	"github.com/launchdarkly/sse-contract-tests/logging"
	"github.com/launchdarkly/sse-contract-tests/testframework"
)

const timestampFormat = "2006-01-02 15:04:05.000"

type ConsoleTestLogger struct {
	DebugOutputOnFailure bool
	DebugOutputOnSuccess bool
}

func (c *ConsoleTestLogger) TestStarted(id testframework.TestID) {
	fmt.Printf("[%s]\n", id)
}

func (c *ConsoleTestLogger) TestError(id testframework.TestID, err error) {
	for _, line := range strings.Split(err.Error(), "\n") {
		fmt.Printf("  %s\n", line)
	}
}

func (c *ConsoleTestLogger) TestFinished(id testframework.TestID, failed bool, debugOutput []logging.CapturedMessage) {
	if failed {
		fmt.Printf("  FAILED: %s\n", id)
	}
	if len(debugOutput) > 0 &&
		((failed && c.DebugOutputOnFailure) || (!failed && c.DebugOutputOnSuccess)) {
		for _, m := range debugOutput {
			fmt.Printf("    DEBUG [%s] %s\n",
				m.Time.Format(timestampFormat),
				m.Message,
			)
		}
	}
}

func (c *ConsoleTestLogger) TestSkipped(id testframework.TestID, reason string) {
	if reason == "" {
		fmt.Printf("  SKIPPED: %s\n", id)
	} else {
		fmt.Printf("  SKIPPED: %s (%s)\n", id, reason)
	}
}
