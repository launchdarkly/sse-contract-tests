package framework

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

const endpointPathPrefix = "/endpoints/"
const httpListenerTimeout = time.Second * 10

type TestHarness struct {
	testServiceBaseURL         string
	testHarnessExternalBaseURL string
	testServiceInfo            TestServiceInfo
	endpoints                  map[string]*MockEndpoint
	lastEndpointID             int
	logger                     Logger
	lock                       sync.Mutex
}

// NewTestServiceClient creates a TestServiceClient instance, and verifies that the test service
// is responding by querying its status resource. It also starts an HTTP listener on the specified
// port to receive callback requests.
func NewTestHarness(
	testServiceBaseURL string,
	testHarnessExternalHostname string,
	testHarnessPort int,
	statusQueryTimeout time.Duration,
	debugLogger Logger,
	startupOutput io.Writer,
) (*TestHarness, error) {
	if debugLogger == nil {
		debugLogger = NullLogger()
	}

	externalBaseUrl := fmt.Sprintf("http://%s:%d", testHarnessExternalHostname, testHarnessPort)

	h := &TestHarness{
		testServiceBaseURL:         testServiceBaseURL,
		testHarnessExternalBaseURL: externalBaseUrl,
		endpoints:                  make(map[string]*MockEndpoint),
		logger:                     debugLogger,
	}

	testServiceInfo, err := queryTestServiceInfo(testServiceBaseURL, statusQueryTimeout, startupOutput)
	if err != nil {
		return nil, err
	}
	h.testServiceInfo = testServiceInfo

	if err = startServer(testHarnessPort, http.HandlerFunc(h.serveHTTP)); err != nil {
		return nil, err
	}

	return h, nil
}

func (h *TestHarness) TestServiceInfo() TestServiceInfo {
	return h.testServiceInfo
}

func (h *TestHarness) TestServiceHasCapability(desired string) bool {
	for _, capability := range h.testServiceInfo.Capabilities {
		if capability == desired {
			return true
		}
	}
	return false
}

func (h *TestHarness) serveHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == "HEAD" {
		w.WriteHeader(200) // we use this to test whether our own listener is active yet
		return
	}

	if !strings.HasPrefix(req.URL.Path, endpointPathPrefix) {
		h.logger.Printf("Received request for unrecognized URL path %s", req.URL.Path)
		w.WriteHeader(404)
		return
	}
	path := strings.TrimPrefix(req.URL.Path, endpointPathPrefix)
	var endpointID string
	slashPos := strings.Index(path, "/")
	if slashPos >= 0 {
		endpointID = path[0:slashPos]
		path = path[slashPos:]
	} else {
		endpointID = path
		path = ""
	}

	h.lock.Lock()
	e := h.endpoints[endpointID]
	h.lock.Unlock()
	if e == nil {
		h.logger.Printf("Received request for unrecognized endpoint %s", req.URL.Path)
		w.WriteHeader(404)
		return
	}

	var body []byte
	if req.Body != nil {
		data, err := ioutil.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			h.logger.Printf("Unexpected error trying to read request body: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body = data
	}

	e.lock.Lock()
	ctx, canceller := context.WithCancel(req.Context())
	cancellerPtr := &canceller
	e.cancels = append(e.cancels, cancellerPtr)
	e.lock.Unlock()

	incoming := IncomingRequestInfo{
		Headers: req.Header,
		Method:  req.Method,
		Body:    body,
		Context: ctx,
	}
	select { // non-blocking push
	case e.newConns <- incoming:
		break
	default:
		h.logger.Printf("Incoming connection channel was full for %s", req.URL)
	}

	transformedReq := req.WithContext(ctx)
	url := *req.URL
	url.Path = path
	transformedReq.URL = &url
	if body != nil {
		transformedReq.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	}

	e.handler.ServeHTTP(w, transformedReq)

	e.lock.Lock()
	for i, c := range e.cancels {
		if c == cancellerPtr { // can't compare functions with ==, but can compare pointers
			e.cancels = append(e.cancels[:i], e.cancels[i+1:]...)
			break
		}
	}
	e.lock.Unlock()
}

func startServer(port int, handler http.Handler) error {
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "HEAD" {
				w.WriteHeader(200)
				return
			}
			handler.ServeHTTP(w, r)
		}),
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	// Wait till the server is definitely listening for requests before we run any tests
	deadline := time.NewTimer(httpListenerTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond * 10)
	defer ticker.Stop()
	for {
		select {
		case <-deadline.C:
			return fmt.Errorf("Could not detect own listener at %s", server.Addr)
		case <-ticker.C:
			resp, err := http.DefaultClient.Head(fmt.Sprintf("http://localhost:%d", port))
			if err == nil && resp.StatusCode == 200 {
				return nil
			}
		}
	}
}
