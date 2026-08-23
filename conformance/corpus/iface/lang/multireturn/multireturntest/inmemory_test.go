// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// A method returning several values beside its error owes the zero for each.
package multireturntest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/multireturn"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/multireturn/multireturntest"
)

// TestWideContract runs the generated checks and this package's own.
func TestWideContract(t *testing.T) {
	t.Parallel()

	multireturntest.RunWide(t, inMemory("in-memory"), wideChecks)
}

// TestWideContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestWideContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	multireturntest.RunWide(t,
		inMemory("in-memory"),
		multireturntest.WideSuite.Without(multireturntest.WideSuite.Checks.Triple.Smoke()),
	)
}

// TestWideChecksCanFail drives every row against its planted defect.
func TestWideChecksCanFail(t *testing.T) {
	t.Parallel()

	multireturntest.ProveWide(t, inMemory("in-memory"), wideChecks)
}

// TestQuadZeroOnErrorReadsEverySlot is a package test rather than a row: a
// subject zeroing only its first slot must fail, or the GENERATED check is
// reading one of three and reporting on all of them.
//
// This is what running the fixture adds over compiling it: a template emitting
// one comparison instead of three compiles identically, and only a subject that
// zeroes some slots and not others can tell the two apart. The check is reached
// as data, since the assertion functions are unexported.
func TestQuadZeroOnErrorReadsEverySlot(t *testing.T) {
	t.Parallel()

	fx := multireturntest.DefaultWideFixture()
	want := multireturntest.WideSuite.Checks.Quad.ZeroOnError()

	var zeroOnError func(tb testing.TB, s multireturn.Wide)
	for _, c := range multireturntest.WideSuite.Suite(fx).Checks {
		if c.ID == want {
			zeroOnError = c.Run
		}
	}
	testkit.True(t, zeroOnError != nil, "the run emits the check this test is about")

	f := testkit.NewFailableTB()
	zeroOnError(f, multireturntest.PartialZero{InMemory: multireturntest.NewInMemory()})

	testkit.True(t, f.Failed(),
		"a non-zero slot beside an error must fail, whichever slot it is")
}

// --- Harnesses ---------------------------------------------------------------

// theAnswer is what a seeded identifier holds. Its LENGTH is the derived slot,
// which is what makes the two answers checkable against each other rather than
// against two literals a subject could satisfy independently.
const theAnswer = "found"

func inMemory(name string) multireturntest.WideHarness[*multireturntest.InMemory] {
	return multireturntest.WideHarness[*multireturntest.InMemory]{Name: name, New: seeded}
}

// seeded holds the fixture's identifier, so a hit has something to hit.
func seeded() *multireturntest.InMemory {
	s := multireturntest.NewInMemory()
	s.Put(multireturntest.DefaultWideFixture().ID(), theAnswer)
	return s
}

// --- The checks: claims, bodies and defects, by name --------------------------
//
// Each row plants a subject that fills SOME of its slots, which is the failure
// a wide return has and a narrow one cannot: every one of these would pass a
// check reading only the first value back.

var wideChecks = multireturntest.WideChecks{
	{
		Method: "Quad", Name: "every-slot-for-a-hit",
		Claim: "Quad returns every slot for a hit",
		Run:   everySlotForAHit,
		ProvenBy: multireturntest.BrokenWide(
			"a subject whose derived slot stays zero on a hit",
			planted(forgetsTheDerivedSlot),
		),
		ProvenReason: "the derived slot agrees",
	},

	{
		Method: "Triple", Name: "absence-through-the-flag",
		Claim: "Triple reports absence through its flag",
		Run:   absenceThroughTheFlag,
		ProvenBy: multireturntest.BrokenWide(
			"a subject whose flag is true whatever it was asked",
			planted(flagsEverythingPresent),
		),
		ProvenReason: "says so through the flag",
	},

	{
		Method: "NoError", Name: "zero-for-an-absent-identifier",
		Claim: "NoError returns the zero for an absent identifier",
		Run:   zeroForAnAbsentIdentifier,
		ProvenBy: multireturntest.BrokenWide(
			"a subject that answers an identifier it does not hold",
			planted(answersOnAMiss),
		),
		ProvenReason: "the value slot is zero",
	},
}

// --- Bodies -------------------------------------------------------------------

func everySlotForAHit(
	tb testing.TB, s multireturn.Wide, fx multireturntest.WideFixture,
) {
	tb.Helper()
	v, n, ok, err := s.Quad(tb.Context(), fx.ID())
	testkit.NoError(tb, err, "a seeded identifier is found")
	testkit.Equal(tb, v, theAnswer, "the value slot carries it")
	testkit.Equal(tb, n, len(theAnswer), "the derived slot agrees")
	testkit.True(tb, ok, "and the flag says so")
}

func absenceThroughTheFlag(
	tb testing.TB, s multireturn.Wide, fx multireturntest.WideFixture,
) {
	tb.Helper()
	_, _, ok := s.Triple(tb.Context(), fx.IDOther())
	testkit.False(tb, ok, "a method with no error slot says so through the flag")
}

func zeroForAnAbsentIdentifier(
	tb testing.TB, s multireturn.Wide, fx multireturntest.WideFixture,
) {
	tb.Helper()
	v, n := s.NoError(tb.Context(), fx.IDOther())
	testkit.Equal(tb, v, "", "the value slot is zero")
	testkit.Equal(tb, n, 0, "and so is the derived one")
}

// --- Planted defects ----------------------------------------------------------

// fault names which slot one planted subject gets wrong.
type fault int

const (
	// forgetsTheDerivedSlot answers the value and leaves the number beside
	// it at zero, which is the shape a subject computing one field and
	// forgetting the other has.
	forgetsTheDerivedSlot fault = iota

	// flagsEverythingPresent reports every identifier as held, so a caller
	// reading the flag processes a zero value as a real one.
	flagsEverythingPresent

	// answersOnAMiss fills the value slots for an identifier it does not
	// hold, which is a stale buffer handed back uncleared.
	answersOnAMiss
)

// planted builds the constructor for one broken subject, holding what the
// harness's own subject holds so a hit has the same thing to hit.
func planted(wrong fault) func() plantedWide {
	return func() plantedWide {
		return plantedWide{wrong: wrong, id: multireturntest.DefaultWideFixture().ID()}
	}
}

type plantedWide struct {
	wrong fault
	id    string
}

func (p plantedWide) Quad(_ context.Context, id string) (string, int, bool, error) {
	if id != p.id {
		return "", 0, false, multireturntest.ErrNotFound
	}
	if p.wrong == forgetsTheDerivedSlot {
		return theAnswer, 0, true, nil
	}
	return theAnswer, len(theAnswer), true, nil
}

func (p plantedWide) Triple(_ context.Context, id string) (string, int, bool) {
	if p.wrong == flagsEverythingPresent {
		return theAnswer, len(theAnswer), true
	}
	if id != p.id {
		return "", 0, false
	}
	return theAnswer, len(theAnswer), true
}

func (p plantedWide) NoError(_ context.Context, id string) (string, int) {
	if p.wrong == answersOnAMiss {
		return theAnswer, len(theAnswer)
	}
	if id != p.id {
		return "", 0
	}
	return theAnswer, len(theAnswer)
}
