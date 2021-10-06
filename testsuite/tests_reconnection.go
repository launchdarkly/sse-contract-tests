package testsuite

import (
	"github.com/launchdarkly/sse-contract-tests/client"
	"github.com/launchdarkly/sse-contract-tests/stream"

	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func DoReconnectionTests(t *TestContext) {
	t.Run("sends ID of last received event", func(t *TestContext) {
		t.WithStreamEndpoint(func(e *stream.Endpoint) {
			opts := client.CreateStreamOpts{
				InitialDelayMS: ldvalue.NewOptionalInt(0),
			}

			t.WithTestClientStreamOpts(e, opts, func(r *client.ResponseStream) {
				cxn1, err := e.AwaitConnection()
				require.NoError(t, err)

				assert.Empty(t, cxn1.Headers().Values("Last-Event-Id"))

				e.SendChunk("id: abc\ndata: Hello\n\n")

				t.RequireSpecificEvents(r,
					client.EventMessage{ID: "abc", Data: "Hello"})

				e.Interrupt()

				cxn2, err := e.AwaitConnection()
				require.NoError(t, err)

				assert.Equal(t, "abc", cxn2.Headers().Get("Last-Event-Id"))
			})
		})
	})

	t.Run("sends ID of last received event that had an ID if later events did not", func(t *TestContext) {
		t.WithStreamEndpoint(func(e *stream.Endpoint) {
			opts := client.CreateStreamOpts{
				InitialDelayMS: ldvalue.NewOptionalInt(0),
			}

			t.WithTestClientStreamOpts(e, opts, func(r *client.ResponseStream) {
				cxn1, err := e.AwaitConnection()
				require.NoError(t, err)

				assert.Empty(t, cxn1.Headers().Values("Last-Event-Id"))

				e.SendChunk("id: abc\ndata: Hello\n\n")
				e.SendChunk("data: World\n\n")

				e1 := t.RequireEvent(r)
				assert.Equal(t, "Hello", e1.Data)
				assert.Equal(t, "abc", e1.ID)

				e2 := t.RequireEvent(r)
				assert.Equal(t, "World", e2.Data)
				if e2.ID != "" {
					assert.Equal(t, "abc", e2.ID)
				}

				e.Interrupt()

				cxn2, err := e.AwaitConnection()
				require.NoError(t, err)

				assert.Equal(t, "abc", cxn2.Headers().Get("Last-Event-Id"))
			})
		})
	})

	t.Run("resends request body if any when reconnecting", func(t *TestContext) {
		t.RequireCapability("post")

		jsonBody := `{"hello": "world"}`

		t.WithStreamEndpoint(func(e *stream.Endpoint) {
			opts := client.CreateStreamOpts{
				Headers: map[string]string{
					"content-type": "application/json; charset=utf-8",
				},
				InitialDelayMS: ldvalue.NewOptionalInt(0),
				Method:         "POST",
				Body:           jsonBody,
			}

			t.WithTestClientStreamOpts(e, opts, func(r *client.ResponseStream) {
				cxn1, err := e.AwaitConnection()
				require.NoError(t, err)

				assert.Equal(t, "POST", cxn1.Method())
				assert.Equal(t, jsonBody, string(cxn1.Body()))

				e.Interrupt()

				cxn2, err := e.AwaitConnection()
				require.NoError(t, err)

				assert.Equal(t, "POST", cxn2.Method())
				assert.Equal(t, jsonBody, string(cxn2.Body()))
			})
		})
	})
}
