// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `updater mode=replace` says a second write to one key replaces rather than
// accumulates, and the row below writes the key first: an update needs
// something to update, and a fresh subject holds nothing.
package updatertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/updater"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/updater/updatertest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	updatertest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	updatertest.RunContract(t,
		inMemory("in-memory"),
		updatertest.ContractSuite.Without(updatertest.ContractSuite.Checks.Put.Smoke()),
	)
}

// TestContractChecksCanFail drives the row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	updatertest.ProveContract(t, contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) updatertest.ContractHarness[*updatertest.InMemory] {
	return updatertest.ContractHarness[*updatertest.InMemory]{
		Name: name, New: updatertest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = updatertest.ContractChecks{
	{
		Method: "Put", Name: "replaces-rather-than-accumulates",
		Claim: "Put replaces rather than accumulates",
		Run:   replacesRatherThanAccumulates,
		ProvenBy: updatertest.BrokenContract(
			"a store that keeps the first write and drops the update", newFirstWriteWins,
		),
		ProvenReason: "carrying the newer value",
	},
}

// --- Bodies -------------------------------------------------------------------

func replacesRatherThanAccumulates(
	tb testing.TB, s updater.Contract, fx updatertest.ContractFixture,
) {
	tb.Helper()
	first := fx.Value()
	testkit.NoError(tb, s.Put(tb.Context(), first), "the first write lands")

	replacement := updater.Value{Key: first.Key, Body: first.Body + replacedSuffix}
	testkit.NoError(tb, s.Put(tb.Context(), replacement), "the update lands")

	got, err := s.Get(tb.Context(), first.Key)
	testkit.NoError(tb, err, "and the key is still there")
	testkit.Equal(tb, got, replacement, "carrying the newer value")
}

// --- Planted defects ----------------------------------------------------------

// replacedSuffix makes the update differ from the write it replaces. Derived
// from the fixture's own body rather than a literal, so the two values differ
// by construction whatever the run supplies.
const replacedSuffix = "-replaced"

// firstWriteWins accepts the update and keeps what it already had, which is an
// insert-if-absent wearing an updater's name. Both writes succeed, so only
// reading the value back catches it.
type firstWriteWins struct{ held map[string]updater.Value }

func newFirstWriteWins() *firstWriteWins {
	return &firstWriteWins{held: map[string]updater.Value{}}
}

func (f *firstWriteWins) Put(_ context.Context, v updater.Value) error {
	if _, held := f.held[v.Key]; held {
		return nil
	}
	f.held[v.Key] = v
	return nil
}

func (f *firstWriteWins) Get(_ context.Context, key string) (updater.Value, error) {
	v, held := f.held[key]
	if !held {
		return updater.Value{}, updatertest.ErrNotFound
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

	updatertest.ContractModelSaturation(t, func() updatertest.Contract { return updatertest.NewInMemory() })
}
