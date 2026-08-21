// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `windowed` counts occurrences inside a moving window, and the harness
// supplies the clock. The row below is what one occurrence settles on a clock
// nobody advanced — and what an unrecorded key answers, which is zero rather
// than a failure.
package windowedtest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/windowed"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/windowed/windowedtest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	windowedtest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	windowedtest.RunMixed(t,
		inMemory("in-memory"),
		windowedtest.MixedSuite.Without(windowedtest.MixedSuite.Checks.Record.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	windowedtest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) windowedtest.MixedHarness[*windowedtest.InMemory] {
	return windowedtest.MixedHarness[*windowedtest.InMemory]{
		Name: name, New: windowedtest.NewInMemory,
		// The window check advances past the window it declared, which
		// only moves a clock the subject was built on.
		OnClock: windowedtest.NewInMemoryOn,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = windowedtest.MixedChecks{
	{
		Method: "CountIn", Name: "counts-inside-the-window",
		Claim: "CountIn counts inside the window and not outside it",
		Run:   countsInsideTheWindow,
		ProvenBy: windowedtest.BrokenMixed(
			"a counter that reports every key as seen once", newCountsEveryKey,
		),
		ProvenReason: "it simply has no occurrences",
	},
}

// --- Bodies -------------------------------------------------------------------

// countsInsideTheWindow records one occurrence on a clock nothing advanced, so
// it is inside the window by construction — then asks about a key nothing
// recorded, which counts zero rather than erroring: an absent key has no
// occurrences, and that is an answer rather than a failure.
func countsInsideTheWindow(
	tb testing.TB, s windowed.Mixed, fx windowedtest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Record(tb.Context(), fx.Key()), "an occurrence is recorded")

	got, err := s.CountIn(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "the count is readable")
	testkit.Equal(tb, got, 1, "the recorded occurrence is inside the window")

	absent, err := s.CountIn(tb.Context(), fx.KeyOther())
	testkit.NoError(tb, err, "an unrecorded key is not an error")
	testkit.Equal(tb, absent, 0, "it simply has no occurrences")
}

// --- Planted defects ----------------------------------------------------------

// countsEveryKey answers one for anything it is asked, which satisfies the
// recorded half of the row and invents occurrences for every key nobody
// touched — the direction a window check with only a positive case misses.
type countsEveryKey struct{}

func newCountsEveryKey() countsEveryKey { return countsEveryKey{} }

func (countsEveryKey) Record(context.Context, string) error { return nil }

func (countsEveryKey) CountIn(context.Context, string) (int, error) { return 1, nil }
