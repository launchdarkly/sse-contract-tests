package stream

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

// Endpoint is a simulation of an SSE server that is instrumented for tests. Each test in the
// test suite will construct one of these and then tell the test service to connect to it. The
// server only allows one active connection and will signal an error if there is ever more than
// one at a time (since SSE clients should not attempt to reconnect unless the connection has
// dropped).
//
// Endpoint does not have any of the usual pub/sub or event formatting logic that would exist
// in a real SSE server. The test suites will simply push chunks of raw data through it to the
// client.
type Endpoint struct {
	owner     *Server
	id        string
	URL       string
	Errors    chan error
	logger    *log.Logger
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
func (e *Endpoint) AwaitConnection() (*IncomingConnection, error) {
	deadline := time.NewTimer(defaultAwaitConnectionTimeout)
	defer deadline.Stop()
	select {
	case cxn := <-e.cxnCh:
		e.activeCxn = cxn
		return cxn, nil
	case err := <-e.Errors:
		return nil, err
	case <-deadline.C:
		return nil, errors.New("timed out waiting for test service to make a stream connection")
	}
}

// SendChunk sends a chunk of data on the stream and flushes the stream.
func (e *Endpoint) SendChunk(data string) {
	e.SendChunkThenWait(data, 0)
}

// SendChunkThenWait sends a chunk of data, flushes it, and then sleeps for an interval.
func (e *Endpoint) SendChunkThenWait(data string, delay time.Duration) {
	if e.activeCxn == nil {
		return
	}
	e.activeCxn.data <- chunk{data: []byte(data), delayAfter: delay}
}

// SendSplit breaks a string into multiple chunks of the specified byte length, and then sends
// and flushes each, with an optional delay in between.
func (e *Endpoint) SendSplit(data string, chunkSize int, delayBetween time.Duration) {
	if e.activeCxn == nil {
		return
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
		e.activeCxn.data <- chunk
	}
}

// Close permanently removes the endpoint.
func (e *Endpoint) Close() {
	delete(e.owner.endpoints, e.id)
	e.Interrupt()
}

// Interrupt closes the current connection.
func (e *Endpoint) Interrupt() {
	if e.activeCxn != nil {
		close(e.activeCxn.data)
		e.activeCxn = nil
	}
}

func (e *Endpoint) serveHTTP(w http.ResponseWriter, req *http.Request) {
	e.lock.Lock()
	if e.active {
		e.lock.Unlock()
		e.Errors <- errors.New("unexpectedly received a connection while the previous connection was still open")
		return
	}
	e.active = true
	e.lock.Unlock()

	closeNotifyCh := req.Context().Done()

	var body []byte
	if req.Body != nil {
		data, err := ioutil.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			e.Errors <- fmt.Errorf("error trying to read request body: %s", err)
			return
		}
		body = data
	}
	dataCh := make(chan chunk)
	cxn := &IncomingConnection{
		headers: req.Header,
		data:    dataCh,
		method:  req.Method,
		body:    body,
	}
	e.cxnCh <- cxn

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
			e.logger.Printf("sending: %s", jsonStr)
			_, err := w.Write(chunk.data)
			if err != nil {
				e.Errors <- err
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

	e.lock.Lock()
	e.active = false
	e.lock.Unlock()
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
