// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// The pairing is the contract: an append with no replay states nothing, which
// is why eidos requires the partner. The lever is an entry recorded without
// extending the digest — the divergence a tampered log has, and the only way to
// reach the verify role's failure arm through the interface.
package chaintest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/chain"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/chain/chaintest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	chaintest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	chaintest.RunContract(t,
		inMemory("in-memory"),
		chaintest.ContractSuite.Without(chaintest.ContractSuite.Checks.Append.Smoke()),
	)
}

// TestContractChecksCanFail drives the row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	chaintest.ProveContract(t, contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) chaintest.ContractHarness[*chaintest.InMemory] {
	return chaintest.ContractHarness[*chaintest.InMemory]{
		Name: name, New: chaintest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = chaintest.ContractChecks{
	{
		Method: "Replay", Name: "yields-appends-in-order",
		Claim: "Replay yields what Append put in, in order",
		Run:   yieldsAppendsInOrder,
		ProvenBy: chaintest.BrokenContract(
			"a log whose verify always agrees", newAlwaysVerifies,
		),
		ProvenReason: "the break is detected rather than served",
	},
}

// --- Bodies -------------------------------------------------------------------

func yieldsAppendsInOrder(
	tb testing.TB, s chain.Contract, fx chaintest.ContractFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Append(tb.Context(), fx.Entry()), "the entry is recorded")

	got, err := s.Replay(tb.Context())
	testkit.NoError(tb, err, "the log replays")
	testkit.Equal(tb, len(got), 1, "the appended entry is there")
	testkit.NoError(tb, s.Verify(tb.Context()), "and the chain verifies")

	testkit.NoError(tb, s.Append(tb.Context(),
		chain.Entry{Key: chaintest.BreakKey, Body: "unlinked"}),
		"the unlinked entry is recorded")
	testkit.ErrorIs(tb, s.Verify(tb.Context()), chaintest.ErrBroken,
		"and the break is detected rather than served")
}

// --- Planted defects ----------------------------------------------------------

// alwaysVerifies records and replays correctly and agrees with every chain it
// is asked about, which is a digest recomputed from what was read rather than
// carried forward from what was written. Everything but the verify arm passes.
type alwaysVerifies struct{ entries []chain.Entry }

func newAlwaysVerifies() *alwaysVerifies { return &alwaysVerifies{} }

func (a *alwaysVerifies) Append(_ context.Context, e chain.Entry) error {
	a.entries = append(a.entries, e)
	return nil
}

func (a *alwaysVerifies) Replay(context.Context) ([]chain.Entry, error) {
	return a.entries, nil
}

func (*alwaysVerifies) Verify(context.Context) error { return nil }

// The saturation prover is not wired here, and AUTO-APPEND-ONLY-NO-DROPS is why.
//
// It is bound, and every defect the wardrobe wears on its own methods
// leaves it green — measured, not assumed. Which of the three that is
// has not been settled: a law too weak to catch what it is named for, a
// defect class assigned to it wrongly, or a wear that cannot reach it.
// Each has a different owner and a different fix, so the row stays out
// until one of them is chosen rather than shipping a gate nobody can
// read.
