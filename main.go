package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/launchdarkly/sse-contract-tests/client"
	"github.com/launchdarkly/sse-contract-tests/framework"
	"github.com/launchdarkly/sse-contract-tests/mockstream"
	"github.com/launchdarkly/sse-contract-tests/ssetests"
)

const defaultPort = 8111

func main() {
	var serviceURL string
	var port int
	var host string
	var runFilter Filter
	var skipFilter Filter
	var debug bool
	var debugAll bool

	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.StringVar(&serviceURL, "url", "", "test service URL")
	fs.StringVar(&host, "host", "localhost", "external hostname of the test harness")
	fs.IntVar(&port, "port", defaultPort, "port that the test harness will listen on")
	fs.Var(&runFilter, "run", "regex pattern(s) to select tests to run")
	fs.Var(&skipFilter, "skip", "regex pattern(s) to select tests not to run")
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

	fmt.Printf("Connecting to test service at %s\n\n", serviceURL)
	var clientLogger framework.CapturingLogger
	client, err := client.NewSSETestClient(
		serviceURL,
		port,
		host,
		time.Second*5,
		&clientLogger,
	)
	if debugAll || (debug && err != nil) {
		clientLogger.Output().Dump(os.Stdout, "")
		fmt.Println()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to test service: %s\n", err)
		os.Exit(1)
	}

	missing := client.MissingCapabilities()
	if len(missing) > 0 {
		fmt.Println("Some tests will be skipped because the test service does not support the following capabilities:")
		fmt.Printf("  %s\n", strings.Join(missing, ", "))
		fmt.Println()
	}

	if runFilter.IsDefined() || skipFilter.IsDefined() {
		fmt.Println("Some tests will be skipped based on the filter criteria for this test run:")
		if runFilter.IsDefined() {
			fmt.Printf("  skip any not matching %s\n", runFilter)
		}
		if skipFilter.IsDefined() {
			fmt.Printf("  skip any matching %s\n", skipFilter)
		}
		fmt.Println()
	}

	streamManager := mockstream.NewStreamManager(host, port)

	startServer(port, client, streamManager)

	filter := func(id framework.TestID) bool {
		name := id.String()
		return (!runFilter.IsDefined() || runFilter.AnyMatch(name)) &&
			!skipFilter.AnyMatch(name)
	}

	fmt.Println("Running test suite")
	testLogger := ConsoleTestLogger{
		DebugOutputOnFailure: debug || debugAll,
		DebugOutputOnSuccess: debugAll,
	}
	results := ssetests.RunTestSuite(client, streamManager, filter, &testLogger)
	if !results.OK() {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "FAILED TESTS (%d):\n", len(results.Failures))
		for _, f := range results.Failures {
			fmt.Fprintf(os.Stderr, "  * %s\n", f.TestID)
		}
		os.Exit(1)
	}
	fmt.Println()
	fmt.Println("All tests passed")
}

func startServer(port int, client *client.TestServiceClient, streamManager *mockstream.StreamManager) {
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "HEAD" {
				w.WriteHeader(200)
				return
			}
			if client.HandleRequest(w, r) {
				return
			}
			if streamManager.HandleRequest(w, r) {
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}),
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	// Wait till the server is definitely listening for requests before we run any tests
	deadline := time.NewTimer(time.Second * 10)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond * 10)
	defer ticker.Stop()
	for {
		select {
		case <-deadline.C:
			panic("Could not detect own listener at " + server.Addr)
		case <-ticker.C:
			resp, err := http.DefaultClient.Head(fmt.Sprintf("http://localhost:%d", port))
			if err == nil && resp.StatusCode == 200 {
				return
			}
		}
	}
}
