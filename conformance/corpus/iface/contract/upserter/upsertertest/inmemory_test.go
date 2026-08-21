// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `upserter` says a repeated write lands on the key it already wrote rather
// than making a second one, which is the difference between an upsert and an
// append nobody deduplicates.
package upsertertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/upserter"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/upserter/upsertertest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	upsertertest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	upsertertest.RunContract(t,
		inMemory("in-memory"),
		upsertertest.ContractSuite.Without(upsertertest.ContractSuite.Checks.Put.Smoke()),
	)
}

// TestContractChecksCanFail drives the row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	upsertertest.ProveContract(t, contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) upsertertest.ContractHarness[*upsertertest.InMemory] {
	return upsertertest.ContractHarness[*upsertertest.InMemory]{
		Name: name, New: upsertertest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = upsertertest.ContractChecks{
	{
		Method: "Put", Name: "repeat-write-is-the-same-key",
		Claim: "Put writes the same key a second time rather than a new one",
		Run:   repeatWriteIsTheSameKey,
		ProvenBy: upsertertest.BrokenContract(
			"a store that re-keys the entry on a repeat", newMovesOnRepeat,
		),
		ProvenReason: "the key is still there",
	},
}

// --- Bodies -------------------------------------------------------------------

func repeatWriteIsTheSameKey(
	tb testing.TB, s upserter.Contract, fx upsertertest.ContractFixture,
) {
	tb.Helper()
	v := fx.Value()
	testkit.NoError(tb, s.Put(tb.Context(), v), "the first write lands")
	testkit.NoError(tb, s.Put(tb.Context(), v), "the repeated write lands")

	got, err := s.Get(tb.Context(), v.Key)
	testkit.NoError(tb, err, "and the key is still there")
	testkit.Equal(tb, got, v, "carrying what it carried before")
}

// --- Planted defects ----------------------------------------------------------

// movesOnRepeat files the repeat under a key of its own and drops the original,
// which is what a store keying on an insertion counter does. Both writes
// succeed and the key the caller asked about is gone, so only reading it back
// catches it.
//
// A defect that KEPT the first entry beside the second would pass this row, and
// it would be right to: with only Put and Get on the interface there is nothing
// to enumerate the leak through. That is a limit of the contract's surface
// rather than of the row.
type movesOnRepeat struct {
	held map[string]upserter.Value
	seen int
}

func newMovesOnRepeat() *movesOnRepeat {
	return &movesOnRepeat{held: map[string]upserter.Value{}}
}

func (m *movesOnRepeat) Put(_ context.Context, v upserter.Value) error {
	m.seen++
	if m.seen > 1 {
		delete(m.held, v.Key)
		v.Key += "-again"
	}
	m.held[v.Key] = v
	return nil
}

func (m *movesOnRepeat) Get(_ context.Context, key string) (upserter.Value, error) {
	v, held := m.held[key]
	if !held {
		return upserter.Value{}, upsertertest.ErrNotFound
	}
	return v, nil
}

// TestContractLawsCanSaturate drives each bound law against defects worn on
// its own methods, with that law as the run's only oracle.
//
// Binding a law is necessary; this is what makes it sufficient. A law
// every worn defect survives is bound and unsaturatable, which reads as
// coverage in the report and is not.
func TestContractLawsCanSaturate(t *testing.T) {
	t.Parallel()

	upsertertest.ContractModelSaturation(t, func() upsertertest.Contract { return upsertertest.NewInMemory() })
}
