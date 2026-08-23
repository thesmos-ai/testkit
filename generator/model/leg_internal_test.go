// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/suite"
)

// legBindings is the smallest derivation the bodies read: a name to
// compose identifiers from, a derived reference to compare against, and
// the rows whose bodies are being asked for.
func legBindings(rows ...projection.CheckPlan) *Bindings {
	b := &Bindings{Rows: rows, Reference: Reference{Oracle: OracleMap, CtorName: "NewMixedModelReference"}}
	b.IfaceName = "Mixed"
	return b
}

// legRows are the two this tier plans: one per leg.
func legRows() []projection.CheckPlan {
	return []projection.CheckPlan{
		{ID: projection.IDPlan{Seg: "laws"}, Body: projection.LawLeg{}},
		{ID: projection.IDPlan{Seg: "differential"}, Body: projection.DifferentialLeg{}},
	}
}

// Every planned row gets a body, and every body belongs to a planned row.
//
// The pairing is the whole invariant. A row with no body reaches the
// runtime setting neither Run nor RunWith, which it refuses by name; a
// body with no row is a function nothing calls, in a file the consumer
// may not edit.
func TestEveryRowGetsABody(t *testing.T) {
	t.Parallel()

	b := legBindings(legRows()...)
	legs := legsFor(b, &suite.Contract{})

	testkit.Len(t, legs, len(b.Rows),
		"a row with no body is a check the runtime refuses, and a body with "+
			"no row is dead code in somebody else's file")
}

// The two legs are two kinds, not one kind with a setting.
//
// They differ in the one thing that matters: the differential runs with
// no law registered, so a disagreement is what ends the run; the laws leg
// runs the same sequences with that verdict off, so the laws are the only
// thing that can. One template behind a conditional would put that reason
// where nobody reading either body can see it.
func TestTheLegsAreDistinctKinds(t *testing.T) {
	t.Parallel()

	seen := map[sdk.Kind]int{}
	for _, l := range legsFor(legBindings(legRows()...), &suite.Contract{}) {
		seen[l.Kind()]++
	}
	testkit.Equal(t, seen[KindDifferentialLeg], 1, "one body per leg, and one only")
	testkit.Equal(t, seen[KindLawsLeg], 1, "for each of the two")
}

// A body is named the way its row names it, because the row's RunWith
// calls it by that name.
func TestABodyIsNamedTheWayItsRowNamesIt(t *testing.T) {
	t.Parallel()

	b := legBindings(legRows()...)
	named := map[string]bool{}
	for _, r := range CheckRows(projection.Token(b.IfaceName), b.Rows) {
		named[r.AssertName] = true
	}

	for _, l := range legsFor(b, &suite.Contract{}) {
		switch body := l.(type) {
		case *DifferentialLeg:
			testkit.True(t, named[body.Assert], "the differential body its row names")
		case *LawsLeg:
			testkit.True(t, named[body.Assert], "and the laws body its row names")
		default:
			t.Fatalf("a leg kind no row can name: %T", l)
		}
	}
}

// A twin floor spells its comparison instance rather than naming one:
// there is no derived constructor to call.
func TestTheTwinFloorNamesNoConstructor(t *testing.T) {
	t.Parallel()

	b := legBindings(legRows()...)
	b.Reference = Reference{Oracle: OracleTwin, TwinWhy: "no shipped oracle models this shape"}

	for _, l := range legsFor(b, &suite.Contract{}) {
		body, ok := l.(*DifferentialLeg)
		if !ok {
			continue
		}
		testkit.True(t, body.Twin(),
			"a constructor named here is one this build never generated")
	}
}

// The sample inputs reach a body only when both halves agree: this tier
// reads them and the harness derives them.
//
// Two questions, because they fail in opposite directions. A parameter
// nothing reads does not compile; a call naming inputs the surrounding
// function has no name for does not either.
func TestABodyDrawsOnlyWhatBothSidesHave(t *testing.T) {
	t.Parallel()

	reads := legBindings(legRows()...)
	reads.LawsUseFixture = true

	testkit.True(t, drawsFixture(reads, &suite.Contract{DrawsFixture: true}),
		"the tier reads the inputs and the surface has them")
	testkit.False(t, drawsFixture(reads, &suite.Contract{}),
		"a surface with no inputs has none to hand over")
	testkit.False(t, drawsFixture(legBindings(), &suite.Contract{DrawsFixture: true}),
		"and a tier reading none takes none, or the parameter does not compile")
}

// The crash claim is refused for a store whose acknowledgement does not
// mean "this key holds this record until something else writes it".
//
// Every arm below is a store that breaks that sentence honestly. Running
// the schedule against one would red correct code rather than find a
// lost write, and a red nobody can fix is a red everybody learns to
// ignore — so the row is not planned and the header carries the reason
// in words.
func TestSimLegRefusesWhatItCannotHold(t *testing.T) {
	t.Parallel()

	keyed := func() *Bindings {
		return &Bindings{Reference: Reference{
			Oracle: OracleMap, CtorName: "newRef", KeyField: fieldKey,
		}}
	}

	t.Run("a plain keyed store carries it", func(t *testing.T) {
		t.Parallel()
		b := keyed()
		b.Actions = []*Action{{Pool: poolKeys}, {Pool: poolValues}}
		testkit.Equal(t, simLegReason(b), "", "nothing here breaks the sentence")
	})

	t.Run("a pinning store does not", func(t *testing.T) {
		t.Parallel()
		b := keyed()
		b.Actions = []*Action{{Pool: poolKeys}, {Pool: poolValues}}
		b.Reference.Pins = true
		testkit.Contains(t, simLegReason(b), "pins it",
			"a later acknowledged write is one the store never promised to answer with")
	})

	t.Run("a deduplicating store does not", func(t *testing.T) {
		t.Parallel()
		b := keyed()
		b.Actions = []*Action{{Pool: poolKeys}, {Pool: poolValues}}
		b.Reference.Dedupe = true
		testkit.Contains(t, simLegReason(b), "acknowledged and not installed",
			"and the schedule holds every acknowledgement to a read")
	})

	t.Run("a store that stamps what it stores does not", func(t *testing.T) {
		t.Parallel()
		b := keyed()
		b.Actions = []*Action{{Pool: poolKeys}, {Pool: poolValues}}
		b.Reference.VersionField = "Version"
		testkit.Contains(t, simLegReason(b), "stamps what it stores",
			"a read answers a record the write did not hand it")
	})

	t.Run("a run with no pool to draw from does not", func(t *testing.T) {
		t.Parallel()
		b := keyed()
		testkit.Contains(t, simLegReason(b), "declares no pool",
			"the schedule draws a key to read and a record to write")
	})

	t.Run("a store with no key projection does not", func(t *testing.T) {
		t.Parallel()
		b := keyed()
		b.Actions = []*Action{{Pool: poolKeys}, {Pool: poolValues}}
		b.Reference.KeyField = ""
		testkit.Contains(t, simLegReason(b), "no projection derives",
			"a write is filed under the key it lands on, and nothing says which member")
	})
}
