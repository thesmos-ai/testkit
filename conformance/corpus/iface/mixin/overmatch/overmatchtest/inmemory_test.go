// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `overmatch` is the model tier's: that a query answers no more than it should
// is a claim about a generated corpus and a generated query. What one subject
// settles is the row below — a repeated append is one element, in order.
package overmatchtest_test

import (
	"context"
	"maps"
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/overmatch"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/overmatch/overmatchtest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	overmatchtest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	overmatchtest.RunMixed(t,
		inMemory("in-memory"),
		overmatchtest.MixedSuite.Without(overmatchtest.MixedSuite.Checks.Add.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	overmatchtest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) overmatchtest.MixedHarness[*overmatchtest.InMemory] {
	return overmatchtest.MixedHarness[*overmatchtest.InMemory]{
		Name: name, New: overmatchtest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = overmatchtest.MixedChecks{
	{
		Method: "Items", Name: "yields-every-append",
		Claim: "Items yields everything Add accepted, and may yield more",
		Run:   yieldsEveryAppend,
		ProvenBy: overmatchtest.BrokenMixed(
			"a collection that overwrites an element sharing a key", newOverwritesByKey,
		),
		ProvenReason: "a repeat is a second element",
	},
}

// --- Bodies -------------------------------------------------------------------

// yieldsEveryAppend puts the same key in twice and a second key that sorts
// ahead of it.
//
// The repeat is what makes the claim testable in the direction this mixin
// points: overmatch permits a drain to yield MORE than was asked of it and
// forbids it yielding less, so a collection that folded the repeat into one
// element would have dropped something it accepted. The out-of-order second
// key is what makes the ordering observable — with one element every order
// is the right one, and a subject answering in arrival order would pass.
func yieldsEveryAppend(
	tb testing.TB, s overmatch.Mixed, _ overmatchtest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Add(tb.Context(), overmatch.Value{Key: lateKey, Body: "last"}),
		"an element is accepted")
	testkit.NoError(tb, s.Add(tb.Context(), overmatch.Value{Key: lateKey, Body: "last"}),
		"and so is the same key again")
	testkit.NoError(tb, s.Add(tb.Context(), overmatch.Value{Key: earlyKey, Body: "first"}),
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

// overwritesByKey folds an element into an earlier one sharing its key,
// which is a collection that drops what it accepted. That is the direction
// overmatch forbids: yielding more than was asked is permitted, and this
// yields less.
type overwritesByKey struct{ items map[string]overmatch.Value }

func newOverwritesByKey() *overwritesByKey {
	return &overwritesByKey{items: map[string]overmatch.Value{}}
}

func (k *overwritesByKey) Add(_ context.Context, v overmatch.Value) error {
	k.items[v.Key] = v
	return nil
}

func (k *overwritesByKey) Items(context.Context) ([]overmatch.Value, error) {
	return slices.Collect(maps.Values(k.items)), nil
}
