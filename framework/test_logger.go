package framework

type TestLogger interface {
	TestStarted(id TestID)
	TestError(id TestID, err error)
	TestFinished(id TestID, failed bool, debugOutput CapturedOutput)
	TestSkipped(id TestID, reason string)
}

type nullTestLogger struct{}

func (n nullTestLogger) TestStarted(TestID)                        {}
func (n nullTestLogger) TestError(TestID, error)                   {}
func (n nullTestLogger) TestFinished(TestID, bool, CapturedOutput) {}
func (n nullTestLogger) TestSkipped(TestID, string)                {}
