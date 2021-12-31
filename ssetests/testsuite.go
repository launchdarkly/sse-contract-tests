package ssetests

import (
	"github.com/launchdarkly/sse-contract-tests/framework/harness"
	"github.com/launchdarkly/sse-contract-tests/framework/ldtest"
)

var AllCapabilities = []string{ //nolint:gochecknoglobals
	"comments",
	"headers",
	"last-event-id",
	"post",
	"read-timeout",
	"report",
	"retry",
}

func RunTestSuite(
	harness *harness.TestHarness,
	filter ldtest.Filter,
	testLogger ldtest.TestLogger,
) ldtest.Results {
	config := ldtest.TestConfiguration{
		Filter:       filter,
		Capabilities: harness.TestServiceInfo().Capabilities,
		TestLogger:   testLogger,
		Context: SSETestContext{
			harness: harness,
		},
	}

	return ldtest.Run(config, func(t *ldtest.T) {
		t.Run("basic parsing", DoBasicParsingTests)
		t.Run("comments", DoCommentTests)
		t.Run("linefeeds", DoLinefeedTests)
		t.Run("HTTP behavior", DoHTTPBehaviorTests)
		t.Run("reconnection", DoReconnectionTests)
	})
}
