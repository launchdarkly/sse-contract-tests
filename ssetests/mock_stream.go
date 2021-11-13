package ssetests

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/launchdarkly/sse-contract-tests/framework"
)

// mockStream is a mock SSE service attached to one of the test harness's mock endpoints.
// It gives the test logic a way to inject stream data that will be served up by the endpoint.
// It is not a multiplexing SSE server-- any data injected by the test logic will only go to
// the most recent client connection.
type mockStream struct {
	endpoint *framework.MockEndpoint
	logger   framework.Logger
	dataCh   chan streamChunk
	lock     sync.Mutex
}

type streamChunk struct {
	data       []byte
	delayAfter time.Duration
}

func newMockStream(
	harness *framework.TestHarness,
	logger framework.Logger,
) *mockStream {
	s := &mockStream{
		dataCh: make(chan streamChunk, 1000),
		logger: logger,
	}
	streamLogger := framework.LoggerWithPrefix(logger, "[mock stream] ")
	s.endpoint = harness.NewMockEndpoint(s, streamLogger)
	return s
}

func (s *mockStream) Close() {
	close(s.dataCh)
	s.endpoint.Close()
}

// SendChunk sends a chunk of data on the stream and flushes the stream.
func (s *mockStream) SendChunk(data string) {
	s.send(streamChunk{data: []byte(data)})
}

// SendChunkThenWait sends a chunk of data, flushes it, and then sleeps for an interval.
func (s *mockStream) SendChunkThenWait(data string, delay time.Duration) {
	s.send(streamChunk{data: []byte(data), delayAfter: delay})
}

// SendSplit breaks a string into multiple chunks of the specified byte length, and then sends
// and flushes each, with an optional delay in between.
func (s *mockStream) SendSplit(data string, chunkSize int, delayBetween time.Duration) {
	bytes := []byte(data)
	for pos := 0; pos < len(bytes); pos += chunkSize {
		max := pos + chunkSize
		if max > len(bytes) {
			max = len(bytes)
		}
		chunk := streamChunk{data: bytes[pos:max]}
		if max < len(bytes) {
			chunk.delayAfter = delayBetween
		}
		s.send(chunk)
	}
}

func (s *mockStream) send(chunk streamChunk) {
	// Grab the channel under a lock because if a new connection arrives, we will replace
	// the channel with a new channel. That ensures that the test data is only ever going
	// to one connection at a time.
	s.lock.Lock()
	ch := s.dataCh
	s.lock.Unlock()
	ch <- chunk
}

// Interrupt closes the current connection.
func (s *mockStream) Interrupt() {
	s.logger.Printf("Deliberately breaking stream connection")
	s.send(streamChunk{data: nil})
}

func (s *mockStream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	closeNotifyCh := r.Context().Done()

	flusher := w.(http.Flusher)
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	flusher.Flush()

	s.lock.Lock() // see comment on send()
	ch := s.dataCh
	s.lock.Unlock()

Loop:
	for {
		select {
		case chunk, ok := <-ch:
			if !ok {
				break Loop
			}
			if chunk.data == nil { // indicates we want to break the connection
				break Loop
			}
			chunkStr := string(chunk.data)
			jsonStr, _ := json.Marshal(chunkStr)
			s.logger.Printf("<< sending: %s", jsonStr)
			_, _ = w.Write(chunk.data)
			flusher.Flush()
			if chunk.delayAfter > 0 {
				time.Sleep(chunk.delayAfter)
			}
		case <-closeNotifyCh:
			break Loop
		}
	}
}