// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Every method here names a parameter `s`, one at Session and one at string.
//
// The fixture keys on the name AND the type, so the checks are handed a Session
// and a string rather than one value the other method could not take. The
// author's own rule is that the two agree: what Put stored under a session's
// identifier is what Get returns for it.
package receivercollisiontest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/receivercollision"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/receivercollision/receivercollisiontest"
)

// TestStoreContract runs the generated checks and this package's own.
func TestStoreContract(t *testing.T) {
	t.Parallel()

	receivercollisiontest.RunStore(t, inMemory("in-memory"), storeChecks)
}

// TestStoreContractSuppression drops a check against the same subject: what is
// under test is the harness declining what it was told to.
func TestStoreContractSuppression(t *testing.T) {
	t.Parallel()

	receivercollisiontest.RunStore(t,
		inMemory("in-memory"),
		receivercollisiontest.StoreSuite.Without(receivercollisiontest.StoreSuite.Checks.Touch.Smoke()),
	)
}

// TestStoreChecksCanFail drives the row against its planted defect.
func TestStoreChecksCanFail(t *testing.T) {
	t.Parallel()

	receivercollisiontest.ProveStore(t, inMemory("in-memory"), storeChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) receivercollisiontest.StoreHarness[*receivercollisiontest.InMemory] {
	return receivercollisiontest.StoreHarness[*receivercollisiontest.InMemory]{
		Name: name, New: receivercollisiontest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var storeChecks = receivercollisiontest.StoreChecks{
	{
		Method: "Get", Name: "reads-back-the-stored-session",
		Claim: "Get returns what Put stored under that identifier",
		Run:   readsBackTheStoredSession,
		ProvenBy: receivercollisiontest.BrokenStore(
			"a store that files every session under one slot", newOneSlotForEverySession,
		),
		ProvenReason: "comes back whole",
	},
}

// --- Bodies -------------------------------------------------------------------

// readsBackTheStoredSession draws both `s` parameters from the one fixture,
// which is the whole point of the collision: they are different fields because
// they are different types.
func readsBackTheStoredSession(
	tb testing.TB, s receivercollision.Store, fx receivercollisiontest.StoreFixture,
) {
	tb.Helper()
	stored := fx.Session()
	testkit.NoError(tb, s.Put(tb.Context(), stored), "the session is stored")

	got, err := s.Get(tb.Context(), stored.ID)
	testkit.NoError(tb, err, "a stored session is found by its own identifier")
	testkit.Equal(tb, got, stored, "and comes back whole")
}

// --- Planted defects ----------------------------------------------------------

// oneSlotForEverySession keeps the last session and answers it for any
// identifier, with the identifier itself cleared — which is a store that took
// the Session and never read its ID. The write and the read both succeed, so
// only comparing the whole value catches it.
type oneSlotForEverySession struct {
	last    receivercollision.Session
	written bool
}

func newOneSlotForEverySession() *oneSlotForEverySession {
	return &oneSlotForEverySession{}
}

func (o *oneSlotForEverySession) Put(
	_ context.Context, s receivercollision.Session,
) error {
	o.last = s
	o.last.ID = ""
	o.written = true
	return nil
}

func (o *oneSlotForEverySession) Get(
	_ context.Context, _ string,
) (receivercollision.Session, error) {
	if !o.written {
		return receivercollision.Session{}, receivercollisiontest.ErrNotFound
	}
	return o.last, nil
}

func (*oneSlotForEverySession) Touch(context.Context, receivercollision.Session) {}
