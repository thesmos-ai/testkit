// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/subject"
	"go.thesmos.sh/testkit/generator/suite"
)

// A refusal is recorded in the generated file, not swallowed.
//
// A directive honoured in every other package and silently dropped in
// this one is the absence nothing else in the output would show — so the
// two cases that cannot be served say so where a reader is already
// looking.
func TestARefusalSaysWhy(t *testing.T) {
	t.Parallel()

	t.Run("a generic harness carries generic checks these rows cannot join", func(t *testing.T) {
		t.Parallel()
		generic := &suite.Contract{}
		generic.TypeParams = []*sdk.EmitTypeParam{{Name: "V"}}

		why := declineReason(&Bindings{}, generic)
		testkit.True(t, len(why) > 0, "the generic case is refused")
		testkit.Contains(t, strings.Join(why, " "), WitnessKey,
			"and names the key whose concrete types are the obstacle")
	})

	t.Run("inputs nothing derives are inputs nothing can draw", func(t *testing.T) {
		t.Parallel()
		reads := &Bindings{LawsUseFixture: true}
		testkit.True(t, len(declineReason(reads, &suite.Contract{})) > 0,
			"a tier reading sample inputs the harness never derived is refused")
		testkit.Len(t, declineReason(reads, &suite.Contract{DrawsFixture: true}), 0,
			"and served the moment the harness has them")
	})
}

// TestAppendLegGuards holds the append leg to its own eligibility: an
// appender whose offsets are not int64, or whose method drives nothing,
// derives no leg.
func TestAppendLegGuards(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	run := func(offset *node.TypeRef) *subject.Method {
		m := projected("Run",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", pkgRef("example.com/a", "Value"))},
			[]golang.Return{res(offset), errRet})
		m.Contracts = []string{"appender"}
		return m
	}

	t.Run("a non-int64 offset keeps the sequential law alone", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Actions: []*Action{{Method: "Run", Pool: poolValues}}}
		a, _ := appendActionOf(b, &subject.Projection{Methods: []subject.Method{*run(namedRef("string"))}})
		testkit.True(t, a == nil, "the shared-history model counts in int64")
	})

	t.Run("an undriven appender derives no leg", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{}
		a, _ := appendActionOf(b, &subject.Projection{Methods: []subject.Method{*run(namedRef("int64"))}})
		testkit.True(t, a == nil, "no action, nothing to interleave")
	})

	t.Run("a driven int64 appender derives the leg", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Actions: []*Action{{Method: "Run", Pool: poolValues}}}
		concurrentOf(b, &subject.Projection{Methods: []subject.Method{*run(namedRef("int64"))}}, nil, nil)
		testkit.Equal(t, b.ConcFamily, concFamilyAppend, "the offsets join one shared history")
		testkit.True(t, b.ConcEntry != nil, "typed at the method's own entry")
	})
}

// TestCASLegGuards holds the cell leg to a whole pair: a VersionedCell
// without its aggregator read interleaves nothing worth checking.
func TestCASLegGuards(t *testing.T) {
	t.Parallel()

	b := &Bindings{
		Reference: Reference{Oracle: OracleContract, ContractStore: "VersionedCell", VersionField: "V"},
		Actions:   []*Action{{Method: "Swap", Shape: shapeCASWriter, Pool: poolKeys}},
	}
	concurrentOf(b, &subject.Projection{}, nil, nil)
	testkit.Equal(t, b.ConcFamily, "", "half a pair drives nothing Porcupine can order")

	b.Actions = append(b.Actions, &Action{Method: "Get", Shape: shapeAggregator})
	concurrentOf(b, &subject.Projection{}, nil, nil)
	testkit.Equal(t, b.ConcFamily, concFamilyCAS, "the whole pair derives the cell leg")

	testkit.Equal(t, (&Bindings{}).CasMismatch(), CtorErr{},
		"no error rows, no identity for the model to match")
}
