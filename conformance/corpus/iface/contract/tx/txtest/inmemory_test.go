// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// tx is the model tier's under ADR-0028: `AUTO-TWO-PHASE-MUTEX` and
// `AUTO-TWO-PHASE-ROLLBACK-AFTER-COMMIT` state it, over the handle Begin
// answers and the terminal pair threads.
//
// What the rows below add is the handle discipline the laws do not draw: two
// open transactions settling independently, and a handle nothing opened
// settling nothing — facts about *which* transaction a terminal operation
// names rather than about the mutex itself.
package txtest_test

import (
	"context"
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/tx"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/tx/txtest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	txtest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	txtest.RunContract(t,
		inMemory("in-memory"),
		txtest.ContractSuite.Without(txtest.ContractSuite.Checks.Begin.Smoke()),
	)
}

// TestContractChecksCanFail drives every row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	txtest.ProveContract(t, inMemory("in-memory"), contractChecks)
}

// TestStagingRefusesASettledHandle is a package test rather than a row: staging
// is on the interface and the generated law drives it, but its refusals belong
// here — a settled handle stages nothing, which is a claim about this subject's
// own error rather than about the contract.
func TestStagingRefusesASettledHandle(t *testing.T) {
	t.Parallel()

	s := txtest.NewInMemory()
	h, err := s.Begin(t.Context())
	testkit.NoError(t, err, "a transaction opens")
	testkit.NoError(t, s.Put(t.Context(), h, "k", tx.Value{Key: "k", Body: "staged"}),
		"the open transaction stages")

	_, err = s.Get(t.Context(), "k")
	testkit.ErrorIs(t, err, tx.ErrNotFound, "and the outside read sees nothing of it")

	testkit.NoError(t, s.Rollback(t.Context(), h), "the transaction rolls back")
	testkit.ErrorIs(t, s.Put(t.Context(), h, "k", tx.Value{Key: "k", Body: "late"}), tx.ErrTxClosed,
		"a settled handle stages nothing")

	h2, err := s.Begin(t.Context())
	testkit.NoError(t, err, "a second transaction opens")
	testkit.NoError(t, s.Put(t.Context(), h2, "k", tx.Value{Key: "k", Body: "kept"}), "and stages")
	testkit.NoError(t, s.Commit(t.Context(), h2), "and commits")

	got, err := s.Get(t.Context(), "k")
	testkit.NoError(t, err, "the committed write is readable")
	testkit.Equal(t, got.Body, "kept", "whole, as staged")
}

// --- Harnesses ---------------------------------------------------------------

// inMemory is the subject with the smallest true history this contract has.
func inMemory(name string) txtest.ContractHarness[*txtest.InMemory] {
	return txtest.ContractHarness[*txtest.InMemory]{Name: name, Start: seeded}
}

// seeded opens one transaction and settles it before the run sees the subject.
//
// Both terminal writers refuse a handle nothing opened, so a subject with no
// history fails the generated checks on the absence of a transaction rather
// than on anything they are about. Start rather than New, because the seed can
// fail and the failure is the test's to report.
func seeded(tb testing.TB) *txtest.InMemory {
	tb.Helper()
	s := txtest.NewInMemory()
	h, err := s.Begin(tb.Context())
	if err != nil {
		tb.Fatalf("open the transaction the seed history needs: %v", err)
	}
	if err := s.Commit(tb.Context(), h); err != nil {
		tb.Fatalf("settle the transaction the seed history needs: %v", err)
	}
	return s
}

// --- The checks: claims, bodies and defects, by name --------------------------
//
// Each row plants the near miss it exists to forbid rather than a subject that
// does nothing: a store that never settles anything reds three of these, and
// "settling one settles one" would then be evidenced by a subject that settles
// neither.

var contractChecks = txtest.ContractChecks{
	{
		Method: "Begin", Name: "settles-once-after-a-commit",
		Claim: "Begin settles once and then refuses both terminals",
		Run:   settlesOnceAfterACommit,
		ProvenBy: txtest.BrokenContract(
			"a store whose commit leaves the handle open", planted(neverSettles),
		),
		ProvenReason: "a second commit has nothing to settle",
	},

	{
		Method: "Begin", Name: "settles-once-after-a-rollback",
		Claim: "Begin rolls back once and then refuses both terminals",
		Run:   settlesOnceAfterARollback,
		ProvenBy: txtest.BrokenContract(
			"a store that settles a commit and not a rollback", planted(forgetsRollback),
		),
		ProvenReason: "a second rollback has nothing to settle",
	},

	{
		Method: "Begin", Name: "settles-two-transactions-independently",
		Claim: "Begin settles two open transactions independently",
		Run:   settlesTwoTransactionsIndependently,
		ProvenBy: txtest.BrokenContract(
			"a store with one transaction for the whole subject",
			planted(sharesOneTransaction),
		),
		ProvenReason: "settling one settles one",
	},

	{
		Method: "Begin", Name: "refuses-an-invented-handle",
		Claim: "Begin refuses a handle that never began",
		Run:   refusesAnInventedHandle,
		ProvenBy: txtest.BrokenContract(
			"a store that settles whatever handle it is given", planted(acceptsAnyHandle),
		),
		ProvenReason: "there is nothing to commit under an invented handle",
	},
}

// --- Bodies -------------------------------------------------------------------

func settlesOnceAfterACommit(tb testing.TB, s tx.Contract, _ txtest.ContractFixture) {
	tb.Helper()
	h, err := s.Begin(tb.Context())
	testkit.NoError(tb, err, "the transaction opens")
	testkit.NoError(tb, s.Commit(tb.Context(), h), "and commits")

	testkit.ErrorIs(tb, s.Commit(tb.Context(), h), tx.ErrTxClosed,
		"a second commit has nothing to settle")
	testkit.ErrorIs(tb, s.Rollback(tb.Context(), h), tx.ErrTxClosed,
		"and a rollback after a commit is the second terminal operation")
}

// settlesOnceAfterARollback is the mirror, because a subject can get one
// direction right and the other wrong: settling through separate paths gives
// the rule two places to be written and one to be forgotten.
func settlesOnceAfterARollback(tb testing.TB, s tx.Contract, _ txtest.ContractFixture) {
	tb.Helper()
	h, err := s.Begin(tb.Context())
	testkit.NoError(tb, err, "the transaction opens")
	testkit.NoError(tb, s.Rollback(tb.Context(), h), "and rolls back")

	testkit.ErrorIs(tb, s.Rollback(tb.Context(), h), tx.ErrTxClosed,
		"a second rollback has nothing to settle")
	testkit.ErrorIs(tb, s.Commit(tb.Context(), h), tx.ErrTxClosed,
		"and a commit after a rollback is the second terminal operation")
}

func settlesTwoTransactionsIndependently(tb testing.TB, s tx.Contract, _ txtest.ContractFixture) {
	tb.Helper()
	first, err := s.Begin(tb.Context())
	testkit.NoError(tb, err, "the first transaction opens")
	second, err := s.Begin(tb.Context())
	testkit.NoError(tb, err, "and a second opens beside it")

	testkit.NoError(tb, s.Commit(tb.Context(), first), "the first commits")
	testkit.NoError(tb, s.Rollback(tb.Context(), second),
		"and the second still rolls back — settling one settles one")
}

func refusesAnInventedHandle(tb testing.TB, s tx.Contract, _ txtest.ContractFixture) {
	tb.Helper()
	testkit.ErrorIs(tb, s.Commit(tb.Context(), tx.Tx{ID: 99}), tx.ErrTxClosed,
		"there is nothing to commit under an invented handle")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted store gets wrong about handle discipline.
//
// One implementation with a switch rather than four, because handle discipline
// is the only thing any of them gets wrong: four copies of the open-handle
// bookkeeping to vary one branch would bury the difference the rows are about.
type fault int

const (
	// neverSettles keeps a handle open through both terminal operations.
	neverSettles fault = iota

	// forgetsRollback settles a commit and leaves a rolled-back handle open,
	// which is what a store with the rule written twice looks like.
	forgetsRollback

	// sharesOneTransaction holds one transaction for the whole subject, so
	// settling any handle settles every open one.
	sharesOneTransaction

	// acceptsAnyHandle settles whatever it is given, including a handle
	// nothing opened.
	acceptsAnyHandle
)

// planted builds the constructor for one broken store.
func planted(wrong fault) func() *plantedTx {
	return func() *plantedTx { return &plantedTx{wrong: wrong, open: map[int64]bool{}} }
}

type plantedTx struct {
	wrong fault
	mu    sync.Mutex
	next  int64
	open  map[int64]bool
}

func (s *plantedTx) Begin(context.Context) (tx.Tx, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	s.open[s.next] = true
	return tx.Tx{ID: s.next}, nil
}

func (s *plantedTx) Commit(_ context.Context, h tx.Tx) error { return s.settle(h, false) }

func (s *plantedTx) Rollback(_ context.Context, h tx.Tx) error { return s.settle(h, true) }

// settle is the handle discipline, and the one place each fault disagrees.
func (s *plantedTx) settle(h tx.Tx, rollback bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch s.wrong {
	case acceptsAnyHandle:
		return nil
	case sharesOneTransaction:
		if len(s.open) == 0 {
			return tx.ErrTxClosed
		}
		s.open = map[int64]bool{}
		return nil
	case neverSettles, forgetsRollback:
		// Both need the handle to be open first, so they are decided below
		// rather than here.
	}
	if !s.open[h.ID] {
		return tx.ErrTxClosed
	}
	if s.wrong == neverSettles || (s.wrong == forgetsRollback && rollback) {
		return nil
	}
	delete(s.open, h.ID)
	return nil
}

// Put and Get answer the way an empty store does. No row here is about staging
// — [TestStagingRefusesASettledHandle] is — so a defect that staged would be
// varying something no proof reads.
func (*plantedTx) Put(context.Context, tx.Tx, string, tx.Value) error { return nil }

func (*plantedTx) Get(context.Context, string) (tx.Value, error) {
	return tx.Value{}, tx.ErrNotFound
}
