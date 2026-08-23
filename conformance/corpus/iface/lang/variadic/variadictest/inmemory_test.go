// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// A variadic parameter is one fixture field, so a generated check witnesses one
// element. Everything about *several* is the author's to state, which is what
// the rows below are.
package variadictest_test

import (
	"context"
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/variadic"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/variadic/variadictest"
)

// TestFinderContract runs the generated checks and this package's own.
func TestFinderContract(t *testing.T) {
	t.Parallel()

	variadictest.RunFinder(t, inMemory("in-memory"), finderChecks)
}

// TestFinderContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestFinderContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	variadictest.RunFinder(t,
		inMemory("in-memory"),
		variadictest.FinderSuite.Without(variadictest.FinderSuite.Checks.Find.Smoke()),
	)
}

// TestFinderChecksCanFail drives every row against its planted defect.
func TestFinderChecksCanFail(t *testing.T) {
	t.Parallel()

	variadictest.ProveFinder(t, inMemory("in-memory"), finderChecks)
}

// --- Harnesses ---------------------------------------------------------------

// The two answers a seeded finder holds, in the order a lookup of both asks
// for them: the rows are about *several*, so one seeded key would leave every
// ordering claim vacuous.
const (
	firstAnswer  = "first"
	secondAnswer = "second"
)

func inMemory(name string) variadictest.FinderHarness[*variadictest.InMemory] {
	return variadictest.FinderHarness[*variadictest.InMemory]{Name: name, New: seeded}
}

// seeded holds both fixture keys, so a lookup of several has several to find.
func seeded() *variadictest.InMemory {
	fx := variadictest.DefaultFinderFixture()
	s := variadictest.NewInMemory()
	s.Put(fx.Keys(), firstAnswer)
	s.Put(fx.KeysOther(), secondAnswer)
	return s
}

// --- The checks: claims, bodies and defects, by name --------------------------

var finderChecks = variadictest.FinderChecks{
	{
		Method: "Find", Name: "one-result-per-held-key",
		Claim: "Find returns one result per key it holds, in order",
		Run:   oneResultPerHeldKey,
		ProvenBy: variadictest.BrokenFinder(
			"a finder that answers both keys in the wrong order", planted(answersOutOfOrder),
		),
		ProvenReason: "in the order asked",
	},

	{
		Method: "Find", Name: "empty-lookup-is-reported",
		Claim: "Find reports a lookup with nothing to look up",
		Run:   emptyLookupIsReported,
		ProvenBy: variadictest.BrokenFinder(
			"a finder that reads no keys as no results", planted(acceptsAnEmptyLookup),
		),
		ProvenReason: "the empty variadic form",
	},

	{
		Method: "FindWithLimit", Name: "truncates-to-the-limit",
		Claim: "FindWithLimit truncates to the limit",
		Run:   truncatesToTheLimit,
		ProvenBy: variadictest.BrokenFinder(
			"a finder that answers every key it was asked about", planted(ignoresTheLimit),
		),
		ProvenReason: "no more than the limit",
	},
}

// --- Bodies -------------------------------------------------------------------

func oneResultPerHeldKey(
	tb testing.TB, s variadic.Finder, fx variadictest.FinderFixture,
) {
	tb.Helper()
	got, err := s.Find(tb.Context(), fx.Keys(), fx.KeysOther())
	testkit.NoError(tb, err, "a lookup of two held keys succeeds")
	testkit.Equal(tb, got, []string{firstAnswer, secondAnswer},
		"several keys are answered in the order asked")
}

func emptyLookupIsReported(
	tb testing.TB, s variadic.Finder, _ variadictest.FinderFixture,
) {
	tb.Helper()
	_, err := s.Find(tb.Context())
	testkit.ErrorIs(tb, err, variadictest.ErrNoKeys,
		"the empty variadic form is the one a derived check cannot reach")
}

func truncatesToTheLimit(
	tb testing.TB, s variadic.Finder, fx variadictest.FinderFixture,
) {
	tb.Helper()
	got, err := s.FindWithLimit(tb.Context(), 1, fx.Keys(), fx.KeysOther())
	testkit.NoError(tb, err, "a limited lookup succeeds")
	testkit.Len(tb, got, 1, "and returns no more than the limit")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted finder gets wrong.
//
// Each answers rather than failing: a defect that returned an error would red
// its row on the NoError above, which every row shares, and a red they all
// share is evidence for none of them.
type fault int

const (
	// answersOutOfOrder finds both keys and hands them back the other way
	// round, which is what looking up through an unordered map does.
	answersOutOfOrder fault = iota

	// acceptsAnEmptyLookup reads a call with no keys as a lookup that found
	// nothing, which a caller cannot tell from a real empty result.
	acceptsAnEmptyLookup

	// ignoresTheLimit answers every key it was asked about, whatever
	// ceiling the call named.
	ignoresTheLimit
)

// planted builds the constructor for one broken finder, holding what the
// harness's own subject holds so a lookup has the same thing to find.
func planted(wrong fault) func() *plantedFinder {
	return func() *plantedFinder {
		fx := variadictest.DefaultFinderFixture()
		return &plantedFinder{
			wrong: wrong,
			held:  map[string]string{fx.Keys(): firstAnswer, fx.KeysOther(): secondAnswer},
		}
	}
}

type plantedFinder struct {
	wrong fault
	held  map[string]string
}

func (f *plantedFinder) Find(_ context.Context, keys ...string) ([]string, error) {
	if len(keys) == 0 {
		if f.wrong == acceptsAnEmptyLookup {
			return nil, nil
		}
		return nil, variadictest.ErrNoKeys
	}
	found := f.lookup(keys)
	if f.wrong == answersOutOfOrder {
		slices.Reverse(found)
	}
	return found, nil
}

func (f *plantedFinder) FindWithLimit(
	_ context.Context, limit int, keys ...string,
) ([]string, error) {
	if len(keys) == 0 {
		return nil, variadictest.ErrNoKeys
	}
	found := f.lookup(keys)
	if f.wrong == ignoresTheLimit {
		return found, nil
	}
	return found[:min(len(found), limit)], nil
}

func (f *plantedFinder) lookup(keys []string) []string {
	found := make([]string, 0, len(keys))
	for _, k := range keys {
		if v, held := f.held[k]; held {
			found = append(found, v)
		}
	}
	return found
}
