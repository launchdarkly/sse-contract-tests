package testsuite

import (
	"time"

	"github.com/launchdarkly/sse-contract-tests/client"
	"github.com/launchdarkly/sse-contract-tests/stream"
)

func DoLinefeedTests(t *TestContext) {
	testInputParsing := func(input string, expectedEvents []client.EventMessage) func(t *TestContext) {
		return func(t *TestContext) {
			t.Run("one chunk", func(t *TestContext) {
				t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
					e.SendChunk(input)
					t.RequireSpecificEvents(r, expectedEvents...)
				})
			})

			t.Run("1-character chunks", func(t *TestContext) {
				t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
					e.SendSplit(input, 1, time.Millisecond*10)
					t.RequireSpecificEvents(r, expectedEvents...)
				})
			})

			t.Run("2-character chunks", func(t *TestContext) {
				t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
					e.SendSplit(input, 2, time.Millisecond*10)
					t.RequireSpecificEvents(r, expectedEvents...)
				})
			})
		}
	}

	testWithTerminator := func(terminator string) func(t *TestContext) {
		return func(t *TestContext) {
			t.Run("one-line event + two-line event", testInputParsing(
				"data: event 1"+terminator+terminator+
					"data: event 2 line 1"+terminator+
					"data: event 2 line 2"+terminator+terminator,
				[]client.EventMessage{
					{Data: "event 1"},
					{Data: "event 2 line 1\nevent 2 line 2"},
				},
			))

			t.Run("3-line event with empty line at beginning", testInputParsing(
				"data:"+terminator+"data: line2"+terminator+"data: line3"+terminator+terminator,
				[]client.EventMessage{
					{Data: "\nline2\nline3"},
				},
			))

			t.Run("3-line event with empty line in middle", testInputParsing(
				"data: line1"+terminator+"data:"+terminator+"data: line3"+terminator+terminator,
				[]client.EventMessage{
					{Data: "line1\n\nline3"},
				},
			))

			t.Run("ignores 1 extra empty line", testInputParsing(
				"data: event 1"+terminator+terminator+terminator+
					"data: event 2"+terminator+terminator,
				[]client.EventMessage{
					{Data: "event 1"},
					{Data: "event 2"},
				},
			))

			t.Run("ignores 2 extra empty lines", testInputParsing(
				"data: event 1"+terminator+terminator+terminator+terminator+
					"data: event 2"+terminator+terminator,
				[]client.EventMessage{
					{Data: "event 1"},
					{Data: "event 2"},
				},
			))
		}
	}

	t.Run("LF separator", testWithTerminator("\n"))

	t.Run("CRLF separator", testWithTerminator("\r\n"))

	if t.HasCapability("cr-only") {
		t.Run("CR separator", testWithTerminator("\r"))
	}

	t.Run("CRLF where CR is end of 1 chunk", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendChunk("data: Hello\r")
			e.SendChunk("\ndata: World\r")
			e.SendChunk("\n\r\ndata: OK\r")
			e.SendChunk("\n")
			e.SendChunk("\r")
			e.SendChunk("\n")
			t.RequireSpecificEvents(r,
				client.EventMessage{Data: "Hello\nWorld"},
				client.EventMessage{Data: "OK"},
			)
		})
	})
}
