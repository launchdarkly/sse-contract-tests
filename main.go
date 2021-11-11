package main

import (
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
	var params commandParams
	if err := params.Read(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid parameters: %s\n", err)
		os.Exit(1)
	}

	if params.outputDockerScriptVersion != "" {
		params.outputDockerScript()
		os.Exit(0)
	}

	mainDebugLogger := framework.NullLogger()
	if params.debugAll {
		mainDebugLogger = log.New(os.Stdout, "", log.LstdFlags)
	}

	harness, err := framework.NewTestHarness(
		params.serviceURL,
		params.host,
		params.port,
		statusQueryTimeout,
		mainDebugLogger,
		os.Stdout,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Test service error: %s\n", err)
		os.Exit(1)
	}

	fmt.Println()
	framework.PrintFilterDescription(harness, params.filters, ssetests.AllCapabilities)

	fmt.Println("Running test suite")

	testLogger := framework.ConsoleTestLogger{
		DebugOutputOnFailure: params.debug || params.debugAll,
		DebugOutputOnSuccess: params.debugAll,
	}

	results := ssetests.RunTestSuite(harness, params.filters.AsFilter, testLogger)

	fmt.Println()
	framework.PrintResults(results)
	if !results.OK() {
		os.Exit(1)
	}
}
