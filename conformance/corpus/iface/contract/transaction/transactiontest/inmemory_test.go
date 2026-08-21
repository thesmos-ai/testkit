// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `transaction` wraps a unit of work, and the claim below is the one the
// signature cannot make: what an ERRORING body leaves behind. The body is the
// row's, because a derived func literal has no way to fail.
package transactiontest_test

import (
	"context"
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/transaction"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/transaction/transactiontest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	transactiontest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	transactiontest.RunContract(t,
		inMemory("in-memory"),
		transactiontest.ContractSuite.Without(transactiontest.ContractSuite.Checks.Run.Smoke()),
	)
}

// TestContractChecksCanFail drives the row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	transactiontest.ProveContract(t, contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) transactiontest.ContractHarness[*transactiontest.InMemory] {
	return transactiontest.ContractHarness[*transactiontest.InMemory]{
		Name: name, New: transactiontest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = transactiontest.ContractChecks{
	{
		Method: "Get", Name: "erroring-body-changes-nothing",
		Claim: "an erroring body leaves the store as it found it",
		Run:   erroringBodyChangesNothing,
		ProvenBy: transactiontest.BrokenContract(
			"a unit of work that commits whatever the body reported", newCommitsOnError,
		),
		ProvenReason: "the erroring run changed no presence",
	},
}

// --- Bodies -------------------------------------------------------------------

func erroringBodyChangesNothing(
	tb testing.TB, s transaction.Contract, _ transactiontest.ContractFixture,
) {
	tb.Helper()
	before, beforeErr := s.Get(tb.Context(), transactiontest.RunKey)

	testkit.ErrorIs(tb,
		s.Run(tb.Context(), func(context.Context) error { return errInduced }),
		errInduced, "the body's error is the run's")

	after, afterErr := s.Get(tb.Context(), transactiontest.RunKey)
	testkit.Equal(tb, afterErr == nil, beforeErr == nil,
		"the erroring run changed no presence")
	testkit.Equal(tb, after, before, "and no value")

	testkit.NoError(tb, s.Run(tb.Context(), nil), "an empty unit of work commits")
	_, err := s.Get(tb.Context(), transactiontest.RunKey)
	testkit.NoError(tb, err, "and its entry is readable")
}

// --- Planted defects ----------------------------------------------------------

// errInduced is what the row's body reports. Its identity does not matter
// beyond being distinguishable, so the row can assert the run handed the
// caller's own error back rather than one of its own.
var errInduced = errors.New("transactiontest_test: induced")

// commitsOnError records the unit of work whatever the body said and hands the
// error back, which is the transaction with the rollback path never written.
// The error is reported faithfully, so only reading the store afterwards
// catches it.
type commitsOnError struct{ held map[string]transaction.Value }

func newCommitsOnError() *commitsOnError {
	return &commitsOnError{held: map[string]transaction.Value{}}
}

func (c *commitsOnError) Run(
	ctx context.Context, body func(ctx context.Context) error,
) error {
	var err error
	if body != nil {
		err = body(ctx)
	}
	c.held[transactiontest.RunKey] = transaction.Value{Key: transactiontest.RunKey}
	return err
}

func (c *commitsOnError) Put(_ context.Context, key string, v transaction.Value) error {
	c.held[key] = v
	return nil
}

func (c *commitsOnError) Get(
	_ context.Context, key string,
) (transaction.Value, error) {
	v, held := c.held[key]
	if !held {
		return transaction.Value{}, transaction.ErrNotFound
	}
	return v, nil
}
