// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `scheduled` is the model tier's: what fires when is a claim about a clock the
// run advances. What one subject settles is the comparison itself, which the
// row below walks from both sides.
package scheduledtest_test

import (
	"context"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/scheduled"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/scheduled/scheduledtest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	scheduledtest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	scheduledtest.RunMixed(t,
		inMemory("in-memory"),
		scheduledtest.MixedSuite.Without(scheduledtest.MixedSuite.Checks.At.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	scheduledtest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) scheduledtest.MixedHarness[*scheduledtest.InMemory] {
	return scheduledtest.MixedHarness[*scheduledtest.InMemory]{
		Name: name, New: scheduledtest.NewInMemory,
		// The firing check advances past a scheduled offset, which only
		// moves a clock the subject was built on.
		OnClock: scheduledtest.NewInMemoryOn,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = scheduledtest.MixedChecks{
	{
		Method: "Fired", Name: "counts-nothing-before-its-instant",
		Claim: "Fired counts nothing before its instant arrives",
		Run:   countsNothingBeforeItsInstant,
		ProvenBy: scheduledtest.BrokenMixed(
			"a scheduler that fires everything it is handed", newFiresImmediately,
		),
		ProvenReason: "an hour has not passed on this clock",
	},
}

// --- Bodies -------------------------------------------------------------------

// countsNothingBeforeItsInstant walks the comparison from both sides.
//
// A task registered for the future has not run on a clock nobody advanced, and
// asserting the zero is what stops this fixture from passing against a
// scheduler that fires everything immediately. A task due now is already due —
// its instant is not AFTER the clock's reading — and without that half a
// scheduler that never fires would pass.
func countsNothingBeforeItsInstant(
	tb testing.TB, s scheduled.Mixed, _ scheduledtest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.At(tb.Context(), time.Hour), "the task registers")

	got, err := s.Fired(tb.Context())
	testkit.NoError(tb, err, "the count is readable")
	testkit.Equal(tb, got, 0, "an hour has not passed on this clock")

	testkit.NoError(tb, s.At(tb.Context(), 0), "a task due now registers")

	got, err = s.Fired(tb.Context())
	testkit.NoError(tb, err, "the count is readable")
	testkit.Equal(tb, got, 1, "and counts the one whose instant has arrived")
}

// --- Planted defects ----------------------------------------------------------

// firesImmediately runs every task as it is registered, which is the scheduler
// with no clock at all — and the one every check that only registers a task and
// asks whether the call succeeded calls correct.
type firesImmediately struct{ fired int }

func newFiresImmediately() *firesImmediately { return &firesImmediately{} }

func (f *firesImmediately) At(context.Context, time.Duration) error {
	f.fired++
	return nil
}

func (f *firesImmediately) Fired(context.Context) (int, error) { return f.fired, nil }
