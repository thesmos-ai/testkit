// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Exhausted is "you have everything"; closed is "you gave up the right to ask".
// A cursor reporting the first for the second hides a bug in the caller's own
// control flow, and the row below is where the difference is stated.
package cursortest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cursor"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cursor/cursortest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	cursortest.RunContract(t, holding("in-memory"), empty("in-memory, empty"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	cursortest.RunContract(t,
		holding("in-memory"),
		cursortest.ContractSuite.Without(cursortest.ContractSuite.Checks.Next.Smoke()),
	)
}

// TestContractChecksCanFail drives the row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	cursortest.ProveContract(t, holding("in-memory"), contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

// held is what the loaded cursor walks. Two values rather than one, so a read
// after Close has something it could plausibly have answered.
var held = []cursor.Value{
	{Key: "a", Body: "one"},
	{Key: "b", Body: "two"},
}

func holding(name string) cursortest.ContractHarness[*cursortest.InMemory] {
	return cursortest.ContractHarness[*cursortest.InMemory]{
		Name: name,
		New:  func() *cursortest.InMemory { return cursortest.NewInMemory(held...) },
	}
}

// empty is the cursor with nothing to walk, so exhaustion is reached on the
// first read rather than after two — which is the state the row's claim is
// most easily confused with.
func empty(name string) cursortest.ContractHarness[*cursortest.InMemory] {
	return cursortest.ContractHarness[*cursortest.InMemory]{
		Name: name,
		New:  func() *cursortest.InMemory { return cursortest.NewInMemory() },
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = cursortest.ContractChecks{
	{
		Method: "Next", Name: "refuses-a-read-after-close",
		Claim: "Next refuses a read after the cursor is closed",
		Run:   refusesAReadAfterClose,
		ProvenBy: cursortest.BrokenContract(
			"a cursor that reads a closed read as exhaustion", newClosedReadsAsExhausted,
		),
		ProvenReason: "a read after it is refused",
	},
}

// --- Bodies -------------------------------------------------------------------

func refusesAReadAfterClose(
	tb testing.TB, s cursor.Contract, _ cursortest.ContractFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Close(tb.Context()), "the cursor closes")

	_, ok, err := s.Next(tb.Context())
	testkit.ErrorIs(tb, err, cursor.ErrClosed, "and a read after it is refused")
	testkit.False(tb, ok, "with no value beside the error")
}

// --- Planted defects ----------------------------------------------------------

// closedReadsAsExhausted answers the clean end-of-stream for a cursor the
// caller closed, which is the collapse the row exists to forbid: a loop reading
// until ok is false terminates either way, and the caller never learns it was
// reading something it had already released.
type closedReadsAsExhausted struct{ closed bool }

func newClosedReadsAsExhausted() *closedReadsAsExhausted {
	return &closedReadsAsExhausted{}
}

func (c *closedReadsAsExhausted) Close(context.Context) error {
	c.closed = true
	return nil
}

func (*closedReadsAsExhausted) Next(context.Context) (cursor.Value, bool, error) {
	return cursor.Value{}, false, nil
}
