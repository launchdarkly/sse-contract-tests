package framework

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// TestServiceInfo is status information returned by the test service from the initial status query.
type TestServiceInfo struct {
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
}

// TestServiceEntity represents some kind of entity that we have asked the test service to create,
// which the test harness will interact with.
type TestServiceEntity struct {
	resourceURL string
	logger      Logger
}

type commandRequestParams struct {
	Command string `json:"command"`
}

func queryTestServiceInfo(url string, timeout time.Duration, output io.Writer) (TestServiceInfo, error) {
	fmt.Fprintf(output, "Connecting to test service at %s", url)

	deadline := time.Now().Add(timeout)
	for {
		fmt.Fprintf(output, ".")
		resp, err := http.DefaultClient.Get(url)
		if err == nil {
			fmt.Fprintln(output)
			if resp.StatusCode != 200 {
				return TestServiceInfo{}, fmt.Errorf("test service returned status code %d", resp.StatusCode)
			}
			if resp.Body == nil {
				fmt.Fprintf(output, "Status query successful, but service provided no metadata\n")
				return TestServiceInfo{}, nil
			}
			respData, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return TestServiceInfo{}, err
			}
			fmt.Fprintf(output, "Status query returned metadata: %s\n", string(respData))
			var info TestServiceInfo
			if err := json.Unmarshal(respData, &info); err != nil {
				return TestServiceInfo{}, fmt.Errorf("malformed status response from test service: %s", string(respData))
			}
			return info, nil
		}
		if !time.Now().Before(deadline) {
			return TestServiceInfo{}, fmt.Errorf("timed out, result of last query was: %w", err)
		}
		time.Sleep(time.Millisecond * 100)
	}
}

// StopService tells the test service that it should exit.
func (h *TestHarness) StopService() error {
	req, _ := http.NewRequest("DELETE", h.testServiceBaseURL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err == nil && resp.StatusCode >= 300 {
		return fmt.Errorf("service returned HTTP %d", resp.StatusCode)
	}
	// It's normal for the request to return an I/O error if the service immediately quit before sending a response
	return nil
}

// NewTestServiceEntity tells the test service to create a new instance of whatever kind of entity
// it manages, based on the parameters we provide. The test harness can interact with it via the
// returned TestServiceEntity. The entity is assumed to remain active inside the test service
// until we explicitly close it.
//
// The format of entityParams is defined by the test harness; this low-level method simply calls
// json.Marshal to convert whatever it is to JSON.
func (h *TestHarness) NewTestServiceEntity(
	entityParams interface{},
	description string,
	logger Logger,
) (*TestServiceEntity, error) {
	if logger == nil {
		logger = NullLogger()
	}

	data, err := json.Marshal(entityParams)
	if err != nil {
		return nil, err
	}
	body := bytes.NewBuffer(data)

	logger.Printf("Creating test service entity (%s) with parameters: %s", description, string(data))
	req, err := http.NewRequest("POST", h.testServiceBaseURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
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
		resourceURL = h.testServiceBaseURL + resourceURL
	}

	e := &TestServiceEntity{
		resourceURL: resourceURL,
		logger:      logger,
	}

	return e, nil
}

// Close tells the test service to dispose of this entity.
func (e *TestServiceEntity) Close() error {
	req, err := http.NewRequest("DELETE", e.resourceURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.Body != nil {
		_ = resp.Body.Close()
	}
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return fmt.Errorf("DELETE request to test service returned HTTP status %d", resp.StatusCode)
	}
	return nil
}

// SendCommand sends a command to the test service entity.
func (e *TestServiceEntity) SendCommand(command string, additionalParams ...map[string]interface{}) error {
	allParams := map[string]interface{}{"command": command}
	for _, p := range additionalParams {
		for k, v := range p {
			allParams[k] = v
		}
	}
	data, _ := json.Marshal(allParams)
	e.logger.Printf("Sending command: %s", string(data))
	resp, err := http.DefaultClient.Post(e.resourceURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	if resp.Body != nil {
		resp.Body.Close()
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("command returned HTTP status %d", resp.StatusCode)
	}
	return nil
}
