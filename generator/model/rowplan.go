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
	lawsClaim  = "every bound law holds over random operation sequences"
	refClaim   = "every operation sequence leaves the subject agreeing with the reference"
	concClaim  = "concurrent operation histories are linearizable"
	simClaim   = "every acknowledged write is still readable after the process is rebuilt over its medium"
	faultClaim = "a write the medium refused leaves nothing behind for a rebuild to find"
)

// Argued rather than Proven is not a formality: the parity gate refuses
// the Proven stamp without a planted defect beside it, so a row that
// carries no evidence has to state its case. The reasons live in
// defect.go, told apart because they are fixed in different places.
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
	//
	// Not on the twin floor WHERE A LAW LEG CARRIES IT. A twin comparison
	// rides every law leg — the actions compare both sides after each call
	// — so a row of its own would rerun the weakest half of what those legs
	// already do, under a claim about a reference comparison it did not
	// make.
	//
	// Where no law bound there are no legs, and the reasoning inverts: this
	// row is then the only thing driving the actions at all. Declining it
	// leaves the tier empty and the shapes it would have exercised
	// untouched — eight detector fixtures lost their whole model tier that
	// way, and with it the only run of the constructors for a lookup, a
	// multi-reader, a mutator, a bool-answering read. Catching
	// nondeterminism and hidden shared state is a smaller claim than a
	// derived oracle makes, and it is not nothing.
	//
	// The twin arm below is why this is worth saying twice. Until now the
	// row was emitted only for a derived or supplied reference, which left
	// fifty-eight twin fixtures comparing against their own factory on law
	// legs and nowhere else — and eight of those bind no law, so their
	// actions were built and never driven. That was an omission rather
	// than a verdict: nothing in the tree argued it.
	switch decline := differentialDecline(b); {
	case decline != "":
		b.Unbound = append(b.Unbound, Skip{
			Method: b.qualifier() + " differential",
			Reason: decline,
		})
	default:
		dropped, writer, why := differentialDefect(b)
		out = append(out, proveOrArgue(projection.CheckPlan{
			ID: projection.IDPlan{
				Family: vocab.FamilyModel, Qualifier: b.qualifier(), Seg: vocab.SegDifferential,
			},
			Class: vocab.ClassDifferential,
			Claim: refClaim,
			Body:  projection.DifferentialLeg{},
		}, b, dropped, writer, why == "", why))
	}
	if why := concLegReason(b); b.Concurrent() && why != "" {
		b.Unbound = append(b.Unbound, Skip{Method: b.ConcFamily + " linearizability", Reason: why})
	} else if b.Concurrent() {
		// Its own row, and on no law leg. The verdict comes from a search
		// for a serialisation of a recorded history rather than from an
		// invariant after a step, and a row folding it into the bundle
		// would report a linearizability violation as a law that failed.
		out = append(out, proveOrArgue(projection.CheckPlan{
			ID: projection.IDPlan{
				Family: vocab.FamilyModel, Qualifier: b.qualifier(), Seg: vocab.SegLinearizable,
			},
			Class: vocab.ClassConcurrent,
			Claim: concClaim,
			Body:  projection.ConcurrentLeg{},
		}, b, nil, nil, false))
	}
	if !b.Sim() && b.SimUnpaired != "" && b.KeysReachAWrite() {
		// Stated only where the interface writes. One that never writes
		// had no crash claim to make, and a line saying so on every
		// read-only fixture is noise a reader learns to skip past.
		b.Unbound = append(b.Unbound, Skip{Method: "crash recovery", Reason: b.SimUnpaired})
	}
	if why := simLegReason(b); b.Sim() && why != "" {
		b.Unbound = append(b.Unbound, Skip{Method: "crash recovery", Reason: why})
	} else if b.Sim() {
		fresh, freshOK := freshMedium(b)
		// Under the sim family rather than the model one. Every other row
		// here judges the subject against something that models it; this
		// judges it against its own acknowledgements across a seam nothing
		// else in the run crosses, and a report grouping the two would put
		// a lost write beside a disagreeing reference.
		out = append(out, proveOrArgue(projection.CheckPlan{
			ID: projection.IDPlan{
				Family: vocab.FamilySim, Qualifier: b.qualifier(), Seg: vocab.SegRecovery,
			},
			Class: vocab.ClassSimRecovery,
			Claim: simClaim,
			Needs: []projection.NeedPlan{{Capability: vocab.CapRecover}},
			Body:  projection.SimLeg{Kind: projection.SimRecovery},
		}, b, fresh, nil, freshOK))

		// The same schedule with the medium free to fail, and its own row
		// because the claim is the other half. Recovery asks what an
		// acknowledged write survives; this asks what a REFUSED one left
		// behind, and nothing else in the run ever refuses one.
		//
		// The failure is induced in the subject rather than answered in
		// front of it. A double that reports the error without calling
		// the subject leaves it ignorant of the write, so it cannot have
		// written anything down and the claim has nothing to be false
		// about — which is a green row that says nothing.
		if b.Faulted() {
			out = append(out, proveOrArgue(projection.CheckPlan{
				ID: projection.IDPlan{
					Family: vocab.FamilySim, Qualifier: b.qualifier(), Seg: vocab.SegFault,
				},
				Class: vocab.ClassSimFault,
				Claim: faultClaim,
				Needs: []projection.NeedPlan{
					{Capability: vocab.CapRecover},
					{Capability: vocab.CapInduce, Sym: b.FaultSym},
				},
				Body: projection.SimLeg{Kind: projection.SimFault},
			}, b, nil, nil, false))
		}
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
// freshMedium is the crash claim's defect: the reference, rebuilt onto
// nothing.
//
// Correct while it runs and amnesiac across the seam, which is the whole
// of what recovery claims. It needs a working implementation to be right
// about everything else — a bare stub stores nothing, so the schedule
// would red on the first read rather than the first read after a crash —
// and the derived reference is one. A run on the twin floor has none,
// and says so.
// No method comes back, and that is the shape rather than an omission:
// this defect overrides nothing, it answers the crash seam differently.
// The plant threads a nil method through for it.
func freshMedium(b *Bindings) (projection.Defect, bool) {
	if !b.Reference.Derived() {
		return nil, false
	}
	return projection.FreshMedium{
		Clause: projection.Clause{Text: "rebuild finds an empty medium"},
		Ref:    projection.Expr(b.Reference.CtorName),
	}, true
}

// differentialDecline is why this run emits no reference comparison,
// empty where it emits one.
//
// Two reasons, and the order matters. A twin whose actions a law leg
// already drives gets nothing new from a row of its own. A run whose every
// driven method answers an error has nothing for either side to disagree
// about, whatever stands behind the reference.
//
// A twin with no laws gets the row. It is the only thing that would drive
// the actions at all, and comparing a subject against a second instance of
// itself catches nondeterminism and hidden shared state — less than a
// derived oracle, and more than nothing.
func differentialDecline(b *Bindings) string {
	if b.Reference.Twin() && len(b.Laws) > 0 {
		return "the reference is the subject's own factory, whose comparison " +
			"already rides each law leg's actions; alone it catches " +
			"nondeterminism and nothing a second instance shares"
	}
	return blindDifferential(b)
}

// blindDifferential reports why a reference comparison could not fail,
// empty where it could.
//
// The comparison reads what each side ANSWERED. Where every driven action
// answers an error and nothing else, both sides return nil for every call
// a correct subject makes, and the row is a claim about a reference that
// nothing consults. The corpus proved it the moment the publisher family
// gained an oracle: the differential appeared and the double that reports
// success and keeps nothing passed it, because keeping nothing and keeping
// everything both return nil from Publish.
//
// A publisher's deliveries ARE observable, just not through an action
// return — they arrive on a channel the differential cannot compare by
// identity. Declining the row is the honest state until an action drives
// that channel; see action.Delivery, which is the comparison that would.
func blindDifferential(b *Bindings) string {
	if len(b.Actions) == 0 {
		return ""
	}
	for _, a := range b.Actions {
		m := methodNamed(b, a.Method)
		if m == nil || !errOnly(m) {
			return ""
		}
	}
	return "every driven method here answers an error and nothing else, so both " +
		"sides return nil for every call a correct subject makes and the " +
		"comparison has nothing to disagree about"
}

func proveOrArgue(
	plan projection.CheckPlan, b *Bindings,
	defect projection.Defect, over *subject.Method, proven bool, why ...string,
) projection.CheckPlan {
	if !proven {
		reason := NoRule
		if len(why) > 0 && why[0] != "" {
			reason = why[0]
		}
		plan.Falsifiable = vocab.Argued(reason)
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
		defect, over, proven, why := defectFor(b, l)
		out = append(out, proveOrArgue(projection.CheckPlan{
			ID: projection.IDPlan{
				Family: vocab.FamilyModel, Qualifier: b.qualifier(), Seg: l.ID,
			},
			Class: class,
			Claim: claim,
			Needs: needsFor(l),
			Body:  projection.LawLeg{Laws: bind},
			Binds: bind,
		}, b, defect, over, proven, why))
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
	var out []projection.NeedPlan
	if l.Clocked {
		out = append(out, projection.NeedPlan{Capability: vocab.CapClock})
	}
	// Every door the law reads. Declared here so the runner refuses a
	// subject supplying none BEFORE the body runs, which is what lets the
	// body read each one unconditionally. Dropping the law instead is
	// what this replaced: it bound, ran nowhere, and appeared in no
	// header — the one absence nothing in the output would show.
	for _, door := range SuppliedBy([]*LawBinding{l}) {
		out = append(out, projection.NeedPlan{Capability: vocab.Capability(door)})
	}
	return out
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
