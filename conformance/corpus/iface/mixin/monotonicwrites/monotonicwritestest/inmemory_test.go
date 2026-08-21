// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// monotonicwrites is the model tier's: that writes land in the order they were
// issued is a claim about a sequence, and a generated history is what states
// it. The row below is what one write settles — that a read lands on the key
// the write did.
package monotonicwritestest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/monotonicwrites"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/monotonicwrites/monotonicwritestest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	monotonicwritestest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	monotonicwritestest.RunMixed(t,
		inMemory("in-memory"),
		monotonicwritestest.MixedSuite.Without(monotonicwritestest.MixedSuite.Checks.Store.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	monotonicwritestest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) monotonicwritestest.MixedHarness[*monotonicwritestest.InMemory] {
	return monotonicwritestest.MixedHarness[*monotonicwritestest.InMemory]{
		Name: name, New: monotonicwritestest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = monotonicwritestest.MixedChecks{
	{
		Method: "Get", Name: "reads-back-what-store-wrote",
		Claim: "Get returns what Store wrote",
		Run:   readsBackWhatStoreWrote,
		ProvenBy: monotonicwritestest.BrokenMixed(
			"a store whose reads land on a key of their own", newAnswersADifferentKey,
		),
		ProvenReason: "answers under the key it was stored with",
	},
}

// --- Bodies -------------------------------------------------------------------

func readsBackWhatStoreWrote(
	tb testing.TB, s monotonicwrites.Mixed, fx monotonicwritestest.MixedFixture,
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

// answersADifferentKey stores under the key it was given and answers a value
// carrying another, which is a store whose read path and write path derive the
// key differently — the one bug an ordering law over a single key never sees.
type answersADifferentKey struct {
	held map[string]monotonicwrites.Value
}

func newAnswersADifferentKey() *answersADifferentKey {
	return &answersADifferentKey{held: map[string]monotonicwrites.Value{}}
}

func (a *answersADifferentKey) Store(
	_ context.Context, v monotonicwrites.Value,
) (monotonicwrites.Value, error) {
	a.held[v.Key] = v
	return v, nil
}

func (a *answersADifferentKey) Get(
	_ context.Context, key string,
) (monotonicwrites.Value, error) {
	v, held := a.held[key]
	if !held {
		return monotonicwrites.Value{}, monotonicwritestest.ErrNotFound
	}
	v.Key += "-elsewhere"
	return v, nil
}
