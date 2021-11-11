package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/launchdarkly/sse-contract-tests/framework"
	"github.com/launchdarkly/sse-contract-tests/ssetests"
)

const defaultPort = 8111
const statusQueryTimeout = time.Second * 10

func main() {
	var serviceURL string
	var port int
	var host string
	var filters framework.RegexFilters
	var debug bool
	var debugAll bool

	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.StringVar(&serviceURL, "url", "", "test service URL")
	fs.StringVar(&host, "host", "localhost", "external hostname of the test harness")
	fs.IntVar(&port, "port", defaultPort, "port that the test harness will listen on")
	fs.Var(&filters.MustMatch, "run", "regex pattern(s) to select tests to run")
	fs.Var(&filters.MustNotMatch, "skip", "regex pattern(s) to select tests not to run")
	fs.BoolVar(&debug, "debug", false, "enable debug logging for failed tests")
	fs.BoolVar(&debugAll, "debug-all", false, "enable debug logging for all tests")

	err := fs.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid parameters: %s\n", err)
		os.Exit(1)
	}
	if serviceURL == "" {
		fmt.Fprintln(os.Stderr, "--url is required")
		os.Exit(1)
	}

	mainDebugLogger := framework.NullLogger()
	if debugAll {
		mainDebugLogger = log.New(os.Stdout, "", log.LstdFlags)
	}

	harness, err := framework.NewTestHarness(
		serviceURL,
		host,
		port,
		statusQueryTimeout,
		mainDebugLogger,
		os.Stdout,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Test service error: %s\n", err)
		os.Exit(1)
	}

	fmt.Println()
	framework.PrintFilterDescription(harness, filters, ssetests.AllCapabilities)

	fmt.Println("Running test suite")

	testLogger := framework.ConsoleTestLogger{
		DebugOutputOnFailure: debug || debugAll,
		DebugOutputOnSuccess: debugAll,
	}

	results := ssetests.RunTestSuite(harness, filters.AsFilter, testLogger)

	fmt.Println()
	framework.PrintResults(results)
	if !results.OK() {
		os.Exit(1)
	}
}
