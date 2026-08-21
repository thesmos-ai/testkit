// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `permutation` is the model tier's: that a drain is a rearrangement of what
// went in is a claim about a whole history, and comparing two multisets needs
// one. The row below is what a single subject settles.
package permutationtest_test

import (
	"context"
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

	permutationtest.ProveMixed(t, mixedChecks)
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
		Method: "Items", Name: "yields-each-append-once",
		Claim: "Items yields what Add put in, once each",
		Run:   yieldsEachAppendOnce,
		ProvenBy: permutationtest.BrokenMixed(
			"a collection that drains its appends as they arrived", newKeepsTheRepeat,
		),
		ProvenReason: "the repeated key is one element",
	},
}

// --- Bodies -------------------------------------------------------------------

// yieldsEachAppendOnce puts the same key in twice and a second key that sorts
// ahead of it.
//
// The repeat is what makes the dedup claim testable: a drain that yielded it
// twice would be reporting its input rather than its contents. The out-of-order
// second key is what makes the ordering observable — with one element every
// order is the right one, and a subject returning map order would pass.
func yieldsEachAppendOnce(
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
	testkit.Equal(tb, len(got), 2, "the repeated key is one element")
	testkit.Equal(tb, got[0].Key, earlyKey, "and the drain is ordered rather than arbitrary")
}

// --- Planted defects ----------------------------------------------------------

// The two keys the row adds, named so the ordering claim reads as one: earlyKey
// sorts ahead of lateKey, and the row adds them the other way round.
const (
	earlyKey = "aa"
	lateKey  = "zz"
)

// keepsTheRepeat stores every append and drains them in the order they arrived,
// which is what a collection backed by a plain slice does. It gets BOTH halves
// wrong at once, which is right for this row: the two are one claim about what
// a drain reports, and a defect wrong about only one would leave the other
// unproven.
type keepsTheRepeat struct{ items []permutation.Value }

func newKeepsTheRepeat() *keepsTheRepeat { return &keepsTheRepeat{} }

func (k *keepsTheRepeat) Add(_ context.Context, v permutation.Value) error {
	k.items = append(k.items, v)
	return nil
}

func (k *keepsTheRepeat) Items(context.Context) ([]permutation.Value, error) {
	return slices.Clone(k.items), nil
}
