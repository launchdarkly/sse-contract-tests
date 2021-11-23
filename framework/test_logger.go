package framework

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

var consoleTestErrorColor = color.New(color.FgYellow)
var consoleTestFailedColor = color.New(color.FgRed)
var consoleTestSkippedColor = color.New(color.Faint, color.FgBlue)
var consoleDebugOutputColor = color.New(color.Faint)
var allTestsPassedColor = color.New(color.FgGreen)

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

type ConsoleTestLogger struct {
	DebugOutputOnFailure bool
	DebugOutputOnSuccess bool
}

func (c ConsoleTestLogger) TestStarted(id TestID) {
	fmt.Printf("[%s]\n", id)
}

func (c ConsoleTestLogger) TestError(id TestID, err error) {
	for _, line := range strings.Split(err.Error(), "\n") {
		consoleTestErrorColor.Printf("  %s\n", line)
	}
}

func (c ConsoleTestLogger) TestFinished(id TestID, failed bool, debugOutput CapturedOutput) {
	if failed {
		consoleTestFailedColor.Printf("  FAILED: %s\n", id)
	}
	if len(debugOutput) > 0 &&
		((failed && c.DebugOutputOnFailure) || (!failed && c.DebugOutputOnSuccess)) {
		consoleDebugOutputColor.Println(debugOutput.ToString("    DEBUG "))
	}
}

func (c ConsoleTestLogger) TestSkipped(id TestID, reason string) {
	if reason == "" {
		consoleTestSkippedColor.Printf("  SKIPPED: %s\n", id)
	} else {
		consoleTestSkippedColor.Printf("  SKIPPED: %s (%s)\n", id, reason)
	}
}

func PrintResults(results Results) {
	if results.OK() {
		allTestsPassedColor.Println("All tests passed")
	} else {
		consoleTestFailedColor.Fprintf(os.Stderr, "FAILED TESTS (%d):\n", len(results.Failures))
		for _, f := range results.Failures {
			consoleTestFailedColor.Fprintf(os.Stderr, "  * %s\n", f.TestID)
		}
	}
}
