package framework

import (
	"fmt"
	"io"
	"sync"
	"time"
)

const timestampFormat = "2006-01-02 15:04:05.000"

type Logger interface {
	Printf(message string, args ...interface{})
}

type nullLogger struct{}

func (n nullLogger) Printf(message string, args ...interface{}) {}

func NullLogger() Logger { return nullLogger{} }

type CapturedMessage struct {
	Time    time.Time
	Message string
}

type CapturedOutput []CapturedMessage

type CapturingLogger struct {
	output []CapturedMessage
	lock   sync.Mutex
}

func (l *CapturingLogger) Printf(message string, args ...interface{}) {
	l.lock.Lock()
	l.output = append(l.output, CapturedMessage{Time: time.Now(), Message: fmt.Sprintf(message, args...)})
	l.lock.Unlock()
}

func (l *CapturingLogger) Output() CapturedOutput {
	l.lock.Lock()
	ret := append([]CapturedMessage(nil), l.output...)
	l.lock.Unlock()
	return ret
}

func (output CapturedOutput) Dump(dest io.Writer, prefix string) {
	for _, m := range output {
		fmt.Fprintf(dest, "%s[%s] %s\n",
			prefix,
			m.Time.Format(timestampFormat),
			m.Message,
		)
	}
}
