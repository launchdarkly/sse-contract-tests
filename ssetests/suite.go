package ssetests

import (
	"github.com/launchdarkly/sse-contract-tests/client"
	"github.com/launchdarkly/sse-contract-tests/framework"
	"github.com/launchdarkly/sse-contract-tests/mockstream"
)

func RunTestSuite(
	client *client.TestServiceClient,
	streamManager *mockstream.StreamManager,
	filter framework.Filter,
	testLogger framework.TestLogger,
) framework.Results {
	return framework.Run(filter, testLogger, func(c *framework.Context) {
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
