package ssetests

import (
	"fmt"
	"net/http"
	"time"

	"github.com/launchdarkly/sse-contract-tests/framework/ldtest"
	"github.com/launchdarkly/sse-contract-tests/servicedef"

	"github.com/launchdarkly/go-test-helpers/v2/httphelpers"

	"github.com/stretchr/testify/assert"
)

func DoHTTPBehaviorTests(t *ldtest.T) {
	t.Run("default method and headers", func(t *ldtest.T) {
		_, stream, _ := NewStreamAndSSEClient(t)
		assert.Equal(t, "GET", stream.RequestInfo.Method, "incorrect request method")
		assert.Equal(t, "text/event-stream", stream.RequestInfo.Headers.Get("Accept"), "missing or incorrect Accept header")
		assert.Equal(t, "no-cache", stream.RequestInfo.Headers.Get("Cache-Control"),
			"missing or incorrect Cache-Control header")
		assert.Empty(t, stream.RequestInfo.Headers.Values("Last-Event-Id"),
			"Last-Event-Id header should not have had a value")
	})

	if t.Capabilities().Has("204") {
		t.Run("204 halts re-connection attempts", func(t *ldtest.T) {
			h := httphelpers.HandlerWithStatus(204)
			rh, requestsCh := httphelpers.RecordingHandler(h)

			endpointReturning204 := requireContext(t).harness.NewMockEndpoint(rh, nil, t.DebugLogger())
			t.Defer(endpointReturning204.Close)

			_ = NewSSEClient(t, WithClientParams(servicedef.CreateStreamParams{
				StreamURL: endpointReturning204.BaseURL(),
			}))

			// Give time for the client to reconnect if it is going to try
			time.Sleep(time.Second)

			assert.Equal(t, 1, len(requestsCh))
		})
	}

	for _, status := range []int{301, 307} {
		t.Run(fmt.Sprintf("client follows %d redirect", status), func(t *ldtest.T) {
			server := NewStreamServer(t)

			headers := make(http.Header)
			headers.Set("Location", server.endpoint.BaseURL())
			handler := httphelpers.HandlerWithResponse(status, headers, nil)
			endpointReturningRedirect := requireContext(t).harness.NewMockEndpoint(handler, nil, t.DebugLogger())
			t.Defer(endpointReturningRedirect.Close)

			client := NewSSEClient(t, WithClientParams(servicedef.CreateStreamParams{
				StreamURL: endpointReturningRedirect.BaseURL(),
			}))

			stream := server.AwaitConnection(t)
			stream.Send("data: hello\n\n")
			client.RequireSpecificEvents(t, EventMessage{Data: "hello"})
		})

		// The intention of these tests are to ensure the client does not, when presented
		// with an empty or missing Location header:
		// 1) Loop infinitely
		// 2) Keep using the current URL without emitting an error
		for _, action := range []string{"empty", "missing"} {
			t.Run(fmt.Sprintf("client handles %s Location header with %d status", action, status), func(t *ldtest.T) {
				headers := make(http.Header)
				if action == "empty" {
					headers.Set("Location", "")
				}
				handler := httphelpers.HandlerWithResponse(status, headers, nil)
				endpointReturningRedirect := requireContext(t).harness.NewMockEndpoint(handler, nil, t.DebugLogger())
				t.Defer(endpointReturningRedirect.Close)

				client := NewSSEClient(t, WithClientParams(servicedef.CreateStreamParams{
					StreamURL: endpointReturningRedirect.BaseURL(),
				}))

				client.RequireError(t)
			})
		}
	}

	t.Run("custom headers", func(t *ldtest.T) {
		t.RequireCapability("headers")

		params := servicedef.CreateStreamParams{
			Headers: map[string]string{
				"header-name-1": "value-1",
				"header-name-2": "value-2",
			},
		}
		_, stream, _ := NewStreamAndSSEClient(t, WithClientParams(params))

		assert.Equal(t, "value-1", stream.RequestInfo.Headers.Get("header-name-1"),
			"missing or incorrect custom header 'header-name-1'")
		assert.Equal(t, "value-2", stream.RequestInfo.Headers.Get("header-name-2"),
			"missing or incorrect custom header 'header-name-1'")
	})

	doRequestWithBody := func(method, capability string) func(*ldtest.T) {
		return func(t *ldtest.T) {
			t.RequireCapability(capability)

			jsonBody := `{"hello": "world"}`

			params := servicedef.CreateStreamParams{
				Headers: map[string]string{
					"content-type": "application/json; charset=utf-8",
				},
				Method: method,
				Body:   jsonBody,
			}
			_, stream, _ := NewStreamAndSSEClient(t, WithClientParams(params))

			assert.Equal(t, method, stream.RequestInfo.Method, "incorrect request method")
			assert.Equal(t, "application/json; charset=utf-8", stream.RequestInfo.Headers.Get("content-type"),
				"incorrect Content-Type header")
			assert.Equal(t, jsonBody, string(stream.RequestInfo.Body), "missing or incorrect request body")
		}
	}

	t.Run("POST request", doRequestWithBody("POST", "post"))

	t.Run("REPORT request", doRequestWithBody("REPORT", "report"))

	t.Run("sends Last-Event-Id in initial request if set", func(t *ldtest.T) {
		t.RequireCapability("last-event-id")

		params := servicedef.CreateStreamParams{
			LastEventID: "abc",
		}
		_, stream, _ := NewStreamAndSSEClient(t, WithClientParams(params))

		assert.Equal(t, "abc", stream.RequestInfo.Headers.Get("Last-Event-Id"), "missing or incorrect Last-Event-Id header")
	})
}
