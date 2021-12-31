package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/launchdarkly/sse-contract-tests/framework"
	"github.com/launchdarkly/sse-contract-tests/framework/harness"
	"github.com/launchdarkly/sse-contract-tests/framework/ldtest"
	"github.com/launchdarkly/sse-contract-tests/ssetests"
)

const defaultPort = 8111
const statusQueryTimeout = time.Second * 10

//go:embed VERSION
var versionString string

func main() {
	fmt.Printf("sse-contract-tests v%s\n", strings.TrimSpace(versionString))

	var params commandParams
	if !params.Read(os.Args) {
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

	harness, err := harness.NewTestHarness(
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
	ldtest.PrintFilterDescription(params.filters, ssetests.AllCapabilities, harness.TestServiceInfo().Capabilities)

	fmt.Println("Running test suite")

	testLogger := ldtest.ConsoleTestLogger{
		DebugOutputOnFailure: params.debug || params.debugAll,
		DebugOutputOnSuccess: params.debugAll,
	}

	results := ssetests.RunTestSuite(harness, params.filters.Match, testLogger)

	fmt.Println()
	ldtest.PrintResults(results)

	if params.stopServiceAtEnd {
		fmt.Println("Stopping test service")
		if err := harness.StopService(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to stop test service: %s\n", err)
		}
	}
	if !results.OK() {
		os.Exit(1)
	}
}
