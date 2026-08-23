// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// A store refusing every write after the first passes the generated check
// without holding a key at all, which is the reading of "refused" the contract
// does not mean. The row below walks all three cases.
package ifabsenttest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	ifabsent "go.thesmos.sh/testkit/conformance/corpus/iface/contract/if-absent"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/if-absent/ifabsenttest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	ifabsenttest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	ifabsenttest.RunContract(t,
		inMemory("in-memory"),
		ifabsenttest.ContractSuite.Without(ifabsenttest.ContractSuite.Checks.Put.Smoke()),
	)
}

// TestContractChecksCanFail drives the row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	ifabsenttest.ProveContract(t, inMemory("in-memory"), contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) ifabsenttest.ContractHarness[*ifabsenttest.InMemory] {
	return ifabsenttest.ContractHarness[*ifabsenttest.InMemory]{
		Name: name, New: ifabsenttest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = ifabsenttest.ContractChecks{
	{
		Method: "Put", Name: "refuses-the-key",
		Claim: "Put refuses the key rather than the call",
		Run:   refusesTheKey,
		ProvenBy: ifabsenttest.BrokenContract(
			"a store that refuses every write after its first", newRefusesAfterTheFirst,
		),
		ProvenReason: "another key nothing holds is still accepted",
	},
}

// --- Bodies -------------------------------------------------------------------

func refusesTheKey(
	tb testing.TB, s ifabsent.Contract, fx ifabsenttest.ContractFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Put(tb.Context(), fx.Value()),
		"a key nothing holds is accepted")
	testkit.Error(tb, s.Put(tb.Context(), fx.Value()),
		"the same key a second time is refused")
	testkit.NoError(tb, s.Put(tb.Context(), fx.ValueOther()),
		"and another key nothing holds is still accepted")
}

// --- Planted defects ----------------------------------------------------------

// refusesAfterTheFirst takes one write and refuses everything after it, which
// satisfies "the same key a second time is refused" while holding nothing — the
// misreading the third assertion exists to catch.
type refusesAfterTheFirst struct{ written bool }

func newRefusesAfterTheFirst() *refusesAfterTheFirst { return &refusesAfterTheFirst{} }

func (r *refusesAfterTheFirst) Put(context.Context, ifabsent.Value) error {
	if r.written {
		return ifabsenttest.ErrPresent
	}
	r.written = true
	return nil
}
