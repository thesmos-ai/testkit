// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/sdk"

	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/core/tiers"
)

// LawBinding is one law, instantiated and filled, in the generated registry.
type LawBinding struct {
	sdk.BaseEmit

	// ID is the identifier the law reports under, for the header.
	ID string

	// Ctor is the law struct, qualified; Args are its type arguments after
	// the subject, resolved against the interface. Ptr addresses the literal,
	// for a stateful law whose Check lives on the pointer.
	Ctor *sdk.Expr
	Args []sdk.Ref
	Ptr  bool

	// Fields fill the struct, each through its shape's template. Clocked
	// marks a binding whose Advance reads the run's test clock, armed only
	// where the ModelClocked option supplies a subject on it. Session marks
	// a per-client law, re-registered on the concurrent leg where its
	// multi-client trace lives.
	Fields  []*LawField
	Clocked bool
	Session bool

	// Supplied names the config fields this law reads — the guarded
	// registration arms it only when every one is set, and the header names
	// the options that would.
	Supplied []string

	// Unarmed names the optional roles nothing declared: the law binds in
	// its unrefined form, and the header says which arm went unexercised —
	// a green bound law that never redelivered reads as more than it
	// proved unless the header confesses it.
	Unarmed []string
}

// Kind returns the one template every binding renders through.
func (*LawBinding) Kind() sdk.Kind { return "model.law" }

// Induces reports that some bound law asks the subject to be put into a
// failure state before it can state its claim.
//
// Read off the leg each law reports under rather than off the stamps: a
// law that binds is one that will run, and the harness field exists for
// the checks that run. A classification could select a law this build
// then refused, and a field for a refused law is one a consumer fills
// for a check that never comes.
func (b *Bindings) Induces() bool {
	for _, l := range b.Laws {
		if class, own := tiers.LegOf(l.ID); own && class == vocab.ClassPoison {
			return true
		}
	}
	return false
}

// LawField is one filled field of a law struct.
type LawField struct {
	sdk.BaseEmit

	// KindName selects the field's template by its closure shape.
	KindName sdk.Kind

	// Name is the struct field, for the composite literal.
	Name string

	// Method is the role method a closure field calls, with TakesCtx saying
	// whether the call forwards the run's context.
	Method   string
	TakesCtx bool

	// Iface, Key and Value spell the closure's parameter and result types at
	// the pools' own types.
	Iface, Key, Value sdk.Ref

	// In and Out spell a closure's own types where they are not the pools':
	// the domain a roundtrip draws, the offset an append answers, the state
	// an observation returns.
	In, Out sdk.Ref

	// Pool names the shared local a generator field reuses, and KeyOfName the
	// shared key projection a handle field reuses — the same values the
	// actions and the derived reference already draw from, which is the
	// one-derivation rule inside the file.
	Pool, KeyOfName string

	// Const is a constant field's qualified value — a sentinel the
	// declaration stamped, rendered where a manifest names its stamp key.
	// Lit is the literal form, for a numeric stamp like a bound.
	Const *sdk.Expr
	Lit   string

	// KeyField names the fixture key a fixture-anchored closure reads or
	// writes — the fixed key an idempotent composite write repeats, the key
	// a keyed observation revisits.
	KeyField string

	// Pairs are the permitted transitions a workflow stamp declares, parsed
	// from its `from>to` list.
	Pairs [][2]string

	// Partner names the sibling method a coordinating closure also calls —
	// the compensation a saga run unwinds through — with PartnerCtx saying
	// whether that call forwards the run's context.
	Partner    string
	PartnerCtx bool
}

// Kind returns the field's template key.
func (f *LawField) Kind() sdk.Kind { return f.KindName }

// ModelPkg surfaces the runner's import path to the field templates, whose
// closures take the runner's *T.
func (*LawField) ModelPkg() string { return ModelPkg }

// LawPool is one pool a law draws that the sequences do not: a wide input
// domain for a stateless claim, the adversarial strings a safety claim needs.
type LawPool struct {
	// Name is the local the property declares; Q is the element's stamp
	// spelling, so two laws asking for one name at two types are caught.
	Name, Q string

	// Elem is the drawn element type; Adversarial selects the hostile-string
	// generator, Offsets the bounded-duration one, instead of the
	// reflective default.
	Elem        sdk.Ref
	Adversarial bool
	Offsets     bool
}
