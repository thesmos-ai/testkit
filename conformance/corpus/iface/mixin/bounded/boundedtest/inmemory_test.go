// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `bounded limit=` declares a ceiling, and the harness hands it to every
// constructor — the real subject's and each planted defect's — so the number
// under test is the declared one rather than a literal repeated here.
package boundedtest_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/bounded"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/bounded/boundedtest"
	"go.thesmos.sh/testkit/engine/suite"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	boundedtest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	boundedtest.RunMixed(t,
		inMemory("in-memory"),
		boundedtest.MixedSuite.Without(boundedtest.MixedSuite.Checks.Add.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	boundedtest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

// The bound is 5 and the row adds one more than that, so the clamp has
// something to clamp. Both numbers are here rather than in the body because
// they are one decision: added must exceed the ceiling or the row is vacuous.
const (
	bound = 5
	added = bound + 2
)

func inMemory(name string) boundedtest.MixedHarness[*boundedtest.InMemory] {
	return boundedtest.MixedHarness[*boundedtest.InMemory]{
		Name: name, New: boundedtest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = boundedtest.MixedChecks{
	{
		Method: "List", Name: "clamped-to-the-declared-bound",
		Claim: "List is bounded by the capacity the declaration gave it",
		Run:   clampedToTheDeclaredBound,
		ProvenBy: boundedtest.BrokenMixed(
			"a collection that answers everything it took", plantedUnbounded,
		),
		ProvenReason: "no more than the declared bound",
	},
}

// --- Bodies -------------------------------------------------------------------

// clampedToTheDeclaredBound adds more than the bound, so a collection that grew
// without answering more is what the mixin claims and this row observes.
func clampedToTheDeclaredBound(
	tb testing.TB, s bounded.Mixed, fx boundedtest.MixedFixture,
) {
	tb.Helper()
	for range added {
		testkit.NoError(tb, s.Add(tb.Context(), fx.Item()), "an item is added")
	}

	got, err := s.List(tb.Context())
	testkit.NoError(tb, err, "the collection is readable")
	testkit.Len(tb, got, bound, "and answers no more than the declared bound")
}

// --- Planted defects ----------------------------------------------------------

// plantedUnbounded builds a collection that takes the declared capacity and
// ignores it, which is what a bound enforced at the write and forgotten at the
// read looks like.
func plantedUnbounded(capacity int) *unboundedList {
	return &unboundedList{capacity: capacity}
}

type unboundedList struct {
	capacity int
	items    []string
}

func (u *unboundedList) Add(_ context.Context, item string) error {
	u.items = append(u.items, item)
	return nil
}

// List answers everything, which is the defect. The capacity it was handed is
// kept and unread, so a run at a different declared bound plants the same one.
func (u *unboundedList) List(context.Context) ([]string, error) {
	return slices.Clone(u.items), nil
}

// --- The negative control -----------------------------------------------------

// TestMixedToleratesADifferentVictim is the specificity half of this
// package's evidence.
//
// Every other test here measures sensitivity: a planted defect must be
// caught. None of them can tell a check that is too STRONG from one that
// is right — a check reddening a legal implementation looks exactly like
// a suite working, until somebody's correct code fails it.
//
// The freedom this control exercises is real and the mixin leaves it
// open. `bounded limit=5` says the reader answers at most five; it says
// nothing about WHICH five. The subject beside it keeps the oldest, so a
// check written from that subject could quietly come to require it.
func TestMixedToleratesADifferentVictim(t *testing.T) {
	t.Parallel()

	// Through the generated entry point rather than prove.Green directly:
	// this way the control meets the checks the run does, including the
	// hand-written ones below, which no caller can bind for themselves.
	boundedtest.GreenMixed(t, suite.Subject[boundedtest.Mixed]{
		Name: "a bounded list that keeps the newest",
		New: func(testing.TB) boundedtest.Mixed {
			return newestFirst(boundedtest.MixedSuite.DeclaredLimit())
		},
	}, mixedChecks)
}

// newestFirst answers the most recent items rather than the earliest,
// which the bound permits and this package's own subject does not do.
type newestFirstList struct {
	capacity int
	items    []string
}

func newestFirst(capacity int) *newestFirstList {
	return &newestFirstList{capacity: capacity}
}

func (n *newestFirstList) Add(ctx context.Context, item string) error {
	if ctx == nil {
		return errors.New("boundedtest: nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	n.items = append(n.items, item)
	return nil
}

// List answers the tail rather than the head. Same count, different
// survivors — legal, and the whole point of the control.
func (n *newestFirstList) List(ctx context.Context) ([]string, error) {
	if ctx == nil {
		return nil, errors.New("boundedtest: nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(n.items) <= n.capacity {
		return slices.Clone(n.items), nil
	}
	return slices.Clone(n.items[len(n.items)-n.capacity:]), nil
}
