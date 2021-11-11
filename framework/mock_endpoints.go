package framework

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const defaultAwaitConnectionTimeout = time.Second * 5

// MockEndpoint represents an endpoint that can receive requests.
type MockEndpoint struct {
	owner       *TestHarness
	id          string
	description string
	basePath    string
	handler     http.Handler
	maxConns    int
	curConns    int
	newConns    chan IncomingRequestInfo
	cancels     []*context.CancelFunc
	logger      Logger
	lock        sync.Mutex
	closing     sync.Once
}

// IncomingRequestInfo contains information about an HTTP request sent by the test service
// to one of the mock endpoints.
type IncomingRequestInfo struct {
	Headers http.Header
	Method  string
	Body    []byte
	Context context.Context
}

// NewEndpoint adds a new endpoint that can receive requests.
//
// The specified handler will be called for all incoming requests to the endpoint's
// base URL or any subpath of it. For instance, if the generated base URL (as reported
// by MockEndpoint.BaseURL()) is http://localhost:8111/endpoints/3, then it can also
// receive requests to http://localhost:8111/endpoints/3/some/subpath.
//
// When the handler is called, the test harness rewrites the request URL first so that
// the handler sees only the subpath. It also attaches a Context to the request whose
// Done channel will be closed if Close is called on the endpoint.
func (h *TestHarness) NewMockEndpoint(
	handler http.Handler,
	maxConnections int,
	logger Logger,
) *MockEndpoint {
	if logger == nil {
		logger = h.logger
	}
	e := &MockEndpoint{
		owner:    h,
		handler:  handler,
		maxConns: maxConnections,
		newConns: make(chan IncomingRequestInfo, maxConnections),
		logger:   logger,
	}
	h.lock.Lock()
	h.lastEndpointID++
	e.id = strconv.Itoa(h.lastEndpointID)
	e.basePath = endpointPathPrefix + e.id
	h.endpoints[e.id] = e
	h.lock.Unlock()

	return e
}

// BaseURL returns the base path of the mock endpoint.
func (e *MockEndpoint) BaseURL() string {
	return e.owner.testHarnessExternalBaseURL + e.basePath
}

// AwaitConnection waits for an incoming request to the endpoint.
func (e *MockEndpoint) AwaitConnection(timeout time.Duration) (IncomingRequestInfo, error) {
	deadline := time.NewTimer(defaultAwaitConnectionTimeout)
	defer deadline.Stop()
	select {
	case cxn := <-e.newConns:
		return cxn, nil
	case <-deadline.C:
		return IncomingRequestInfo{}, fmt.Errorf("timed out waiting for an incoming request to %s", e.description)
	}
}

// Close unregisters the endpoint. Any subsequent requests to it will receive 404 errors.
// It also cancels the Context for every active request to that endpoint.
func (e *MockEndpoint) Close() {
	e.closing.Do(func() {
		e.owner.lock.Lock()
		delete(e.owner.endpoints, e.id)
		e.owner.lock.Unlock()

		e.lock.Lock()
		cancellers := e.cancels
		e.cancels = nil
		close(e.newConns)
		e.lock.Unlock()

		for _, cancel := range cancellers {
			(*cancel)()
		}
	})
}
