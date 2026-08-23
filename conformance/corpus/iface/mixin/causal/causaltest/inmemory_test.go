// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// causal is the model tier's: ordering is a claim about a sequence of writes
// and the tokens threaded between them, which needs generated histories to
// state. What one subject settles is the row below — that the acknowledgement
// names what it acknowledged, so anything downstream has something to order
// against.
package causaltest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/causal"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/causal/causaltest"
	"go.thesmos.sh/testkit/engine/model/law"
	"go.thesmos.sh/testkit/engine/suite"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	causaltest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	causaltest.RunMixed(t,
		inMemory("in-memory"),
		causaltest.MixedSuite.Without(causaltest.MixedSuite.Checks.Store.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	causaltest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) causaltest.MixedHarness[*causaltest.InMemory] {
	return causaltest.MixedHarness[*causaltest.InMemory]{
		Name: name, New: causaltest.NewInMemory,
		// AUTO-CAUSAL-ORDERING needs the causal relation over writes, and
		// no shape yields it: the directive names Rev as the stamp, and
		// what a stamp ORDERS is the declaration's to say.
		Provide: map[suite.Capability]any{"happensBefore": storeOrder},
	}
}

// storeOrder is the causal relation Rev carries.
//
// One store assigns every revision from one counter, so its writes are
// totally ordered and an earlier revision causally precedes a later one.
// That relation is transitively closed, which is what the checker needs
// and does not compute for itself. A store whose stamp were per-key, or
// a vector clock, would need a different answer here — which is why the
// law asks rather than deriving it.
func storeOrder(a, b law.ClientOp[string]) bool { return a.Version < b.Version }

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = causaltest.MixedChecks{
	{
		Method: "Get", Name: "reads-back-what-store-wrote",
		Claim: "Get returns what Store wrote",
		Run:   readsBackWhatStoreWrote,
		ProvenBy: causaltest.BrokenMixed(
			"a store whose acknowledgement names no key", newLosesTheKeyOnStore,
		),
		ProvenReason: "under the key it was given",
	},
}

// --- Bodies -------------------------------------------------------------------

func readsBackWhatStoreWrote(
	tb testing.TB, s causal.Mixed, fx causaltest.MixedFixture,
) {
	tb.Helper()
	written := fx.Value()
	// Store answers the state it wrote beside its error, which is what
	// makes this an answering writer rather than a plain one.
	stored, err := s.Store(tb.Context(), written)
	testkit.NoError(tb, err, "the value is stored")
	testkit.Equal(tb, stored.Key, written.Key, "under the key it was given")

	got, err := s.Get(tb.Context(), written.Key)
	testkit.NoError(tb, err, "the written key is present")
	testkit.Equal(tb, got.Key, written.Key,
		"and Get answers under the key it was stored with")
}

// --- Planted defects ----------------------------------------------------------

// losesTheKeyOnStore writes correctly and answers an acknowledgement with the
// key slot empty, which leaves a caller holding a receipt for something it
// cannot name — and nothing downstream able to order against it.
type losesTheKeyOnStore struct{ held map[string]causal.Value }

func newLosesTheKeyOnStore() *losesTheKeyOnStore {
	return &losesTheKeyOnStore{held: map[string]causal.Value{}}
}

func (l *losesTheKeyOnStore) Store(_ context.Context, v causal.Value) (causal.Value, error) {
	l.held[v.Key] = v
	return causal.Value{}, nil
}

func (l *losesTheKeyOnStore) Get(_ context.Context, key string) (causal.Value, error) {
	v, held := l.held[key]
	if !held {
		return causal.Value{}, causaltest.ErrNotFound
	}
	return v, nil
}
