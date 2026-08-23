// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Two subjects rather than one, which is what a conformance suite is for.
//
// Purity is a property of the method, not of any particular receiver, so both
// implementations answer to one statement of the contract — and a check written
// once runs against each. The generated half is a single smoke call; the law
// the shape carries is that repeated calls agree, and that needs two of them.
package puretest_test

import (
	"strconv"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/pure"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/pure/puretest"
)

// TestPureContract runs the generated check and this package's own, against two
// implementations of the same contract.
func TestPureContract(t *testing.T) {
	t.Parallel()

	puretest.RunPure(t, labelled("in-memory", "first"),
		labelled("in-memory, relabelled", "second"), pureChecks)
}

// TestPureContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestPureContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	puretest.RunPure(t,
		labelled("in-memory", "first"),
		puretest.PureSuite.Without(puretest.PureSuite.Checks.Describe.Smoke()),
	)
}

// TestPureChecksCanFail drives every row against its planted defect.
func TestPureChecksCanFail(t *testing.T) {
	t.Parallel()

	puretest.ProvePure(t, labelled("in-memory", "first"), pureChecks)
}

// --- Harnesses ---------------------------------------------------------------

// labelled is one subject under its own label, which is what makes the pair a
// pair: the same contract, two receivers that answer differently and agree with
// themselves.
func labelled(name, label string) puretest.PureHarness[*puretest.InMemory] {
	return puretest.PureHarness[*puretest.InMemory]{
		Name: name,
		New:  func() *puretest.InMemory { return puretest.NewInMemory(label) },
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var pureChecks = puretest.PureChecks{
	{
		Method: "Describe", Name: "agrees-with-itself",
		Claim: "Describe agrees with itself",
		Run:   agreesWithItself,
		ProvenBy: puretest.BrokenPure(
			"a description that counts the times it was asked", planted(countsItsCalls),
		),
		ProvenReason: "the same value from the same receiver",
	},

	{
		Method: "Describe", Name: "says-something",
		Claim: "Describe says something",
		Run:   saysSomething,
		ProvenBy: puretest.BrokenPure(
			"a description that is empty", planted(saysNothing),
		),
		ProvenReason: "says nothing",
	},
}

// --- Bodies -------------------------------------------------------------------

// agreesWithItself is the whole of the shape's law: nothing was observed
// between the two calls, so nothing may differ between the two answers.
func agreesWithItself(tb testing.TB, s pure.Pure, _ puretest.PureFixture) {
	tb.Helper()
	testkit.Equal(tb, s.Describe(), s.Describe(),
		"repeated calls derive the same value from the same receiver")
}

// saysSomething keeps the row above from being satisfied by returning "".
func saysSomething(tb testing.TB, s pure.Pure, _ puretest.PureFixture) {
	tb.Helper()
	testkit.False(tb, s.Describe() == "",
		"a description that is empty agrees with itself and says nothing")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted description gets wrong.
//
// The pair is deliberate: each defect satisfies the OTHER row. A description
// that counts its calls is never empty, and one that is empty agrees with
// itself — which is why two rows and not one.
type fault int

const (
	// countsItsCalls derives its answer from how often it was asked rather
	// than from the receiver, which is what a memoised value with a leaky
	// counter looks like.
	countsItsCalls fault = iota

	// saysNothing answers the empty string, which agrees with itself
	// perfectly.
	saysNothing
)

// planted builds the constructor for one broken description.
func planted(wrong fault) func() *plantedPure {
	return func() *plantedPure { return &plantedPure{wrong: wrong} }
}

type plantedPure struct {
	wrong fault
	asked int
}

func (p *plantedPure) Describe() string {
	if p.wrong == saysNothing {
		return ""
	}
	p.asked++
	return "asked " + strconv.Itoa(p.asked) + " times"
}
