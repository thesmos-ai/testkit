// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"strings"

	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/suite"
)

// The emit kinds and template names for the property surface a consumer
// writes their own drawn-input checks through.
//
// Three contributions in three regions, which is what the surface is:
// the alias a consumer names in their own signature, the fields they set
// on a row, and the dispatch that turns one into a body the runner
// calls. Any one without the others is worse than none — a field nothing
// dispatches is a body a consumer writes and nothing runs, and it
// reports green.
const (
	KindPropAlias    sdk.Kind = "model.prop.alias"
	KindPropFields   sdk.Kind = "model.prop.fields"
	KindPropDispatch sdk.Kind = "model.prop.dispatch"
)

// PropSugar is one method's drawn-input field: `PropPut`, taking what
// Put takes, drawn from the pool the sequential legs draw it from.
//
// One per action rather than one per method. An action is a method this
// tier already knows how to draw an argument for, and a sugar for a
// method it cannot draw for would be a field whose only honest body
// ignores its own parameter.
type PropSugar struct {
	// Field is the row field — `PropPut`. Method is the source
	// identifier it fixes the check's scope to, and MethodConst the
	// generated constant holding that name.
	Field, Method, MethodConst string

	// Param is the drawn parameter's name and ParamType its type, as the
	// consumer's own signature will spell them.
	Param     string
	ParamType sdk.Ref

	// Pool is the function yielding the generator the argument is drawn
	// from, and Label the draw's name in a shrunk counterexample.
	Pool, Label string

	// TakesCtx reports whether the method takes a context. The body a
	// consumer writes never sees one — it has the run's through the
	// property state — but the closure calling their method does.
	TakesCtx bool
}

// prop is what all three contributions share.
type prop struct {
	sdk.BaseEmit

	// Subject is the interface as the harness file spells it, and Fixture
	// the type the un-sugared body receives.
	Subject, Fixture string

	// FixtureIdent is the local the bind method holds the run's sample
	// inputs in, so a draw expression names what is in scope.
	FixtureIdent string

	// Alias is the property state's local name — the one thing a consumer
	// writes that would otherwise oblige them to import the engine.
	Alias string

	// MethodsVar is the generated name set a row's Method is checked
	// against, so the un-sugared body refuses a typo the way Run does.
	MethodsVar string

	// Sugars are the per-method fields, empty where no action draws an
	// argument this tier can name.
	Sugars []PropSugar

	// The packages the surface reaches, so a template asks rather than
	// spells.
	Vocab, ModelPkg string
}

// PropAlias is the property state under a local name.
type PropAlias struct{ prop }

// Kind returns the template this contribution renders through.
func (*PropAlias) Kind() sdk.Kind { return KindPropAlias }

// PropFields is the row fields a consumer sets.
type PropFields struct{ prop }

// Kind returns the template this contribution renders through.
func (*PropFields) Kind() sdk.Kind { return KindPropFields }

// PropDispatch is what turns one of those fields into a body the runner
// calls.
type PropDispatch struct{ prop }

// Kind returns the template this contribution renders through.
func (*PropDispatch) Kind() sdk.Kind { return KindPropDispatch }

// propFor is the property surface for one interface, or nil where this
// tier has nothing to offer one.
//
// Gated on the same fact everything else here is: the directive. The
// alias is `= model.T`, so contributing it puts the engine and rapid
// behind it into the consumer's build — which is exactly what the
// directive gates, and why this surface cannot be the harness
// generator's however much it looks like part of the row type.
func propFor(b *Bindings, harness *suite.Contract) (
	alias *PropAlias, fields *PropFields, dispatch *PropDispatch,
) {
	token := projection.Token(b.IfaceName)
	p := prop{
		BaseEmit:     b.BaseEmit,
		Subject:      harness.SubjectType(),
		Fixture:      harness.Fixture.TypeName + harness.TypeArgs,
		FixtureIdent: fixtureIdent,
		Alias:        propAliasName,
		MethodsVar:   projection.MethodsVar(token),
		Vocab:        VocabPkg,
		ModelPkg:     ModelPkg,
		Sugars:       sugarsFor(b),
	}
	return &PropAlias{prop: p}, &PropFields{prop: p}, &PropDispatch{prop: p}
}

// propAliasName is the property state's local name.
//
// Short and not the engine's own, because a consumer writes it in every
// property body's signature and `*model.T` would put the engine in their
// import block — the one thing the directive gating exists to keep out of
// a package that did not ask for it.
const propAliasName = "PropT"

// PropSugarsOf is [sugarsFor] for a census that has to count what a
// consumer is offered.
//
// Exported for the conformance gate and nothing else: whether a fixture
// offers a sugared field is a fact about this derivation, and a census
// re-deriving it from the generated text would be measuring its own
// reading rather than the tier's answer.
func PropSugarsOf(b *Bindings) []PropSugar { return sugarsFor(b) }

// sugarsFor is one field per action drawing from a shared pool this run
// actually declares.
//
// Two conditions and each has bitten. The pool has to be one of the
// shared pair: a multi-argument writer draws a value per position from
// its own fixture accessors, and a field for it would take a parameter
// list a consumer has to read the generated file to learn — the
// un-sugared Prop covers that method with the fixture in hand, which is
// the same power spelled once. And the run has to DECLARE that pool: an
// action naming one is not the same fact as a file declaring it, and a
// draw expression calling a function nobody emitted is a compile error
// over generated code the consumer may not edit.
func sugarsFor(b *Bindings) []PropSugar {
	token := projection.Token(b.IfaceName)
	var out []PropSugar
	for _, a := range b.Actions {
		s := PropSugar{
			Field:       "Prop" + a.Method,
			Method:      a.Method,
			MethodConst: projection.MethodConst(token, a.Method),
			TakesCtx:    a.TakesCtx,
		}
		switch {
		case a.Pool == poolKeys && b.NeedsKeysPool():
			s.Param, s.ParamType, s.Pool, s.Label = "key", a.Key, b.KeysFuncName(), "key"
		case a.Pool == poolValues && b.NeedsValuesPool():
			s.Param, s.ParamType, s.Pool, s.Label = "value", a.Value, b.ValuesFuncName(), "value"
		default:
			continue
		}
		if s.ParamType == nil {
			continue
		}
		out = append(out, s)
	}
	return out
}

// Fields is the listing OneBody offers a row that set the wrong number
// of bodies, as this tier's contribution extends it.
//
// Composed here rather than in the template because it has to match the
// fields rendered beside it exactly: a message offering PropPut to a row
// with no PropPut field sends a reader looking for something that is not
// there, and the two would drift the moment a sugar is added.
func (p prop) Fields() string {
	names := make([]string, 0, len(p.Sugars)+1)
	names = append(names, "Prop")
	for _, s := range p.Sugars {
		names = append(names, s.Field)
	}
	return ", " + strings.Join(names, ", ")
}
