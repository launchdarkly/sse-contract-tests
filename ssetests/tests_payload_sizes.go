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
// Constants were calibrated empirically against an Android emulator's adb-tunneled
// throughput (~3 MiB/s round-trip observed). Prior values of base=5s, rate=5 MiB/s were
// tuned for localhost throughput and were too tight for larger payloads on realistic
// mobile/emulator infrastructure -- they timed out at 64 MiB (17.8s allowed) and 128 MiB
// (30.6s allowed) even for correct implementations.
//
// Current values give ~10s for small payloads (well under the 60+ second stall the
// harness is designed to catch) and ~74s for 128 MiB (comfortably above what the
// emulator needs to transfer at ~3 MiB/s round-trip). At sizes that large, the
// buffer-mismatch stall pattern can't manifest anyway -- TCP frames arrive continuously
// -- so the timeout crossing the ~60s stall boundary at the top end does not reduce
// the sweep's ability to catch the class of bug it targets.
func timeoutForSize(size int) time.Duration {
	const base = 10 * time.Second
	const bytesPerSecond = 2 * 1024 * 1024 // 2 MiB/s effective allowance (~1.5x safety over observed emulator throughput)
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
