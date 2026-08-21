// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `associative` is the model tier's: that folding in any grouping lands on the
// same total is a claim about a generated sequence of deltas. The row below is
// what one delta settles — that the fold reaches the total at all.
package associativetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/associative"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/associative/associativetest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	associativetest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	associativetest.RunMixed(t,
		inMemory("in-memory"),
		associativetest.MixedSuite.Without(associativetest.MixedSuite.Checks.Apply.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	associativetest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) associativetest.MixedHarness[*associativetest.InMemory] {
	return associativetest.MixedHarness[*associativetest.InMemory]{
		Name: name, New: associativetest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = associativetest.MixedChecks{
	{
		Method: "Total", Name: "reflects-what-apply-folded",
		Claim: "Total reports what Apply folded in",
		Run:   reflectsWhatApplyFolded,
		ProvenBy: associativetest.BrokenMixed(
			"a fold that accepts every delta and keeps none", newFoldsNothing,
		),
		ProvenReason: "reflects the applied delta",
	},
}

// --- Bodies -------------------------------------------------------------------

// reflectsWhatApplyFolded applies the delta itself: nothing seeds a subject now
// but its own constructor, and a total of zero against an untouched fold would
// state nothing.
func reflectsWhatApplyFolded(
	tb testing.TB, s associative.Mixed, fx associativetest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Apply(tb.Context(), fx.Delta()), "the delta is folded in")

	got, err := s.Total(tb.Context())
	testkit.NoError(tb, err, "the total is readable")
	testkit.NotEqual(tb, got, 0, "and reflects the applied delta")
}

// --- Planted defects ----------------------------------------------------------

// foldsNothing takes every delta and answers the identity, which is
// associative, commutative and every other law a grouping argument can state —
// and reports nothing about the deltas it was given.
type foldsNothing struct{}

func newFoldsNothing() foldsNothing { return foldsNothing{} }

func (foldsNothing) Apply(context.Context, associative.Delta) error { return nil }

func (foldsNothing) Total(context.Context) (int, error) { return 0, nil }
