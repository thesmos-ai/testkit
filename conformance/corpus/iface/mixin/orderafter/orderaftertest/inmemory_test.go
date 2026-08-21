// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `orderafter after=Prepare unready=` names a prerequisite and the sentinel a
// premature call owes, and the row below is both halves of the ordering claim.
package orderaftertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/orderafter"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/orderafter/orderaftertest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	orderaftertest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	orderaftertest.RunMixed(t,
		inMemory("in-memory"),
		orderaftertest.MixedSuite.Without(orderaftertest.MixedSuite.Checks.Prepare.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	orderaftertest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) orderaftertest.MixedHarness[*orderaftertest.InMemory] {
	return orderaftertest.MixedHarness[*orderaftertest.InMemory]{
		Name: name, New: orderaftertest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = orderaftertest.MixedChecks{
	{
		Method: "Commit", Name: "accepted-after-the-prerequisite",
		Claim: "Commit succeeds once the prerequisite has run",
		Run:   acceptedAfterThePrerequisite,
		ProvenBy: orderaftertest.BrokenMixed(
			"a subject that commits without preparing", newCommitsUnprepared,
		),
		ProvenReason: "committing before the prerequisite is refused",
	},
}

// --- Bodies -------------------------------------------------------------------

// acceptedAfterThePrerequisite asserts both halves, because the ordering claim
// needs both: that a commit is refused before Prepare, and that it is accepted
// after. Nothing says the constraint is the only reason a commit could fail, so
// the second half is not implied by the first.
func acceptedAfterThePrerequisite(
	tb testing.TB, s orderafter.Mixed, _ orderaftertest.MixedFixture,
) {
	tb.Helper()
	testkit.Error(tb, s.Commit(tb.Context()),
		"committing before the prerequisite is refused")

	testkit.NoError(tb, s.Prepare(tb.Context()), "preparing succeeds")
	testkit.NoError(tb, s.Commit(tb.Context()), "and then so does committing")
}

// --- Planted defects ----------------------------------------------------------

// commitsUnprepared takes a commit whatever has happened before it, which is
// the constraint missing rather than the constraint wrong — and the shape every
// generated check on either method passes, since each meets a fresh subject and
// calls one thing.
type commitsUnprepared struct{}

func newCommitsUnprepared() commitsUnprepared { return commitsUnprepared{} }

func (commitsUnprepared) Prepare(context.Context) error { return nil }

func (commitsUnprepared) Commit(context.Context) error { return nil }

// TestMixedLawsCanSaturate drives each bound law against defects worn on
// its own methods, with that law as the run's only oracle.
//
// Binding a law is necessary; this is what makes it sufficient. A law
// every worn defect survives is bound and unsaturatable, which reads as
// coverage in the report and is not.
func TestMixedLawsCanSaturate(t *testing.T) {
	t.Parallel()

	orderaftertest.MixedModelSaturation(t, func() orderaftertest.Mixed { return orderaftertest.NewInMemory() })
}
