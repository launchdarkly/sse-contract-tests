package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/launchdarkly/sse-contract-tests/client"
	"github.com/launchdarkly/sse-contract-tests/stream"
	"github.com/launchdarkly/sse-contract-tests/testsuite"
)

const defaultPort = 8111

func main() {
	var serviceURL string
	var port int
	var host string
	var debug bool

	fs := flag.NewFlagSet("", flag.ExitOnError)

	fs.StringVar(&serviceURL, "url", "", "test service URL")
	fs.BoolVar(&debug, "debug", false, "enable debug logging")
	fs.StringVar(&host, "host", "localhost", "external hostname of the test harness")
	fs.IntVar(&port, "port", defaultPort, "port that the test harness will listen on")

	err := fs.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid parameters: %s\n", err)
		os.Exit(1)
	}

	var debugLogger *log.Logger
	if debug {
		debugLogger = log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lmicroseconds)
	} else {
		debugLogger = log.New(ioutil.Discard, "", 0)
	}

	client, err := client.NewSSETestClient(serviceURL, time.Second*5, debugLogger)
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

	server := stream.NewServer(host, port)

	fmt.Println("Running test suite")
	result := testsuite.RunTestSuite(client, server, &ConsoleTestLogger{}, debugLogger)
	if !result.OK() {
		fmt.Fprintf(os.Stderr, "%d failed tests\n", len(result.Failures))
		os.Exit(1)
	}
	fmt.Println("All tests passed")
}
