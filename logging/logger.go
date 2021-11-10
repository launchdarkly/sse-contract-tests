package logging

import (
	"fmt"
	"sync"
	"time"
)

type Logger interface {
	Printf(message string, args ...interface{})
}

type CapturedMessage struct {
	Time    time.Time
	Message string
}

type CapturingLogger struct {
	output []CapturedMessage
	lock   sync.Mutex
}

func (l *CapturingLogger) Printf(message string, args ...interface{}) {
	l.lock.Lock()
	l.output = append(l.output, CapturedMessage{Time: time.Now(), Message: fmt.Sprintf(message, args...)})
	l.lock.Unlock()
}

func (l *CapturingLogger) Output() []CapturedMessage {
	l.lock.Lock()
	ret := append([]CapturedMessage(nil), l.output...)
	l.lock.Unlock()
	return ret
}
