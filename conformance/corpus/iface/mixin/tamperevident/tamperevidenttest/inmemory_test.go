// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `tamperevident` is the model tier's: detecting an alteration needs one
// induced, which is the model tier's to arrange. Corrupt is on the interface
// here precisely so a consumer can arrange it too, and the row below does.
package tamperevidenttest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/tamperevident"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/tamperevident/tamperevidenttest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	tamperevidenttest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	tamperevidenttest.RunMixed(t,
		inMemory("in-memory"),
		tamperevidenttest.MixedSuite.Without(tamperevidenttest.MixedSuite.Checks.Store.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	tamperevidenttest.ProveMixed(t, mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) tamperevidenttest.MixedHarness[*tamperevidenttest.InMemory] {
	return tamperevidenttest.MixedHarness[*tamperevidenttest.InMemory]{
		Name: name, New: tamperevidenttest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = tamperevidenttest.MixedChecks{
	{
		Method: "Verify", Name: "detects-an-alteration",
		Claim: "Verify detects a value altered behind its back",
		Run:   detectsAnAlteration,
		ProvenBy: tamperevidenttest.BrokenMixed(
			"a subject whose verify always agrees", newAlwaysVerifies,
		),
		ProvenReason: "detected rather than served",
	},
}

// --- Bodies -------------------------------------------------------------------

// detectsAnAlteration stores first, because Corrupt reaches past the interface
// and there has to be something for it to reach.
func detectsAnAlteration(
	tb testing.TB, s tamperevident.Mixed, fx tamperevidenttest.MixedFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Store(tb.Context(), fx.Body()), "a value is stored")

	testkit.NoError(tb, s.Verify(tb.Context()), "an untouched value verifies")
	testkit.NoError(tb, s.Corrupt(tb.Context()), "the bytes are altered")
	testkit.Error(tb, s.Verify(tb.Context()),
		"and the alteration is detected rather than served")
}

// --- Planted defects ----------------------------------------------------------

// alwaysVerifies answers clean whatever happened to the bytes, which is a
// checksum computed from what was read rather than from what was written — the
// classic way to build a tamper check that detects nothing.
type alwaysVerifies struct{ body string }

func newAlwaysVerifies() *alwaysVerifies { return &alwaysVerifies{} }

func (a *alwaysVerifies) Store(_ context.Context, body string) error {
	a.body = body
	return nil
}

func (a *alwaysVerifies) Corrupt(context.Context) error {
	a.body += "-altered"
	return nil
}

func (*alwaysVerifies) Verify(context.Context) error { return nil }

// TestMixedLawsCanSaturate drives each bound law against defects worn on
// its own methods, with that law as the run's only oracle.
//
// Binding a law is necessary; this is what makes it sufficient. A law
// every worn defect survives is bound and unsaturatable, which reads as
// coverage in the report and is not.
func TestMixedLawsCanSaturate(t *testing.T) {
	t.Parallel()

	tamperevidenttest.MixedModelSaturation(t, func() tamperevidenttest.Mixed { return tamperevidenttest.NewInMemory() })
}
