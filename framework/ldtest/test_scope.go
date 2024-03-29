package ldtest

import (
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/launchdarkly/sse-contract-tests/framework"
)

type environment struct {
	config  TestConfiguration
	results Results
}

// T represents a test scope. It is very similar to Go's testing.T type.
type T struct {
	env         *environment
	id          TestID
	debugLogger framework.CapturingLogger
	failed      bool
	skipped     bool
	skipReason  string
	cleanups    []func()
	errors      []error
}

// TestConfiguration contains options for the entire test run.
type TestConfiguration struct {
	// Filter is an optional function for determining which tests to run based on their names.
	Filter Filter

	// TestLogger receives status information about each test.
	TestLogger TestLogger

	// Context is an optional value of any type defined by the application which can be accessed from tests.
	Context interface{}

	// Capabilities is a list of strings which are used by T.HasCapability and T.RequireCapability.
	Capabilities []string
}

// Run starts a top-level test scope.
func Run(
	config TestConfiguration,
	action func(*T),
) Results {
	if config.TestLogger == nil {
		config.TestLogger = nullTestLogger{}
	}
	env := &environment{
		config: config,
	}
	t := &T{env: env}
	t.run(action)
	return env.results
}

func (t *T) run(action func(*T)) {
	defer func() {
		if r := recover(); r != nil {
			if t.skipped {
				return
			}
			t.failed = true
			var addError error
			if _, ok := r.(*T); ok {
				if len(t.errors) == 0 {
					addError = errors.New("test failed with no failure message")
				}
			} else {
				addError = fmt.Errorf("unexpected panic in test: %+v\n%s", r, string(debug.Stack()))
			}
			if addError != nil {
				t.errors = append(t.errors, addError)
				t.env.config.TestLogger.TestError(t.id, addError)
			}
		}
		result := TestResult{TestID: t.id, Errors: t.errors}
		t.env.results.Tests = append(t.env.results.Tests, result)
		if t.failed {
			t.env.results.Failures = append(t.env.results.Failures, result)
		}
		for i := len(t.cleanups) - 1; i >= 0; i-- {
			t.cleanups[i]()
		}
	}()

	action(t)
}

// ID returns the full name of the current test.
func (t *T) ID() TestID {
	return t.id
}

// Run runs a subtest in its own scope.
//
// This is equivalent to Go's testing.T.Run.
func (t *T) Run(name string, action func(*T)) {
	id := t.id.Plus(name)

	t.env.config.TestLogger.TestStarted(id)
	if t.env.config.Filter != nil && !t.env.config.Filter(id) {
		t.env.config.TestLogger.TestSkipped(id, "excluded by filter parameters")
		return
	}
	c1 := &T{
		id:  id,
		env: t.env,
	}
	c1.run(action)
	if c1.skipped {
		t.env.config.TestLogger.TestSkipped(id, c1.skipReason)
	} else {
		t.env.config.TestLogger.TestFinished(id, c1.failed, c1.debugLogger.Output())
	}
}

// Errorf reports a test failure. It is equivalent to Go's testing.T.Errorf. It does not cause the test
// to terminate, but adds the failure message to the output and marks the test as failed.
//
// You will rarely use this method directly; it is part of this type's implementation of the base
// interfaces testing.T and assert.TestingT, allowing it to be called from assertion helpers.
func (t *T) Errorf(format string, args ...interface{}) {
	t.failed = true
	err := fmt.Errorf(format, args...)
	t.errors = append(t.errors, err)
	t.env.config.TestLogger.TestError(t.id, reformatError(err))
}

// FailNow causes the test to immediately terminate and be marked as failed.
//
// You will rarely use this method directly; it is part of this type's implementation of the base
// interfaces testing.T and assert.TestingT, allowing it to be called from assertion helpers.
func (t *T) FailNow() {
	panic(t)
}

// Skip causes the test to immediately terminate and be marked as skipped.
func (t *T) Skip() {
	t.skipped = true
	panic(t)
}

// SkipWithReason is equivalent to Skip but provides a message.
func (t *T) SkipWithReason(reason string) {
	t.skipReason = reason
	t.Skip()
}

// Debug writes a message to the output for this test scope.
func (t *T) Debug(message string, args ...interface{}) {
	t.debugLogger.Printf(message, args...)
}

// DebugLogger returns a Logger instance for writing output for this test scope.
func (t *T) DebugLogger() framework.Logger {
	return &t.debugLogger
}

// Defer schedules a cleanup function which is guaranteed to be called when this test scope
// exits for any reason. Unlike a Go defer statement, Defer can be used from within helper
// functions.
func (t *T) Defer(cleanupFn func()) {
	t.cleanups = append(t.cleanups, cleanupFn)
}

// Context returns the application-defined context value, if any, that was specified in the
// TestConfiguration.
func (t *T) Context() interface{} {
	return t.env.config.Context
}

// Capabilities returns the capabilities reported by the test service.
func (t *T) Capabilities() framework.Capabilities {
	return append(framework.Capabilities(nil), t.env.config.Capabilities...)
}

// RequireCapability causes the test to be skipped if HasCapability(name) returns false.
func (t *T) RequireCapability(name string) {
	if !t.Capabilities().Has(name) {
		t.SkipWithReason(fmt.Sprintf("test service does not have capability %q", name))
	}
}
