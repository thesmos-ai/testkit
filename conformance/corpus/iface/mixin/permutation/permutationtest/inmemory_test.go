// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `permutation` is the model tier's: that a drain is a rearrangement of what
// went in is a claim about a whole history, and comparing two multisets needs
// one. The row below is what a single subject settles.
package permutationtest_test

import (
	"context"
	"maps"
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/permutation"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/permutation/permutationtest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	permutationtest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	permutationtest.RunMixed(t,
		inMemory("in-memory"),
		permutationtest.MixedSuite.Without(permutationtest.MixedSuite.Checks.Add.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	permutationtest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) permutationtest.MixedHarness[*permutationtest.InMemory] {
	return permutationtest.MixedHarness[*permutationtest.InMemory]{
		Name: name, New: permutationtest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = permutationtest.MixedChecks{
	{
		Method: "Items", Name: "yields-every-append-once",
		Claim: "Items yields exactly what Add accepted, in some order",
		Run:   yieldsEveryAppendOnce,
		ProvenBy: permutationtest.BrokenMixed(
			"a collection that overwrites an element sharing a key", newOverwritesByKey,
		),
		ProvenReason: "a repeat is a second element",
	},
}

// --- Bodies -------------------------------------------------------------------

// yieldsEveryAppendOnce puts the same key in twice and a second key that
// sorts ahead of it.
//
// The repeat is what makes the claim testable: a permutation is of what went
// IN, so two appends of one key are two elements and a collection folding
// them into one has lost an element it accepted. The out-of-order second key
// is what makes the ordering observable — with one element every order is
// the right one, and a subject answering in arrival order would pass.
func yieldsEveryAppendOnce(
	tb testing.TB, s permutation.Mixed, _ permutationtest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Add(tb.Context(), permutation.Value{Key: lateKey, Body: "last"}),
		"an element is accepted")
	testkit.NoError(tb, s.Add(tb.Context(), permutation.Value{Key: lateKey, Body: "last"}),
		"and so is the same key again")
	testkit.NoError(tb, s.Add(tb.Context(), permutation.Value{Key: earlyKey, Body: "first"}),
		"and a second that sorts ahead of it")

	got, err := s.Items(tb.Context())
	testkit.NoError(tb, err, "the drain succeeds")
	testkit.Equal(tb, len(got), 3, "a repeat is a second element, not the same one again")
	testkit.Equal(tb, got[0].Key, earlyKey, "and the drain is ordered rather than arbitrary")
}

// --- Planted defects ----------------------------------------------------------

// The two keys the row adds, named so the ordering claim reads as one: earlyKey
// sorts ahead of lateKey, and the row adds them the other way round.
const (
	earlyKey = "aa"
	lateKey  = "zz"
)

// overwritesByKey folds an element into an earlier one sharing its key, so
// its drain is a permutation of something smaller than what went in. It gets
// BOTH halves wrong at once, which is right for this row: the count and the
// order are one claim about what a drain reports, and a defect wrong about
// only one would leave the other unproven.
type overwritesByKey struct{ items map[string]permutation.Value }

func newOverwritesByKey() *overwritesByKey {
	return &overwritesByKey{items: map[string]permutation.Value{}}
}

func (k *overwritesByKey) Add(_ context.Context, v permutation.Value) error {
	k.items[v.Key] = v
	return nil
}

func (k *overwritesByKey) Items(context.Context) ([]permutation.Value, error) {
	return slices.Collect(maps.Values(k.items)), nil
}
