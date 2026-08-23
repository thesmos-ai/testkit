// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// monotonicreads is the model tier's: that a reader never goes backwards is a
// claim about a sequence of reads, which needs a generated history. What one
// write and one read settle is the row below.
package monotonicreadstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/monotonicreads"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/monotonicreads/monotonicreadstest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	monotonicreadstest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	monotonicreadstest.RunMixed(t,
		inMemory("in-memory"),
		monotonicreadstest.MixedSuite.Without(monotonicreadstest.MixedSuite.Checks.Store.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	monotonicreadstest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) monotonicreadstest.MixedHarness[*monotonicreadstest.InMemory] {
	return monotonicreadstest.MixedHarness[*monotonicreadstest.InMemory]{
		Name: name, New: monotonicreadstest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = monotonicreadstest.MixedChecks{
	{
		Method: "Get", Name: "reads-back-what-store-wrote",
		Claim: "Get returns what Store wrote",
		Run:   readsBackWhatStoreWrote,
		ProvenBy: monotonicreadstest.BrokenMixed(
			"a store whose reads answer a value with no key", newAnswersTheZeroKey,
		),
		ProvenReason: "answers under the key it was stored with",
	},
}

// --- Bodies -------------------------------------------------------------------

func readsBackWhatStoreWrote(
	tb testing.TB, s monotonicreads.Mixed, fx monotonicreadstest.MixedFixture,
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
// never filled in, which a caller matching answers to requests by key reads as
// a record belonging to nothing.
type answersTheZeroKey struct {
	held map[string]monotonicreads.Value
}

func newAnswersTheZeroKey() *answersTheZeroKey {
	return &answersTheZeroKey{held: map[string]monotonicreads.Value{}}
}

func (a *answersTheZeroKey) Store(_ context.Context, v monotonicreads.Value) error {
	a.held[v.Key] = v
	return nil
}

func (a *answersTheZeroKey) Get(
	_ context.Context, key string,
) (monotonicreads.Value, error) {
	v, held := a.held[key]
	if !held {
		return monotonicreads.Value{}, monotonicreadstest.ErrNotFound
	}
	v.Key = ""
	return v, nil
}
