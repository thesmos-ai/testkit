// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `total` is the model tier's: totality over a generated corpus needs the
// corpus. The row below carries the one input a consumer can name that a
// derivation never draws — the empty string.
package totaltest_test

import (
	"context"
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/total"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/total/totaltest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	totaltest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	totaltest.RunMixed(t,
		inMemory("in-memory"),
		totaltest.MixedSuite.Without(totaltest.MixedSuite.Checks.Classify.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	totaltest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) totaltest.MixedHarness[*totaltest.InMemory] {
	return totaltest.MixedHarness[*totaltest.InMemory]{
		Name: name, New: totaltest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = totaltest.MixedChecks{
	{
		Method: "Classify", Name: "answers-for-the-empty-string",
		Claim: "Classify answers for the empty string as readily as for any other",
		Run:   answersForTheEmptyString,
		ProvenBy: totaltest.BrokenMixed(
			"a subject total over non-empty strings", newRefusesTheEmpty,
		),
		ProvenReason: "the empty string is in the domain",
	},
}

// --- Bodies -------------------------------------------------------------------

// answersForTheEmptyString names the input derivation never draws. A subject
// that refused it would be total over "non-empty strings", which is a different
// claim.
func answersForTheEmptyString(
	tb testing.TB, s total.Mixed, fx totaltest.MixedFixture,
) {
	tb.Helper()
	got, err := s.Classify(tb.Context(), "")
	testkit.NoError(tb, err, "the empty string is in the domain")
	testkit.Equal(tb, got, "empty", "and is classified rather than refused")

	_, err = s.Classify(tb.Context(), fx.In())
	testkit.NoError(tb, err, "and so is anything else")
}

// --- Planted defects ----------------------------------------------------------

// errUnclassifiable is what refusesTheEmpty answers with. The interface
// declares no sentinel for it, which is the point of the row: nothing in this
// domain should ever be unclassifiable.
var errUnclassifiable = errors.New("totaltest_test: nothing to classify")

// refusesTheEmpty is total over every input a generator would think to draw and
// not over the domain it declared, which is the gap the row exists to close.
type refusesTheEmpty struct{}

func newRefusesTheEmpty() refusesTheEmpty { return refusesTheEmpty{} }

func (refusesTheEmpty) Classify(_ context.Context, in string) (string, error) {
	if in == "" {
		return "", errUnclassifiable
	}
	return "nonempty", nil
}

func (refusesTheEmpty) Normalize(in string) string { return in }

// TestMixedLawsCanSaturate drives each bound law against defects worn on
// its own methods, with that law as the run's only oracle.
//
// Binding a law is necessary; this is what makes it sufficient. A law
// every worn defect survives is bound and unsaturatable, which reads as
// coverage in the report and is not.
func TestMixedLawsCanSaturate(t *testing.T) {
	t.Parallel()

	totaltest.MixedModelSaturation(t, func() totaltest.Mixed { return totaltest.NewInMemory() })
}
