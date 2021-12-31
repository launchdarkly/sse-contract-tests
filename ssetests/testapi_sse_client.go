package ssetests

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/launchdarkly/sse-contract-tests/framework"
	"github.com/launchdarkly/sse-contract-tests/framework/harness"
	"github.com/launchdarkly/sse-contract-tests/framework/ldtest"
	"github.com/launchdarkly/sse-contract-tests/servicedef"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const awaitMessageTimeout = time.Second * 5

type SSEClient struct {
	service         *harness.TestServiceEntity
	outputCh        chan messageOrError
	callbackQueue   *harness.MessageSortingQueue
	ignoreNextError bool
	logger          framework.Logger
}

type SSEClientConfigurer interface {
	ApplyConfiguration(*servicedef.CreateStreamParams)
}

// ReceivedMessage is a single message sent to us by the test service.
type ReceivedMessage struct {
	// Kind is "event", "comment", or "error".
	Kind string `json:"kind"`

	// Event is non-nil if Kind is "event". It contains an SSE event that was received by the
	// test service's SSE client.
	Event *EventMessage `json:"event,omitempty"`

	// Comment contains an SSE comment that was received by the test service's SSE client,
	// if Kind is "comment". Not all SSE implementations are able to return comments.
	Comment string `json:"comment,omitempty"`

	// Error contains an error message from the test service, if Kind is "error".
	Error string `json:"error,omitempty"`

	raw string // The original JSON, for debug logging
}

func (m ReceivedMessage) String() string { return m.raw }

// EventMessage contains the fields of an SSE event, exactly as it was received from the
// test service's SSE client.
type EventMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
	ID   string `json:"id"`
}

func (e EventMessage) String() string {
	data, _ := json.Marshal(e)
	return string(data)
}

type messageOrError struct {
	message ReceivedMessage
	err     error
}

type clientParamsConfigurer servicedef.CreateStreamParams

func NewSSEClient(t *ldtest.T, configurers ...SSEClientConfigurer) *SSEClient {
	testHarness := requireContext(t).harness

	params := servicedef.CreateStreamParams{}
	for _, conf := range configurers {
		conf.ApplyConfiguration(&params)
	}
	if params.StreamURL == "" {
		require.Fail(t, "StreamURL was not set in stream parameters; did you forget to reference the StreamServer?")
	}
	params.Tag = t.ID().String()
	c := &SSEClient{
		outputCh:      make(chan messageOrError, 100),
		callbackQueue: harness.NewMessageSortingQueue(100),
		logger:        t.DebugLogger(),
	}
	t.Defer(c.callbackQueue.Close)

	callbackEndpoint := testHarness.NewMockEndpoint(http.HandlerFunc(c.handleCallback), nil, t.DebugLogger())
	t.Defer(callbackEndpoint.Close)

	params.CallbackURL = callbackEndpoint.BaseURL()

	service, err := testHarness.NewTestServiceEntity(params, "SSE client", t.DebugLogger())
	require.NoError(t, err)
	t.Defer(func() {
		_ = service.Close()
	})
	c.service = service

	go c.consumeCallbacks()

	return c
}

func (c *SSEClient) handleCallback(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		c.outputError(errors.New("got callback request with no body"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer func() { _ = req.Body.Close() }()
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		c.outputError(fmt.Errorf("error reading callback request body: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.URL.Path != "" || req.URL.Path == "/" {
		counter, err := strconv.Atoi(req.URL.Path[1:])
		if err == nil {
			c.callbackQueue.Accept(counter, data)
			w.WriteHeader(http.StatusAccepted)
			return
		}
	}
	c.outputError(fmt.Errorf("callback request had invalid path %q", req.URL.Path))
	w.WriteHeader(http.StatusBadRequest)
}

// AwaitMessage waits until the test service sends a message.
func (c *SSEClient) AwaitMessage(timeout time.Duration) (ReceivedMessage, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		select {
		case item, ok := <-c.outputCh:
			if !ok {
				return ReceivedMessage{}, errors.New("callback endpoint was already closed")
			}
			if c.ignoreNextError {
				c.ignoreNextError = false
				if item.message.Kind == "error" {
					continue
				}
			}
			return item.message, nil
		case <-deadline.C:
			return ReceivedMessage{}, errors.New("timed out waiting for message from test service entity")
		}
	}
}

// RequireMessage waits for the SSE client in the test service to send us some kind of information.
//
// The test fails and immediately exits if it times out without receiving anything.
func (c *SSEClient) RequireMessage(t *ldtest.T) ReceivedMessage {
	m, err := c.AwaitMessage(awaitMessageTimeout)
	require.NoError(t, err)
	return m
}

func (c *SSEClient) requireMessageOfKind(t *ldtest.T, kind string) ReceivedMessage {
	m := c.RequireMessage(t)
	if m.Kind != kind {
		require.Fail(t, "received an unexpected message", "expected %q but got: %s", kind, m)
	}
	return m
}

// RequireEvent waits for the SSE client in the test service to tell us that it received an event.
//
// The test fails and immediately exits if it times out without receiving anything, or if what we
// receive from the test service us is not an event.
func (c *SSEClient) RequireEvent(t *ldtest.T) EventMessage {
	return *(c.requireMessageOfKind(t, "event").Event)
}

// RequireError waits for the SSE client in the test service to tell us that it received an error.
//
// The test fails and immediately exits if it times out without receiving anything, or if what we
// receive from the test service us is not an error.
func (c *SSEClient) RequireError(t *ldtest.T) string {
	return c.requireMessageOfKind(t, "error").Error
}

// IgnoreErrorHere specifies that the next message from the client should be ignored if and
// only if it is an error.
//
// Some SSE client implementations report an unexpected end of stream as an error; others do not.
// In tests that are not actually trying to produce an error, but are simply checking for proper
// reconnection behavior after a connection is dropped, you can use this method after dropping
// the connection to ensure that the rest of the test behaves correctly either way if you are
// trying to read events.
func (c *SSEClient) IgnoreErrorHere() {
	c.ignoreNextError = true
}

// RequireSpecificEvents waits for the SSE client in the test service to tell us that it received
// a series of events, which must match the specified events.
//
// Since some SSE implementations do not properly set the default event type to "message", this
// method uses a loose comparison where event types of "message" and "" are equal. We can check
// the default event type behavior in a more specific test, so lack of compliance on that point
// won't cause all sorts of other tests to fail.
func (c *SSEClient) RequireSpecificEvents(t *ldtest.T, events ...EventMessage) {
	for _, expected := range events {
		if expected.Type == "" {
			expected.Type = "message"
		}
		actual := c.RequireEvent(t)
		if actual.Type == "" {
			actual.Type = "message"
		}
		assert.Equal(t, expected, actual)
	}
}

// RequireComment waits for the SSE client in the test service to tell us that it received a comment.
//
// The test fails and immediately exits if it times out without receiving anything, or if what we
// receive from the test service us is not a comment.
func (c *SSEClient) RequireComment(t *ldtest.T) string {
	return c.requireMessageOfKind(t, "comment").Comment
}

// Restart tells the SSE client in the test service to immediately disconnect and retry.
// Not all SSE implementations support this.
func (c *SSEClient) Restart(t *ldtest.T) {
	require.NoError(t, c.service.SendCommand("restart", c.logger, nil))
}

// BePreparedToReceiveEventType tells the SSE client in the test service that it should be ready to
// receive an event with the specified type. This is only necessary for SSE implementations that
// require you to explicitly listen for each event type.
func (c *SSEClient) BePreparedToReceiveEventType(t *ldtest.T, eventType string) {
	if !t.Capabilities().Has("event-type-listeners") {
		// If the test service doesn't advertise this capability, then it is able to receive
		// events of any type without specifically listening for them.
		return
	}
	require.NoError(t, c.service.SendCommandWithParams(
		servicedef.CommandParams{
			Command: "listen",
			Listen:  &servicedef.ListenParams{Type: eventType},
		},
		c.logger,
		nil))
}

func (c *SSEClient) consumeCallbacks() {
	for data := range c.callbackQueue.C {
		message := ReceivedMessage{raw: string(data)}
		if err := json.Unmarshal(data, &message); err != nil {
			c.outputError(fmt.Errorf("malformed JSON data from test service: %s", message.raw))
			continue
		}
		c.logger.Printf("Received: %s", string(data))
		c.outputCh <- messageOrError{message: message}
	}
}

func (c *SSEClient) outputError(err error) {
	c.logger.Printf("Error: %s", err)
	c.outputCh <- messageOrError{err: err}
}

func (c clientParamsConfigurer) ApplyConfiguration(p *servicedef.CreateStreamParams) {
	*p = servicedef.CreateStreamParams(c)
}

func WithClientParams(params servicedef.CreateStreamParams) SSEClientConfigurer {
	return clientParamsConfigurer(params)
}
