package testframework

import (
	"github.com/launchdarkly/sse-contract-tests/logging"
)

type TestLogger interface {
	TestStarted(id TestID)
	TestError(id TestID, err error)
	TestFinished(id TestID, failed bool, debugOutput []logging.CapturedMessage)
	TestSkipped(id TestID, reason string)
}

type NullLogger struct{}

func (n NullLogger) TestStarted(id TestID) {}

func (n NullLogger) TestError(id TestID, err error) {}

func (n NullLogger) TestFinished(id TestID, failed bool, debugOutput []logging.CapturedMessage) {}

func (n NullLogger) TestSkipped(id TestID, reason string) {}
