package testsuite

import (
	"log"

	"github.com/launchdarkly/sse-contract-tests/client"
	"github.com/launchdarkly/sse-contract-tests/stream"
)

func RunTestSuite(
	client *client.SSETestClient,
	server *stream.Server,
	logger TestLogger,
	debugLogger *log.Logger,
) Result {
	var result Result

	Run(client, server, &result, logger, debugLogger, func(t *TestContext) {
		t.Run("basic parsing", DoBasicParsingTests)
		t.Run("linefeeds", DoLinefeedTests)
		t.Run("comments", DoCommentTests)
		t.Run("HTTP request", DoHTTPRequestTests)
		t.Run("reconnection", DoReconnectionTests)
		t.Run("read timeout", DoReadTimeoutTests)
	})

	return result
}
