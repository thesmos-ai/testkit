// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// The per-write lifetime is the point of this fixture. Its sibling `ttl`
// fixes one lifetime for every write through `duration=`; here each entry
// carries its own, which is the shape that gives a defect a field to
// reach for.
package ttlperwritetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/ttlperwrite/ttlperwritetest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	ttlperwritetest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedChecksCanFail drives each row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	ttlperwritetest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) ttlperwritetest.MixedHarness[*ttlperwritetest.InMemory] {
	return ttlperwritetest.MixedHarness[*ttlperwritetest.InMemory]{
		Name: name, New: ttlperwritetest.NewInMemory,
		// The map and the clock both outlive the instance holding them,
		// which is what makes a rebuild over them mean anything: an entry
		// written before the crash still expires when it was going to.
		Recover: ttlperwritetest.Reopen,
	}
}

// --- The checks -----------------------------------------------------------

// None yet, and the generated header says why: the TTL law reads a
// duration= stamp this declaration does not carry, so no clocked row is
// emitted and no clock door opens.
//
// The claim this fixture exists for — two entries written together
// expiring apart, because they asked to — needs the clock a bound law
// would open. Writing it here with no clock would mean asserting
// something weaker and calling it the per-write claim.
var mixedChecks = ttlperwritetest.MixedChecks{}
