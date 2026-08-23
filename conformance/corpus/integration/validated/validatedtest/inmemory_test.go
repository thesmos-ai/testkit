// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Three generators over one declaration, wired by a consumer in one statement.
//
// The suite supplies the contract and the subjects. The stub supplies the
// second run, wrapping each subject so anything the wrapper fails that the
// subject passes is the double lying. The builder supplies the variant the
// custom check below needs — one field changed, the rest still acceptable.
//
// What binds them is AccountDefaults: builder seeds NewAccount with it and the
// suite's fixture takes it over anything it could derive, so a team states the
// valid shape of an Account once and every generator that needs one uses it.
package validatedtest_test

import (
	"context"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/integration/validated"
	"go.thesmos.sh/testkit/conformance/corpus/integration/validated/validatedtest"
)

// TestStoreContract runs the generated checks and this package's own.
func TestStoreContract(t *testing.T) {
	t.Parallel()

	validatedtest.RunStore(t, inMemory("in-memory"), storeChecks)
}

// TestStoreChecksCanFail drives every row against its planted defect.
func TestStoreChecksCanFail(t *testing.T) {
	t.Parallel()

	validatedtest.ProveStore(t, inMemory("in-memory"), storeChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) validatedtest.StoreHarness[*validatedtest.InMemory] {
	return validatedtest.StoreHarness[*validatedtest.InMemory]{
		Name: name, New: validatedtest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var storeChecks = validatedtest.StoreChecks{
	{
		Method: "Get", Name: "reads-back-the-stored-account",
		Claim: "Get returns the account Put stored",
		Run:   readsBackTheStoredAccount,
		ProvenBy: validatedtest.BrokenStore(
			"a store that hands back an account missing a field",
			planted(dropsAField),
		),
		ProvenReason: "comes back whole",
	},

	{
		Method: "Put", Name: "refuses-an-address-with-no-at",
		Claim: "Put refuses an address with no @",
		Run:   refusesAnAddressWithNoAt,
		ProvenBy: validatedtest.BrokenStore(
			"a store that takes any address it is given", planted(takesAnyAddress),
		),
		ProvenReason: "must refuse an account whose email is not one",
	},
}

// --- Bodies -------------------------------------------------------------------

// readsBackTheStoredAccount is a domain rule no classification implies. It runs
// against every subject and through the double, from this one row.
func readsBackTheStoredAccount(
	tb testing.TB, s validated.Store, _ validatedtest.StoreFixture,
) {
	tb.Helper()
	want := validated.AccountDefaults()
	testkit.NoError(tb, s.Put(tb.Context(), want), "a valid account stores")

	got, err := s.Get(tb.Context(), want.ID)
	testkit.NoError(tb, err, "a stored account is found by its own identifier")
	testkit.Equal(tb, got, want, "and comes back whole")
}

// refusesAnAddressWithNoAt builds its bad account from the valid one rather
// than writing it out: a literal restating every field goes stale the moment
// one is added, and would then be refused for the wrong reason.
func refusesAnAddressWithNoAt(
	tb testing.TB, s validated.Store, fx validatedtest.StoreFixture,
) {
	tb.Helper()
	bad := validatedtest.NewAccountFrom(fx.Account()).WithEmail(notAnAddress).Build()
	testkit.ErrorIs(tb, s.Put(tb.Context(), bad), validatedtest.ErrInvalid,
		"Put must refuse an account whose email is not one")
}

// --- Planted defects ----------------------------------------------------------

// notAnAddress is the one field the second row changes: everything else about
// the account stays whatever the builder's defaults say is valid.
const notAnAddress = "no-at-sign"

// fault names what one planted store gets wrong.
type fault int

const (
	// dropsAField keeps the account under its identifier and answers with
	// the email cleared, which is a projection that forgot a column — and
	// the failure a check reading only the identifier back would miss.
	dropsAField fault = iota

	// takesAnyAddress stores whatever it is given, which is the guard
	// missing rather than the guard wrong.
	takesAnyAddress
)

// planted builds the constructor for one broken store.
func planted(wrong fault) func() *plantedStore {
	return func() *plantedStore {
		return &plantedStore{wrong: wrong, held: map[string]validated.Account{}}
	}
}

type plantedStore struct {
	wrong fault
	held  map[string]validated.Account
}

func (p *plantedStore) Put(_ context.Context, a validated.Account) error {
	if !strings.Contains(a.Email, "@") && p.wrong != takesAnyAddress {
		return validatedtest.ErrInvalid
	}
	p.held[a.ID] = a
	return nil
}

func (p *plantedStore) Get(_ context.Context, id string) (validated.Account, error) {
	a, held := p.held[id]
	if !held {
		return validated.Account{}, validatedtest.ErrNotFound
	}
	if p.wrong == dropsAField {
		a.Email = ""
	}
	return a, nil
}
