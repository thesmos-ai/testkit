// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Both derived values are admitted, so the generated run only ever exercises
// "accepts what Match admits". A stale body is what the contract exists to
// catch, and the row writes the key first: a fresh subject holds nothing to be
// stale against.
package ifmatchtest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	ifmatch "go.thesmos.sh/testkit/conformance/corpus/iface/contract/if-match"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/if-match/ifmatchtest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	ifmatchtest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	ifmatchtest.RunContract(t,
		inMemory("in-memory"),
		ifmatchtest.ContractSuite.Without(ifmatchtest.ContractSuite.Checks.Put.Smoke()),
	)
}

// TestContractChecksCanFail drives the row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	ifmatchtest.ProveContract(t, inMemory("in-memory"), contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) ifmatchtest.ContractHarness[*ifmatchtest.InMemory] {
	return ifmatchtest.ContractHarness[*ifmatchtest.InMemory]{
		Name: name, New: ifmatchtest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = ifmatchtest.ContractChecks{
	{
		Method: "Put", Name: "refuses-a-declined-value",
		Claim: "Put refuses what the predicate declines",
		Run:   refusesADeclinedValue,
		ProvenBy: ifmatchtest.BrokenContract(
			"a store whose write ignores its own predicate", newWritesWhatItDeclines,
		),
		ProvenReason: "the write conditional on it is refused",
	},
}

// --- Bodies -------------------------------------------------------------------

func refusesADeclinedValue(
	tb testing.TB, s ifmatch.Contract, fx ifmatchtest.ContractFixture,
) {
	tb.Helper()
	held := fx.Value()
	testkit.NoError(tb, s.Put(tb.Context(), held), "the key is written")

	stale := ifmatch.Value{Key: held.Key, Body: held.Body + staleSuffix}

	allowed, err := s.Match(tb.Context(), stale)
	testkit.NoError(tb, err, "the predicate answers about a key the store holds")
	testkit.False(tb, allowed, "and declines a body the store does not hold")

	testkit.Error(tb, s.Put(tb.Context(), stale),
		"so the write conditional on it is refused")
}

// --- Planted defects ----------------------------------------------------------

// staleSuffix makes a body the store does not hold. Derived from the held one
// rather than a literal, so the two differ by construction whatever the run
// supplies.
const staleSuffix = "-stale"

// writesWhatItDeclines answers the predicate correctly and writes anyway, which
// is the guard on the wrong side of the door — and the one thing a check
// calling only Match would miss.
type writesWhatItDeclines struct{ held map[string]ifmatch.Value }

func newWritesWhatItDeclines() *writesWhatItDeclines {
	return &writesWhatItDeclines{held: map[string]ifmatch.Value{}}
}

func (w *writesWhatItDeclines) Match(
	_ context.Context, v ifmatch.Value,
) (bool, error) {
	held, present := w.held[v.Key]
	if !present {
		return false, ifmatchtest.ErrAbsent
	}
	return held.Body == v.Body, nil
}

func (w *writesWhatItDeclines) Put(_ context.Context, v ifmatch.Value) error {
	w.held[v.Key] = v
	return nil
}
