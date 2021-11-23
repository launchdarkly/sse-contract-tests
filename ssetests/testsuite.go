package ssetests

import (
	"github.com/launchdarkly/sse-contract-tests/framework"
)

func RunTestSuite(
	harness *framework.TestHarness,
	filter framework.Filter,
	testLogger framework.TestLogger,
) framework.Results {
	return framework.Run(filter, testLogger, func(c *framework.Context) {
		t := newTestScope(c, harness)

		t.Run("basic parsing", DoBasicParsingTests)
		t.Run("comments", DoCommentTests)
		t.Run("linefeeds", DoLinefeedTests)
		t.Run("HTTP behavior", DoHTTPBehaviorTests)
		t.Run("reconnection", DoReconnectionTests)
	})
}
