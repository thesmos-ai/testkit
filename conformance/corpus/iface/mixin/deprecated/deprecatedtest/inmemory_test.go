// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `deprecated replacedby=New` names what took over, and the row below is the
// only claim worth making about the old spelling: that it has not quietly
// diverged from what replaced it.
package deprecatedtest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/deprecated"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/deprecated/deprecatedtest"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	deprecatedtest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	deprecatedtest.RunMixed(t,
		inMemory("in-memory"),
		deprecatedtest.MixedSuite.Without(deprecatedtest.MixedSuite.Checks.Old.Smoke()),
	)
}

// TestMixedChecksCanFail drives the row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	deprecatedtest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

// storedBody is what the seeded key holds under both spellings.
const storedBody = "stored"

// inMemory seeds the subject: nothing on this interface writes, so the seed is
// the constructor's — both spellings read, neither stores.
func inMemory(name string) deprecatedtest.MixedHarness[*deprecatedtest.InMemory] {
	return deprecatedtest.MixedHarness[*deprecatedtest.InMemory]{Name: name, New: seeded}
}

func seeded() *deprecatedtest.InMemory {
	s := deprecatedtest.NewInMemory()
	s.Put(deprecatedtest.DefaultMixedFixture().Key(), storedBody)
	return s
}

// --- The checks: claims, bodies and defects, by name --------------------------

var mixedChecks = deprecatedtest.MixedChecks{
	{
		Method: "Old", Name: "agrees-with-the-replacement",
		Claim: "Old answers as the replacement does",
		Run:   agreesWithTheReplacement,
		ProvenBy: deprecatedtest.BrokenMixed(
			"a subject whose old spelling drifted", newDrifted,
		),
		ProvenReason: "the two agree",
	},
}

// --- Bodies -------------------------------------------------------------------

func agreesWithTheReplacement(
	tb testing.TB, s deprecated.Mixed, fx deprecatedtest.MixedFixture,
) {
	tb.Helper()
	old, oldErr := s.Old(tb.Context(), fx.Key())
	replacement, newErr := s.New(tb.Context(), fx.Key())
	testkit.NoError(tb, oldErr, "the deprecated spelling still works")
	testkit.NoError(tb, newErr, "and so does the replacement")
	testkit.Equal(tb, old, replacement, "and the two agree")
}

// --- Planted defects ----------------------------------------------------------

// drifted answers both spellings without failing and answers them differently,
// which is what a deprecated path left behind when its replacement was fixed
// looks like — and what every check calling one method cannot see.
type drifted struct{}

func newDrifted() drifted { return drifted{} }

func (drifted) Old(_ context.Context, key string) (string, error) {
	return key + "-old", nil
}

func (drifted) New(_ context.Context, key string) (string, error) {
	return key + "-new", nil
}
