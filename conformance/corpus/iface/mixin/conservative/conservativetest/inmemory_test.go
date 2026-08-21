// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `conservative` is the model tier's: that a quantity survives every operation
// is a claim about a whole history. The row below is what one transfer settles
// — that the sum reads as it did at birth.
package conservativetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/conservative"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/conservative/conservativetest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	conservativetest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	conservativetest.RunMixed(t,
		inMemory("in-memory"),
		conservativetest.MixedSuite.Without(conservativetest.MixedSuite.Checks.Apply.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	conservativetest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) conservativetest.MixedHarness[*conservativetest.InMemory] {
	return conservativetest.MixedHarness[*conservativetest.InMemory]{
		Name: name, New: conservativetest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = conservativetest.MixedChecks{
	{
		Method: "Total", Name: "transfer-conserves-the-sum",
		Claim: "Total holds the conserved sum through a transfer",
		Run:   transferConservesTheSum,
		ProvenBy: conservativetest.BrokenMixed(
			"a ledger that credits without debiting", newMintsOnTransfer,
		),
		ProvenReason: "the transfer conserved it",
	},
}

// --- Bodies -------------------------------------------------------------------

// transferConservesTheSum reads the total after a transfer: Apply moves
// quantity rather than adding it, so the conserved sum must still read as it
// did at birth — a non-zero total is quantity minted from nothing, which is the
// mixin's violation.
func transferConservesTheSum(
	tb testing.TB, s conservative.Mixed, fx conservativetest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Apply(tb.Context(), fx.Delta()), "the transfer is applied")

	got, err := s.Total(tb.Context())
	testkit.NoError(tb, err, "the total is readable")
	testkit.Equal(tb, got, 0, "and the transfer conserved it")
}

// --- Planted defects ----------------------------------------------------------

// mintsOnTransfer credits the destination and forgets the debit, which is the
// ledger bug this mixin is named for — and one every check on Apply alone
// passes, since the call itself succeeds.
type mintsOnTransfer struct{ total int }

func newMintsOnTransfer() *mintsOnTransfer { return &mintsOnTransfer{} }

func (m *mintsOnTransfer) Apply(_ context.Context, d conservative.Delta) error {
	m.total += d.Amount
	return nil
}

func (m *mintsOnTransfer) Total(context.Context) (int, error) { return m.total, nil }

// TestMixedLawsCanSaturate drives each bound law against defects worn on
// its own methods, with that law as the run's only oracle.
//
// Binding a law is necessary; this is what makes it sufficient. A law
// every worn defect survives is bound and unsaturatable, which reads as
// coverage in the report and is not.
func TestMixedLawsCanSaturate(t *testing.T) {
	t.Parallel()

	conservativetest.MixedModelSaturation(t, func() conservativetest.Mixed { return conservativetest.NewInMemory() })
}
