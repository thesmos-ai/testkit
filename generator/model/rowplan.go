// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// The claims this tier's own rows state.
//
// Worded here because they are claims about SEQUENCES, and no
// classification states one: a law says what holds, and these say that
// it holds over sequences nobody wrote. A per-law claim comes from
// lawid's table instead, where the law's own wording lives.
const (
	lawsClaim = "every bound law holds over random operation sequences"
	refClaim  = "every operation sequence leaves the subject agreeing with the reference"
)

// unproven is why neither row claims to have been shown able to fail.
//
// Argued rather than Proven, and not a formality: the parity gate refuses
// the Proven stamp without a planted defect beside it, and the defects
// this package plants break one method of a stub at a time. A sequence
// claim needs a subject that is wrong over a history — the saturation
// prover's job, which runs against a surface this tier does not emit yet.
// Claiming proof without the evidence is refused in both directions, so
// the honest record is the argument.
const unproven = "no mechanical rule plants a defect for this claim; the ones that " +
	"would are domain composites, which no rule reaches from shape and stamps alone"

// unspellable is the other refusal: a rule reached the row and this run
// could not write the defect out.
//
// Told apart from [unproven] because they are fixed in different places.
// "Nobody wrote a rule" sends a reader to the rule table; "your signature
// yields no value to plant" sends them to their own declaration.
const unspellable = "a defect for this claim was derived and this run could not spell it: " +
	"the overridden method's types yield no value to plant"

// PlanRows is the checks this tier owns for one interface.
//
// Planned here rather than by the harness generator. That generator
// planned them once, because the harness's capability fields were
// projected from the plans and a clocked law's row was what opened the
// clock — so the rows and the field had to be worked out together. They
// no longer do: this tier contributes the field itself, so it can plan
// the row that needs it.
//
// One row per leg, not per law. A law with a leg of its own reports
// under it; the rest ride the shared sequences under one claim, because
// what a reader wants to know is which oracle failed, and every law on
// the shared leg failed the same way.
func PlanRows(b *Bindings) []projection.CheckPlan {
	var out []projection.CheckPlan

	if bundle := b.bundledLaws(); len(bundle) > 0 {
		out = append(out, proveOrArgue(projection.CheckPlan{
			ID:    projection.IDPlan{Family: vocab.FamilyModel, Qualifier: b.qualifier(), Seg: vocab.SegLaws},
			Class: vocab.ClassLaws,
			Claim: lawsClaim,
			Body:  projection.LawLeg{Laws: bundle},
			Binds: bundle,
		}, b, nil, nil, false))
	}

	// The differential is the strongest oracle this tier has, and it runs
	// with no law registered so nothing competes with it: a disagreement
	// is what ends the run.
	if b.Reference.Derived() || b.Reference.Supplied() {
		dropped, writer, drops := differentialDefect(b)
		out = append(out, proveOrArgue(projection.CheckPlan{
			ID: projection.IDPlan{
				Family: vocab.FamilyModel, Qualifier: b.qualifier(), Seg: vocab.SegDifferential,
			},
			Class: vocab.ClassDifferential,
			Claim: refClaim,
			Body:  projection.DifferentialLeg{},
		}, b, dropped, writer, drops))
	}
	return append(out, ownLegRows(b)...)
}

// proveOrArgue settles one plan's falsifiability from a rule's verdict:
// a plan the rule reached is Proven and carries its defect, one it
// declined stays Argued and says why.
//
// The two fields move together or not at all, which is why they are
// settled here rather than at each call site. The parity gate refuses a
// Proven row with no defect as firmly as an Argued row that plants one,
// so a site setting one and forgetting the other fails in the generated
// package rather than here.
//
// The overriding method rides along on the binding, because a defect is
// planted THROUGH one and the plan alone does not say which.
func proveOrArgue(
	plan projection.CheckPlan, b *Bindings,
	defect projection.Defect, over *subject.Method, proven bool,
) projection.CheckPlan {
	if !proven {
		plan.Falsifiable = vocab.Argued(unproven)
		return plan
	}
	plan.Falsifiable = vocab.Proven()
	plan.Defect = defect
	b.overrides = append(b.overrides, override{ID: plan.ID, Method: over})
	return plan
}

// ownLegRows is one row per law the shared sequences cannot carry.
//
// Its own row rather than a line in the bundle, because what a reader
// wants from a red is which claim broke: every law on the shared leg
// failed the same way, and these did not. Each reports under the law's
// own identity and states the law's own sentence, so the report names the
// claim rather than the machinery.
//
// A law whose claim will not word is refused here and named in the
// header. The alternative is a row stating a sentence this package
// invented for a law somebody else defined, which is how a manifest comes
// to promise something nothing checks.
func ownLegRows(b *Bindings) []projection.CheckPlan {
	token := projection.Token(b.IfaceName)
	var out []projection.CheckPlan
	for _, l := range b.RowedLaws() {
		// The leg's class where the law rides one of its own, and the
		// plain laws class otherwise: a worded law with no special leg
		// still states its own sentence over the shared sequences.
		class, own := tiers.LegOf(l.ID)
		if !own {
			class = vocab.ClassLaws
		}
		claim, err := subject.ClaimOf(l.ID, token, l.Carriers())
		if err != nil {
			b.Unbound = append(b.Unbound, Skip{Method: l.ID, Reason: err.Error()})
			continue
		}
		bind := []projection.Bind{{Law: l.ID}}
		defect, over, proven := defectFor(b, l)
		out = append(out, proveOrArgue(projection.CheckPlan{
			ID: projection.IDPlan{
				Family: vocab.FamilyModel, Qualifier: b.qualifier(), Seg: l.ID,
			},
			Class: class,
			Claim: claim,
			Needs: needsFor(l),
			Body:  projection.LawLeg{Laws: bind},
			Binds: bind,
		}, b, defect, over, proven))
	}
	return out
}

// needsFor is what a row demands of the harness beyond a constructor.
//
// Read off the LAW rather than off the row's class, because the two do
// not line up and the corpus proved it: the windowed law advances past
// the window it declared, so its binding is clocked, but it reports
// under the plain laws class and a class-keyed answer left it asking for
// no clock while its leg demanded one. The law is what moves time.
//
// One capability. Every other leg provokes what it needs through methods
// the interface already declares, so it asks for nothing — a field a
// consumer must fill for a check that never reads it is worse than no
// field at all.
func needsFor(l *LawBinding) []projection.NeedPlan {
	if l.Clocked {
		return []projection.NeedPlan{{Capability: vocab.CapClock}}
	}
	return nil
}

// qualifier is the interface's word inside a family-scoped identity.
//
// Composed through the same policy the harness generator's index spells
// it with, because the two have to agree: the index names the row and
// the row reports under the name, and a slug derived twice diverges the
// moment an interface name has two words.
func (b *Bindings) qualifier() string { return projection.IDQualifier(b.IfaceName) }

// bundledLaws are the laws riding the shared sequences, as the row's
// Binds column spells them.
//
// Read off [Bindings.LegLaws] rather than filtered again here, so the row
// names exactly the laws the leg beside it registers. Two derivations of
// one set is how a manifest comes to promise a law nothing runs.
func (b *Bindings) bundledLaws() []projection.Bind {
	legLaws := b.LegLaws()
	out := make([]projection.Bind, 0, len(legLaws))
	for _, l := range legLaws {
		out = append(out, projection.Bind{Law: l.ID})
	}
	return out
}
