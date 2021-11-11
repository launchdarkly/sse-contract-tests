package ssetests

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/launchdarkly/sse-contract-tests/framework"
)

// mockStream is a mock SSE service attached to one of the test harness's mock endpoints.
// It only allows a single connection from the test service at a time, and gives the test
// logic a way to inject data into that connection.
type mockStream struct {
	endpoint *framework.MockEndpoint
	logger   framework.Logger
	dataCh   chan streamChunk
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
	s.endpoint = harness.NewMockEndpoint(s, 1, streamLogger)
	return s
}

func (s *mockStream) Close() {
	close(s.dataCh)
	s.endpoint.Close()
}

// SendChunk sends a chunk of data on the stream and flushes the stream.
func (s *mockStream) SendChunk(data string) {
	s.SendChunkThenWait(data, 0)
}

// SendChunkThenWait sends a chunk of data, flushes it, and then sleeps for an interval.
func (s *mockStream) SendChunkThenWait(data string, delay time.Duration) {
	s.dataCh <- streamChunk{data: []byte(data), delayAfter: delay}
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
		s.dataCh <- chunk
	}
}

// Interrupt closes the current connection.
func (s *mockStream) Interrupt() {
	s.logger.Printf("Deliberately breaking stream connection")
	s.dataCh <- streamChunk{data: nil}
}

func (s *mockStream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	closeNotifyCh := r.Context().Done()

	flusher := w.(http.Flusher)
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	flusher.Flush()

Loop:
	for {
		select {
		case chunk, ok := <-s.dataCh:
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
