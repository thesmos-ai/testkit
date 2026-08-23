// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `sample by=NewInput` names the builder that produces a legal input, and the
// constraint is what makes the mixin worth having: a method taking any string
// needs no builder at all.
package sampletest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sample"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sample/sampletest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	sampletest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	sampletest.RunMixed(t,
		inMemory("in-memory"),
		sampletest.MixedSuite.Without(sampletest.MixedSuite.Checks.Process.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	sampletest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) sampletest.MixedHarness[*sampletest.InMemory] {
	return sampletest.MixedHarness[*sampletest.InMemory]{
		Name: name, New: sampletest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = sampletest.MixedChecks{
	{
		Method: "Process", Name: "accepts-only-what-the-builder-made",
		Claim: "Process refuses an input the builder did not produce",
		Run:   acceptsOnlyWhatTheBuilderMade,
		ProvenBy: sampletest.BrokenMixed(
			"a subject that processes any string it is handed", newProcessesAnything,
		),
		ProvenReason: "an input outside the shape is refused",
	},
}

// --- Bodies -------------------------------------------------------------------

func acceptsOnlyWhatTheBuilderMade(
	tb testing.TB, s sample.Mixed, fx sampletest.MixedFixture,
) {
	tb.Helper()
	built, err := s.NewInput(tb.Context())
	testkit.NoError(tb, err, "the builder produces an input")

	_, err = s.Process(tb.Context(), built)
	testkit.NoError(tb, err, "which the method accepts")

	_, err = s.Process(tb.Context(), unshapedPrefix+fx.Input())
	testkit.Error(tb, err, "an input outside the shape is refused")
}

// --- Planted defects ----------------------------------------------------------

// unshapedPrefix makes an input the builder could not have produced. A prefix
// rather than a literal, so the value still varies with the fixture and only
// its shape is wrong.
const unshapedPrefix = "unshaped-"

// processesAnything takes whatever it is given, which is the constraint absent
// rather than wrong — and which the builder's own output satisfies, so every
// check that only round-trips a built input calls it correct.
type processesAnything struct{}

func newProcessesAnything() processesAnything { return processesAnything{} }

func (processesAnything) NewInput(context.Context) (string, error) {
	return "shaped-input", nil
}

func (processesAnything) Process(_ context.Context, input string) (string, error) {
	return input, nil
}
