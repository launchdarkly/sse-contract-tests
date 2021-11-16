package framework

import (
	"sort"
	"sync"
)

type MessageSortingQueue struct {
	C           chan []byte
	lastCounter int
	deferred    []deferredMessage
	lock        sync.Mutex
	closeOnce   sync.Once
}

type deferredMessage struct {
	counter int
	message []byte
}

func NewMessageSortingQueue(channelSize int) *MessageSortingQueue {
	return &MessageSortingQueue{C: make(chan []byte, channelSize)}
}

func (q *MessageSortingQueue) Accept(counter int, message []byte) {
	q.lock.Lock()
	if counter > q.lastCounter+1 {
		q.deferred = append(q.deferred, deferredMessage{counter: counter, message: message})
		sort.Slice(q.deferred, func(i, j int) bool { return q.deferred[i].counter < q.deferred[j].counter })
		q.lock.Unlock()
		return
	}
	q.lastCounter = counter
	q.C <- message
	for len(q.deferred) > 0 {
		next := q.deferred[0]
		if next.counter != q.lastCounter+1 {
			break
		}
		q.deferred = q.deferred[1:]
		q.lastCounter++
		q.C <- next.message
	}
	q.lock.Unlock()
}

func (q *MessageSortingQueue) Deferred() [][]byte {
	q.lock.Lock()
	ret := make([][]byte, 0, len(q.deferred))
	for _, d := range q.deferred {
		ret = append(ret, d.message)
	}
	q.lock.Unlock()
	return ret
}

func (q *MessageSortingQueue) Close() {
	q.closeOnce.Do(func() {
		close(q.C)
		q.C = nil
	})
}
