package testsuite

type TestLogger interface {
	TestStarted(id TestID)
	TestError(id TestID, err error)
	TestFinished(id TestID, failed bool)
	TestSkipped(id TestID)
}

type NullLogger struct{}

func (n NullLogger) TestStarted(id TestID)               {}
func (n NullLogger) TestError(id TestID, err error)      {}
func (n NullLogger) TestFinished(id TestID, failed bool) {}
func (n NullLogger) TestSkipped(id TestID)               {}
