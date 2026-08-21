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

// A tier with nothing to say contributes nothing, rather than an
// expression yielding an empty slice.
//
// An empty call is a line in the generated file saying this tier ran and
// found nothing — which reads as coverage. An absence belongs in the
// header, where it comes with a reason.
func TestNoRowsIsNoContribution(t *testing.T) {
	t.Parallel()

	call := rowCallFor(sdk.NewProvenance(Name), rowIface(), &Bindings{}, &suite.Contract{})
	testkit.True(t, call == nil, "no planned row is no contributed expression")
}

// The call names the parameter only where both halves agree: this tier
// reads the sample inputs, and the function it renders inside declares
// them.
//
// Two questions rather than one, because they fail in opposite
// directions. The call renders within the harness generator's own
// function and can name nothing that function does not declare; the
// declarations it reaches take a parameter only where something reads
// it, and one nothing reads does not compile.
func TestTheCallNamesOnlyADeclaredParameter(t *testing.T) {
	t.Parallel()

	reads := rowBindings(projection.CheckPlan{})
	reads.LawsUseFixture = true

	both := rowCallFor(sdk.NewProvenance(Name), rowIface(), reads, &suite.Contract{DrawsFixture: true})
	testkit.Equal(t, both.Fixture, fixtureIdent,
		"the tier reads the inputs and the surface has them, so the call passes them")

	undeclared := rowCallFor(sdk.NewProvenance(Name), rowIface(), reads, &suite.Contract{})
	testkit.Equal(t, undeclared.Fixture, "",
		"a surface with no inputs has nothing to pass, whatever this tier wants")

	unread := rowCallFor(sdk.NewProvenance(Name), rowIface(),
		rowBindings(projection.CheckPlan{}), &suite.Contract{DrawsFixture: true})
	testkit.Equal(t, unread.Fixture, "",
		"and a tier that reads none takes none, or the parameter does not compile")
}

// Every plan reaches the index, because the index names what runs.
func TestTheContributionCarriesItsPlans(t *testing.T) {
	t.Parallel()

	plans := []projection.CheckPlan{{}, {}}
	b := rowBindings(plans...)

	call := rowCallFor(sdk.NewProvenance(Name), rowIface(), b, &suite.Contract{})
	testkit.Len(t, call.CheckPlans(), len(plans),
		"a plan dropped here leaves a check the index cannot name")
}

// rowBindings is the smallest value the row emission reads: a name to
// compose the function from, and the rows it yields.
func rowBindings(rows ...projection.CheckPlan) *Bindings {
	b := &Bindings{Rows: rows}
	b.IfaceName = "Mixed"
	return b
}

// rowIface is the declaration the contribution is attributed to.
func rowIface() *sdk.Interface { return &sdk.Interface{Name: "Mixed"} }

// The declaration the call names is the one this tier emits, taking the
// same parameter and returning the rows the call yields.
func TestTheDeclarationMatchesTheCall(t *testing.T) {
	t.Parallel()

	b := rowBindings(projection.CheckPlan{})
	b.LawsUseFixture = true
	h := &suite.Contract{DrawsFixture: true}
	h.Fixture.TypeName = "MixedFixture"

	call := rowCallFor(sdk.NewProvenance(Name), rowIface(), b, h)
	decl := rowDeclFor(sdk.NewProvenance(Name), rowIface(), b, h)

	testkit.Equal(t, decl.Func, call.Func,
		"an expression naming a function nobody declared is a compile error "+
			"over generated code a consumer cannot edit")
	testkit.Equal(t, decl.FixtureParam, call.Fixture, "and the argument it is passed")
	testkit.Len(t, decl.Rows, len(call.CheckPlans()), "and the rows it returns")
}
