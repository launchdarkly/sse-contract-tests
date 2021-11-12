package ssetests

import (
	"github.com/stretchr/testify/assert"
)

func DoCommentTests(t *T) {
	t.RequireCapability("comments")

	t.Run("single comment", func(t *T) {
		t.StartSSEClient()
		t.SendOnStream(":Hello\n")
		c := t.RequireComment()
		assert.Equal(t, "Hello", c)
	})

	t.Run("two comments in a row", func(t *T) {
		t.StartSSEClient()

		t.SendOnStream(":Hello\n")
		c1 := t.RequireComment()
		assert.Equal(t, "Hello", c1)

		t.SendOnStream(":World\n")
		c2 := t.RequireComment()
		assert.Equal(t, "World", c2)
	})

	t.Run("comment before event", func(t *T) {
		t.StartSSEClient()

		t.SendOnStream(":Hello\n")
		t.SendOnStream("data: Hello\n\n")

		c := t.RequireComment()
		assert.Equal(t, "Hello", c)

		ev := t.RequireEvent()
		assert.Equal(t, "message", ev.Type)
		assert.Equal(t, "Hello", ev.Data)
	})

	t.Run("comment after event", func(t *T) {
		t.StartSSEClient()

		t.SendOnStream("data: Hello\n\n")
		t.SendOnStream(":Hello\n")

		ev := t.RequireEvent()
		assert.Equal(t, "message", ev.Type)
		assert.Equal(t, "Hello", ev.Data)

		c := t.RequireComment()
		assert.Equal(t, "Hello", c)
	})
}
