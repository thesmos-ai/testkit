// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `indexed by=Len` names the sizer a position is meaningful against, and the
// row below walks the boundary it fixes: one past the last element is not a
// position, and the last element is.
package indexedtest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/indexed"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/indexed/indexedtest"
)

// TestRankedContract runs the generated checks and this package's own.
func TestRankedContract(t *testing.T) {
	t.Parallel()

	indexedtest.RunRanked(t, inMemory("in-memory"), rankedChecks)
}

// TestRankedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestRankedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	indexedtest.RunRanked(t,
		inMemory("in-memory"),
		indexedtest.RankedSuite.Without(indexedtest.RankedSuite.Checks.Add.Smoke()),
	)
}

// TestRankedChecksCanFail drives the row against its planted defect.
func TestRankedChecksCanFail(t *testing.T) {
	t.Parallel()

	indexedtest.ProveRanked(t, rankedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) indexedtest.RankedHarness[*indexedtest.InMemory] {
	return indexedtest.RankedHarness[*indexedtest.InMemory]{
		Name: name, New: indexedtest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var rankedChecks = indexedtest.RankedChecks{
	{
		Method: "At", Name: "one-past-the-end-is-not-a-position",
		Claim: "At misses a position the collection does not hold",
		Run:   onePastTheEndIsNotAPosition,
		ProvenBy: indexedtest.BrokenRanked(
			"a collection that clamps an out-of-range position to its last", newClampsToTheLast,
		),
		ProvenReason: "one past the last element is not a position",
	},
}

// --- Bodies -------------------------------------------------------------------

// onePastTheEndIsNotAPosition adds first, so the boundary it walks to has an
// element on the inside of it as well as the outside — a collection that
// refused everything would satisfy half of this and hold nothing.
func onePastTheEndIsNotAPosition(
	tb testing.TB, s indexed.Ranked, fx indexedtest.RankedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Add(tb.Context(), fx.Value()), "an element is added")

	n, err := s.Len(tb.Context())
	testkit.NoError(tb, err, "the size is readable")
	testkit.Equal(tb, n, 1, "and counts what was added")

	_, err = s.At(tb.Context(), n)
	testkit.ErrorIs(tb, err, indexedtest.ErrOutOfRange,
		"one past the last element is not a position")

	_, err = s.At(tb.Context(), n-1)
	testkit.NoError(tb, err, "the last element is")
}

// --- Planted defects ----------------------------------------------------------

// clampsToTheLast answers the final element for any position past the end,
// which is the off-by-one that never crashes and quietly duplicates the tail
// for every caller walking to Len inclusive.
type clampsToTheLast struct{ items []indexed.Value }

func newClampsToTheLast() *clampsToTheLast { return &clampsToTheLast{} }

func (c *clampsToTheLast) Add(_ context.Context, v indexed.Value) error {
	c.items = append(c.items, v)
	return nil
}

func (c *clampsToTheLast) Len(context.Context) (int, error) { return len(c.items), nil }

func (c *clampsToTheLast) At(_ context.Context, i int) (indexed.Value, error) {
	if len(c.items) == 0 {
		return indexed.Value{}, indexedtest.ErrOutOfRange
	}
	if i >= len(c.items) {
		i = len(c.items) - 1
	}
	return c.items[i], nil
}

// TestRankedLawsCanSaturate drives each bound law against defects worn on
// its own methods, with that law as the run's only oracle.
//
// Binding a law is necessary; this is what makes it sufficient. A law
// every worn defect survives is bound and unsaturatable, which reads as
// coverage in the report and is not.
func TestRankedLawsCanSaturate(t *testing.T) {
	t.Parallel()

	indexedtest.RankedModelSaturation(t, func() indexedtest.Ranked { return indexedtest.NewInMemory() })
}
