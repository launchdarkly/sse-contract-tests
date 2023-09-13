package ssetests

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/launchdarkly/sse-contract-tests/framework"
	"github.com/launchdarkly/sse-contract-tests/framework/harness"
	"github.com/launchdarkly/sse-contract-tests/framework/ldtest"
	"github.com/launchdarkly/sse-contract-tests/servicedef"
)

const awaitConnectionTimeout = time.Second * 5

type StreamServer struct {
	endpoint *harness.MockEndpoint
	logger   framework.Logger
}

type StreamConnection struct {
	RequestInfo harness.IncomingRequestInfo
	sendCh      chan<- streamChunk
	logger      framework.Logger
}

type streamContextKeyType string

const streamContextKey streamContextKeyType = "ssetests.streamContext"

type streamContext struct {
	dataCh chan streamChunk
}

type streamChunk struct {
	data       []byte
	delayAfter time.Duration
}

func NewStreamServer(t *ldtest.T) *StreamServer {
	endpoint := requireContext(t).harness.NewMockEndpoint(
		streamHandler(t.DebugLogger()),
		addStreamContext,
		t.DebugLogger(),
	)
	t.Defer(func() {
		endpoint.Close()
	})
	return &StreamServer{endpoint: endpoint, logger: t.DebugLogger()}
}

func (s *StreamServer) ApplyConfiguration(params *servicedef.CreateStreamParams) {
	params.StreamURL = s.endpoint.BaseURL()
}

func (s *StreamServer) AwaitConnection(t *ldtest.T) *StreamConnection {
	sc, err := s.AwaitConnectionWithTimeout(t, awaitConnectionTimeout)
	if err != nil {
		t.Errorf("error: %s", err.Error())
		t.FailNow()
	}
	return sc
}

func (s *StreamServer) AwaitConnectionWithTimeout(t *ldtest.T, timeout time.Duration) (*StreamConnection, error) {
	requestInfo, err := s.endpoint.AwaitConnection(timeout)
	if err != nil {
		t.Errorf("error: %s", err.Error())
		t.FailNow()
	}
	dataCh := streamContextFromContext(requestInfo.Context).dataCh
	return &StreamConnection{
		RequestInfo: requestInfo,
		sendCh:      dataCh,
		logger:      s.logger,
	}, nil
}

func (sc *StreamConnection) Send(data string) {
	sc.sendCh <- streamChunk{data: []byte(data)}
}

func (sc *StreamConnection) SendInChunks(data string, chunkSize int, delayBetween time.Duration) {
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
		sc.sendCh <- chunk
	}
}

// BreakConnection closes the current connection.
func (sc *StreamConnection) BreakConnection() {
	sc.logger.Printf("Deliberately breaking stream connection")
	sc.sendCh <- streamChunk{data: nil}
}

func addStreamContext(c context.Context) context.Context {
	dataCh := make(chan streamChunk, 1000)
	sc := streamContext{dataCh: dataCh}
	return context.WithValue(c, streamContextKey, sc)
}

func streamContextFromContext(c context.Context) streamContext {
	if sc, ok := c.Value(streamContextKey).(streamContext); ok {
		return sc
	}
	panic("streamContext was not added to request context; this is a mistake in the program logic")
}

func streamHandler(logger framework.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		closeNotifyCh := r.Context().Done()

		sc, ok := r.Context().Value(streamContextKey).(streamContext)
		if !ok {
			panic("streamContext was not added to request context; this is a mistake in the program logic")
		}

		flusher := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		flusher.Flush()

	Loop:
		for {
			select {
			case chunk, ok := <-sc.dataCh:
				if !ok {
					break Loop
				}
				if chunk.data == nil { // indicates we want to break the connection
					break Loop
				}
				chunkStr := string(chunk.data)
				jsonStr, _ := json.Marshal(chunkStr)
				logger.Printf("<< sending: %s", jsonStr)
				_, _ = w.Write(chunk.data)
				flusher.Flush()
				if chunk.delayAfter > 0 {
					time.Sleep(chunk.delayAfter)
				}
			case <-closeNotifyCh:
				break Loop
			}
		}

	})
}
