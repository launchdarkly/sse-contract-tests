package ssetests

import (
	"fmt"
	"time"

	"github.com/launchdarkly/sse-contract-tests/framework"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const awaitConnectionTimeout = time.Second * 5
const awaitMessageTimeout = time.Second * 5

var AllCapabilities = []string{
	"comments",
	"headers",
	"last-event-id",
	"post",
	"read-timeout",
	"retry",
	"report",
}

type CreateStreamOpts struct {
	InitialDelayMS ldvalue.OptionalInt `json:"initialDelayMs,omitempty"`
	LastEventID    string              `json:"lastEventId,omitempty"`
	Method         string              `json:"method,omitempty"`
	Body           string              `json:"body,omitempty"`
	Headers        map[string]string   `json:"headers,omitempty"`
	ReadTimeoutMS  ldvalue.OptionalInt `json:"readTimeoutMs,omitempty"`
}

type createSSEClientOpts struct {
	CreateStreamOpts
	CallbackURL string `json:"callbackUrl"`
	StreamURL   string `json:"streamUrl"`
	Tag         string `json:"tag"`
}

// T represents a test or subtest in our SSE test suite.
//
// It implements the same basic functionality as Go's testing.T, but in an environment that is outside
// of the Go test runner, and with some extra features such as debug logging that are convenient for
// our use case. Those features are provided by our lower-level framework package.
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
	context          *framework.Context
	harness          *framework.TestHarness
	stream           *mockStream
	callbackReceiver *callbackReceiver
	sseClientEntity  *framework.TestServiceEntity
}

func newTestScope(context *framework.Context, harness *framework.TestHarness) *T {
	t := &T{
		context: context,
		harness: harness,
	}
	t.stream = newMockStream(t.harness, context.DebugLogger())
	t.callbackReceiver = newCallbackReceiver(t.harness, context.DebugLogger())
	return t
}

func (t *T) close() {
	if t.sseClientEntity != nil {
		t.sseClientEntity.Close()
	}
	if t.callbackReceiver != nil {
		t.callbackReceiver.Close()
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
	var t1 *T
	t.context.Run(name, func(c *framework.Context) {
		t1 = newTestScope(c, t.harness)
		action(t1)
	})
	if t1 != nil {
		t1.close()
	}
}

// Debug logs some debug output for the test. The output will be passed to the test logger at
// the end of the test.
func (t *T) Debug(format string, args ...interface{}) {
	t.context.Debug(format, args...)
}

// RequireCapability skips this test if the test service did not declare that it supports the
// specified capability.
func (t *T) RequireCapability(capability string) {
	if !t.harness.TestServiceHasCapability(capability) {
		t.context.SkipWithReason(fmt.Sprintf("test service does not have capability %q", capability))
	}
}

// StartSSEClient tells the test service to start an SSE client with default options. All
// subsequent calls to methods like RequireEvent will refer to this client.
//
// This also causes the test to wait for the client to connect to the mock stream. It will fail
// and immediately exit the test if it times out while waiting. It returns information about
// the incoming connection.
func (t *T) StartSSEClient() framework.IncomingRequestInfo {
	return t.StartSSEClientOptions(CreateStreamOpts{})
}

// StartSSEClientOptions tells the test service to start an SSE client with the specified options.
// All subsequent calls to methods like RequireEvent will refer to this client.
//
// This also causes the test to wait for the client to connect to the mock stream. It will fail
// and immediately exit the test if it times out while waiting. It returns information about
// the incoming connection.
func (t *T) StartSSEClientOptions(opts CreateStreamOpts) framework.IncomingRequestInfo {
	clientOpts := createSSEClientOpts{
		CreateStreamOpts: opts,
		StreamURL:        t.stream.endpoint.BaseURL(),
		CallbackURL:      t.callbackReceiver.endpoint.BaseURL(),
		Tag:              t.context.ID().String(),
	}
	sseClient, err := t.harness.NewTestServiceEntity(clientOpts, "SSE client", t.context.DebugLogger())
	require.NoError(t, err)
	t.sseClientEntity = sseClient

	return t.AwaitNewConnectionToStream()
}

// AwaitNewConnectionToStream waits for the SSE client to connect to the mock stream. It will fail
// and immediately exit the test if it times out while waiting. It returns information about
// the incoming connection.
//
// Tests only need to call this method if they expect another connection after the first one.
func (t *T) AwaitNewConnectionToStream() framework.IncomingRequestInfo {
	cxn, err := t.stream.endpoint.AwaitConnection(awaitConnectionTimeout)
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
	require.NotNil(t, t.sseClientEntity, "test tried to communicate with the SSE client before starting one")
}

// RequireMessage waits for the SSE client in the test service to send us some kind of information.
//
// The test fails and immediately exits if it times out without receiving anything.
func (t *T) RequireMessage() ReceivedMessage {
	t.requireSSEClientStarted()
	m, err := t.callbackReceiver.AwaitMessage(awaitMessageTimeout)
	require.NoError(t, err)
	return m
}

func (t *T) requireMessageOfKind(kind string) ReceivedMessage {
	m := t.RequireMessage()
	if m.Kind != kind {
		require.Fail(t, "received an unexpected message", "expected %q but got: %s", kind, m)
	}
	return m
}

// RequireEvent waits for the SSE client in the test service to tell us that it received an event.
//
// The test fails and immediately exits if it times out without receiving anything, or if what we
// receive from the test service us is not an event.
func (t *T) RequireEvent() EventMessage {
	return *(t.requireMessageOfKind("event").Event)
}

// RequireError waits for the SSE client in the test service to tell us that it received an error.
//
// The test fails and immediately exits if it times out without receiving anything, or if what we
// receive from the test service us is not an error.
func (t *T) RequireError() string {
	return t.requireMessageOfKind("error").Error
}

// RequireSpecificEvents waits for the SSE client in the test service to tell us that it received
// a series of events, which must match the specified events.
//
// Since some SSE implementations do not properly set the default event type to "message", this
// method uses a loose comparison where event types of "message" and "" are equal. We can check
// the default event type behavior in a more specific test, so lack of compliance on that point
// won't cause all sorts of other tests to fail.
func (t *T) RequireSpecificEvents(events ...EventMessage) {
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
	return t.requireMessageOfKind("comment").Comment
}

// RestartClient tells the SSE client in the test service to immediately disconnect and retry.
// Not all SSE implementations support this.
func (t *T) RestartClient() {
	t.requireSSEClientStarted()
	require.NoError(t, t.sseClientEntity.SendCommand("restart"))
}

// TellClientToExpectEventType tells the SSE client in the test service that it should be ready to
// receive an event with the specified type. This is only necessary for SSE implementations that
// require you to explicitly listen for each event type.
func (t *T) TellClientToExpectEventType(eventType string) {
	if !t.harness.TestServiceHasCapability("event-type-listeners") {
		// If the test service doesn't advertise this capability, then it is able to receive
		// events of any type without specifically listening for them.
		return
	}
	t.requireSSEClientStarted()
	require.NoError(t, t.sseClientEntity.SendCommand("listen",
		map[string]interface{}{"type": eventType}))
}
