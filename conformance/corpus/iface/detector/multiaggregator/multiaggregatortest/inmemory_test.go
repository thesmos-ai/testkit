// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Two value slots beside an error and still four checks, not five.
//
// The zero-value check would have more to say here than for the single
// aggregator — two slots can disagree with the error independently — and it is
// still not generated, for the same reason: Stats takes nothing, so the harness
// has no input to choose and could only demand failure from a correct
// implementation. Both halves of the claim are written below instead.
package multiaggregatortest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/multiaggregator"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/multiaggregator/multiaggregatortest"
)

// TestMultiAggregatorContract runs the generated checks and this package's own.
func TestMultiAggregatorContract(t *testing.T) {
	t.Parallel()

	multiaggregatortest.RunMultiAggregator(t, inMemory("in-memory"), multiAggregatorChecks)
}

// TestMultiAggregatorContractWithoutSmoke drops a check through the typed index
// rather than a string, so a check that is renamed or stops being emitted
// breaks this compile instead of silently declining nothing.
func TestMultiAggregatorContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	multiaggregatortest.RunMultiAggregator(t,
		inMemory("in-memory"),
		multiaggregatortest.MultiAggregatorSuite.Without(
			multiaggregatortest.MultiAggregatorSuite.Checks.Stats.Smoke(),
		),
	)
}

// TestMultiAggregatorChecksCanFail drives the row against its planted defect.
func TestMultiAggregatorChecksCanFail(t *testing.T) {
	t.Parallel()

	multiaggregatortest.ProveMultiAggregator(t, inMemory("in-memory"), multiAggregatorChecks)
}

// --- Harnesses ---------------------------------------------------------------

// seededValue is the one number the constructor puts there, so the count and
// the sum are different numbers and a defect cannot satisfy both by accident.
const seededValue = 4

func inMemory(
	name string,
) multiaggregatortest.MultiAggregatorHarness[*multiaggregatortest.InMemory] {
	return multiaggregatortest.MultiAggregatorHarness[*multiaggregatortest.InMemory]{
		Name: name, New: seeded,
	}
}

func seeded() *multiaggregatortest.InMemory {
	s := multiaggregatortest.NewInMemory()
	s.Add(seededValue)
	return s
}

// --- The checks: claims, bodies and defects, by name --------------------------

var multiAggregatorChecks = multiaggregatortest.MultiAggregatorChecks{
	{
		Method: "Stats", Name: "reduces-to-both-numbers",
		Claim: "Stats reduces the collection to both numbers",
		Run:   reducesToBothNumbers,
		ProvenBy: multiaggregatortest.BrokenMultiAggregator(
			"an aggregator that counts and does not add", newCountsOnly,
		),
		ProvenReason: "the sum agrees with it",
	},
}

// --- Bodies -------------------------------------------------------------------

func reducesToBothNumbers(
	tb testing.TB, s multiaggregator.MultiAggregator,
	_ multiaggregatortest.MultiAggregatorFixture,
) {
	tb.Helper()
	count, sum, err := s.Stats(tb.Context())
	testkit.NoError(tb, err, "reducing a healthy collection succeeds")
	testkit.Equal(tb, count, 1, "the count reports what the constructor put there")
	testkit.Equal(tb, sum, seededValue, "and the sum agrees with it")
}

// --- Planted defects ----------------------------------------------------------

// countsOnly reduces one slot and leaves the other at zero, which is the
// failure a two-slot return has and a single-value one cannot: a check reading
// only the count calls it correct.
type countsOnly struct{}

func newCountsOnly() countsOnly { return countsOnly{} }

func (countsOnly) Stats(context.Context) (int, int, error) { return 1, 0, nil }
