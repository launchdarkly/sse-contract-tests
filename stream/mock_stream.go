package stream

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/launchdarkly/sse-contract-tests/logging"
)

// MockStream is a simulation of an SSE server that is instrumented for tests. Each test in the
// test suite will construct one of these and then tell the test service to connect to it. The
// server only allows one active connection and will signal an error if there is ever more than
// one at a time (since SSE clients should not attempt to reconnect unless the connection has
// dropped).
//
// Endpoint does not have any of the usual pub/sub or event formatting logic that would exist
// in a real SSE server. The test suites will simply push chunks of raw data through it to the
// client.
type MockStream struct {
	owner     *StreamManager
	id        string
	URL       string
	Errors    chan error
	logger    logging.Logger
	active    bool
	cxnCh     chan *IncomingConnection
	activeCxn *IncomingConnection
	lock      sync.Mutex
}

// IncomingConnection contains information about the HTTP request sent by the test service.
type IncomingConnection struct {
	headers http.Header
	method  string
	body    []byte
	data    chan<- chunk
}

type chunk struct {
	data       []byte
	delayAfter time.Duration
}

// AwaitConnection waits until the test service has connected to the server.
func (m *MockStream) AwaitConnection() (*IncomingConnection, error) {
	deadline := time.NewTimer(defaultAwaitConnectionTimeout)
	defer deadline.Stop()
	select {
	case cxn := <-m.cxnCh:
		m.activeCxn = cxn
		return cxn, nil
	case err := <-m.Errors:
		return nil, err
	case <-deadline.C:
		return nil, errors.New("timed out waiting for test service to make a stream connection")
	}
}

// SendChunk sends a chunk of data on the stream and flushes the stream.
func (m *MockStream) SendChunk(data string) {
	m.SendChunkThenWait(data, 0)
}

// SendChunkThenWait sends a chunk of data, flushes it, and then sleeps for an interval.
func (m *MockStream) SendChunkThenWait(data string, delay time.Duration) {
	if m.activeCxn == nil {
		panic("tried to send data on the stream before we got a connection")
	}
	m.activeCxn.data <- chunk{data: []byte(data), delayAfter: delay}
}

// SendSplit breaks a string into multiple chunks of the specified byte length, and then sends
// and flushes each, with an optional delay in between.
func (m *MockStream) SendSplit(data string, chunkSize int, delayBetween time.Duration) {
	if m.activeCxn == nil {
		panic("tried to send data on the stream before we got a connection")
	}
	bytes := []byte(data)
	for pos := 0; pos < len(bytes); pos += chunkSize {
		max := pos + chunkSize
		if max > len(bytes) {
			max = len(bytes)
		}
		chunk := chunk{data: bytes[pos:max]}
		if max < len(bytes) {
			chunk.delayAfter = delayBetween
		}
		m.activeCxn.data <- chunk
	}
}

// Close permanently removes the endpoint.
func (m *MockStream) Close() {
	m.owner.forgetStream(m.id)
	m.Interrupt()
}

// Interrupt closes the current connection.
func (m *MockStream) Interrupt() {
	if m.activeCxn != nil {
		m.logger.Printf("Deliberately breaking stream connection")
		close(m.activeCxn.data)
		m.activeCxn = nil
	}
}

func (m *MockStream) serveHTTP(w http.ResponseWriter, req *http.Request) {
	m.logger.Printf("Got connection from SSE client; headers follow")
	for k, v := range req.Header {
		m.logger.Printf("  %s: %s", k, strings.Join(v, ", "))
	}

	m.lock.Lock()
	if m.active {
		m.lock.Unlock()
		m.Errors <- errors.New("unexpectedly received a connection while the previous connection was still open")
		return
	}
	m.active = true
	m.lock.Unlock()

	closeNotifyCh := req.Context().Done()

	var body []byte
	if req.Body != nil {
		data, err := ioutil.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			m.Errors <- fmt.Errorf("error trying to read request body: %s", err)
			return
		}
		body = data
	}
	dataCh := make(chan chunk, 100)
	cxn := &IncomingConnection{
		headers: req.Header,
		data:    dataCh,
		method:  req.Method,
		body:    body,
	}
	m.cxnCh <- cxn

	flusher := w.(http.Flusher)
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	flusher.Flush()

Loop:
	for {
		select {
		case chunk, ok := <-dataCh:
			if !ok {
				break Loop
			}
			chunkStr := string(chunk.data)
			jsonStr, _ := json.Marshal(chunkStr)
			m.logger.Printf("<< sending: %s", jsonStr)
			_, err := w.Write(chunk.data)
			if err != nil {
				m.Errors <- err
				break Loop
			}
			flusher.Flush()
			if chunk.delayAfter > 0 {
				time.Sleep(chunk.delayAfter)
			}
		case <-closeNotifyCh:
			break Loop
		}
	}

	m.lock.Lock()
	m.active = false
	m.lock.Unlock()
}

func (c *IncomingConnection) Headers() http.Header {
	return c.headers
}

func (c *IncomingConnection) Method() string {
	return c.method
}

func (c *IncomingConnection) Body() []byte {
	return append([]byte(nil), c.body...)
}
