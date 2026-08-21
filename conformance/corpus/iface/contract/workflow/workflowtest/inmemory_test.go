// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `transitions=Draft>Live` declares two states, and what the directive cannot
// say is that each key walks them on its own. Both rows below are about that.
package workflowtest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/workflow"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/workflow/workflowtest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	workflowtest.RunContract(t, inMemory("in-memory"), withoutTheMiss(), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	workflowtest.RunContract(t,
		inMemory("in-memory"),
		withoutTheMiss(),
		workflowtest.ContractSuite.Without(workflowtest.ContractSuite.Checks.Run.Smoke()),
	)
}

// TestContractChecksCanFail drives every row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	workflowtest.ProveContract(t, contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

// The two states `transitions=Draft>Live` declares. Named here because the
// planted defects have to walk the same ladder the subject does.
const (
	draft = "Draft"
	live  = "Live"
)

func inMemory(name string) workflowtest.ContractHarness[*workflowtest.InMemory] {
	return workflowtest.ContractHarness[*workflowtest.InMemory]{
		Name: name, New: workflowtest.NewInMemory,
	}
}

// withoutTheMiss drops the derived miss check.
//
// A key nothing has run is in the first state, not absent: this workflow has no
// "unknown", so State answers Draft rather than the zero. The reader shape
// cannot see that, and the subject is not wrong — which is what a drop is for.
func withoutTheMiss() workflowtest.ContractRunOpt {
	return workflowtest.ContractSuite.Without(workflowtest.ContractSuite.Checks.State.Miss())
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = workflowtest.ContractChecks{
	{
		Method: "Run", Name: "refuses-a-transition-out-of-the-last-state",
		Claim: "Run refuses a transition out of the last state",
		Run:   refusesATransitionOutOfTheLastState,
		ProvenBy: workflowtest.BrokenContract(
			"a workflow that cycles back to its first state", planted(wrapsAround),
		),
		ProvenReason: "no transition out of where it left the key",
	},

	{
		Method: "Run", Name: "advances-one-key-not-another",
		Claim: "Run advances one key without advancing another",
		Run:   advancesOneKeyNotAnother,
		ProvenBy: workflowtest.BrokenContract(
			"a workflow holding one state for every key", planted(sharesOneState),
		),
		ProvenReason: "another key still has both its transitions left",
	},
}

// --- Bodies -------------------------------------------------------------------

// refusesATransitionOutOfTheLastState walks the key to the last declared state
// and asks for one more.
func refusesATransitionOutOfTheLastState(
	tb testing.TB, s workflow.Contract, fx workflowtest.ContractFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Run(tb.Context(), fx.Key()), "a fresh key starts")
	testkit.NoError(tb, s.Run(tb.Context(), fx.Key()), "the declared transition runs")
	testkit.ErrorIs(tb, s.Run(tb.Context(), fx.Key()), workflowtest.ErrTerminal,
		"and there is no transition out of where it left the key")
}

// advancesOneKeyNotAnother catches the subject the row above cannot: one
// holding a single state for the whole workflow rather than one per key passes
// every single-key check and settles every caller's work at once.
func advancesOneKeyNotAnother(
	tb testing.TB, s workflow.Contract, fx workflowtest.ContractFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Run(tb.Context(), fx.KeyOther()), "a fresh key starts")
	testkit.NoError(tb, s.Run(tb.Context(), fx.KeyOther()), "and advances to the last state")
	testkit.NoError(tb, s.Run(tb.Context(), fx.Key()),
		"while another key still has both its transitions left")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted workflow gets wrong.
type fault int

const (
	// wrapsAround cycles a key that reached the last state back to the
	// first, which is a state machine with the modulo in the wrong place —
	// and one that never refuses anything.
	wrapsAround fault = iota

	// sharesOneState holds one position for the whole workflow rather than
	// one per key, which settles every caller's work the moment one of them
	// finishes.
	sharesOneState
)

// planted builds the constructor for one broken workflow.
func planted(wrong fault) func() *plantedWorkflow {
	return func() *plantedWorkflow {
		return &plantedWorkflow{wrong: wrong, at: map[string]int{}}
	}
}

// plantedWorkflow counts transitions rather than naming states: two declared
// states means one legal transition, so the position is what matters and the
// name is only what State answers with.
type plantedWorkflow struct {
	wrong  fault
	at     map[string]int
	shared int
}

func (p *plantedWorkflow) Run(_ context.Context, key string) error {
	if p.wrong == sharesOneState {
		if p.shared >= 2 {
			return workflowtest.ErrTerminal
		}
		p.shared++
		return nil
	}
	if p.at[key] >= 2 {
		if p.wrong == wrapsAround {
			p.at[key] = 1
			return nil
		}
		return workflowtest.ErrTerminal
	}
	p.at[key]++
	return nil
}

func (p *plantedWorkflow) State(_ context.Context, key string) (string, error) {
	at := p.at[key]
	if p.wrong == sharesOneState {
		at = p.shared
	}
	if at >= 2 {
		return live, nil
	}
	return draft, nil
}

// TestContractLawsCanSaturate drives each bound law against defects worn on
// its own methods, with that law as the run's only oracle.
//
// Binding a law is necessary; this is what makes it sufficient. A law
// every worn defect survives is bound and unsaturatable, which reads as
// coverage in the report and is not.
func TestContractLawsCanSaturate(t *testing.T) {
	t.Parallel()

	workflowtest.ContractModelSaturation(t, func() workflowtest.Contract { return workflowtest.NewInMemory() })
}
