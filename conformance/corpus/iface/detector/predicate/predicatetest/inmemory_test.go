// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// A bare predicate earns one derived check: it takes nothing to vary and
// reports nothing to compare, so the smoke call is the whole signature family.
// Which way it should answer is the row's.
package predicatetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/predicate"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/predicate/predicatetest"
)

// TestPredicateContract runs the generated check and this package's own.
func TestPredicateContract(t *testing.T) {
	t.Parallel()

	predicatetest.RunPredicate(t, inMemory("in-memory"), predicateChecks)
}

// TestPredicateContractWithoutSmoke drops a check through the typed index
// rather than a string, so a check that is renamed or stops being emitted
// breaks this compile instead of silently declining nothing.
func TestPredicateContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	predicatetest.RunPredicate(t,
		inMemory("in-memory"),
		predicatetest.PredicateSuite.Without(predicatetest.PredicateSuite.Checks.IsEmpty.Smoke()),
	)
}

// TestPredicateChecksCanFail drives the row against its planted defect.
func TestPredicateChecksCanFail(t *testing.T) {
	t.Parallel()

	predicatetest.ProvePredicate(t, predicateChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) predicatetest.PredicateHarness[*predicatetest.InMemory] {
	return predicatetest.PredicateHarness[*predicatetest.InMemory]{
		Name: name, New: predicatetest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var predicateChecks = predicatetest.PredicateChecks{
	{
		Method: "IsEmpty", Name: "true-on-a-fresh-subject",
		Claim: "IsEmpty reports true for a fresh subject",
		Run:   trueOnAFreshSubject,
		ProvenBy: predicatetest.BrokenPredicate(
			"a predicate that is never empty", newNeverEmpty,
		),
		ProvenReason: "nothing has been added yet",
	},
}

// --- Bodies -------------------------------------------------------------------

func trueOnAFreshSubject(
	tb testing.TB, s predicate.Predicate, _ predicatetest.PredicateFixture,
) {
	tb.Helper()
	testkit.True(tb, s.IsEmpty(), "nothing has been added yet")
}

// --- Planted defects ----------------------------------------------------------

// neverEmpty answers false whatever has happened to it, which is the one thing
// a smoke call cannot tell from a correct predicate: both survive being asked.
type neverEmpty struct{}

func newNeverEmpty() neverEmpty { return neverEmpty{} }

func (neverEmpty) IsEmpty() bool { return false }

// TestPredicateLawsCanSaturate drives each bound law against defects worn on
// its own methods, with that law as the run's only oracle.
//
// Binding a law is necessary; this is what makes it sufficient. A law
// every worn defect survives is bound and unsaturatable, which reads as
// coverage in the report and is not.
func TestPredicateLawsCanSaturate(t *testing.T) {
	t.Parallel()

	predicatetest.PredicateModelSaturation(t, func() predicatetest.Predicate { return predicatetest.NewInMemory() })
}
