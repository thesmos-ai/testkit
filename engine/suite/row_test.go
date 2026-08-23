// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/suite"
)

// rowMethods is the interface a row in these tests is written against.
var rowMethods = suite.NewNameSet("Store", "Put", "Get")

// rowOf is the row every test here starts from: named, claimed, and with
// no body, so each test adds exactly the one it is about.
func rowOf(name string) suite.Row[int, string] {
	return suite.Row[int, string]{Name: name, Claim: "a claim"}
}

// TestBindRowNamesAScopedBodyByItsMethod pins the identity a Run body
// gets: the method it was filed under, then its own name.
//
// The two halves matter separately. A row filed under nothing would be
// unfindable in a rerun, and one filed under a method the interface does
// not declare would be filed where nobody looks — which is what the name
// set is checked against.
func TestBindRowNamesAScopedBodyByItsMethod(t *testing.T) {
	t.Parallel()

	r := rowOf("keeps-what-it-took")
	r.Method = "Put"
	r.Run = func(testing.TB, int, string) {}

	got, err := suite.BindRow(r, "fx", rowMethods).Seal(r.Method)
	testkit.NoError(t, err, "a Run body under a declared method binds")
	testkit.Equal(t, string(got.ID), "Put/keeps-what-it-took", "filed under its method")
	testkit.Equal(t, got.Class, suite.ClassHandWritten, "and grouped as hand-written by default")
}

// TestBindRowRefusesAMethodItDoesNotDeclare is the typo guard.
//
// A misspelled method name looks like any other name, so nothing but the
// interface's own set can tell them apart — and a check filed under a
// method that does not exist is one its author cannot find or drop.
func TestBindRowRefusesAMethodItDoesNotDeclare(t *testing.T) {
	t.Parallel()

	r := rowOf("mine")
	r.Method = "Puut"
	r.Run = func(testing.TB, int, string) {}

	_, err := suite.BindRow(r, "fx", rowMethods).Seal(r.Method)
	testkit.Error(t, err, "the refusal names the method that was not found")
	testkit.Contains(t, err.Error(), "Puut", "the refusal names the method that was not found")
}

// TestBindRowRefusesTwoBodies holds the one-body rule across the seam.
//
// The rule is why this is a builder: a generated struct adds bodies of its
// own after this package has handed the row over, and a count kept on
// either side alone would miss the pair that spans both.
func TestBindRowRefusesTwoBodies(t *testing.T) {
	t.Parallel()

	r := rowOf("two-bodies")
	r.RunWith = func(testing.TB, suite.Subject[int], string) {}

	b := suite.BindRow(r, "fx", rowMethods)
	b.Offers("Prop")
	b.Fixed(suite.HandRowID("two-bodies"), func(testing.TB, suite.Subject[int]) {})

	_, err := b.Seal(r.Method)
	testkit.Error(t, err, "the refusal lists the fields a row may set, this one included")
	testkit.Contains(t, err.Error(), "Prop", "the refusal lists the fields a row may set, this one included")
}

// TestBindRowRefusesNoBody catches the row that declares a claim and
// checks nothing — which reads in a report exactly like one that passed.
func TestBindRowRefusesNoBody(t *testing.T) {
	t.Parallel()

	_, err := suite.BindRow(rowOf("empty"), "fx", rowMethods).Seal("")
	testkit.Error(t, err, "the refusal names the row")
	testkit.Contains(t, err.Error(), "empty", "the refusal names the row")
}

// TestBindRowRefusesMethodBesideAFixedScope covers the two ways a check
// can be named disagreeing.
//
// A body that fixes its own scope has already decided what the check is
// called. A Method beside it is a second answer, and taking either would
// file the check somewhere its author was not expecting.
func TestBindRowRefusesMethodBesideAFixedScope(t *testing.T) {
	t.Parallel()

	r := rowOf("fixed-and-scoped")
	r.Method = "Put"
	r.RunWith = func(testing.TB, suite.Subject[int], string) {}

	_, err := suite.BindRow(r, "fx", rowMethods).Seal(r.Method)
	testkit.Error(t, err, "the refusal names the shorter of the two edits")
	testkit.Contains(t, err.Error(), "drop Method", "the refusal names the shorter of the two edits")
}

// TestBindRowRefusesAProofClaimedBothWays keeps the falsifiability record
// honest: a row cannot both name the implementation that breaks it and
// argue that none can be built.
func TestBindRowRefusesAProofClaimedBothWays(t *testing.T) {
	t.Parallel()

	r := rowOf("both-ways")
	r.Run = func(testing.TB, int, string) {}
	r.Proven, r.Argued = true, "no such implementation exists"

	_, err := suite.BindRow(r, "fx", rowMethods).Seal("")
	testkit.Error(t, err, "the refusal names the row")
	testkit.Contains(t, err.Error(), "both-ways", "the refusal names the row")
}

// TestBindRowHandsTheFixtureToTheBody is the whole reason a row is bound
// to a run rather than read straight: a check written by hand draws the
// values the generated ones draw, and sees an override to them.
func TestBindRowHandsTheFixtureToTheBody(t *testing.T) {
	t.Parallel()

	var saw string
	r := rowOf("reads-the-fixture")
	r.Method = "Get"
	r.Run = func(_ testing.TB, _ int, fx string) { saw = fx }

	got, err := suite.BindRow(r, "the-run's-inputs", rowMethods).Seal(r.Method)
	testkit.NoError(t, err, "the row binds")
	got.Run(t, 0)
	testkit.Equal(t, saw, "the-run's-inputs", "the body was handed this run's fixture")
}
