// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// tx-with-retry stacks the tx contract with the retrysucceeds mixin, and this
// fixture exists because retry changes what the contract's terminal-state rule
// means: a commit that failed either settled the transaction or did not, and
// the two readings produce opposite suites.
//
// `tx` is owned by no tier and `retrysucceeds` names no attempt count, so
// nothing is generated for either. Every claim below is statable through the
// interface, so each is a row rather than a package test — and each names the
// same planted defect, because a subject whose methods return nil is what all
// five exist to reject.
package txwithretrytest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	txwithretry "go.thesmos.sh/testkit/conformance/corpus/iface/composite/tx-with-retry"
	"go.thesmos.sh/testkit/conformance/corpus/iface/composite/tx-with-retry/txwithretrytest"
)

// TestTxWithRetryContract runs the generated checks and this package's own.
func TestTxWithRetryContract(t *testing.T) {
	t.Parallel()

	txwithretrytest.RunTxWithRetry(t, inMemory("in-memory"), txWithRetryChecks)
}

// TestTxWithRetryContractWithoutSmoke drops a check through the typed index
// rather than a string, so a check that is renamed or stops being emitted
// breaks this compile instead of silently declining nothing.
func TestTxWithRetryContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	txwithretrytest.RunTxWithRetry(t,
		inMemory("in-memory"),
		txwithretrytest.TxWithRetrySuite.Without(
			txwithretrytest.TxWithRetrySuite.Checks.Begin.Smoke(),
		),
	)
}

// TestTxWithRetryChecksCanFail drives every row against its planted defect.
//
// The reason matters as much as the rejection: a stand-in failing for some
// unrelated reason would satisfy a boolean guard while the check's own
// assertion never ran.
func TestTxWithRetryChecksCanFail(t *testing.T) {
	t.Parallel()

	txwithretrytest.ProveTxWithRetry(t, inMemory("in-memory"), txWithRetryChecks)
}

// --- Harnesses ---------------------------------------------------------------

// transientCommits is how many times a subject's first commits fail before one
// succeeds.
//
// One rather than several: `retrysucceeds` names no attempt count, so any
// number here is this package's choice. One is the smallest that makes the
// retry a retry.
const transientCommits = 1

// inMemory is the subject whose first commit fails transiently.
func inMemory(name string) txwithretrytest.TxWithRetryHarness[*txwithretrytest.InMemory] {
	return txwithretrytest.TxWithRetryHarness[*txwithretrytest.InMemory]{
		Name: name,
		New:  func() *txwithretrytest.InMemory { return txwithretrytest.NewInMemory(transientCommits) },
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------
//
// Every row plants the same defect and names its own reason for rejecting it.
// One subject rather than five, because what they have in common is the thing
// worth proving: a check whose only assertions are NoError reads as coverage
// while asserting nothing.

var txWithRetryChecks = txwithretrytest.TxWithRetryChecks{
	{
		Method: "Commit", Name: "retries-the-same-terminal",
		Claim:        "Commit retries the same terminal operation",
		Run:          retriesTheSameCommit,
		ProvenBy:     doesNothing(),
		ProvenReason: "the first commit fails transiently",
	},

	{
		Method: "Commit", Name: "refuses-an-unbegun-transaction",
		Claim:        "Commit refuses a transaction that never began",
		Run:          refusesAnUnbegunCommit,
		ProvenBy:     doesNothing(),
		ProvenReason: "there is nothing to commit",
	},

	{
		Method: "Begin", Name: "settles-once",
		Claim:        "Begin settles once and then refuses both terminals",
		Run:          settlesOnce,
		ProvenBy:     doesNothing(),
		ProvenReason: "a second rollback has nothing to settle",
	},

	{
		Method: "Rollback", Name: "reopens-after-settling",
		Claim:        "Rollback reopens after settling",
		Run:          reopensAfterSettling,
		ProvenBy:     doesNothing(),
		ProvenReason: "is settled once, like the first",
	},

	{
		Method: "Begin", Name: "refuses-a-second-open",
		Claim:        "Begin refuses a second open",
		Run:          refusesASecondOpen,
		ProvenBy:     doesNothing(),
		ProvenReason: "does not silently replace it",
	},
}

// --- Bodies -------------------------------------------------------------------

// retriesTheSameCommit is the fixture's whole question.
//
// A transient failure leaves the transaction open, so the retry continues the
// same terminal operation rather than starting a second one. Read the other way
// the tx contract refuses the retry, and the suite fails an implementation that
// did exactly what the two directives said.
func retriesTheSameCommit(
	tb testing.TB, s txwithretry.TxWithRetry, _ txwithretrytest.TxWithRetryFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Begin(tb.Context()), "the transaction opens")

	testkit.ErrorIs(tb, s.Commit(tb.Context()), txwithretry.ErrTransient,
		"the first commit fails transiently")
	testkit.ErrorIsNot(tb, s.Commit(tb.Context()), txwithretry.ErrClosed,
		"and the retry is not refused as a second terminal operation")

	testkit.ErrorIs(tb, s.Commit(tb.Context()), txwithretry.ErrClosed,
		"the transaction is settled once the commit succeeded")
}

// refusesAnUnbegunCommit holds the other half of the contract's rule: a
// terminal operation needs a transaction to be terminal for.
func refusesAnUnbegunCommit(
	tb testing.TB, s txwithretry.TxWithRetry, _ txwithretrytest.TxWithRetryFixture,
) {
	tb.Helper()
	testkit.ErrorIs(tb, s.Commit(tb.Context()), txwithretry.ErrClosed,
		"there is nothing to commit")
}

// settlesOnce holds the tx contract's terminal-state rule against both
// directions.
//
// A subject settling through separate paths has the rule written twice and one
// place to forget it, which is why the rollback-after-commit case is asserted
// rather than assumed from the commit-after-commit one.
func settlesOnce(
	tb testing.TB, s txwithretry.TxWithRetry, _ txwithretrytest.TxWithRetryFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Begin(tb.Context()), "the transaction opens")
	testkit.NoError(tb, s.Rollback(tb.Context()), "and rolls back")

	testkit.ErrorIs(tb, s.Rollback(tb.Context()), txwithretry.ErrClosed,
		"a second rollback has nothing to settle")
	testkit.ErrorIs(tb, s.Commit(tb.Context()), txwithretry.ErrClosed,
		"and a commit after a rollback is the second terminal operation")
}

// reopensAfterSettling holds the handle usable more than once.
//
// A subject refusing every Begin after the first passes every check above and
// serves exactly one transaction.
//
// It ends on a refusal rather than on the reopen, and that is the difference
// between a check and a description: three NoErrors are satisfied by a subject
// whose methods all return nil, so the second transaction has to be shown
// settling — and settling only once.
func reopensAfterSettling(
	tb testing.TB, s txwithretry.TxWithRetry, _ txwithretrytest.TxWithRetryFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Begin(tb.Context()), "the transaction opens")
	testkit.NoError(tb, s.Rollback(tb.Context()), "and rolls back")

	testkit.NoError(tb, s.Begin(tb.Context()), "and the handle opens a new one")
	testkit.NoError(tb, s.Rollback(tb.Context()), "which settles in its own right")
	testkit.ErrorIs(tb, s.Rollback(tb.Context()), txwithretry.ErrClosed,
		"and is settled once, like the first")
}

// refusesASecondOpen keeps a running transaction from being replaced.
//
// A subject that let Begin reopen would strand whatever the first transaction
// had staged, and every terminal check above would still pass — they settle a
// transaction without caring which one it is.
func refusesASecondOpen(
	tb testing.TB, s txwithretry.TxWithRetry, _ txwithretrytest.TxWithRetryFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Begin(tb.Context()), "the transaction opens")
	testkit.ErrorIs(tb, s.Begin(tb.Context()), txwithretry.ErrClosed,
		"and a second Begin does not silently replace it")
}

// --- Planted defects ----------------------------------------------------------

// nullSubject implements the interface and does nothing, which is the
// implementation every check here exists to reject.
//
// A check whose only assertions are NoError passes against this, and reads as
// coverage while asserting nothing. Naming it and driving it is what turns "the
// check looks right" into evidence.
type nullSubject struct{}

func newNullSubject() nullSubject { return nullSubject{} }

func (nullSubject) Begin(context.Context) error    { return nil }
func (nullSubject) Commit(context.Context) error   { return nil }
func (nullSubject) Rollback(context.Context) error { return nil }

// doesNothing is the planted defect five rows share, spelled once.
func doesNothing() txwithretrytest.TxWithRetryHarness[nullSubject] {
	return txwithretrytest.BrokenTxWithRetry("a subject that does nothing", newNullSubject)
}
