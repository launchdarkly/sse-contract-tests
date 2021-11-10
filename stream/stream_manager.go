package stream

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/launchdarkly/sse-contract-tests/logging"
)

const defaultAwaitConnectionTimeout = time.Second * 5
const streamPathPrefix = "/streams/"

type StreamManager struct {
	baseURL string
	streams map[string]*MockStream
	lastID  int
	lock    sync.Mutex
}

func NewStreamManager(host string, port int) *StreamManager {
	return &StreamManager{
		baseURL: fmt.Sprintf("http://%s:%d", host, port),
		streams: make(map[string]*MockStream),
	}
}

func (s *StreamManager) NewMockStream(logger logging.Logger) *MockStream {
	s.lastID++
	endpointID := strconv.Itoa(s.lastID)

	m := &MockStream{
		owner:  s,
		id:     endpointID,
		Errors: make(chan error, 10),
		logger: logger,
		cxnCh:  make(chan *IncomingConnection, 10),
		URL:    strings.TrimSuffix(s.baseURL, "/") + streamPathPrefix + endpointID,
	}
	s.lock.Lock()
	s.streams[endpointID] = m
	s.lock.Unlock()

	return m
}

func (s *StreamManager) forgetStream(id string) {
	s.lock.Lock()
	delete(s.streams, id)
	s.lock.Unlock()
}

func (s *StreamManager) HandleRequest(w http.ResponseWriter, req *http.Request) bool {
	if !strings.HasPrefix(req.URL.Path, streamPathPrefix) {
		return false
	}
	if req.Method == "HEAD" {
		w.WriteHeader(http.StatusOK)
		return true
	}
	endpointID := strings.TrimPrefix(req.URL.Path, streamPathPrefix)
	if m := s.streams[endpointID]; m != nil {
		m.serveHTTP(w, req)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
	return true
}
