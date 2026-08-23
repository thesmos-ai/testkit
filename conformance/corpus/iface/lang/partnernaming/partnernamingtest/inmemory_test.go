// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// partnernaming is the language axis's: which effect/observer pair a
// generator can write a check for from the source alone. Two of the three
// here it cannot, and the generated header says which and why.
//
// The row below is the one the generator does NOT write and this package
// must: Emit files under a bucket the signature never mentions, so the
// pairing is the domain's and only a hand-written check can state it.
package partnernamingtest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/partnernaming"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/partnernaming/partnernamingtest"
)

// TestStoreContract runs the generated checks and this package's own.
func TestStoreContract(t *testing.T) {
	t.Parallel()

	partnernamingtest.RunStore(t, inMemory("in-memory"), storeChecks)
}

// TestStoreChecksCanFail drives every planted defect through the check it is
// evidence for.
func TestStoreChecksCanFail(t *testing.T) {
	t.Parallel()

	partnernamingtest.ProveStore(t, inMemory("in-memory"), storeChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) partnernamingtest.StoreHarness[*partnernamingtest.InMemory] {
	return partnernamingtest.StoreHarness[*partnernamingtest.InMemory]{
		Name: name, New: partnernamingtest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var storeChecks = partnernamingtest.StoreChecks{
	{
		Method: "Emit", Name: "lands-in-its-bucket",
		Claim: "Emit files the identifier under the bucket the domain assigns it",
		Run:   landsInItsBucket,
		ProvenBy: partnernamingtest.BrokenStore(
			"a store whose emit records nothing", newEmitsNothing,
		),
		ProvenReason: "the bucket holds it",
	},
}

// --- Bodies -------------------------------------------------------------------

// landsInItsBucket states the pairing the generator refused.
//
// Refused for an honest reason: a generated check receives Emit's arguments
// and nothing else, and Count takes an int no argument of Emit's carries.
// Which bucket an identifier belongs to is a fact about this store, so this
// is where it gets written down.
func landsInItsBucket(
	tb testing.TB, s partnernaming.Store, _ partnernamingtest.StoreFixture,
) {
	tb.Helper()
	const id = "abcd"
	before, err := s.Count(tb.Context(), len(id))
	testkit.NoError(tb, err, "the bucket is readable before the emit")

	testkit.NoError(tb, s.Emit(tb.Context(), id), "the emit is accepted")

	after, err := s.Count(tb.Context(), len(id))
	testkit.NoError(tb, err, "and readable after it")
	testkit.Equal(tb, after, before+1, "and the bucket holds it")
}

// --- Planted defects ----------------------------------------------------------

// emitsNothing accepts the emit and files nothing, which is the effect whose
// observer never sees it — the whole failure the sideeffect claim is about,
// on the one pair no generated check covers.
type emitsNothing struct{ partnernaming.Store }

func newEmitsNothing() *emitsNothing {
	return &emitsNothing{Store: partnernamingtest.NewInMemory()}
}

func (*emitsNothing) Emit(context.Context, string) error { return nil }
