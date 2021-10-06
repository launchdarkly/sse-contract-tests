package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
)

const defaultStreamTimeout = time.Second * 5

// SSETestClient manages REST requests to the test service.
type SSETestClient struct {
	url          string
	logger       *log.Logger
	capabilities []string
}

// CreateStreamOpts contains options for SSETestClient.CreateStream.
type CreateStreamOpts struct {
	URL            string              `json:"url"`
	Tag            string              `json:"tag"`
	InitialDelayMS ldvalue.OptionalInt `json:"initialDelayMs,omitempty"`
	LastEventID    string              `json:"lastEventId,omitempty"`
	Method         string              `json:"method,omitempty"`
	Body           string              `json:"body,omitempty"`
	Headers        map[string]string   `json:"headers,omitempty"`
	ReadTimeoutMS  ldvalue.OptionalInt `json:"readTimeoutMs,omitempty"`
}

type clientStatusResponse struct {
	Capabilities []string `json:"capabilities"`
}

var allCapabilities = []string{
	"comments",
	"cr-only",
	"headers",
	"last-event-id",
	"post",
	"read-timeout",
	"report",
}

// NewSSETestClient creates an SSETestClient instance, and verifies that the test service is
// responding by querying its status resource.
func NewSSETestClient(baseURL string, timeout time.Duration, logger *log.Logger) (*SSETestClient, error) {
	deadline := time.Now().Add(timeout)
	c := &SSETestClient{
		url:    baseURL,
		logger: logger,
	}
WaitLoop:
	for {
		logger.Printf("Making request to %s", baseURL)
		resp, err := http.DefaultClient.Get(baseURL)
		if err == nil && resp.StatusCode == 200 {
			logger.Printf("Got 200 status from %s", baseURL)
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
func (c *SSETestClient) Capabilities() []string {
	return append([]string(nil), c.capabilities...)
}

func (c *SSETestClient) HasCapability(desired string) bool {
	for _, capability := range c.Capabilities() {
		if capability == desired {
			return true
		}
	}
	return false
}

func (c *SSETestClient) MissingCapabilities() []string {
	var ret []string
	for _, capability := range allCapabilities {
		if !c.HasCapability(capability) {
			ret = append(ret, capability)
		}
	}
	return ret
}

// CreateStream tells the test service to start an SSE stream client, returning a ResponseStream
// that the test service will use to send status information back to the test harness.
func (c *SSETestClient) CreateStream(opts CreateStreamOpts) (*ResponseStream, error) {
	data, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	body := bytes.NewBuffer(data)
	ctx, canceller := context.WithCancel(context.Background())

	prefix := opts.Tag + " >> "
	subLogger := log.New(c.logger.Writer(), prefix, c.logger.Flags())

	subLogger.Println("Creating test client stream")
	req, err := http.NewRequestWithContext(ctx, "POST", c.url, body)
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
	subLogger.Println("Test client stream created")
	return newResponseStream(resp.Body, subLogger, canceller, defaultStreamTimeout), nil
}
