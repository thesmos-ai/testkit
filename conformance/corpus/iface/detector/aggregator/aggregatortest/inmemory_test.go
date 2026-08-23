// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// A nullary reduction: nothing to vary, so nothing to choose, and the derived
// family stops at the smoke call. That the number means anything is the row's.
package aggregatortest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/aggregator"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/aggregator/aggregatortest"
)

// TestAggregatorContract runs the generated checks and this package's own.
func TestAggregatorContract(t *testing.T) {
	t.Parallel()

	aggregatortest.RunAggregator(t, inMemory("in-memory"), aggregatorChecks)
}

// TestAggregatorContractWithoutSmoke drops a check through the typed index
// rather than a string, so a check that is renamed or stops being emitted
// breaks this compile instead of silently declining nothing.
func TestAggregatorContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	aggregatortest.RunAggregator(t,
		inMemory("in-memory"),
		aggregatortest.AggregatorSuite.Without(aggregatortest.AggregatorSuite.Checks.Count.Smoke()),
	)
}

// TestAggregatorChecksCanFail drives the row against its planted defect.
func TestAggregatorChecksCanFail(t *testing.T) {
	t.Parallel()

	aggregatortest.ProveAggregator(t, inMemory("in-memory"), aggregatorChecks)
}

// --- Harnesses ---------------------------------------------------------------

// seededItem is the one thing the constructor puts there, so the count has
// something to be wrong about.
const seededItem = "seeded"

// inMemory seeds the subject: Aggregator declares no writer, so nothing is
// derived to seed through and the seed is the constructor's.
func inMemory(name string) aggregatortest.AggregatorHarness[*aggregatortest.InMemory] {
	return aggregatortest.AggregatorHarness[*aggregatortest.InMemory]{Name: name, New: seeded}
}

func seeded() *aggregatortest.InMemory {
	s := aggregatortest.NewInMemory()
	s.Add(seededItem)
	return s
}

// --- The checks: claims, bodies and defects, by name --------------------------

var aggregatorChecks = aggregatortest.AggregatorChecks{
	{
		Method: "Count", Name: "counts-what-it-holds",
		Claim: "Count counts what the collection holds",
		Run:   countsWhatItHolds,
		ProvenBy: aggregatortest.BrokenAggregator(
			"an aggregator that always reports an empty collection", newAlwaysZero,
		),
		ProvenReason: "reports what the constructor put there",
	},
}

// --- Bodies -------------------------------------------------------------------

func countsWhatItHolds(
	tb testing.TB, s aggregator.Aggregator, _ aggregatortest.AggregatorFixture,
) {
	tb.Helper()
	got, err := s.Count(tb.Context())
	testkit.NoError(tb, err, "counting a healthy collection succeeds")
	testkit.Equal(tb, got, 1, "and reports what the constructor put there")
}

// --- Planted defects ----------------------------------------------------------

// alwaysZero answers zero with no error, which the smoke call cannot tell from
// a correct reduction of an empty collection — and which is why the row seeds
// one item rather than none.
type alwaysZero struct{}

func newAlwaysZero() alwaysZero { return alwaysZero{} }

func (alwaysZero) Count(context.Context) (int, error) { return 0, nil }
