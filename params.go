package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/launchdarkly/sse-contract-tests/framework/ldtest"

	"github.com/alessio/shellescape"
)

type commandParams struct {
	serviceURL       string
	port             int
	host             string
	filters          ldtest.RegexFilters
	stopServiceAtEnd bool
	debug            bool
	debugAll         bool
}

func (c *commandParams) Read(args []string) bool {
	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.StringVar(&c.serviceURL, "url", "", "test service URL")
	fs.StringVar(&c.host, "host", "localhost", "external hostname of the test harness")
	fs.IntVar(&c.port, "port", defaultPort, "port that the test harness will listen on")
	fs.Var(&c.filters.MustMatch, "run", "regex pattern(s) to select tests to run")
	fs.Var(&c.filters.MustNotMatch, "skip", "regex pattern(s) to select tests not to run")
	fs.BoolVar(&c.stopServiceAtEnd, "stop-service-at-end", false, "tell test service to exit after the test run")
	fs.BoolVar(&c.debug, "debug", false, "enable debug logging for failed tests")
	fs.BoolVar(&c.debugAll, "debug-all", false, "enable debug logging for all tests")

	if err := fs.Parse(args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		fs.Usage()
		return false
	}
	if c.serviceURL == "" {
		fmt.Fprintln(os.Stderr, "-url is required")
		fs.Usage()
		return false
	}
	return true
}

type commandBuilder []string

func (b *commandBuilder) add(args ...string) {
	for _, a := range args {
		*b = append(*b, shellescape.Quote(a))
	}
}

func (b commandBuilder) String() string {
	return strings.Join(b, " ")
}
