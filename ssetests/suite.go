package ssetests

import (
	"github.com/launchdarkly/sse-contract-tests/client"
	"github.com/launchdarkly/sse-contract-tests/stream"
	"github.com/launchdarkly/sse-contract-tests/testframework"
)

func RunTestSuite(
	client *client.TestServiceClient,
	streamManager *stream.StreamManager,
	filter testframework.Filter,
	testLogger testframework.TestLogger,
) testframework.Results {
	return testframework.Run(filter, testLogger, func(c *testframework.Context) {
		t := &T{
			context: c,
			env: &environment{
				client:        client,
				streamManager: streamManager,
			},
		}

		t.Run("basic parsing", DoBasicParsingTests)
		t.Run("comments", DoCommentTests)
		t.Run("linefeeds", DoLinefeedTests)
		t.Run("HTTP behavior", DoHTTPBehaviorTests)
		t.Run("reconnection", DoReconnectionTests)
	})
}
