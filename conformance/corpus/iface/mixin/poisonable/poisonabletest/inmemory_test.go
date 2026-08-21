// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `poisonable induce=Fail` names the door a run drives a subject through, and
// what a poisoned subject then owes is the row below: the state it was driven
// into is one it keeps.
package poisonabletest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/poisonable"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/poisonable/poisonabletest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	poisonabletest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	poisonabletest.RunMixed(t,
		inMemory("in-memory"),
		poisonabletest.MixedSuite.Without(poisonabletest.MixedSuite.Checks.Fail.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	poisonabletest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) poisonabletest.MixedHarness[*poisonabletest.InMemory] {
	return poisonabletest.MixedHarness[*poisonabletest.InMemory]{
		Name: name, New: poisonabletest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = poisonabletest.MixedChecks{
	{
		Method: "Probe", Name: "latches-the-state-it-was-driven-into",
		Claim: "Probe keeps reporting the state it was driven into",
		Run:   latchesTheStateItWasDrivenInto,
		ProvenBy: poisonabletest.BrokenMixed(
			"a subject whose failure clears on being read", newClearsOnRead,
		),
		ProvenReason: "reading it does not clear it",
	},
}

// --- Bodies -------------------------------------------------------------------

func latchesTheStateItWasDrivenInto(
	tb testing.TB, s poisonable.Mixed, _ poisonabletest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Probe(), "a fresh subject is healthy")
	testkit.NoError(tb, s.Fail(tb.Context()), "and can be driven to fail")

	testkit.Error(tb, s.Probe(), "the failure is reported")
	testkit.Error(tb, s.Probe(), "and reading it does not clear it")
}

// --- Planted defects ----------------------------------------------------------

// clearsOnRead reports the failure once and forgets it, which is a status flag
// consumed by the reader rather than latched. Every check making ONE probe
// calls it correct, which is why the row makes two.
type clearsOnRead struct{ failed bool }

func newClearsOnRead() *clearsOnRead { return &clearsOnRead{} }

func (c *clearsOnRead) Fail(context.Context) error {
	c.failed = true
	return nil
}

func (c *clearsOnRead) Probe() error {
	if !c.failed {
		return nil
	}
	c.failed = false
	return poisonabletest.ErrPoisoned
}
