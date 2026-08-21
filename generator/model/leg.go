// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/suite"
)

// The emit kinds and template names for the bodies this tier's rows run.
//
// A kind per leg, the way the doors are a kind per capability. The two
// legs share their inputs and almost nothing else: the differential runs
// with no law registered, so a disagreement is what ends the run; the
// laws leg runs the same sequences with that verdict switched off, so the
// laws are the only thing that can. One template carrying both behind a
// conditional would put the reason for the split where nobody reading
// either body can see it.
const (
	KindDifferentialLeg sdk.Kind = "model.leg.differential"
	KindLawsLeg         sdk.Kind = "model.leg.laws"
	KindOwnLeg          sdk.Kind = "model.leg.own"
	KindClockedLeg      sdk.Kind = "model.leg.clocked"
)

// leg is what every body needs and no template can work out for itself.
type leg struct {
	sdk.BaseEmit

	// Assert is the function this leg declares — the one the row's
	// RunWith calls.
	Assert string

	// Subject is the interface as the harness file spells it, for the
	// runtime Subject the body receives.
	Subject string

	// Draws says the leg takes the run's sample inputs; Fixture is the
	// parameter's name and FixtureType its type. False where nothing in
	// the tier reads the fixture, in which case a parameter for it would
	// not compile.
	Draws                bool
	Fixture, FixtureType string

	// Actions names the function yielding the operation vocabulary both
	// legs drive, and Laws the one yielding the bound laws.
	Actions, Laws string

	// Ref is the derived reference's constructor and RefExpr the one the
	// directive named; both are empty for the twin floor, where the
	// subject's own factory stands in and the body spells it.
	Ref     string
	RefExpr *sdk.Expr

	// History is the append log the recording actions fill, empty where no
	// law reads one, and HistoryElem what it holds. The leg declares it,
	// hands it to the actions and resets it each iteration: a law comparing
	// against a history the actions never wrote into is a law over an empty
	// record.
	History     string
	HistoryElem sdk.Ref

	// Factory says a bundled law builds instances of its own — a merge
	// claim compares two replicas, and no observation over one states it.
	// The leg has the subject and hands the factory over.
	Factory bool

	// The packages a body reaches, so a template asks rather than spells.
	Vocab, LegsPkg, ModelPkg, HistoryPkg, LawPath string
}

// Twin reports the floor where no independent reference exists: the leg
// builds its comparison instance from the subject's own factory.
func (l leg) Twin() bool { return l.Ref == "" && l.RefExpr == nil }

// DifferentialLeg is the body of the row claiming the subject and the
// reference agree over every sequence.
type DifferentialLeg struct{ leg }

// Kind returns the template this body renders through.
func (*DifferentialLeg) Kind() sdk.Kind { return KindDifferentialLeg }

// LawsLeg is the body of the row claiming every bundled law holds over
// the same sequences.
type LawsLeg struct{ leg }

// Kind returns the template this body renders through.
func (*LawsLeg) Kind() sdk.Kind { return KindLawsLeg }

// OwnLeg is the body of a row carrying one law the shared sequences
// cannot: it needs a lifecycle probe, a poisoned subject, or writes of
// its own, and the bundle provides none of those.
type OwnLeg struct {
	leg

	// Laws are the bindings this leg registers — every binding of the
	// one law its row claims. Rendered inline rather than through the
	// bundle's function, because a clocked law's Advance reads a local
	// the leg declares and a literal spelled outside that function names
	// nothing.
	Laws []*LawBinding

	// Drain is the publisher sweep a delivery law reads, empty for every
	// other law. Declared by whichever body registers the law, which is
	// this one now that a worded law has a leg of its own.
	Drain string

	// Builds says the law builds subjects of its own — an algebraic
	// claim comparing two folds, a freshness claim wanting an untouched
	// one — and Coalesces that it counts its own invocations under a
	// lock. Both are locals the bundle declared and this leg now owes.
	Builds, Coalesces bool

	// Keys and Values say which shared pools the law draws, and Pools the
	// ones it brings of its own. A local for a pool this law does not
	// read would not compile.
	Keys, Values bool
	Pools        []LawPool

	// KeysFunc and ValuesFunc name the pool constructors the locals call.
	KeysFunc, ValuesFunc string
}

// Kind returns the template this body renders through.
func (*OwnLeg) Kind() sdk.Kind { return KindOwnLeg }

// ClockedLeg is [OwnLeg] for a law that moves time: the subject is built
// on a clock the law advances, which is the one capability this tier asks
// a consumer for.
type ClockedLeg struct {
	OwnLeg

	// Clock is the controllable clock's package, whose test clock the
	// factory builds and the law's Advance moves.
	Clock string
}

// Kind returns the template this body renders through.
func (*ClockedLeg) Kind() sdk.Kind { return KindClockedLeg }

// KindDeclined is the emit kind and template for a refusal: the interface
// asked for this tier and the harness it has cannot carry it.
const KindDeclined sdk.Kind = "model.declined"

// Declined is that refusal, rendered where the rows would have been.
//
// In the generated file rather than in the generator's diagnostics,
// because the question a reader asks is "why does this package have no
// model checks", and they ask it while reading the package. A directive
// that was honoured everywhere else and silently dropped here is the one
// absence nothing else in the output would show.
type Declined struct {
	sdk.BaseEmit

	// Iface is the interface that asked.
	Iface string

	// Why is the reason it cannot be given, one comment line per element.
	//
	// Wrapped at the source rather than here, because the only correct
	// place to break a sentence is where its author would break it, and a
	// column count in a template does not know that.
	Why []string
}

// Kind returns the template this refusal renders through.
func (*Declined) Kind() sdk.Kind { return KindDeclined }

// drawsFixture reports whether this tier's declarations take the run's
// sample inputs.
//
// Both halves, and one answer for the row, the declaration and every leg.
// The tier has to want them — a parameter nothing reads does not compile —
// and the harness has to have them, because the appended expression is
// spelled inside that generator's function and can only name what it
// declares. Asking the two questions separately is how a call and a
// signature come to disagree in a file nobody may edit.
func drawsFixture(b *Bindings, harness *suite.Contract) bool {
	return b.NeedsFixture() && harness.DrawsFixture
}

// historyIdent is the append log's name inside a leg, spelled to match
// what the recording action's own template reads. The action renders
// inside the function this hands it to, so the two agree here or the
// generated file names an undeclared identifier.
const historyIdent = "appendHist"

// legsFor is the body each planned row runs, in plan order.
//
// Driven off the rows rather than off the bindings, so a row without a
// body and a body without a row are both impossible: the runtime refuses
// a check setting neither Run nor RunWith by name, and a function nothing
// calls does not compile in a file the consumer may not edit.
func legsFor(b *Bindings, harness *suite.Contract) []sdk.EmitNode {
	base := leg{
		BaseEmit:    b.BaseEmit,
		Subject:     harness.SubjectType(),
		Draws:       drawsFixture(b, harness),
		Fixture:     fixtureIdent,
		FixtureType: harness.Fixture.TypeName,
		Actions:     b.ActionsFuncName(),
		Laws:        b.LawsFuncName(),
		Factory:     b.LawsNeedFactory(),
		Vocab:       VocabPkg,
		LegsPkg:     LegsPkg,
		ModelPkg:    ModelPkg,
		HistoryPkg:  HistoryPkg,
		LawPath:     LawPkg,
	}
	switch {
	case b.Reference.Supplied():
		base.RefExpr = b.Reference.SuppliedCtor
	case !b.Reference.Twin():
		base.Ref = b.Reference.CtorName
	}
	if b.RecordsHistory {
		base.History, base.HistoryElem = historyIdent, b.HistoryElem
	}

	// The own-leg laws by identity, because a row for one names it in the
	// segment of its own ID — which is how a body finds the law it is the
	// body of without re-running the selection.
	own := map[string]*LawBinding{}
	for _, l := range b.RowedLaws() {
		own[l.ID] = l
	}

	out := make([]sdk.EmitNode, 0, len(b.Rows))
	for _, p := range b.Rows {
		l := base
		l.Assert = rowAssertName(projection.Token(b.IfaceName), p.ID)
		law, single := own[p.ID.Seg]
		switch {
		case single:
			out = append(out, ownLegFor(b, l, law))
		case p.Body.BodyKind() == projection.DifferentialLeg{}.BodyKind():
			out = append(out, &DifferentialLeg{leg: l})
		default:
			out = append(out, &LawsLeg{leg: l})
		}
	}
	return out
}

// ownLegFor is the body of a row carrying one law: the clocked shape
// where the law moves time, the plain one otherwise.
func ownLegFor(b *Bindings, base leg, law *LawBinding) sdk.EmitNode {
	one := b.BindingsOf(law.ID)
	body := OwnLeg{
		leg:        base,
		Laws:       one,
		Keys:       LawsDraw(one, poolKeys),
		Values:     LawsDraw(one, poolValues),
		Pools:      b.PoolsFor(one),
		KeysFunc:   b.KeysFuncName(),
		ValuesFunc: b.ValuesFuncName(),
	}
	if b.Publisher != nil && LawsDrawDrain(one) {
		body.Drain = b.Publisher.DrainName
	}
	body.Builds = LawsNeed(one, factoryFieldKind)
	body.Coalesces = LawsNeed(one, sdk.Kind(LawFieldKindPrefix+"Compute")) ||
		LawsNeed(one, sdk.Kind(LawFieldKindPrefix+"Counter"))
	if law.Clocked {
		return &ClockedLeg{OwnLeg: body, Clock: ClockPkg}
	}
	return &body
}
