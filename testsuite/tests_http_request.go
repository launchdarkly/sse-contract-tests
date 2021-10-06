package testsuite

import (
	"github.com/launchdarkly/sse-contract-tests/client"
	"github.com/launchdarkly/sse-contract-tests/stream"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func DoHTTPRequestTests(t *TestContext) {
	t.Run("default method and headers", func(t *TestContext) {
		t.WithStreamEndpoint(func(e *stream.Endpoint) {
			t.WithTestClientStream(e, func(r *client.ResponseStream) {
				cxn, err := e.AwaitConnection()
				require.NoError(t, err)

				assert.Equal(t, "GET", cxn.Method())
				assert.Equal(t, "text/event-stream", cxn.Headers().Get("Accept"))
				assert.Equal(t, "no-cache", cxn.Headers().Get("Cache-Control"))
				assert.Empty(t, cxn.Headers().Values("Last-Event-Id"))
			})
		})
	})

	t.Run("custom headers", func(t *TestContext) {
		t.RequireCapability("headers")

		t.WithStreamEndpoint(func(e *stream.Endpoint) {
			opts := client.CreateStreamOpts{
				Headers: map[string]string{
					"header-name-1": "value-1",
					"header-name-2": "value-2",
				},
			}
			t.WithTestClientStreamOpts(e, opts, func(r *client.ResponseStream) {
				cxn, err := e.AwaitConnection()
				require.NoError(t, err)

				assert.Equal(t, "value-1", cxn.Headers().Get("header-name-1"))
				assert.Equal(t, "value-2", cxn.Headers().Get("header-name-2"))
			})
		})
	})

	doRequestWithBody := func(method, capability string) func(*TestContext) {
		return func(t *TestContext) {
			t.RequireCapability(capability)

			jsonBody := `{"hello": "world"}`

			t.WithStreamEndpoint(func(e *stream.Endpoint) {
				opts := client.CreateStreamOpts{
					Headers: map[string]string{
						"content-type": "application/json; charset=utf-8",
					},
					Method: method,
					Body:   jsonBody,
				}
				t.WithTestClientStreamOpts(e, opts, func(r *client.ResponseStream) {
					cxn, err := e.AwaitConnection()
					require.NoError(t, err)

					assert.Equal(t, method, cxn.Method())
					assert.Equal(t, "application/json; charset=utf-8", cxn.Headers().Get("content-type"))
					assert.Equal(t, jsonBody, string(cxn.Body()))
				})
			})
		}
	}

	t.Run("POST request", doRequestWithBody("POST", "post"))

	t.Run("REPORT request", doRequestWithBody("REPORT", "report"))

	t.Run("sends Last-Event-Id in initial request if set", func(t *TestContext) {
		t.RequireCapability("last-event-id")
		t.WithStreamEndpoint(func(e *stream.Endpoint) {
			opts := client.CreateStreamOpts{
				LastEventID: "abc",
			}
			t.WithTestClientStreamOpts(e, opts, func(r *client.ResponseStream) {
				cxn, err := e.AwaitConnection()
				require.NoError(t, err)

				assert.Equal(t, "abc", cxn.Headers().Get("Last-Event-Id"))
			})
		})
	})
}
