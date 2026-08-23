// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Leadership is a group property reached from a surface that hands out one
// subject, so the second harness builds the contention rather than the row:
// a constructor may make any starting state, and it runs before anything wraps
// it.
package leaderelectiontest_test

import (
	"context"
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	leaderelection "go.thesmos.sh/testkit/conformance/corpus/iface/contract/leader-election"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/leader-election/leaderelectiontest"
)

// TestContractContract runs the generated checks and this package's own,
// against a lone candidate and one campaigning against a standing leader.
func TestContractContract(t *testing.T) {
	t.Parallel()

	leaderelectiontest.RunContract(t,
		inMemory("in-memory"),
		contended("in-memory, contended"),
		contractChecks,
	)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	leaderelectiontest.RunContract(t,
		inMemory("in-memory"),
		leaderelectiontest.ContractSuite.Without(
			leaderelectiontest.ContractSuite.Checks.Campaign.Smoke(),
		),
	)
}

// TestContractChecksCanFail drives every row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	leaderelectiontest.ProveContract(t, inMemory("in-memory"), contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) leaderelectiontest.ContractHarness[*leaderelectiontest.InMemory] {
	return leaderelectiontest.ContractHarness[*leaderelectiontest.InMemory]{
		Name: name, New: leaderelectiontest.NewInMemory,
	}
}

// contended is a candidate whose registry already has a leader in it.
func contended(name string) leaderelectiontest.ContractHarness[*leaderelectiontest.InMemory] {
	return leaderelectiontest.ContractHarness[*leaderelectiontest.InMemory]{
		Name: name, Start: seatAnIncumbent,
	}
}

// seatAnIncumbent hands back a candidate campaigning against one that already
// won. Start rather than New, because the incumbent's campaign needs the test's
// context and its failure is the test's to report.
func seatAnIncumbent(tb testing.TB) *leaderelectiontest.InMemory {
	tb.Helper()
	r := leaderelectiontest.NewRegistry()
	if err := r.Candidate().Campaign(tb.Context()); err != nil {
		tb.Fatalf("seat the incumbent that makes the contention real: %v", err)
	}
	return r.Candidate()
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = leaderelectiontest.ContractChecks{
	{
		Method: "IsLeader", Name: "answers-a-caller-who-gave-up",
		Claim: "IsLeader answers a caller who already gave up",
		Run:   answersACallerWhoGaveUp,
		ProvenBy: leaderelectiontest.BrokenContract(
			"a candidate that claims leadership whatever the caller asked with",
			planted(ignoresTheCancellation),
		),
		ProvenReason: "told nothing rather than told yes",
	},

	{
		Method: "Campaign", Name: "loses-to-a-standing-leader",
		Claim: "Campaign loses to a standing leader",
		Run:   losesToAStandingLeader,
		ProvenBy: leaderelectiontest.BrokenContract(
			"a candidate that wins and will not stand down", planted(willNotResign),
		),
		ProvenReason: "which gives it up",
	},
}

// --- Bodies -------------------------------------------------------------------

// answersACallerWhoGaveUp is the claim the generated family cannot make:
// IsLeader returns no error, so it asks only that the method survive a nil
// context — a cancelled one it never passes.
//
// A leader that answered "yes" to a caller who had already stopped waiting
// would have them act on a claim nobody is holding.
func answersACallerWhoGaveUp(
	tb testing.TB, s leaderelection.Contract, _ leaderelectiontest.ContractFixture,
) {
	tb.Helper()
	ctx, cancel := context.WithCancel(tb.Context())
	cancel()
	testkit.False(tb, s.IsLeader(ctx),
		"a cancelled caller is told nothing rather than told yes")
}

// losesToAStandingLeader is true of the contended subject and not of the lone
// one, which is why both are declared: a check that held for either alone would
// state half the contract.
func losesToAStandingLeader(
	tb testing.TB, s leaderelection.Contract, _ leaderelectiontest.ContractFixture,
) {
	tb.Helper()
	if err := s.Campaign(tb.Context()); err != nil {
		testkit.ErrorIs(tb, err, leaderelectiontest.ErrHeld,
			"a campaign that loses says who to")
		testkit.False(tb, s.IsLeader(tb.Context()), "and does not claim otherwise")
		testkit.NoError(tb, s.Resign(tb.Context()),
			"while standing down from something never held is not a failure")
		return
	}
	testkit.True(tb, s.IsLeader(tb.Context()), "a campaign that wins takes the leadership")
	testkit.NoError(tb, s.Resign(tb.Context()), "and can stand down")
	testkit.False(tb, s.IsLeader(tb.Context()), "which gives it up")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted candidate gets wrong.
type fault int

const (
	// ignoresTheCancellation reads its own state and never the caller's, so
	// somebody who stopped waiting is still told they lead.
	ignoresTheCancellation fault = iota

	// willNotResign takes the leadership and keeps it, which is the shape a
	// candidate releasing its lease asynchronously has — and the reason the
	// row reads IsLeader after Resign rather than trusting the error.
	willNotResign
)

// planted builds the constructor for one broken candidate.
//
// Both start uncontended: the proof drives one row against one subject, and a
// lone candidate is what the winning arm of each body needs.
func planted(wrong fault) func() *plantedCandidate {
	return func() *plantedCandidate { return &plantedCandidate{wrong: wrong} }
}

type plantedCandidate struct {
	wrong   fault
	mu      sync.Mutex
	leading bool
}

func (p *plantedCandidate) Campaign(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.leading = true
	return nil
}

func (p *plantedCandidate) Resign(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.wrong != willNotResign {
		p.leading = false
	}
	return nil
}

func (p *plantedCandidate) IsLeader(ctx context.Context) bool {
	if ctx.Err() != nil && p.wrong != ignoresTheCancellation {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.wrong == ignoresTheCancellation {
		// The whole defect: a candidate that never campaigned still says
		// yes, because it read its own optimism rather than its state.
		return true
	}
	return p.leading
}
