// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `cas` is the model tier's: that a compare-and-set sequence never loses a
// write is a claim about a generated history. The row below is what one fresh
// cell settles, and the derived fixture cannot know the cell's dialect — every
// check gets a subject whose version is back at the start, so the row spells
// which version that is.
package castest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cas"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cas/castest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	castest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	castest.RunContract(t,
		inMemory("in-memory"),
		castest.ContractSuite.Without(castest.ContractSuite.Checks.Put.Smoke()),
	)
}

// TestContractChecksCanFail drives the row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	castest.ProveContract(t, inMemory("in-memory"), contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) castest.ContractHarness[*castest.InMemory] {
	return castest.ContractHarness[*castest.InMemory]{
		Name: name, New: castest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = castest.ContractChecks{
	{
		Method: "Put", Name: "fresh-cell-takes-version-zero",
		Claim: "Put takes version zero against a fresh cell and refuses any other",
		Run:   freshCellTakesVersionZero,
		ProvenBy: castest.BrokenContract(
			"a cell that takes every version it is offered", newAcceptsAnyVersion,
		),
		ProvenReason: "the same version a second time is stale",
	},
}

// --- Bodies -------------------------------------------------------------------

func freshCellTakesVersionZero(
	tb testing.TB, s cas.Contract, fx castest.ContractFixture,
) {
	tb.Helper()
	first := fx.Value()
	first.Version = freshVersion
	testkit.NoError(tb, s.Put(tb.Context(), first), "version zero lands on a fresh cell")

	stale := fx.Value()
	stale.Version = freshVersion
	testkit.Error(tb, s.Put(tb.Context(), stale),
		"and the same version a second time is stale")
}

// --- Planted defects ----------------------------------------------------------

// freshVersion is what a cell nobody has written reads as.
const freshVersion = 0

// acceptsAnyVersion takes every write and never compares, which is
// compare-and-set with the compare left out — and the last writer silently
// wins, which is the whole failure this contract exists to prevent.
type acceptsAnyVersion struct{ held cas.Value }

func newAcceptsAnyVersion() *acceptsAnyVersion { return &acceptsAnyVersion{} }

func (a *acceptsAnyVersion) Put(_ context.Context, v cas.Value) error {
	a.held = v
	return nil
}

func (a *acceptsAnyVersion) Get(context.Context) (cas.Value, error) {
	if a.held.Version == freshVersion {
		return cas.Value{}, castest.ErrEmpty
	}
	return a.held, nil
}
