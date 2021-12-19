package ssetests

import (
	"github.com/launchdarkly/sse-contract-tests/framework/ldtest"
	"github.com/launchdarkly/sse-contract-tests/servicedef"

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
