// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `timeaware` says a subject reads its clock rather than the wall, and the
// harness supplies the clock. What one subject settles is the row below: on a
// clock nobody advanced, nothing moves.
package timeawaretest_test

import (
	"context"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/timeaware"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/timeaware/timeawaretest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	timeawaretest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	timeawaretest.RunMixed(t,
		inMemory("in-memory"),
		timeawaretest.MixedSuite.Without(timeawaretest.MixedSuite.Checks.Touch.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	timeawaretest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) timeawaretest.MixedHarness[*timeawaretest.InMemory] {
	return timeawaretest.MixedHarness[*timeawaretest.InMemory]{
		Name: name, New: timeawaretest.NewInMemory,
		// The clock door. The one claim this mixin states on its own is
		// that the reading moves when the run moves time, and a subject
		// built on its own clock could never be asked.
		OnClock: timeawaretest.NewInMemoryOn,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = timeawaretest.MixedChecks{
	{
		Method: "AgeOf", Name: "answers-from-the-clock",
		Claim: "AgeOf answers from the clock rather than the wall",
		Run:   answersFromTheClock,
		ProvenBy: timeawaretest.BrokenMixed(
			"a subject that ages against the wall", newReadsTheWall,
		),
		ProvenReason: "no time has passed on this clock",
	},
}

// --- Bodies -------------------------------------------------------------------

// answersFromTheClock reads the age twice.
//
// On a clock nothing advanced the age is zero, and reading it twice gives the
// same answer — which a wall-clock subject could not promise, because time
// passes between the two calls whether the run asked it to or not.
func answersFromTheClock(
	tb testing.TB, s timeaware.Mixed, fx timeawaretest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Touch(tb.Context(), fx.Key()), "the key is recorded")

	first, err := s.AgeOf(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "the touched key has an age")
	testkit.Equal(tb, first, int64(0), "no time has passed on this clock")

	again, err := s.AgeOf(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "and it is still readable")
	testkit.Equal(tb, again, first, "a clock nobody advanced does not move")
}

// --- Planted defects ----------------------------------------------------------

// readsTheWall ages against real time, which is what a subject calling
// time.Now directly does. It is the one defect a run cannot make deterministic
// from outside, and the reason the mixin exists: the age it answers depends on
// how long the test took rather than on what the run asked for.
type readsTheWall struct{ touched map[string]time.Time }

func newReadsTheWall() *readsTheWall {
	return &readsTheWall{touched: map[string]time.Time{}}
}

func (r *readsTheWall) Touch(_ context.Context, key string) error {
	// Stamped in the past, so the age is non-zero from the first read
	// rather than depending on how fast the two reads happen to run.
	r.touched[key] = time.Now().Add(-time.Hour)
	return nil
}

func (r *readsTheWall) AgeOf(_ context.Context, key string) (int64, error) {
	at, seen := r.touched[key]
	if !seen {
		return 0, timeawaretest.ErrUnseen
	}
	return int64(time.Since(at).Seconds()), nil
}
