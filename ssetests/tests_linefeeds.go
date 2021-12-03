package ssetests

import (
	"time"
)

func DoLinefeedTests(t *T) {
	testInputParsing := func(input string, expectedEvents []EventMessage) func(t *T) {
		return func(t *T) {
			t.Run("one chunk", func(t *T) {
				t.StartSSEClient()
				t.SendOnStream(input)
				t.RequireSpecificEvents(expectedEvents...)
			})

			t.Run("1-character chunks", func(t *T) {
				t.StartSSEClient()
				t.SendOnStreamInChunks(input, 1, time.Millisecond*10)
				t.RequireSpecificEvents(expectedEvents...)
			})

			t.Run("2-character chunks", func(t *T) {
				t.StartSSEClient()
				t.SendOnStreamInChunks(input, 2, time.Millisecond*10)
				t.RequireSpecificEvents(expectedEvents...)
			})
		}
	}

	testWithTerminator := func(terminator string) func(t *T) {
		return func(t *T) {
			t.Run("one-line event", testInputParsing(
				"data: event 1"+terminator+terminator,
				[]EventMessage{
					{Data: "event 1"},
				},
			))

			t.Run("one-line event + two-line event", testInputParsing(
				"data: event 1"+terminator+terminator+
					"data: event 2 line 1"+terminator+
					"data: event 2 line 2"+terminator+terminator,
				[]EventMessage{
					{Data: "event 1"},
					{Data: "event 2 line 1\nevent 2 line 2"},
				},
			))

			t.Run("3-line event with empty line at beginning", testInputParsing(
				"data:"+terminator+"data: line2"+terminator+"data: line3"+terminator+terminator,
				[]EventMessage{
					{Data: "\nline2\nline3"},
				},
			))

			t.Run("3-line event with empty line in middle", testInputParsing(
				"data: line1"+terminator+"data:"+terminator+"data: line3"+terminator+terminator,
				[]EventMessage{
					{Data: "line1\n\nline3"},
				},
			))

			t.Run("ignores 1 extra empty line", testInputParsing(
				"data: event 1"+terminator+terminator+terminator+
					"data: event 2"+terminator+terminator,
				[]EventMessage{
					{Data: "event 1"},
					{Data: "event 2"},
				},
			))

			t.Run("ignores 2 extra empty lines", testInputParsing(
				"data: event 1"+terminator+terminator+terminator+terminator+
					"data: event 2"+terminator+terminator,
				[]EventMessage{
					{Data: "event 1"},
					{Data: "event 2"},
				},
			))
		}
	}

	t.Run("LF separator", testWithTerminator("\n"))

	t.Run("CRLF separator", testWithTerminator("\r\n"))

	t.Run("CR separator", testWithTerminator("\r"))

	t.Run("CRLF where CR is end of 1 chunk", func(t *T) {
		t.StartSSEClient()
		t.SendOnStream("data: Hello\r")
		t.SendOnStream("\ndata: World\r")
		t.SendOnStream("\n\r\ndata: OK\r")
		t.SendOnStream("\n")
		t.SendOnStream("\r")
		t.SendOnStream("\n")
		t.RequireSpecificEvents(
			EventMessage{Data: "Hello\nWorld"},
			EventMessage{Data: "OK"},
		)
	})
}
