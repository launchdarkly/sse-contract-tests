package ssetests

import (
	"time"

	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"

	"github.com/stretchr/testify/assert"
)

func DoReconnectionTests(t *T) {
	t.Run("caller can trigger a restart", func(t *T) {
		t.RequireCapability("restart")

		opts := CreateStreamOpts{
			InitialDelayMS: ldvalue.NewOptionalInt(0),
		}
		t.StartSSEClientOptions(opts)

		t.SendOnStream("data: Hello\n\n")
		t.RequireSpecificEvents(EventMessage{Data: "Hello"})

		t.RestartClient()

		t.AwaitNewConnectionToStream()

		t.SendOnStream("data: Thanks\n\n")
		t.RequireSpecificEvents(EventMessage{Data: "Thanks"})
	})

	t.Run("sends ID of last received event", func(t *T) {
		opts := CreateStreamOpts{
			InitialDelayMS: ldvalue.NewOptionalInt(0),
		}
		cxn1 := t.StartSSEClientOptions(opts)

		assert.Empty(t, cxn1.Headers.Values("Last-Event-Id"))

		t.SendOnStream("id: abc\ndata: Hello\n\n")

		t.RequireSpecificEvents(EventMessage{ID: "abc", Data: "Hello"})

		t.BreakStreamConnection()

		cxn2 := t.AwaitNewConnectionToStream()

		assert.Equal(t, "abc", cxn2.Headers.Get("Last-Event-Id"), "reconnection request did not send expected Last-Event-Id")
	})

	t.Run("sends ID of last received event that had an ID if later events did not", func(t *T) {
		opts := CreateStreamOpts{
			InitialDelayMS: ldvalue.NewOptionalInt(0),
		}
		cxn1 := t.StartSSEClientOptions(opts)

		assert.Empty(t, cxn1.Headers.Values("Last-Event-Id"))

		t.SendOnStream("id: abc\ndata: Hello\n\n")
		t.SendOnStream("data: World\n\n")

		e1 := t.RequireEvent()
		assert.Equal(t, "Hello", e1.Data)
		assert.Equal(t, "abc", e1.ID)

		e2 := t.RequireEvent()
		assert.Equal(t, "World", e2.Data)

		t.BreakStreamConnection()

		cxn2 := t.AwaitNewConnectionToStream()

		assert.Equal(t, "abc", cxn2.Headers.Get("Last-Event-Id"), "reconnection request did not send expected Last-Event-Id")
	})

	t.Run("last event ID can be explicitly overridden with an empty value", func(t *T) {
		opts := CreateStreamOpts{
			InitialDelayMS: ldvalue.NewOptionalInt(0),
		}
		cxn1 := t.StartSSEClientOptions(opts)

		assert.Empty(t, cxn1.Headers.Values("Last-Event-Id"))

		t.SendOnStream("id: abc\ndata: Hello\n\n")
		t.SendOnStream("id: \ndata: World\n\n")

		e1 := t.RequireEvent()
		assert.Equal(t, "Hello", e1.Data)
		assert.Equal(t, "abc", e1.ID)

		e2 := t.RequireEvent()
		assert.Equal(t, "World", e2.Data)
		assert.Equal(t, "", e2.ID)

		t.BreakStreamConnection()

		cxn2 := t.AwaitNewConnectionToStream()

		_, ok := cxn2.Headers["Last-Event-Id"]
		assert.False(t, ok,
			"reconnection request should not have sent a Last-Event-Id header, but did (value was %q)",
			cxn2.Headers.Get("Last-Event-Id"))
	})

	t.Run("resends request body if any when reconnecting", func(t *T) {
		t.RequireCapability("post")

		jsonBody := `{"hello": "world"}`

		opts := CreateStreamOpts{
			Headers: map[string]string{
				"content-type": "application/json; charset=utf-8",
			},
			InitialDelayMS: ldvalue.NewOptionalInt(0),
			Method:         "POST",
			Body:           jsonBody,
		}
		cxn1 := t.StartSSEClientOptions(opts)

		assert.Equal(t, "POST", cxn1.Method)
		assert.Equal(t, jsonBody, string(cxn1.Body))

		t.BreakStreamConnection()

		cxn2 := t.AwaitNewConnectionToStream()

		assert.Equal(t, "POST", cxn2.Method)
		assert.Equal(t, jsonBody, string(cxn2.Body))
	})

	t.Run("can set read timeout", func(t *T) {
		t.RequireCapability("read-timeout")

		opts := CreateStreamOpts{
			InitialDelayMS: ldvalue.NewOptionalInt(0),
			ReadTimeoutMS:  ldvalue.NewOptionalInt(500),
		}
		t.StartSSEClientOptions(opts)

		t.SendOnStream("data: Hello\n\n")
		time.Sleep(time.Second)

		t.RequireSpecificEvents(EventMessage{Data: "Hello"})

		t.RequireError()

		t.AwaitNewConnectionToStream()
	})

	t.Run("discards partial messages on retry", func(t *T) {
		opts := CreateStreamOpts{
			InitialDelayMS: ldvalue.NewOptionalInt(0),
		}
		_ = t.StartSSEClientOptions(opts)

		t.SendOnStream("id: abc\ndata: Hello\n\nid: def\ndata: Goodbye")
		t.RequireSpecificEvents(EventMessage{ID: "abc", Data: "Hello"})

		t.BreakStreamConnection()

		t.AwaitNewConnectionToStream()
		t.SendOnStream("data: We meet again\n\n")

		e := t.RequireEvent()
		assert.Equal(t, "We meet again", e.Data)
		assert.NotEqual(t, "def", e.ID)
		// The correct ID value here should be "abc", but we're not checking for that here because if the SSE
		// client has a bug making it not correctly retain the last ID from a previous event, we already have
		// a more specific test for that; we don't want it to cause a misleading failure in this test. We
		// just want to prove that it did *not* pick up the "def" from the partial event.
	})
}
