// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `pointintime` is the model tier's: that a read pinned to an instant answers
// what was true then is a claim about a history and a clock. The row below is
// what one write and one read settle.
package pointintimetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/pointintime"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/pointintime/pointintimetest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	pointintimetest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	pointintimetest.RunMixed(t,
		inMemory("in-memory"),
		pointintimetest.MixedSuite.Without(pointintimetest.MixedSuite.Checks.Store.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	pointintimetest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) pointintimetest.MixedHarness[*pointintimetest.InMemory] {
	return pointintimetest.MixedHarness[*pointintimetest.InMemory]{
		Name: name, New: pointintimetest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = pointintimetest.MixedChecks{
	{
		Method: "Get", Name: "reads-back-what-store-wrote",
		Claim: "Get returns what Store wrote",
		Run:   readsBackWhatStoreWrote,
		ProvenBy: pointintimetest.BrokenMixed(
			"a store whose reads answer a value with no key", newAnswersTheZeroKey,
		),
		ProvenReason: "answers under the key it was stored with",
	},
}

// --- Bodies -------------------------------------------------------------------

func readsBackWhatStoreWrote(
	tb testing.TB, s pointintime.Mixed, fx pointintimetest.MixedFixture,
) {
	tb.Helper()
	written := fx.Value()
	testkit.NoError(tb, s.Store(tb.Context(), written), "the value is stored")

	got, err := s.Get(tb.Context(), written.Key)
	testkit.NoError(tb, err, "the written key is present")
	testkit.Equal(tb, got.Key, written.Key,
		"and Get answers under the key it was stored with")
}

// --- Planted defects ----------------------------------------------------------

// answersTheZeroKey finds the record and hands back a value whose key slot was
// never filled in, which a caller pinning reads to an instant reads as a record
// belonging to no key at all.
type answersTheZeroKey struct{ held map[string]pointintime.Value }

func newAnswersTheZeroKey() *answersTheZeroKey {
	return &answersTheZeroKey{held: map[string]pointintime.Value{}}
}

func (a *answersTheZeroKey) Store(_ context.Context, v pointintime.Value) error {
	a.held[v.Key] = v
	return nil
}

func (a *answersTheZeroKey) Get(
	_ context.Context, key string,
) (pointintime.Value, error) {
	v, held := a.held[key]
	if !held {
		return pointintime.Value{}, pointintimetest.ErrNotFound
	}
	v.Key = ""
	return v, nil
}
