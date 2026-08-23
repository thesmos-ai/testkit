// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// A teardown returning nothing: there is no error slot, so the derived family
// is the smoke call and the row below is all that is left of the lifecycle law.
package voidlifecycletest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/voidlifecycle"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/voidlifecycle/voidlifecycletest"
)

// TestVoidLifecycleContract runs the generated check and this package's own.
func TestVoidLifecycleContract(t *testing.T) {
	t.Parallel()

	voidlifecycletest.RunVoidLifecycle(t, inMemory("in-memory"), voidLifecycleChecks)
}

// TestVoidLifecycleContractWithoutSmoke drops a check through the typed index
// rather than a string, so a check that is renamed or stops being emitted
// breaks this compile instead of silently declining nothing.
func TestVoidLifecycleContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	voidlifecycletest.RunVoidLifecycle(t,
		inMemory("in-memory"),
		voidlifecycletest.VoidLifecycleSuite.Without(
			voidlifecycletest.VoidLifecycleSuite.Checks.Stop.Smoke(),
		),
	)
}

// TestVoidLifecycleChecksCanFail drives the row against its planted defect.
func TestVoidLifecycleChecksCanFail(t *testing.T) {
	t.Parallel()

	voidlifecycletest.ProveVoidLifecycle(t, inMemory("in-memory"), voidLifecycleChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) voidlifecycletest.VoidLifecycleHarness[*voidlifecycletest.InMemory] {
	return voidlifecycletest.VoidLifecycleHarness[*voidlifecycletest.InMemory]{
		Name: name, New: voidlifecycletest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var voidLifecycleChecks = voidlifecycletest.VoidLifecycleChecks{
	{
		Method: "Stop", Name: "second-stop-is-safe",
		Claim: "Stop is idempotent",
		Run:   secondStopIsSafe,
		ProvenBy: voidlifecycletest.BrokenVoidLifecycle(
			"a teardown that panics on being repeated", newPanicsOnRepeat,
		),
		// With no return there is nothing to assert against, so the only way
		// this row can fail is the only way the subject can misbehave: a panic.
		// The harness records a panicking check as a failed leg, which is what
		// makes a claim with no assertion in it falsifiable at all.
		ProvenReason: "panicked",
	},
}

// --- Bodies -------------------------------------------------------------------

// secondStopIsSafe is all that is left of the lifecycle law once the error
// return goes: a second Stop must be safe, and safe is all it can be, since
// there is nothing to report.
func secondStopIsSafe(
	tb testing.TB, s voidlifecycle.VoidLifecycle,
	_ voidlifecycletest.VoidLifecycleFixture,
) {
	tb.Helper()
	s.Stop()
	s.Stop()
}

// --- Planted defects ----------------------------------------------------------

// panicsOnRepeat treats shutdown as a state change rather than a state, and has
// nowhere to say so — which is how a void teardown expresses the bug a
// returning one reports.
type panicsOnRepeat struct{ stopped bool }

func newPanicsOnRepeat() *panicsOnRepeat { return &panicsOnRepeat{} }

func (p *panicsOnRepeat) Stop() {
	if p.stopped {
		panic("voidlifecycletest_test: stopped twice")
	}
	p.stopped = true
}
