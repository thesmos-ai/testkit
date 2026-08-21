// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `commutative` is the model tier's: that order does not change the total is a
// claim about two orderings of a generated sequence. The row below is what one
// delta settles — that the fold reaches the total at all.
package commutativetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/commutative"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/commutative/commutativetest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	commutativetest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	commutativetest.RunMixed(t,
		inMemory("in-memory"),
		commutativetest.MixedSuite.Without(commutativetest.MixedSuite.Checks.Apply.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	commutativetest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) commutativetest.MixedHarness[*commutativetest.InMemory] {
	return commutativetest.MixedHarness[*commutativetest.InMemory]{
		Name: name, New: commutativetest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = commutativetest.MixedChecks{
	{
		Method: "Total", Name: "reflects-what-apply-folded",
		Claim: "Total reports what Apply folded in",
		Run:   reflectsWhatApplyFolded,
		ProvenBy: commutativetest.BrokenMixed(
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
	tb testing.TB, s commutative.Mixed, fx commutativetest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Apply(tb.Context(), fx.Delta()), "the delta is folded in")

	got, err := s.Total(tb.Context())
	testkit.NoError(tb, err, "the total is readable")
	testkit.NotEqual(tb, got, 0, "and reflects the applied delta")
}

// --- Planted defects ----------------------------------------------------------

// foldsNothing takes every delta and answers the identity, which commutes
// perfectly with itself in any order — and reports nothing about the deltas it
// was given. It is the reason the law needs this row beneath it: a fold nobody
// can observe satisfies every ordering argument.
type foldsNothing struct{}

func newFoldsNothing() foldsNothing { return foldsNothing{} }

func (foldsNothing) Apply(context.Context, commutative.Delta) error { return nil }

func (foldsNothing) Total(context.Context) (int, error) { return 0, nil }
