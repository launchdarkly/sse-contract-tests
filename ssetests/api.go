package ssetests

import (
	"fmt"
	"time"

	"github.com/launchdarkly/sse-contract-tests/client"
	"github.com/launchdarkly/sse-contract-tests/stream"
	"github.com/launchdarkly/sse-contract-tests/testframework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T represents a test or subtest in our SSE test suite.
//
// It implements the same basic functionality as Go's testing.T, but in an environment that is outside
// of the Go test runner, and with some extra features such as debug logging that are convenient for
// our use case. Those features are provided by our lower-level testframework package.
//
// It also provides functionality that is specific to SSE testing. Every T instance maintains a mock
// stream (implemented in our stream package), and a reference to an SSE client in the test service.
// It has methods for interacting with both of those.
//
// To make test assertions, you can use the assert and require packages, passing the *T as if it were
// a *testing.T. There are also assertions built into many of the stream/client interaction methods,
// causing the test to immediately fail if something unexpected happens, to reduce the amount of
// boilerplate logic in tests.
type T struct {
	context   *testframework.Context
	env       *environment
	stream    *stream.MockStream
	sseClient *client.TestServiceEntity
}

type environment struct {
	client        *client.TestServiceClient
	streamManager *stream.StreamManager
}

func (t *T) close() {
	if t.sseClient != nil {
		t.sseClient.Close()
	}
	if t.stream != nil {
		t.stream.Close()
	}
}

// Errorf is called by assertions to log a test failure. It does not cause an immediate exit.
func (t *T) Errorf(format string, args ...interface{}) {
	t.context.Errorf(format, args...)
}

// FailNow is called by assertions when a test should fail and immediately exit. The methods in
// the require package call FailNow.
func (t *T) FailNow() {
	t.context.FailNow()
}

// Run runs a subtest. This is equivalent to the Run method of testing.T.
//
// The specified function receives a new T instance, with its own mock stream.
func (t *T) Run(name string, action func(*T)) {
	t1 := &T{env: t.env}

	t.context.Run(name, func(c *testframework.Context) {
		t1.context = c
		t1.stream = t.env.streamManager.NewMockStream(c.DebugLogger())
		action(t1)
	})

	t1.close()
}

// Debug logs some debug output for the test. The output will be passed to the test logger at
// the end of the test.
func (t *T) Debug(format string, args ...interface{}) {
	t.context.Debug(format, args...)
}

// RequireCapability skips this test if the test service did not declare that it supports the
// specified capability.
func (t *T) RequireCapability(capability string) {
	if !t.env.client.HasCapability(capability) {
		t.context.SkipWithReason(fmt.Sprintf("test service does not have capability %q", capability))
	}
}

// StartSSEClient tells the test service to start an SSE client with default options. All
// subsequent calls to methods like RequireEvent will refer to this client.
//
// This also causes the test to wait for the client to connect to the mock stream. It will fail
// and immediately exit the test if it times out while waiting. It returns information about
// the incoming connection.
func (t *T) StartSSEClient() *stream.IncomingConnection {
	return t.StartSSEClientOptions(client.CreateStreamOpts{})
}

// StartSSEClientOptions tells the test service to start an SSE client with the specified options.
// All subsequent calls to methods like RequireEvent will refer to this client.
//
// This also causes the test to wait for the client to connect to the mock stream. It will fail
// and immediately exit the test if it times out while waiting. It returns information about
// the incoming connection.
func (t *T) StartSSEClientOptions(opts client.CreateStreamOpts) *stream.IncomingConnection {
	opts.StreamURL = t.stream.URL
	opts.Tag = t.context.ID().String()
	sseClient, err := t.env.client.CreateEntity(opts, t.context.DebugLogger())
	require.NoError(t, err)
	t.sseClient = sseClient

	m, err := sseClient.AwaitMessage()
	require.NoError(t, err)
	require.Equal(t, "hello", m.Kind, `test service did not send the expected "hello" message`)

	return t.AwaitNewConnectionToStream()
}

// AwaitNewConnectionToStream waits for the SSE client to connect to the mock stream. It will fail
// and immediately exit the test if it times out while waiting. It returns information about
// the incoming connection.
//
// Tests only need to call this method if they expect another connection after the first one.
func (t *T) AwaitNewConnectionToStream() *stream.IncomingConnection {
	cxn, err := t.stream.AwaitConnection()
	require.NoError(t, err)

	return cxn
}

// BreakStreamConnection causes the mock stream to disconnect from the SSE client.
func (t *T) BreakStreamConnection() {
	t.stream.Interrupt()
}

// SendOnStream tells the mock stream to send a piece of data.
func (t *T) SendOnStream(data string) {
	t.stream.SendChunk(data)
}

// SendOnStreamInChunks tells the mock stream to send some data broken into chunks of the specified number
// of bytes, with an optional delay between chunks.
func (t *T) SendOnStreamInChunks(data string, chunkSize int, delayBetween time.Duration) {
	t.stream.SendSplit(data, chunkSize, delayBetween)
}

func (t *T) requireSSEClientStarted() {
	require.NotNil(t, t.sseClient, "test tried to communicate with the SSE client before starting one")
}

// RequireMessage waits for the SSE client in the test service to send us some kind of information.
//
// The test fails and immediately exits if it times out without receiving anything.
func (t *T) RequireMessage() client.ReceivedMessage {
	t.requireSSEClientStarted()
	m, err := t.sseClient.AwaitMessage()
	require.NoError(t, err)
	return m
}

// RequireEvent waits for the SSE client in the test service to tell us that it received an event.
//
// The test fails and immediately exits if it times out without receiving anything, or if what we
// receive from the test service us is not an event.
func (t *T) RequireEvent() client.EventMessage {
	m := t.RequireMessage()
	if m.Kind != "event" {
		require.Fail(t, "expected an event but got: %s", m.Kind)
	}
	return *m.Event
}

// RequireEvent waits for the SSE client in the test service to tell us that it received an error.
//
// The test fails and immediately exits if it times out without receiving anything, or if what we
// receive from the test service us is not an error.
func (t *T) RequireError() string {
	m := t.RequireMessage()
	if m.Kind != "error" {
		require.Fail(t, "expected an error but got: %s", m.Kind)
	}
	return m.Error
}

// RequireSpecificEvents waits for the SSE client in the test service to tell us that it received
// a series of events, which must match the specified events.
//
// Since some SSE implementations do not properly set the default event type to "message", this
// method uses a loose comparison where event types of "message" and "" are equal. We can check
// the default event type behavior in a more specific test, so lack of compliance on that point
// won't cause all sorts of other tests to fail.
func (t *T) RequireSpecificEvents(events ...client.EventMessage) {
	for _, expected := range events {
		if expected.Type == "" {
			expected.Type = "message"
		}
		actual := t.RequireEvent()
		if actual.Type == "" {
			actual.Type = "message"
		}
		assert.Equal(t, expected, actual)
	}
}

// RequireEvent waits for the SSE client in the test service to tell us that it received a comment.
//
// The test fails and immediately exits if it times out without receiving anything, or if what we
// receive from the test service us is not a comment.
func (t *T) RequireComment() string {
	t.requireSSEClientStarted()
	m := t.RequireMessage()
	if m.Kind != "comment" {
		require.Fail(t, "expected a comment but got: %s", m.Kind)
	}
	return m.Comment
}

// RestartClient tells the SSE client in the test service to immediately disconnect and retry.
// Not all SSE implementations support this.
func (t *T) RestartClient() {
	t.requireSSEClientStarted()
	require.NoError(t, t.sseClient.SendCommand("restart"))
}
