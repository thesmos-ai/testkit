// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Whether the source named its results changes the declaration and nothing
// else, so all three spellings are held to one contract.
//
// The author's own rule rides along in the same run: that the three agree. No
// classification says so — it is a fact about this interface that a reader of
// the source would assume and nothing would check.
package namedreturnstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/namedreturns"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/namedreturns/namedreturnstest"
)

// TestServiceContract runs the generated checks and this package's own.
func TestServiceContract(t *testing.T) {
	t.Parallel()

	namedreturnstest.RunService(t, inMemory("in-memory"), serviceChecks)
}

// TestServiceContractSuppression drops a check, which is how a consumer keeps a
// suite they would otherwise abandon. Run against the same subject, because
// what is under test is the suppression rather than the implementation.
func TestServiceContractSuppression(t *testing.T) {
	t.Parallel()

	namedreturnstest.RunService(t,
		inMemory("in-memory"),
		namedreturnstest.ServiceSuite.Without(namedreturnstest.ServiceSuite.Checks.Named.Smoke()),
	)
}

// TestServiceChecksCanFail drives the row against its planted defect.
func TestServiceChecksCanFail(t *testing.T) {
	t.Parallel()

	namedreturnstest.ProveService(t, inMemory("in-memory"), serviceChecks)
}

// --- Harnesses ---------------------------------------------------------------

// storedBody is what the seeded identifier holds under all three spellings.
const storedBody = "stored"

// inMemory seeds the subject: nothing on this interface writes, so the seed is
// the constructor's. The identifier comes off the fixture rather than being
// written out, so it and the row cannot disagree.
func inMemory(name string) namedreturnstest.ServiceHarness[*namedreturnstest.InMemory] {
	return namedreturnstest.ServiceHarness[*namedreturnstest.InMemory]{
		Name: name, New: seeded,
	}
}

func seeded() *namedreturnstest.InMemory {
	s := namedreturnstest.NewInMemory()
	s.Put(namedreturnstest.DefaultServiceFixture().ID(), storedBody)
	return s
}

// --- The checks: claims, bodies and defects, by name --------------------------

var serviceChecks = namedreturnstest.ServiceChecks{
	{
		Method: "Named", Name: "three-spellings-agree",
		Claim: "Named agrees with the other two spellings",
		Run:   threeSpellingsAgree,
		ProvenBy: namedreturnstest.BrokenService(
			"a subject whose unnamed spelling drifted", newSpellingsDisagree,
		),
		ProvenReason: "the anonymous form answers alike",
	},
}

// --- Bodies -------------------------------------------------------------------

func threeSpellingsAgree(
	tb testing.TB, s namedreturns.Service, fx namedreturnstest.ServiceFixture,
) {
	tb.Helper()
	named, err := s.Named(tb.Context(), fx.ID())
	testkit.NoError(tb, err, "a seeded identifier is found")

	unnamed, _ := s.Unnamed(tb.Context(), fx.ID())
	partial, _ := s.PartiallyNamed(tb.Context(), fx.ID())
	testkit.Equal(tb, unnamed, named, "the anonymous form answers alike")
	testkit.Equal(tb, partial, named, "and so does the partially named one")
}

// --- Planted defects ----------------------------------------------------------

// spellingsDisagree answers all three without failing and answers them
// differently, which is what three implementations of one lookup drift into.
// Every generated check calls one method, so none of them can see it.
type spellingsDisagree struct{}

func newSpellingsDisagree() spellingsDisagree { return spellingsDisagree{} }

func (spellingsDisagree) Named(_ context.Context, id string) (string, error) {
	return id + "-named", nil
}

func (spellingsDisagree) Unnamed(_ context.Context, id string) (string, error) {
	return id + "-unnamed", nil
}

func (spellingsDisagree) PartiallyNamed(_ context.Context, id string) (string, error) {
	return id + "-partial", nil
}
