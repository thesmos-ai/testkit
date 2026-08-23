// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `pure` is the model tier's: purity over a generated corpus of inputs needs
// the corpus. What one input settles is the whole of the mixin's law in its
// smallest form, and a claim one call cannot make.
package puretest_test

import (
	"strconv"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/pure"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/pure/puretest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	puretest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	puretest.RunMixed(t,
		inMemory("in-memory"),
		puretest.MixedSuite.Without(puretest.MixedSuite.Checks.Derive.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	puretest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) puretest.MixedHarness[*puretest.InMemory] {
	return puretest.MixedHarness[*puretest.InMemory]{
		Name: name, New: puretest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = puretest.MixedChecks{
	{
		Method: "Derive", Name: "agrees-with-itself",
		Claim: "Derive agrees with itself",
		Run:   agreesWithItself,
		ProvenBy: puretest.BrokenMixed(
			"a derivation that counts the times it was asked", newCountsItsCalls,
		),
		ProvenReason: "repeated calls on one input agree",
	},
}

// --- Bodies -------------------------------------------------------------------

// agreesWithItself is the whole of the mixin's law: nothing was observed
// between the two calls, so nothing may differ between the two answers.
func agreesWithItself(tb testing.TB, s pure.Mixed, fx puretest.MixedFixture) {
	tb.Helper()
	testkit.Equal(tb, s.Derive(fx.Input()), s.Derive(fx.Input()),
		"repeated calls on one input agree")
}

// --- Planted defects ----------------------------------------------------------

// countsItsCalls folds the call number into its answer, which is what a
// derivation reading a counter, a cache generation or a clock does — and which
// a single call cannot tell from a correct one.
type countsItsCalls struct{ asked int }

func newCountsItsCalls() *countsItsCalls { return &countsItsCalls{} }

func (c *countsItsCalls) Derive(input string) string {
	c.asked++
	return input + "-" + strconv.Itoa(c.asked)
}
