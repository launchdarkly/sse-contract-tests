package ssetests

import (
	"time"

	"github.com/launchdarkly/sse-contract-tests/framework/ldtest"
	"github.com/launchdarkly/sse-contract-tests/servicedef"

	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"

	"github.com/stretchr/testify/assert"
)

func DoReconnectionTests(t *ldtest.T) {
	t.Run("caller can trigger a restart", func(t *ldtest.T) {
		t.RequireCapability("restart")

		params := servicedef.CreateStreamParams{
			InitialDelayMS: ldvalue.NewOptionalInt(0),
		}
		server, stream1, client := NewStreamAndSSEClient(t, WithClientParams(params))

		stream1.Send("data: Hello\n\n")
		client.RequireSpecificEvents(t, EventMessage{Data: "Hello"})

		client.Restart(t)

		stream2 := server.AwaitConnection(t)

		stream2.Send("data: Thanks\n\n")
		client.RequireSpecificEvents(t, EventMessage{Data: "Thanks"})
	})

	t.Run("sends ID of last received event", func(t *ldtest.T) {
		params := servicedef.CreateStreamParams{
			InitialDelayMS: ldvalue.NewOptionalInt(0),
		}
		server, stream1, client := NewStreamAndSSEClient(t, WithClientParams(params))

		assert.Empty(t, stream1.RequestInfo.Headers.Values("Last-Event-Id"))

		stream1.Send("id: abc\ndata: Hello\n\n")

		client.RequireSpecificEvents(t, EventMessage{ID: "abc", Data: "Hello"})

		stream1.BreakConnection()

		stream2 := server.AwaitConnection(t)

		assert.Equal(t, "abc", stream2.RequestInfo.Headers.Get("Last-Event-Id"),
			"reconnection request did not send expected Last-Event-Id")
	})

	t.Run("sends ID of last received event that had an ID if later events did not", func(t *ldtest.T) {
		params := servicedef.CreateStreamParams{
			InitialDelayMS: ldvalue.NewOptionalInt(0),
		}
		server, stream1, client := NewStreamAndSSEClient(t, WithClientParams(params))

		assert.Empty(t, stream1.RequestInfo.Headers.Values("Last-Event-Id"))

		stream1.Send("id: abc\ndata: Hello\n\n")
		stream1.Send("data: World\n\n")

		e1 := client.RequireEvent(t)
		assert.Equal(t, "Hello", e1.Data)
		assert.Equal(t, "abc", e1.ID)

		e2 := client.RequireEvent(t)
		assert.Equal(t, "World", e2.Data)
		if e2.ID != "" {
			assert.Equal(t, "abc", e2.ID)
		}

		stream1.BreakConnection()

		stream2 := server.AwaitConnection(t)

		assert.Equal(t, "abc", stream2.RequestInfo.Headers.Get("Last-Event-Id"),
			"reconnection request did not send expected Last-Event-Id")
	})

	t.Run("resends request body if any when reconnecting", func(t *ldtest.T) {
		t.RequireCapability("post")

		jsonBody := `{"hello": "world"}`

		params := servicedef.CreateStreamParams{
			Headers: map[string]string{
				"content-type": "application/json; charset=utf-8",
			},
			InitialDelayMS: ldvalue.NewOptionalInt(0),
			Method:         "POST",
			Body:           jsonBody,
		}
		server, stream1, _ := NewStreamAndSSEClient(t, WithClientParams(params))

		assert.Equal(t, "POST", stream1.RequestInfo.Method)
		assert.Equal(t, jsonBody, string(stream1.RequestInfo.Body))

		stream1.BreakConnection()

		stream2 := server.AwaitConnection(t)

		assert.Equal(t, "POST", stream2.RequestInfo.Method)
		assert.Equal(t, jsonBody, string(stream2.RequestInfo.Body))
	})

	t.Run("can set read timeout", func(t *ldtest.T) {
		t.RequireCapability("read-timeout")

		params := servicedef.CreateStreamParams{
			InitialDelayMS: ldvalue.NewOptionalInt(0),
			ReadTimeoutMS:  ldvalue.NewOptionalInt(500),
		}
		server, stream, client := NewStreamAndSSEClient(t, WithClientParams(params))

		stream.Send("data: Hello\n\n")
		time.Sleep(time.Second)

		client.RequireSpecificEvents(t, EventMessage{Data: "Hello"})

		client.RequireError(t)

		server.AwaitConnection(t)
	})
}
