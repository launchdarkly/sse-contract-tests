package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"time"
)

// ResponseStream provides one-way communication from the test service back to the test harness
// after starting an SSE client with TestClient.CreateStream. Individual tests in the test suite
// should use methods such as AwaitEvent to verify that the test service is providing the
// expected output.
type ResponseStream struct {
	reader    io.Reader
	logger    *log.Logger
	canceller context.CancelFunc
	timeout   time.Duration
	items     chan streamItem
}

type streamItem struct {
	message Message
	err     error
}

// Message is a single message provided by the test service.
type Message struct {
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

func (m Message) String() string { return m.raw }

// EventMessage contains the fields of an SSE event, exactly as it was received from the
// test service's SSE client.
type EventMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
	ID   string `json:"id"`
}

func newResponseStream(reader io.Reader, logger *log.Logger, canceller context.CancelFunc, timeout time.Duration) *ResponseStream {
	r := &ResponseStream{
		reader:    reader,
		logger:    logger,
		canceller: canceller,
		timeout:   timeout,
		items:     make(chan streamItem, 100),
	}
	go r.readStream()
	return r
}

// Close closes the connection, which should cause the test service to stop its SSE client.
func (r *ResponseStream) Close() {
	r.canceller()
}

func (r *ResponseStream) sendError(err error) {
	r.logger.Printf("Error: %s", err)
	r.items <- streamItem{err: err}
}

func (r *ResponseStream) readStream() {
	defer close(r.items)
	buf := bytes.NewBuffer(nil)
	for {
		var chunk [1000]byte
		n, err := r.reader.Read(chunk[:])
		if err != nil {
			if err == context.Canceled {
				return
			}
			if err != io.EOF {
				err = fmt.Errorf("I/O error reading test service stream: %w", err)
			}
			r.sendError(err)
			return
		}
		buf.Write(chunk[0:n])
		for {
			messageStr, err := buf.ReadString('\n')
			if err != nil {
				buf.Reset()
				buf.WriteString(messageStr)
				break
			}
			message := Message{raw: messageStr}
			r.logger.Printf("Received: %s", messageStr)
			if err := json.Unmarshal([]byte(messageStr), &message); err != nil {
				r.sendError(fmt.Errorf("malformed JSON data from test service: %s", messageStr))
				return
			}
			r.items <- streamItem{message: message}
		}
	}
}

// AwaitMessage waits until the test service sends a message of the specified kind. It
// returns an error if it times out or if the message is of a different kind.
func (r *ResponseStream) AwaitMessage(kind string) (Message, error) {
	deadline := time.NewTimer(r.timeout)
	defer deadline.Stop()
	select {
	case item, ok := <-r.items:
		if !ok {
			return Message{}, errors.New("connection was closed by test service")
		}
		if item.err != nil {
			return Message{}, item.err
		}
		if item.message.Kind != kind {
			return Message{}, fmt.Errorf("expected message of kind %q but got: %s", kind, item.message)
		}
		return item.message, nil
	case <-deadline.C:
		return Message{}, errors.New("timed out waiting for message from test service")
	}
}

// AwaitEvent is equivalent to AwaitMessage but requires the message to be an SSE event.
func (r *ResponseStream) AwaitEvent() (EventMessage, error) {
	m, err := r.AwaitMessage("event")
	if err != nil {
		return EventMessage{}, err
	}
	if m.Event == nil {
		return EventMessage{}, errors.New(`received message with kind "event" but no event`)
	}
	return *m.Event, nil
}

// AwaitEvent is equivalent to AwaitMessage but requires the message to be an SSE comment.
func (r *ResponseStream) AwaitComment() (string, error) {
	m, err := r.AwaitMessage("comment")
	if err != nil {
		return "", err
	}
	return m.Comment, nil
}
