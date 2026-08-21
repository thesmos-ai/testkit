// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `leakfree` is the model tier's: that every acquire is matched is a claim
// about a whole history of them. The row below is one cycle, which is what a
// single subject settles — and the refusal that keeps the count from going
// negative and hiding an asymmetry.
package leakfreetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/leakfree"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/leakfree/leakfreetest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	leakfreetest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	leakfreetest.RunMixed(t,
		inMemory("in-memory"),
		leakfreetest.MixedSuite.Without(leakfreetest.MixedSuite.Checks.Acquire.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	leakfreetest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) leakfreetest.MixedHarness[*leakfreetest.InMemory] {
	return leakfreetest.MixedHarness[*leakfreetest.InMemory]{
		Name: name, New: leakfreetest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = leakfreetest.MixedChecks{
	{
		Method: "Release", Name: "release-without-acquire-is-refused",
		Claim: "Release refuses a release nothing acquired",
		Run:   releaseWithoutAcquireIsRefused,
		ProvenBy: leakfreetest.BrokenMixed(
			"a pool that counts a release it never lent", newCountsBelowZero,
		),
		ProvenReason: "refused rather than counted",
	},
}

// --- Bodies -------------------------------------------------------------------

// releaseWithoutAcquireIsRefused states the cycle once: acquire, release, and
// the balance is back to zero. A second release then has nothing to give back,
// which is what keeps the count from going negative and hiding an asymmetry.
func releaseWithoutAcquireIsRefused(
	tb testing.TB, s leakfree.Mixed, _ leakfreetest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Acquire(tb.Context()), "the resource is available")
	testkit.NoError(tb, s.Release(tb.Context()), "and giving it back succeeds")

	held, err := s.Outstanding(tb.Context())
	testkit.NoError(tb, err, "the balance is readable")
	testkit.Equal(tb, held, 0, "a completed cycle leaves nothing outstanding")

	testkit.Error(tb, s.Release(tb.Context()),
		"a release with nothing held is refused rather than counted")
}

// --- Planted defects ----------------------------------------------------------

// countsBelowZero takes any release and decrements, so an unmatched one leaves
// the balance negative — which cancels a real leak later and makes the whole
// count useless. It is the reason the row ends on a refusal rather than on the
// zero above it.
type countsBelowZero struct{ held int }

func newCountsBelowZero() *countsBelowZero { return &countsBelowZero{} }

func (c *countsBelowZero) Acquire(context.Context) error {
	c.held++
	return nil
}

func (c *countsBelowZero) Release(context.Context) error {
	c.held--
	return nil
}

func (c *countsBelowZero) Outstanding(context.Context) (int, error) {
	return c.held, nil
}

// TestMixedLawsCanSaturate drives each bound law against defects worn on
// its own methods, with that law as the run's only oracle.
//
// Binding a law is necessary; this is what makes it sufficient. A law
// every worn defect survives is bound and unsaturatable, which reads as
// coverage in the report and is not.
func TestMixedLawsCanSaturate(t *testing.T) {
	t.Parallel()

	leakfreetest.MixedModelSaturation(t, func() leakfreetest.Mixed { return leakfreetest.NewInMemory() })
}
