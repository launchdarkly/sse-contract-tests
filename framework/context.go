package framework

import (
	"errors"
	"fmt"
	"runtime/debug"
)

type environment struct {
	results    Results
	testLogger TestLogger
	filter     Filter
}

type Context struct {
	env         *environment
	id          TestID
	debugLogger CapturingLogger
	failed      bool
	skipped     bool
	skipReason  string
	errors      []error
}

func Run(
	filter func(TestID) bool,
	testLogger TestLogger,
	action func(*Context),
) Results {
	if testLogger == nil {
		testLogger = nullTestLogger{}
	}
	env := &environment{
		filter:     filter,
		testLogger: testLogger,
	}
	c := &Context{env: env}
	c.run(action)
	return env.results
}

func (c *Context) run(action func(*Context)) {
	defer func() {
		if r := recover(); r != nil {
			if c.skipped {
				return
			}
			c.failed = true
			var addError error
			if _, ok := r.(*Context); ok {
				if len(c.errors) == 0 {
					addError = errors.New("test failed with no failure message")
				}
			} else {
				addError = fmt.Errorf("unexpected panic in test: %+v\n%s", r, string(debug.Stack()))
			}
			if addError != nil {
				c.errors = append(c.errors, addError)
				c.env.testLogger.TestError(c.id, addError)
			}
		}
		result := TestResult{TestID: c.id, Errors: c.errors}
		c.env.results.Tests = append(c.env.results.Tests, result)
		if c.failed {
			c.env.results.Failures = append(c.env.results.Failures, result)
		}
	}()

	action(c)
}

func (c *Context) ID() TestID {
	return c.id
}

func (c *Context) Run(name string, action func(*Context)) {
	id := TestID{Path: append(c.id.Path, name)}

	c.env.testLogger.TestStarted(id)
	if c.env.filter != nil && !c.env.filter(id) {
		c.env.testLogger.TestSkipped(id, "excluded by filter parameters")
		return
	}
	c1 := &Context{
		id:  id,
		env: c.env,
	}
	c1.run(action)
	if c1.skipped {
		c.env.testLogger.TestSkipped(id, c1.skipReason)
	} else {
		c.env.testLogger.TestFinished(id, c1.failed, c1.debugLogger.Output())
	}
}

func (c *Context) Errorf(format string, args ...interface{}) {
	c.failed = true
	err := fmt.Errorf(format, args...)
	c.errors = append(c.errors, err)
	c.env.testLogger.TestError(c.id, reformatError(err))
}

func (c *Context) FailNow() {
	panic(c)
}

func (c *Context) Skip() {
	c.skipped = true
	panic(c)
}

func (c *Context) SkipWithReason(reason string) {
	c.skipReason = reason
	c.Skip()
}

func (c *Context) Debug(message string, args ...interface{}) {
	c.debugLogger.Printf(message, args...)
}

func (c *Context) DebugLogger() Logger {
	return &c.debugLogger
}
