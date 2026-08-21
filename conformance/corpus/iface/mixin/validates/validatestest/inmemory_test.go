// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// The whole wiring a consumer writes: one subject, and the checks the generator
// has no classification to derive.
//
// Every value the run uses is derived — from each parameter's own type — and so
// is the double, which comes from the //testkit:stub on the same interface.
package validatestest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/validates"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/validates/validatestest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	validatestest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	validatestest.RunMixed(t,
		inMemory("in-memory"),
		validatestest.MixedSuite.Without(validatestest.MixedSuite.Checks.Store.Smoke()),
	)
}

// TestMixedChecksCanFail drives every row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	validatestest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

// storedBody is what the read-after-write row writes, so the comparison is
// against something rather than against a zero either way.
const storedBody = "read-after-write"

func inMemory(name string) validatestest.MixedHarness[*validatestest.InMemory] {
	return validatestest.MixedHarness[*validatestest.InMemory]{
		Name: name, New: validatestest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = validatestest.MixedChecks{
	{
		Method: "Store", Name: "refuses-what-validate-refuses",
		Claim: "Store refuses what its own validator refuses",
		Run:   refusesWhatValidateRefuses,
		ProvenBy: validatestest.BrokenMixed(
			"a store that takes what its own validator rejects",
			planted(storesWhatItRefuses),
		),
		ProvenReason: "Store refuses it for the same reason",
	},

	{
		Method: "Read", Name: "reads-back-what-store-wrote",
		Claim: "Read returns what Store wrote",
		Run:   readsBackWhatStoreWrote,
		ProvenBy: validatestest.BrokenMixed(
			"a store whose reads answer the zero payload", planted(readsBackTheZero),
		),
		ProvenReason: "comes back whole",
	},
}

// --- Bodies -------------------------------------------------------------------

// refusesWhatValidateRefuses is the mixin's own law, written by hand until the
// generator reads the classification: a payload with no key is one Validate
// rejects, and Store must not take it.
func refusesWhatValidateRefuses(
	tb testing.TB, s validates.Mixed, fx validatestest.MixedFixture,
) {
	tb.Helper()
	invalid := validates.Payload{Body: fx.Payload().Body}
	testkit.ErrorIs(tb, s.Validate(invalid), validatestest.ErrInvalid,
		"a payload with no key does not validate")
	testkit.ErrorIs(tb, s.Store(tb.Context(), invalid), validatestest.ErrInvalid,
		"and Store refuses it for the same reason")
}

// readsBackWhatStoreWrote writes its own precondition rather than assuming one:
// a check that read what something else left behind would break the moment a
// subject supplied its own.
func readsBackWhatStoreWrote(
	tb testing.TB, s validates.Mixed, fx validatestest.MixedFixture,
) {
	tb.Helper()
	want := validates.Payload{Key: fx.Key(), Body: storedBody}
	testkit.NoError(tb, s.Store(tb.Context(), want),
		"a valid payload stores under its own key")

	got, err := s.Read(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "and is found under it")
	testkit.Equal(tb, got, want, "and comes back whole")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted store gets wrong.
type fault int

const (
	// storesWhatItRefuses validates correctly and writes anyway, which is
	// the shape a store with the guard on the wrong side of the door has —
	// and the one a check calling only Validate would miss.
	storesWhatItRefuses fault = iota

	// readsBackTheZero keeps the payload and answers with an empty one,
	// which a check reading only the error cannot see.
	readsBackTheZero
)

// planted builds the constructor for one broken store.
func planted(wrong fault) func() *plantedStore {
	return func() *plantedStore {
		return &plantedStore{wrong: wrong, held: map[string]validates.Payload{}}
	}
}

type plantedStore struct {
	wrong fault
	held  map[string]validates.Payload
}

// validatePayload is the verdict both the method and Store read, so the defect
// is which one ACTS on it. A store with two copies of the rule could disagree
// with itself for a reason the row is not about.
func validatePayload(v validates.Payload) error {
	if v.Key == "" {
		return validatestest.ErrInvalid
	}
	return nil
}

// Validate is correct in both defects: a store whose validator was wrong would
// red the first row on the wrong assertion, and prove nothing about Store.
func (*plantedStore) Validate(v validates.Payload) error { return validatePayload(v) }

func (p *plantedStore) Store(_ context.Context, v validates.Payload) error {
	if err := validatePayload(v); err != nil && p.wrong != storesWhatItRefuses {
		return err
	}
	p.held[v.Key] = v
	return nil
}

func (p *plantedStore) Read(_ context.Context, key string) (validates.Payload, error) {
	v, held := p.held[key]
	if !held {
		return validates.Payload{}, validatestest.ErrNotFound
	}
	if p.wrong == readsBackTheZero {
		return validates.Payload{}, nil
	}
	return v, nil
}

// TestMixedLawsCanSaturate drives each bound law against defects worn on
// its own methods, with that law as the run's only oracle.
//
// Binding a law is necessary; this is what makes it sufficient. A law
// every worn defect survives is bound and unsaturatable, which reads as
// coverage in the report and is not.
func TestMixedLawsCanSaturate(t *testing.T) {
	t.Parallel()

	validatestest.MixedModelSaturation(t, func() validatestest.Mixed { return validatestest.NewInMemory() })
}
