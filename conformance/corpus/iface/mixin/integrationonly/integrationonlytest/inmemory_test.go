// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `integrationonly` marks a method a unit run cannot exercise for real, and
// what is left is what it refuses. A well-formed target is the row's to supply:
// the derived draw is a plausible string rather than a URL, which is the half
// of this claim derivation cannot reach.
package integrationonlytest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/integrationonly"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/integrationonly/integrationonlytest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	integrationonlytest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	integrationonlytest.RunMixed(t,
		inMemory("in-memory"),
		integrationonlytest.MixedSuite.Without(integrationonlytest.MixedSuite.Checks.Connect.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	integrationonlytest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) integrationonlytest.MixedHarness[*integrationonlytest.InMemory] {
	return integrationonlytest.MixedHarness[*integrationonlytest.InMemory]{
		Name: name, New: integrationonlytest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = integrationonlytest.MixedChecks{
	{
		Method: "Connect", Name: "refuses-an-unparseable-target",
		Claim: "Connect refuses a target it cannot parse",
		Run:   refusesAnUnparseableTarget,
		ProvenBy: integrationonlytest.BrokenMixed(
			"a client that accepts any string as a target", newAcceptsAnything,
		),
		ProvenReason: "a target with no scheme is refused",
	},
}

// --- Bodies -------------------------------------------------------------------

func refusesAnUnparseableTarget(
	tb testing.TB, s integrationonly.Mixed, _ integrationonlytest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Connect(tb.Context(), wellFormed),
		"a target with a scheme is accepted")

	testkit.ErrorIs(tb, s.Connect(tb.Context(), unparseable), integrationonlytest.ErrBadDSN,
		"a target with no scheme is refused")
}

// --- Planted defects ----------------------------------------------------------

// The two targets the row supplies. Both are the row's rather than the
// fixture's: a derived draw is a plausible string, and neither half of this
// claim is about plausible strings.
const (
	wellFormed  = "postgres://localhost/primary"
	unparseable = "not-a-dsn"
)

// acceptsAnything takes every target and fails at first use rather than at
// configuration, which is the wrong end of the deployment — and exactly what a
// check that only calls Connect with a derived string calls correct.
type acceptsAnything struct{}

func newAcceptsAnything() acceptsAnything { return acceptsAnything{} }

func (acceptsAnything) Connect(context.Context, string) error { return nil }
