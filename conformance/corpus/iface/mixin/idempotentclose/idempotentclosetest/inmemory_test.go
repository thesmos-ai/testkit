// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `idempotentclose` is the model tier's: that a repeated teardown changes
// nothing needs state observed on both sides of it. Stats is what makes that
// observable here, and the row below reads it twice.
package idempotentclosetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/idempotentclose"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/idempotentclose/idempotentclosetest"
)

// TestCloserContract runs the generated checks and this package's own.
func TestCloserContract(t *testing.T) {
	t.Parallel()

	idempotentclosetest.RunCloser(t, inMemory("in-memory"), closerChecks)
}

// TestCloserContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestCloserContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	idempotentclosetest.RunCloser(t,
		inMemory("in-memory"),
		idempotentclosetest.CloserSuite.Without(idempotentclosetest.CloserSuite.Checks.Close.Smoke()),
	)
}

// TestCloserChecksCanFail drives the row against its planted defect.
func TestCloserChecksCanFail(t *testing.T) {
	t.Parallel()

	idempotentclosetest.ProveCloser(t, inMemory("in-memory"), closerChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) idempotentclosetest.CloserHarness[*idempotentclosetest.InMemory] {
	return idempotentclosetest.CloserHarness[*idempotentclosetest.InMemory]{
		Name: name, New: idempotentclosetest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var closerChecks = idempotentclosetest.CloserChecks{
	{
		Method: "Close", Name: "second-close-changes-nothing",
		Claim: "Close leaves the same state the first one did",
		Run:   secondCloseChangesNothing,
		ProvenBy: idempotentclosetest.BrokenCloser(
			"a teardown that reopens what it closed", newReopensOnSecondClose,
		),
		ProvenReason: "still nothing is open",
	},
}

// --- Bodies -------------------------------------------------------------------

func secondCloseChangesNothing(
	tb testing.TB, s idempotentclose.Closer, _ idempotentclosetest.CloserFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Close(tb.Context()), "the teardown runs")
	open, err := s.Stats(tb.Context())
	testkit.NoError(tb, err, "the state is readable after it")
	testkit.Equal(tb, open, 0, "and nothing is left open")

	testkit.NoError(tb, s.Close(tb.Context()), "closing again is silent")
	again, err := s.Stats(tb.Context())
	testkit.NoError(tb, err, "the state is still readable")
	testkit.Equal(tb, again, 0, "and still nothing is open")
}

// --- Planted defects ----------------------------------------------------------

// reopensOnSecondClose returns cleanly from every teardown and leaves something
// open on the second, which is what a close that re-runs its setup path does.
// The error is nil throughout, so a check reading only that calls it idempotent.
type reopensOnSecondClose struct {
	closes int
	open   int
}

func newReopensOnSecondClose() *reopensOnSecondClose { return &reopensOnSecondClose{} }

func (r *reopensOnSecondClose) Close(context.Context) error {
	r.closes++
	if r.closes > 1 {
		r.open = 1
	}
	return nil
}

func (r *reopensOnSecondClose) Stats(context.Context) (int, error) { return r.open, nil }
