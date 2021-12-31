package ssetests

import (
	"github.com/launchdarkly/sse-contract-tests/framework/harness"
	"github.com/launchdarkly/sse-contract-tests/framework/ldtest"
)

type SSETestContext struct {
	harness *harness.TestHarness
}

func requireContext(t *ldtest.T) SSETestContext {
	if c, ok := t.Context().(SSETestContext); ok {
		return c
	}
	panic("SSETestContext was not included in the global test configuration!" +
		" This is a basic mistake in the initialization logic.")
}

func NewStreamAndSSEClient(
	t *ldtest.T,
	configurers ...SSEClientConfigurer,
) (*StreamServer, *StreamConnection, *SSEClient) {
	server := NewStreamServer(t)
	allConfigurers := append(append([]SSEClientConfigurer(nil), configurers...), server)
	client := NewSSEClient(t, allConfigurers...)
	stream := server.AwaitConnection(t)
	return server, stream, client
}
