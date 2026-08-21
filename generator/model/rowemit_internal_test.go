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

// The call names the parameter only where the function it renders inside
// declares one.
//
// It renders within the harness generator's own function and can name
// nothing that function does not — the harness's flag decides, not this
// tier's fixture.
func TestTheCallNamesOnlyADeclaredParameter(t *testing.T) {
	t.Parallel()

	b := rowBindings(projection.CheckPlan{})

	drawn := rowCallFor(sdk.NewProvenance(Name), rowIface(), b, &suite.Contract{DrawsFixture: true})
	testkit.Equal(t, drawn.Fixture, fixtureIdent, "the surface takes a fixture, so the call passes it")

	bare := rowCallFor(sdk.NewProvenance(Name), rowIface(), b, &suite.Contract{})
	testkit.Equal(t, bare.Fixture, "",
		"and where it does not, passing one names an identifier nothing declared")
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
