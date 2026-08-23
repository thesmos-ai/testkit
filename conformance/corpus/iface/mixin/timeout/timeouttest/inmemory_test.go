// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `timeout duration=` gates the budget check on a declared number, and this
// subject declares none. What is left is the path a budget check cannot
// exercise: that one measures a subject built to spend, and the row below is
// about one built to spend nothing.
package timeouttest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/timeout"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/timeout/timeouttest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	timeouttest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	timeouttest.RunMixed(t,
		inMemory("in-memory"),
		timeouttest.MixedSuite.Without(timeouttest.MixedSuite.Checks.Slow.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	timeouttest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) timeouttest.MixedHarness[*timeouttest.InMemory] {
	return timeouttest.MixedHarness[*timeouttest.InMemory]{
		Name: name, New: timeouttest.NewInMemory,
		// The deadline check advances past the deadline it set, which
		// only moves a clock the subject was built on.
		OnClock: timeouttest.NewInMemoryOn,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = timeouttest.MixedChecks{
	{
		Method: "Slow", Name: "answers-at-once-with-nothing-to-wait-for",
		Claim: "Slow answers at once when it has nothing to wait for",
		Run:   answersAtOnceWithNothingToWaitFor,
		ProvenBy: timeouttest.BrokenMixed(
			"a subject that reports a timeout it never spent", newTimesOutRegardless,
		),
		ProvenReason: "answers without waiting",
	},
}

// --- Bodies -------------------------------------------------------------------

func answersAtOnceWithNothingToWaitFor(
	tb testing.TB, s timeout.Mixed, fx timeouttest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Slow(tb.Context(), fx.Key()),
		"a subject with no delay answers without waiting")
}

// --- Planted defects ----------------------------------------------------------

// timesOutRegardless reports the deadline error without ever having waited,
// which is a budget compared against the wrong side of zero. A check that only
// asked whether a slow call CAN fail would call it correct.
type timesOutRegardless struct{}

func newTimesOutRegardless() timesOutRegardless { return timesOutRegardless{} }

func (timesOutRegardless) Slow(context.Context, string) error {
	return context.DeadlineExceeded
}
