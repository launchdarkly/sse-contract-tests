package harness

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeItemData(counter int) []byte {
	return []byte(fmt.Sprintf("item-%d", counter))
}

func acceptTestItems(q *MessageSortingQueue, counters ...int) {
	for _, c := range counters {
		q.Accept(c, fakeItemData(c))
	}
}

func expectTestItems(t *testing.T, q *MessageSortingQueue, counters ...int) {
	for _, c := range counters {
		select {
		case item := <-q.C:
			assert.Equal(t, string(fakeItemData(c)), string(item))
		case <-time.After(time.Second):
			var deferredList []string
			for _, d := range q.Deferred() {
				deferredList = append(deferredList, string(d))
			}
			require.Fail(t, "timed out waiting for item from queue",
				"was waiting for item %d; deferred items were [%v]", strings.Join(deferredList, ","))
		}
	}
}

func expectDeferredItems(t *testing.T, q *MessageSortingQueue, counters ...int) {
	var expected, actual []string
	for _, c := range counters {
		expected = append(expected, string(fakeItemData(c)))
	}
	for _, d := range q.Deferred() {
		actual = append(actual, string(d))
	}
	assert.Equal(t, expected, actual, "did not see expected items in deferred list")
}

func TestMessageSortingQueueWithMessagesInOrder(t *testing.T) {
	q := NewMessageSortingQueue(10)
	acceptTestItems(q, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	expectDeferredItems(t, q) // should be empty
	expectTestItems(t, q, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
}

func TestMessageSortingQueueWithMessagesOutOfOrder(t *testing.T) {
	q := NewMessageSortingQueue(10)

	acceptTestItems(q, 3)
	expectDeferredItems(t, q, 3)

	acceptTestItems(q, 2)
	expectDeferredItems(t, q, 2, 3)

	acceptTestItems(q, 6)
	expectDeferredItems(t, q, 2, 3, 6)

	acceptTestItems(q, 1)
	expectTestItems(t, q, 1, 2, 3)
	expectDeferredItems(t, q, 6)

	acceptTestItems(q, 5)
	expectDeferredItems(t, q, 5, 6)

	acceptTestItems(q, 4)
	expectTestItems(t, q, 4, 5, 6)
	expectDeferredItems(t, q) // empty
}
