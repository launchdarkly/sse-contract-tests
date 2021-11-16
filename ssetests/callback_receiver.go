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
)

type callbackReceiver struct {
	endpoint       *framework.MockEndpoint
	output         chan entityOutput
	sortedMessages *framework.MessageSortingQueue
	logger         framework.Logger
}

type entityOutput struct {
	message ReceivedMessage
	err     error
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

func newCallbackReceiver(harness *framework.TestHarness, logger framework.Logger) *callbackReceiver {
	c := &callbackReceiver{
		output:         make(chan entityOutput, 100),
		sortedMessages: framework.NewMessageSortingQueue(100),
		logger:         logger,
	}
	c.endpoint = harness.NewMockEndpoint(c, nil, logger)
	go c.consumeMessages()
	return c
}

func (c *callbackReceiver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		c.sendError(errors.New("got callback request with no body"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		c.sendError(fmt.Errorf("error reading callback request body: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if r.URL.Path != "" || r.URL.Path == "/" {
		counter, err := strconv.Atoi(r.URL.Path[1:])
		if err == nil {
			c.sortedMessages.Accept(counter, data)
			w.WriteHeader(http.StatusAccepted)
			return
		}
	}
	c.sendError(fmt.Errorf("callback request had invalid path %q", r.URL.Path))
	w.WriteHeader(http.StatusBadRequest)
}

func (c *callbackReceiver) consumeMessages() {
	for data := range c.sortedMessages.C {
		message := ReceivedMessage{raw: string(data)}
		if err := json.Unmarshal(data, &message); err != nil {
			c.sendError(fmt.Errorf("malformed JSON data from test service: %s", message.raw))
			continue
		}
		c.logger.Printf("Received: %s", string(data))
		c.output <- entityOutput{message: message}
	}
}

func (c *callbackReceiver) sendError(err error) {
	c.logger.Printf("Error: %s", err)
	c.output <- entityOutput{err: err}
}

func (c *callbackReceiver) Close() {
	c.endpoint.Close()
	c.sortedMessages.Close()
}

// AwaitMessage waits until the test service sends a message.
func (c *callbackReceiver) AwaitMessage(timeout time.Duration) (ReceivedMessage, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	select {
	case item, ok := <-c.output:
		if !ok {
			return ReceivedMessage{}, errors.New("callback endpoint was already closed")
		}
		return item.message, nil
	case <-deadline.C:
		return ReceivedMessage{}, errors.New("timed out waiting for message from test service entity")
	}
}
