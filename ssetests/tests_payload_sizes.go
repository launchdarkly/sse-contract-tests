package ssetests

import (
	"fmt"
	"strings"
	"time"

	"github.com/launchdarkly/sse-contract-tests/framework/ldtest"

	"github.com/stretchr/testify/assert"
)

// payloadSizeSweep is the set of `data:` payload lengths (in bytes) exercised by the
// stress test. Sizes cluster around common buffer-size boundaries in the runtimes we've
// observed:
//
//   - 1024 bytes  -- .NET's StreamReader default buffer
//   - 4096 bytes  -- .NET's BufferedStream default buffer, page size on most systems
//   - 8192 bytes  -- Okio Segment.SIZE (used by Android's OkHttp fork), doubled StreamReader
//
// The intent is to catch buffer-mismatch stalls where the caller-side read buffer is
// smaller than an intermediate wrapping layer's buffer, causing the intermediate layer
// to over-drain the underlying stream and force an extra blocking read.
//
// Sizes here are of the data string only; wire size = len("data: ") + size + len("\n\n").
var payloadSizeSweep = []int{
	1,
	128,
	512,
	1023, 1024, 1025,
	2048,
	3999, 4094, 4096, 4098,
	6144,
	8189, 8192, 8195,
	12288,
	16384,
	32768,
	65536,
	262144,
	1048576,
}

// payloadArrivalTimeout is how long we wait for the test service to report the event.
// Deliberately shorter than the default awaitMessageTimeout so that stalls fail fast
// rather than eating the full default. For localhost test services this is more than
// enough headroom; for very slow environments (real-device testing) it may need to
// grow, but any stall of the class we're looking for (60+ seconds) will still trip it.
const payloadArrivalTimeout = 3 * time.Second

// DoPayloadSizeStressTests sends events at a variety of payload sizes to catch buffer-mismatch
// stalls in SSE client implementations. This covers a class of bug where the client's read
// buffer size interacts pathologically with an intermediate stream-wrapping layer, producing
// long stalls when the payload lands in a specific size zone.
//
// This test is gated behind the "payload-size-stress-testable" capability because:
//  1. It is a stress / robustness check, not a spec-conformance test.
//  2. Some SSE implementations may have latent buffer-mismatch behavior that isn't
//     immediately fixable, and we don't want their existing CI to fail on adoption.
//     Teams opt in by declaring the capability once they've validated the sweep.
//  3. The largest sizes (1 MiB) may exceed reasonable defaults for some environments;
//     opting in signals that the test service is prepared to handle them.
func DoPayloadSizeStressTests(t *ldtest.T) {
	t.RequireCapability("payload-size-stress-testable")

	for _, size := range payloadSizeSweep {
		size := size // capture for closure
		t.Run(fmt.Sprintf("payload of %d bytes is delivered promptly", size), func(t *ldtest.T) {
			_, stream, client := NewStreamAndSSEClient(t)
			data := strings.Repeat("x", size)
			stream.Send("data: " + data + "\n\n")
			actual := client.RequireEventWithin(t, payloadArrivalTimeout)
			// Avoid printing megabytes of data on failure.
			if actual.Data != data {
				assert.Failf(t, "payload data did not match",
					"expected %d bytes, got %d bytes", size, len(actual.Data))
			}
		})
	}
}
