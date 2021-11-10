package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/launchdarkly/sse-contract-tests/logging"

	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
)

const callbackPathPrefix = "/callbacks/"

// TestServiceClient manages communication with the test service. This includes both the ability to send
// commands to the test service, and an HTTP listener for receiving callback requests from the test
// service.
type TestServiceClient struct {
	testServiceBaseURL string
	callbackBaseURL    string
	logger             logging.Logger
	capabilities       []string
	activeEntities     map[string]*TestServiceEntity
	lastID             int
	lock               sync.Mutex
}

// CreateStreamOpts contains options for SSETestClient.CreateStream.
type CreateStreamOpts struct {
	StreamURL      string              `json:"streamUrl"`
	Tag            string              `json:"tag"`
	InitialDelayMS ldvalue.OptionalInt `json:"initialDelayMs,omitempty"`
	LastEventID    string              `json:"lastEventId,omitempty"`
	Method         string              `json:"method,omitempty"`
	Body           string              `json:"body,omitempty"`
	Headers        map[string]string   `json:"headers,omitempty"`
	ReadTimeoutMS  ldvalue.OptionalInt `json:"readTimeoutMs,omitempty"`
}

type createStreamRequestParams struct {
	CreateStreamOpts
	CallbackURL string `json:"callbackUrl"`
}

type clientStatusResponse struct {
	Capabilities []string `json:"capabilities"`
}

var allCapabilities = []string{
	"comments",
	"headers",
	"last-event-id",
	"post",
	"read-timeout",
	"report",
}

// NewTestServiceClient creates a TestServiceClient instance, and verifies that the test service
// is responding by querying its status resource. It also starts an HTTP listener on the specified
// port to receive callback requests.
func NewSSETestClient(
	testServiceBaseURL string,
	listenerPort int,
	externalHostname string,
	timeout time.Duration,
	logger *log.Logger,
) (*TestServiceClient, error) {
	deadline := time.Now().Add(timeout)
	c := &TestServiceClient{
		testServiceBaseURL: testServiceBaseURL,
		callbackBaseURL:    fmt.Sprintf("http://%s:%d", externalHostname, listenerPort),
		logger:             logger,
		activeEntities:     make(map[string]*TestServiceEntity),
	}
WaitLoop:
	for {
		logger.Printf("Making request to %s", testServiceBaseURL)
		resp, err := http.DefaultClient.Get(testServiceBaseURL)
		if err == nil && resp.StatusCode == 200 {
			logger.Printf("Got 200 status from %s", testServiceBaseURL)
			if resp.Body != nil {
				respData, err := ioutil.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					return nil, err
				}
				logger.Printf("Metadata: %s", string(respData))
				var statusResp clientStatusResponse
				if err := json.Unmarshal(respData, &statusResp); err != nil {
					return nil, fmt.Errorf("malformed status response from test service: %s", string(respData))
				}
				c.capabilities = statusResp.Capabilities
			}
			break WaitLoop
		}
		if !time.Now().Before(deadline) {
			if err == nil {
				err = fmt.Errorf("status code %d", resp.StatusCode)
			}
			return nil, fmt.Errorf("result of last query was: %s", err)
		}
		time.Sleep(time.Millisecond * 20)
	}

	return c, nil
}

// Capabilities returns the list of capabilities, if any, provided by the test service's
// status resource.
func (c *TestServiceClient) Capabilities() []string {
	return append([]string(nil), c.capabilities...)
}

func (c *TestServiceClient) HasCapability(desired string) bool {
	for _, capability := range c.Capabilities() {
		if capability == desired {
			return true
		}
	}
	return false
}

func (c *TestServiceClient) MissingCapabilities() []string {
	var ret []string
	for _, capability := range allCapabilities {
		if !c.HasCapability(capability) {
			ret = append(ret, capability)
		}
	}
	return ret
}

// CreateEntity tells the test service to create a new instance of the kind of entity it
// manages (in this case an SSE stream client), returning a TestServiceEntity to use for
// communicating about that instance.
func (c *TestServiceClient) CreateEntity(opts CreateStreamOpts, logger logging.Logger) (*TestServiceEntity, error) {
	if logger == nil {
		logger = c.logger
	}

	c.lock.Lock()
	c.lastID++
	entityID := fmt.Sprintf("%d", c.lastID)
	callbackURL := c.callbackBaseURL + callbackPathPrefix + entityID
	entity := newTestServiceEntity(c, entityID, logger)
	c.activeEntities[entityID] = entity
	c.lock.Unlock()

	success := false
	defer func() {
		if !success {
			_ = entity.Close()
		}
	}()

	params := createStreamRequestParams{CreateStreamOpts: opts}
	params.CallbackURL = callbackURL

	data, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	body := bytes.NewBuffer(data)

	logger.Printf("Creating test SSE client")
	req, err := http.NewRequest("POST", c.testServiceBaseURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		var message string
		if resp.Body != nil {
			data, _ = ioutil.ReadAll(resp.Body)
			message = ": " + string(data)
			resp.Body.Close()
		}
		return nil, fmt.Errorf("unexpected response status %d from test service%s", resp.StatusCode, message)
	}
	resourceURL := resp.Header.Get("Location")
	if resourceURL == "" {
		return nil, errors.New("test service did not return a Location header with a resource URL")
	}
	if !strings.HasPrefix(resourceURL, "http:") {
		resourceURL = c.testServiceBaseURL + resourceURL
	}
	entity.setResourceURL(resourceURL)

	success = true
	logger.Printf("Test SSE client created")
	return entity, nil
}

func (c *TestServiceClient) forgetEntity(id string) {
	c.lock.Lock()
	delete(c.activeEntities, id)
	c.lock.Unlock()
}

func (c *TestServiceClient) HandleRequest(w http.ResponseWriter, r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, callbackPathPrefix) {
		return false
	}
	entityID := strings.TrimPrefix(r.URL.Path, callbackPathPrefix)
	c.lock.Lock()
	entity := c.activeEntities[entityID]
	c.lock.Unlock()
	if entity == nil {
		w.WriteHeader(http.StatusNotFound)
	} else {
		entity.handleRequest(w, r)
	}
	return true
}
