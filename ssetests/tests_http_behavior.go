package ssetests

import (
	"github.com/stretchr/testify/assert"
)

func DoHTTPBehaviorTests(t *T) {
	t.Run("default method and headers", func(t *T) {
		cxn := t.StartSSEClient()
		assert.Equal(t, "GET", cxn.Method, "incorrect request method")
		assert.Equal(t, "text/event-stream", cxn.Headers.Get("Accept"), "missing or incorrect Accept header")
		assert.Equal(t, "no-cache", cxn.Headers.Get("Cache-Control"), "missing or incorrect Cache-Control header")
		assert.Empty(t, cxn.Headers.Values("Last-Event-Id"), "Last-Event-Id header should not have had a value")
	})

	t.Run("custom headers", func(t *T) {
		t.RequireCapability("headers")

		opts := CreateStreamOpts{
			Headers: map[string]string{
				"header-name-1": "value-1",
				"header-name-2": "value-2",
			},
		}
		cxn := t.StartSSEClientOptions(opts)

		assert.Equal(t, "value-1", cxn.Headers.Get("header-name-1"), "missing or incorrect custom header 'header-name-1'")
		assert.Equal(t, "value-2", cxn.Headers.Get("header-name-2"), "missing or incorrect custom header 'header-name-1'")
	})

	doRequestWithBody := func(method, capability string) func(*T) {
		return func(t *T) {
			t.RequireCapability(capability)

			jsonBody := `{"hello": "world"}`

			opts := CreateStreamOpts{
				Headers: map[string]string{
					"content-type": "application/json; charset=utf-8",
				},
				Method: method,
				Body:   jsonBody,
			}
			cxn := t.StartSSEClientOptions(opts)

			assert.Equal(t, method, cxn.Method, "incorrect request method")
			assert.Equal(t, "application/json; charset=utf-8", cxn.Headers.Get("content-type"), "incorrect Content-Type header")
			assert.Equal(t, jsonBody, string(cxn.Body), "missing or incorrect request body")
		}
	}

	t.Run("POST request", doRequestWithBody("POST", "post"))

	t.Run("REPORT request", doRequestWithBody("REPORT", "report"))

	t.Run("sends Last-Event-Id in initial request if set", func(t *T) {
		t.RequireCapability("last-event-id")

		opts := CreateStreamOpts{
			LastEventID: "abc",
		}
		cxn := t.StartSSEClientOptions(opts)
		assert.Equal(t, "abc", cxn.Headers.Get("Last-Event-Id"), "missing or incorrect Last-Event-Id header")
	})
}
