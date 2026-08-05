package ssetests

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/launchdarkly/sse-contract-tests/framework/ldtest"

	"github.com/stretchr/testify/assert"
)

// payloadSizeSweep is the set of `data:` payload lengths (in bytes) exercised by the
// stress test. Generated as every power of two from 1 (2^0) through 128 MiB (2^27),
// plus one byte on either side of each power of two. Duplicates (e.g. where pow+1
// meets the next pow-1) are removed.
//
// Sizes near the power-of-two boundaries matter because most stream-wrapping layers
// choose buffer sizes at powers of two (1 KiB, 4 KiB, 8 KiB, ...); the +/-1 probes
// catch sharp transitions we've observed in this class of bug -- sizes right below
// a runtime buffer boundary stalling while sizes right at or above it don't.
//
// Sizes here are of the data string only; wire size = len("data: ") + size + len("\n\n").
var payloadSizeSweep = generatePayloadSweep() //nolint:gochecknoglobals

func generatePayloadSweep() []int {
	seen := make(map[int]bool)
	var sizes []int
	add := func(n int) {
		if n <= 0 || seen[n] {
			return
		}
		seen[n] = true
		sizes = append(sizes, n)
	}
	// 2^0 = 1 through 2^27 = 128 MiB
	for i := 0; i <= 27; i++ {
		pow := 1 << i
		add(pow - 1)
		add(pow)
		add(pow + 1)
	}
	sort.Ints(sizes)
	return sizes
}

// timeoutForSize returns how long to wait for a payload of the given size to be delivered.
//
// A generous base timeout plus a per-byte allowance keeps the timeout tight enough to
// catch stalls (which manifest as 60+ second waits) while allowing legitimate transfer
// time for large payloads on modest connections. For a 128 MiB payload at the assumed
// 5 MiB/s throughput, the timeout is ~31 seconds -- comfortably shorter than the 60+
// second stall it is designed to catch.
func timeoutForSize(size int) time.Duration {
	const base = 5 * time.Second
	const bytesPerSecond = 5 * 1024 * 1024 // 5 MiB/s allowance beyond base
	return base + time.Duration(size)*time.Second/bytesPerSecond
}

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
//  3. The largest sizes (up to 128 MiB) may exceed reasonable memory/bandwidth
//     defaults for some test environments; opting in signals that the test service
//     is prepared to handle them.
func DoPayloadSizeStressTests(t *ldtest.T) {
	t.RequireCapability("payload-size-stress-testable")

	for _, size := range payloadSizeSweep {
		size := size // capture for closure
		t.Run(fmt.Sprintf("payload of %d bytes is delivered promptly", size), func(t *ldtest.T) {
			_, stream, client := NewStreamAndSSEClient(t)
			data := strings.Repeat("x", size)
			stream.Send("data: " + data + "\n\n")
			actual := client.RequireEventWithin(t, timeoutForSize(size))
			// Avoid printing megabytes of data on failure.
			if actual.Data != data {
				assert.Failf(t, "payload data did not match",
					"expected %d bytes, got %d bytes", size, len(actual.Data))
			}
		})
	}
}
