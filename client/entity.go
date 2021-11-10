package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/launchdarkly/sse-contract-tests/framework"
)

const defaultAwaitTimeout = time.Second * 5

// TestServiceEntity represents the entity within the test service that was created by calling
// TestServiceClient.CreateEntity-- that is, an active SSE client within the test service.
// It provides two-way communication between the test harness and the test service with regard
// to this specific instance. Individual tests in the test suite should use methods such as
// AwaitEvent to verify that the test service is providing the expected output.
type TestServiceEntity struct {
	owner   *TestServiceClient
	id      string
	url     string
	logger  framework.Logger
	timeout time.Duration
	output  chan entityOutput
	lock    sync.Mutex
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

type commandRequestParams struct {
	Command string `json:"command"`
}

func newTestServiceEntity(owner *TestServiceClient, id string, logger framework.Logger) *TestServiceEntity {
	return &TestServiceEntity{
		owner:   owner,
		id:      id,
		logger:  logger,
		timeout: defaultAwaitTimeout,
		output:  make(chan entityOutput, 1000),
	}
}

func (e *TestServiceEntity) setResourceURL(url string) {
	e.lock.Lock()
	e.url = url
	e.lock.Unlock()
}

func (e *TestServiceEntity) getResourceURL() string {
	e.lock.Lock()
	defer e.lock.Unlock()
	return e.url
}

func (e *TestServiceEntity) sendError(err error) {
	e.logger.Printf("Error: %s", err)
	e.output <- entityOutput{err: err}
}

func (e *TestServiceEntity) Close() error {
	e.owner.forgetEntity(e.id)

	url := e.getResourceURL()
	if url == "" {
		return nil
	}

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.Body != nil {
		_ = resp.Body.Close()
	}
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return fmt.Errorf("DELETE request to test service returned HTTP status %d", resp.StatusCode)
	}

	return nil
}

func (e *TestServiceEntity) handleRequest(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		e.sendError(errors.New("got callback request with no body"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		e.sendError(fmt.Errorf("error reading callback request body: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	message := ReceivedMessage{raw: string(data)}
	e.logger.Printf("Received: %s", string(data))
	if err := json.Unmarshal(data, &message); err != nil {
		e.sendError(fmt.Errorf("malformed JSON data from test service: %s", message.raw))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	e.output <- entityOutput{message: message}
	w.WriteHeader(http.StatusAccepted)
}

// AwaitMessage waits until the test service sends a message.
func (e *TestServiceEntity) AwaitMessage() (ReceivedMessage, error) {
	deadline := time.NewTimer(e.timeout)
	defer deadline.Stop()
	select {
	case item, ok := <-e.output:
		if !ok {
			return ReceivedMessage{}, errors.New("callback endpoint was already closed")
		}
		return item.message, nil
	case <-deadline.C:
		return ReceivedMessage{}, errors.New("timed out waiting for message from test service entity")
	}
}

// SendCommand sends a command to the test service entity.
func (e *TestServiceEntity) SendCommand(command string) error {
	data, _ := json.Marshal(commandRequestParams{Command: command})
	resp, err := http.DefaultClient.Post(e.getResourceURL(), "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	if resp.Body != nil {
		resp.Body.Close()
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("command returned HTTP status %d", resp.StatusCode)
	}
	return nil
}
