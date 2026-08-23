// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// A saga's two rows are the two halves nothing derives: that compensating undoes
// exactly the step it names, and that the fingerprint a fully compensated saga
// answers is the one it started from.
package sagatest_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/saga"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/saga/sagatest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	sagatest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	sagatest.RunContract(t,
		inMemory("in-memory"),
		sagatest.ContractSuite.Without(sagatest.ContractSuite.Checks.Step.Smoke()),
	)
}

// TestContractChecksCanFail drives every row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	sagatest.ProveContract(t, inMemory("in-memory"), contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) sagatest.ContractHarness[*sagatest.InMemory] {
	return sagatest.ContractHarness[*sagatest.InMemory]{
		Name: name, New: sagatest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = sagatest.ContractChecks{
	{
		Method: "Compensate", Name: "undoes-an-applied-step",
		Claim: "Compensate undoes the step that was applied",
		Run:   undoesAnAppliedStep,
		ProvenBy: sagatest.BrokenContract(
			"a saga that lets a step be compensated twice",
			planted(compensatesTwice),
		),
		ProvenReason: "has nothing to undo",
	},

	{
		Method: "State", Name: "fingerprints-in-application-order",
		Claim: "State fingerprints in application order",
		Run:   fingerprintsInApplicationOrder,
		ProvenBy: sagatest.BrokenContract(
			"a saga whose fingerprint keeps a step it compensated",
			planted(remembersWhatItUndid),
		),
		ProvenReason: "restored the fingerprint",
	},
}

// --- Bodies -------------------------------------------------------------------

func undoesAnAppliedStep(tb testing.TB, s saga.Contract, fx sagatest.ContractFixture) {
	tb.Helper()
	testkit.NoError(tb, s.Step(tb.Context(), fx.Value()), "a step applies")
	testkit.NoError(tb, s.Compensate(tb.Context(), fx.Value()),
		"the applied step is compensated")
	testkit.ErrorIs(tb, s.Compensate(tb.Context(), fx.Value()), sagatest.ErrNotApplied,
		"and compensating it again has nothing to undo")
}

// fingerprintsInApplicationOrder walks the saga out and back: two steps change
// the fingerprint, and compensating both in reverse restores it.
func fingerprintsInApplicationOrder(
	tb testing.TB, s saga.Contract, fx sagatest.ContractFixture,
) {
	tb.Helper()
	before, err := s.State(tb.Context())
	testkit.NoError(tb, err, "the state is readable")

	first, second := fx.Value(), fx.ValueOther()
	testkit.NoError(tb, s.Step(tb.Context(), first), "the first step applies")
	testkit.NoError(tb, s.Step(tb.Context(), second), "and the second after it")

	stepped, err := s.State(tb.Context())
	testkit.NoError(tb, err, "the state is still readable")
	testkit.NotEqual(tb, stepped, before, "two applied steps changed the fingerprint")

	testkit.NoError(tb, s.Compensate(tb.Context(), second), "the newest step compensates")
	testkit.NoError(tb, s.Compensate(tb.Context(), first), "then the one before it")

	after, err := s.State(tb.Context())
	testkit.NoError(tb, err, "and the state is readable at the end")
	testkit.Equal(tb, after, before, "full compensation restored the fingerprint")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted saga gets wrong.
type fault int

const (
	// compensatesTwice accepts an undo for a step it already undid, which
	// lets a retried rollback run the compensation a second time.
	compensatesTwice fault = iota

	// remembersWhatItUndid keeps a compensated step in the fingerprint, so
	// a fully rolled-back saga never reads as untouched — the failure a
	// check that only counted steps could not see.
	remembersWhatItUndid
)

// planted builds the constructor for one broken saga.
func planted(wrong fault) func() *plantedSaga {
	return func() *plantedSaga { return &plantedSaga{wrong: wrong} }
}

// plantedSaga keeps the applied steps in order and fingerprints them by joining
// their keys, which is the smallest thing that makes order observable.
type plantedSaga struct {
	wrong   fault
	applied []string
	ever    []string
}

func (p *plantedSaga) Step(_ context.Context, v saga.Value) error {
	if slices.Contains(p.applied, v.Key) {
		return sagatest.ErrAlreadyApplied
	}
	p.applied = append(p.applied, v.Key)
	p.ever = append(p.ever, v.Key)
	return nil
}

func (p *plantedSaga) Compensate(_ context.Context, v saga.Value) error {
	at := slices.Index(p.applied, v.Key)
	if at < 0 && p.wrong != compensatesTwice {
		return sagatest.ErrNotApplied
	}
	if at >= 0 {
		p.applied = slices.Delete(p.applied, at, at+1)
	}
	return nil
}

func (p *plantedSaga) State(context.Context) (string, error) {
	if p.wrong == remembersWhatItUndid {
		return strings.Join(p.ever, ">"), nil
	}
	return strings.Join(p.applied, ">"), nil
}
