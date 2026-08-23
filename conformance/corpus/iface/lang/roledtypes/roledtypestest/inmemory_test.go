// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// roledtypes is the language axis's: a role written on a named type rather
// than inferred from a parameter's position. The row below is what the pair
// comes to once both roles draw — a value written under a key is the value
// that key reads back.
package roledtypestest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/roledtypes"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/roledtypes/roledtypestest"
)

// TestStoreContract runs the generated checks and this package's own.
func TestStoreContract(t *testing.T) {
	t.Parallel()

	roledtypestest.RunStore(t, inMemory("in-memory"), storeChecks)
}

// TestStoreChecksCanFail drives every planted defect through the check it is
// evidence for.
func TestStoreChecksCanFail(t *testing.T) {
	t.Parallel()

	roledtypestest.ProveStore(t, inMemory("in-memory"), storeChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) roledtypestest.StoreHarness[*roledtypestest.InMemory] {
	return roledtypestest.StoreHarness[*roledtypestest.InMemory]{
		Name: name, New: roledtypestest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var storeChecks = roledtypestest.StoreChecks{
	{
		Method: "Get", Name: "reads-back-what-was-written",
		Claim: "Get answers the payload Put wrote under the key",
		Run:   readsBackWhatWasWritten,
		ProvenBy: roledtypestest.BrokenStore(
			"a store that keeps the key and drops the payload", newDropsThePayload,
		),
		ProvenReason: "the payload comes back",
	},
}

// --- Bodies -------------------------------------------------------------------

// readsBackWhatWasWritten pairs the two roles, which is what makes this
// fixture's point checkable: a run drawing the key alone proves the key role
// resolved, and only a read of what was written proves the payload's did too.
func readsBackWhatWasWritten(
	tb testing.TB, s roledtypes.Store, fx roledtypestest.StoreFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Put(tb.Context(), fx.Key(), fx.Payload()), "the pair is written")

	got, err := s.Get(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "and the key reads")
	testkit.Equal(tb, got, fx.Payload(), "and the payload comes back")
}

// --- Planted defects ----------------------------------------------------------

// dropsThePayload records that a key was written and forgets what was
// written under it — the store whose key role works and whose payload role
// goes nowhere, which is exactly the half this fixture exists to separate.
type dropsThePayload struct{ keys map[roledtypes.Key]bool }

func newDropsThePayload() *dropsThePayload {
	return &dropsThePayload{keys: map[roledtypes.Key]bool{}}
}

func (d *dropsThePayload) Put(_ context.Context, key roledtypes.Key, _ roledtypes.Payload) error {
	d.keys[key] = true
	return nil
}

func (d *dropsThePayload) Get(_ context.Context, key roledtypes.Key) (roledtypes.Payload, error) {
	if !d.keys[key] {
		return roledtypes.Payload{}, roledtypestest.ErrNotFound
	}
	return roledtypes.Payload{}, nil
}
