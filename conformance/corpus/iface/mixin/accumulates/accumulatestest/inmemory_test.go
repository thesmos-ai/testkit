// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// The effect axis's two positions in one file, split across two tiers.
//
// Add/accumulates is derived and says what one subject and a fixed sequence
// settle: the second call is taken rather than refused. That is the half a
// coalescing store gets wrong first, and it is all the generated check can
// say — the mixin names no observer, and nothing in Add's signature points at
// Total.
//
// The compounding itself is the first row below. It needs something to read the
// effect through, which is the same reason `idempotent`'s real law is the model
// tier's rather than its repeat probe.
package accumulatestest_test

import (
	"context"
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/accumulates"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/accumulates/accumulatestest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	accumulatestest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	accumulatestest.RunMixed(t,
		inMemory("in-memory"),
		accumulatestest.MixedSuite.Without(accumulatestest.MixedSuite.Checks.Add.Smoke()),
	)
}

// TestMixedChecksCanFail drives every row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	accumulatestest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

// amount is what each addition adds. Both additions add the SAME amount, so a
// store that replaced rather than compounded answers a number the row can name
// — half the total — rather than one that only happens to differ.
const amount = 3

func inMemory(name string) accumulatestest.MixedHarness[*accumulatestest.InMemory] {
	return accumulatestest.MixedHarness[*accumulatestest.InMemory]{
		Name: name, New: accumulatestest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = accumulatestest.MixedChecks{
	{
		Method: "Total", Name: "two-adds-compound",
		Claim: "Total reports the sum of the additions rather than the last one",
		Run:   twoAddsCompound,
		ProvenBy: accumulatestest.BrokenMixed(
			"a store where a second addition replaces the first",
			planted(replacesRatherThanAdds),
		),
		ProvenReason: "counts both additions",
	},

	{
		Method: "Total", Name: "unadded-key-totals-zero",
		Claim: "Total reports zero for a key nothing added to",
		Run:   unaddedKeyTotalsZero,
		ProvenBy: accumulatestest.BrokenMixed(
			"a store that reports a key nothing added to as missing",
			planted(refusesAnUnaddedKey),
		),
		ProvenReason: "an unadded key is not an error",
	},
}

// --- Bodies -------------------------------------------------------------------

// twoAddsCompound is the whole of the mixin, and the assertion that separates
// it from idempotent over the same two methods: a store that replaced would
// answer the amount once.
func twoAddsCompound(
	tb testing.TB, s accumulates.Mixed, fx accumulatestest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Add(tb.Context(), fx.Key(), amount), "the first addition lands")
	testkit.NoError(tb, s.Add(tb.Context(), fx.Key(), amount), "and so does the second")

	got, err := s.Total(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "the total is readable")
	testkit.Equal(tb, got, 2*amount, "and counts both additions")
}

// unaddedKeyTotalsZero is the sum of no additions, which is an answer rather
// than a failure — and what makes Total/miss's zero the right claim rather than
// a sentinel.
func unaddedKeyTotalsZero(
	tb testing.TB, s accumulates.Mixed, fx accumulatestest.MixedFixture,
) {
	tb.Helper()
	got, err := s.Total(tb.Context(), fx.KeyOther())
	testkit.NoError(tb, err, "an unadded key is not an error")
	testkit.Equal(tb, got, 0, "it simply has nothing summed")
}

// --- Planted defects ----------------------------------------------------------

// errUnadded is what refusesAnUnaddedKey answers with. The interface declares
// no sentinel for it — which is the point of the row: an unadded key has an
// answer rather than a failure, so nothing here should ever need one.
var errUnadded = errors.New("accumulatestest_test: nothing was added under that key")

// fault names what one planted store gets wrong.
type fault int

const (
	// replacesRatherThanAdds keeps the last amount instead of the sum,
	// which is `idempotent` wearing this mixin's name — and the confusion
	// the first row exists to settle.
	replacesRatherThanAdds fault = iota

	// refusesAnUnaddedKey treats a key nothing added to as missing, which
	// makes every caller check an error where the answer is zero.
	refusesAnUnaddedKey
)

// planted builds the constructor for one broken store.
func planted(wrong fault) func() *plantedStore {
	return func() *plantedStore {
		return &plantedStore{wrong: wrong, held: map[string]int{}}
	}
}

type plantedStore struct {
	wrong fault
	held  map[string]int
}

func (p *plantedStore) Add(_ context.Context, key string, by int) error {
	if p.wrong == replacesRatherThanAdds {
		p.held[key] = by
		return nil
	}
	p.held[key] += by
	return nil
}

func (p *plantedStore) Total(_ context.Context, key string) (int, error) {
	total, added := p.held[key]
	if !added && p.wrong == refusesAnUnaddedKey {
		return 0, errUnadded
	}
	return total, nil
}
