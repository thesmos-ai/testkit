// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `wrappedvia` names the cause a failure must unwrap to. Which INPUT a method
// refuses is exactly what no signature says, so the failing name is the
// subject's rather than the fixture's.
package wrappedviatest_test

import (
	"context"
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/wrappedvia"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/wrappedvia/wrappedviatest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	wrappedviatest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	wrappedviatest.RunMixed(t,
		inMemory("in-memory"),
		wrappedviatest.MixedSuite.Without(wrappedviatest.MixedSuite.Checks.Open.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	wrappedviatest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) wrappedviatest.MixedHarness[*wrappedviatest.InMemory] {
	return wrappedviatest.MixedHarness[*wrappedviatest.InMemory]{
		Name: name, New: wrappedviatest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = wrappedviatest.MixedChecks{
	{
		Method: "Open", Name: "wraps-the-cause",
		Claim: "Open wraps the cause it reports",
		Run:   wrapsTheCause,
		ProvenBy: wrappedviatest.BrokenMixed(
			"a subject that reports a failure of its own", newLosesTheCause,
		),
		ProvenReason: "unwraps to the cause",
	},
}

// --- Bodies -------------------------------------------------------------------

func wrapsTheCause(
	tb testing.TB, s wrappedvia.Mixed, _ wrappedviatest.MixedFixture,
) {
	tb.Helper()
	err := s.Open(tb.Context(), wrappedviatest.FailingName())
	testkit.ErrorIs(tb, err, wrappedviatest.ErrUnderlying,
		"what Open returns unwraps to the cause")
	testkit.ErrorIs(tb, s.Cause(tb.Context()), wrappedviatest.ErrUnderlying,
		"and Cause reports the same one")
}

// --- Planted defects ----------------------------------------------------------

// losesTheCause fails for the right input and reports an error of its own
// making, which is fmt.Errorf without the %w — the single most common way a
// cause stops being reachable, and one that reads identically at the call site.
type losesTheCause struct{}

func newLosesTheCause() losesTheCause { return losesTheCause{} }

func (losesTheCause) Open(_ context.Context, name string) error {
	if name == wrappedviatest.FailingName() {
		return errors.New("wrappedviatest_test: opening " + name + " failed")
	}
	return nil
}

func (losesTheCause) Cause(context.Context) error { return wrappedviatest.ErrUnderlying }
