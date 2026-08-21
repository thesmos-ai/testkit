// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `persister` is the model tier's: durability across a restart is what its laws
// state, and only a run that can kill the medium can ask. The row below is what
// one write and one read settle.
package persistertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/persister"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/persister/persistertest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	persistertest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	persistertest.RunContract(t,
		inMemory("in-memory"),
		persistertest.ContractSuite.Without(persistertest.ContractSuite.Checks.Put.Smoke()),
	)
}

// TestContractChecksCanFail drives the row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	persistertest.ProveContract(t, contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) persistertest.ContractHarness[*persistertest.InMemory] {
	return persistertest.ContractHarness[*persistertest.InMemory]{
		Name: name, New: persistertest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = persistertest.ContractChecks{
	{
		Method: "Get", Name: "reads-back-what-put-wrote",
		Claim: "Get returns what Put wrote under that key",
		Run:   readsBackWhatPutWrote,
		ProvenBy: persistertest.BrokenContract(
			"a store that files every write under one key", newOneSlotForEverything,
		),
		ProvenReason: "carrying what was filed under it",
	},
}

// --- Bodies -------------------------------------------------------------------

func readsBackWhatPutWrote(
	tb testing.TB, s persister.Contract, fx persistertest.ContractFixture,
) {
	tb.Helper()
	written := fx.Value()
	testkit.NoError(tb, s.Put(tb.Context(), written), "the value is stored")

	got, err := s.Get(tb.Context(), written.Key)
	testkit.NoError(tb, err, "the written key is found")
	testkit.Equal(tb, got, written, "carrying what was filed under it")
}

// --- Planted defects ----------------------------------------------------------

// oneSlotForEverything keeps the last write and answers it for any key, which
// is a store whose key never reached the medium. One write and one read of the
// SAME key would not notice, which is why the row compares the whole value.
type oneSlotForEverything struct{ last persister.Value }

func newOneSlotForEverything() *oneSlotForEverything { return &oneSlotForEverything{} }

func (o *oneSlotForEverything) Put(_ context.Context, v persister.Value) error {
	o.last = v
	o.last.Key = ""
	return nil
}

func (o *oneSlotForEverything) Get(
	_ context.Context, _ string,
) (persister.Value, error) {
	if o.last == (persister.Value{}) {
		return persister.Value{}, persistertest.ErrNotFound
	}
	return o.last, nil
}
