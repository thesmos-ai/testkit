// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/lifecycleafterclose"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
	"go.thesmos.sh/testkit/generator/suite"
)

// The planted defects this tier's rows are proven by.
//
// A check that always passes is indistinguishable from a working one
// until something breaks in production, and a run reporting "12 legs, 12
// passed" does not tell the two apart. Each rule below is the smallest
// implementation that breaks exactly one claim — the generated double
// with a single method overridden — and the row it belongs to ships
// Proven only because the rule reached it.
//
// Mechanical, from shape and stamps alone. The residue ships Argued and
// says so: a domain composite no rule reaches is an honest gap, and a
// proof nobody derived worn as a stamp is not.

// The stamps a rule reads, from the plugins that own them.
const (
	mixinAfterClose         = lifecycleafterclose.Name
	mixinAfterCloseSentinel = lifecycleafterclose.ParamSentinel

	// mixinPoisonable is the stamp that names a latch and the probe that
	// reads it — the other road to the poison claims.
	mixinPoisonable = "poisonable"
)

// defectFor is the planted defect for one law's row, and the method the
// defect overrides — nil where no rule reaches the law, or where the
// rule's target is not on this interface.
//
// The method travels with the defect because a defect is planted THROUGH
// one: the double's option is per-method, and a rule that picked its
// target then handed back only the plan would leave the caller to guess
// which method it meant.
func defectFor(b *Bindings, l *LawBinding) (projection.Defect, *subject.Method, bool, string) {
	rule, ruled := lawDefects()[l.ID]
	if !ruled {
		return nil, nil, false, NoRule
	}
	defect, over, planted := rule(b, l)
	if !planted {
		return nil, nil, false, RuleDeclined
	}
	return defect, over, true, ""
}

// The reasons a row goes Argued, told apart because they are fixed in
// different places.
//
// One sentence used to serve every case and was false for several of
// them: it said no rule exists, on rows where a rule exists and this
// declaration could not supply it. A reader sent to the rule table for a
// gap that is in their own stamp loses the time twice.
const (
	// NoRule is the honest residue: nothing in the table reaches this
	// claim from shape and stamps alone.
	NoRule = "no mechanical rule plants a defect for this claim; the ones that " +
		"would are domain composites, which no rule reaches from shape and stamps alone"

	// RuleDeclined is the other half: a rule exists and this declaration
	// did not give it what it needs — a method it names, a stamp it
	// reads, or a reference to be correct against.
	RuleDeclined = "a rule for this claim exists and this declaration does not supply " +
		"what it needs: the method it plants through, the stamp it reads, or a " +
		"derived reference for the defect to be right about everything else"
)

// lawDefectRule plants one law's defect from the interface's own stamps.
type lawDefectRule func(b *Bindings, l *LawBinding) (projection.Defect, *subject.Method, bool)

// lawDefects is the rule table, keyed by law. A law without a row is the
// honest residue — the domain composites, like the cursor's hand-built
// handle types and the lease's lying accounting, that no mechanical rule
// reaches.
func lawDefects() map[string]lawDefectRule {
	return map[string]lawDefectRule{
		lawid.LifecycleAfterClose:      afterCloseOutlives,
		lawid.PoisonConsistent:         poisonHeals,
		lawid.PoisonNilOnFresh:         poisonBornFailing,
		lawid.PoisonIdempotentRead:     poisonAnswersTwoWays,
		lawid.AppenderMonotonicOffsets: appenderFreezes,
		// A frozen answer is a subject ignoring the clock, which is
		// exactly what this claim forbids: the law advances and demands
		// the reading move, and one pinned to a constant cannot.
		lawid.TimeawareMoves: appenderFreezes,

		// The carrier-does-nothing family. Each of these laws reads what
		// one method answers — a value read back, a delivery, a refusal
		// the context asked for — so a carrier that acts on nothing and
		// reports success breaks it directly. One statement, several
		// claims, which is what [projection.AnswersAnyway] is named for.
		//
		// The comparison laws are deliberately absent. A carrier that
		// answers nothing looks, to a law comparing subject against
		// reference, exactly like a subject that has nothing to report —
		// and the corpus measured it slipping through on 22 of 30
		// fixtures. Whatever breaks those is not this statement, and a
		// rule that does not redden is not evidence.
		lawid.WriteObservable:          carrierDoesNothing,
		lawid.LifecycleRespectsContext: carrierDoesNothing,
		lawid.PublisherDelivers:        carrierDoesNothing,
		lawid.PublisherAtLeastOnce:     carrierDoesNothing,
		lawid.PersisterRetrievable:     carrierDoesNothing,
		lawid.StreamReflectsMutations:  carrierDoesNothing,
		lawid.StreamOverMatch:          carrierDoesNothing,
		lawid.StreamPermutation:        carrierDoesNothing,
		lawid.WatcherReturnsOnChange:   carrierDoesNothing,
		lawid.Roundtrip:                carrierDoesNothing,

		// The repeat family: the first call lands and the second
		// refuses, which is what an idempotence claim forbids and what
		// no total defect can state.
		lawid.IdempotentWrite:     carrierRefusesItsRepeat,
		lawid.UpserterIdempotent:  carrierRefusesItsRepeat,
		lawid.IdempotentLifecycle: carrierRefusesItsRepeat,
	}
}

// carrierDoesNothing plants the method that stamped the law acting on
// nothing and reporting success.
//
// The carrier rather than a writer picked by shape: the law reads what
// ITS method answers, and a defect on some other method is one the law
// has no reason to notice. The corpus settled that for the appender
// already — its law appends through a closure of its own, over a method
// the sequences never drive.
func carrierDoesNothing(b *Bindings, l *LawBinding) (projection.Defect, *subject.Method, bool) {
	m := methodNamed(b, l.carrier.Name)
	if m == nil {
		return nil, nil, false
	}
	return projection.AnswersAnyway{
		Clause: projection.Clause{Text: suite.DropsWriteClause(*m)},
		Option: projection.OptionName(b.IfaceName, m.Name),
	}, m, true
}

// carrierRefusesItsRepeat plants the first call landing and the second
// refusing — the one shape an idempotence claim can see.
//
// A total defect cannot state it: a method that always fails breaks the
// first call too, which is a different claim, and one that always
// succeeds is what the law expects.
func carrierRefusesItsRepeat(b *Bindings, l *LawBinding) (projection.Defect, *subject.Method, bool) {
	m := methodNamed(b, l.carrier.Name)
	if m == nil {
		return nil, nil, false
	}
	return projection.SecondCallErrs{
		Clause: projection.Clause{Text: m.Name + " refuses its repeat"},
		Option: projection.OptionName(b.IfaceName, m.Name),
	}, m, true
}

// differentialDefect is the rule that proves the reference comparison: a
// write acknowledged and dropped.
//
// It reddens because the reference kept what the subject threw away, and
// the differential compares the two after every call. That holds for any
// interface with a writer, whatever it writes, which is what makes this
// mechanical.
//
// It does NOT prove the bundled laws row, and the corpus is where that
// was settled: with the differential verdict off, whether anything
// notices a dropped write depends on which laws bound and which methods
// their closures drive, and 23 of 75 corpus interfaces bind a set that
// tolerates it. Proving a bundled leg means aiming a defect at one law
// and silencing its siblings, which is the saturation prover's job.
//
// The writer is the one the sequences already drive, so the defect
// breaks a method the run actually calls. False where nothing writes:
// there is no dropped write to plant.
func differentialDefect(b *Bindings) (projection.Defect, *subject.Method, bool) {
	m := writerCarrier(b)
	if m == nil {
		return nil, nil, false
	}
	return projection.AnswersAnyway{
		// The harness generator's own wording for this double, because it
		// plants the same one: two sentences for one planted statement
		// read as two different defects in a report.
		Clause: projection.Clause{Text: suite.DropsWriteClause(*m)},
		Option: projection.OptionName(b.IfaceName, m.Name),
	}, m, true
}

// afterCloseOutlives plants the after-close defect: one stamped method
// keeps working past Close.
//
// Exactly one. A double ignoring Close entirely reddens a law probing
// any method and proves nothing about whether the probe set is wide
// enough, which is what the claim is really about.
func afterCloseOutlives(b *Bindings, _ *LawBinding) (projection.Defect, *subject.Method, bool) {
	m := carrierOf(b, func(m *subject.Method) bool { return m.HasMixin(mixinAfterClose) })
	if m == nil {
		return nil, nil, false
	}
	return projection.PartialOutlive{Option: projection.OptionName(b.IfaceName, m.Name)}, m, true
}

// poisonHeals plants the un-sticky poison the law forbids: the subject
// reports the stamped sentinel once and then heals.
//
// The sentinel is the same declaration that licensed the law, so the
// defect cannot break the claim by naming a different one.
func poisonHeals(b *Bindings, _ *LawBinding) (projection.Defect, *subject.Method, bool) {
	for i := range b.Methods {
		m := &b.Methods[i]
		v, stamped := m.MixinParam(mixinAfterClose, mixinAfterCloseSentinel)
		if !stamped || v == "" {
			continue
		}
		return projection.SentinelOnce{
			Clause:   projection.Clause{Text: m.Name + " reports it once and heals"},
			Sentinel: projection.Expr(v),
		}, m, true
	}
	// The other road to the same law. `poisonable induce=` stamps the
	// PROBE, and the law asks only that its answer stay non-nil once the
	// induction has taken — so the defect reports something once and
	// heals, and what it reports does not matter.
	if m := poisonProbe(b); m != nil {
		return projection.SentinelOnce{
			Clause: projection.Clause{Text: m.Name + " reports the state once and heals"},
		}, m, true
	}
	return nil, nil, false
}

// poisonProbe is the method the poison mixin is stamped on: the one that
// reads the state, which is what every poison law observes through.
func poisonProbe(b *Bindings) *subject.Method {
	return carrierOf(b, func(m *subject.Method) bool { return m.HasMixin(mixinPoisonable) })
}

// poisonBornFailing plants a probe that reports poison from the start.
//
// The claim is that a freshly built subject is clean, and the law builds
// one and reads it. A probe that always answers an error breaks exactly
// that and nothing else: no induction has run, so there is no stickiness
// to confuse it with.
func poisonBornFailing(b *Bindings, _ *LawBinding) (projection.Defect, *subject.Method, bool) {
	m := poisonProbe(b)
	if m == nil {
		return nil, nil, false
	}
	return projection.RefusesAlways{
		Clause: projection.Clause{Text: m.Name + " reports poison on a subject nothing touched"},
		Option: projection.OptionName(b.IfaceName, m.Name),
	}, m, true
}

// poisonAnswersTwoWays plants a probe whose two consecutive reads
// disagree.
//
// Read purity rather than stickiness: the law calls the probe twice and
// compares. A probe that answers clean once and poisoned after breaks
// the pair without ever being induced, which is what keeps this defect
// off the other two claims.
func poisonAnswersTwoWays(b *Bindings, _ *LawBinding) (projection.Defect, *subject.Method, bool) {
	m := poisonProbe(b)
	if m == nil {
		return nil, nil, false
	}
	return projection.SecondCallErrs{
		Clause: projection.Clause{Text: m.Name + " answers differently the second time"},
		Option: projection.OptionName(b.IfaceName, m.Name),
	}, m, true
}

// appenderFreezes plants the frozen position: every append lands and
// every one answers the same offset.
func appenderFreezes(b *Bindings, l *LawBinding) (projection.Defect, *subject.Method, bool) {
	// The law's own carrier, not a driven writer: this law appends
	// through a closure of its own, and the corpus's appender fixture
	// drives nothing but a reader. A defect over a method the law never
	// calls is one it cannot notice.
	m := methodNamed(b, l.carrier.Name)
	if m == nil {
		return nil, nil, false
	}
	return projection.FreezeReturn{Option: projection.OptionName(b.IfaceName, m.Name)}, m, true
}

// writerCarrier is the driven method a write-shaped defect overrides.
//
// Read off the actions rather than the methods, because a defect has to
// break something the sequences call: a writer this tier declined to
// drive is one no row would notice being wrong.
func writerCarrier(b *Bindings) *subject.Method {
	for _, a := range b.Actions {
		switch a.Shape {
		case shapeWriter, shapeAnsweringWriter, shapeCompositeWriter:
			return methodNamed(b, a.Method)
		}
	}
	return nil
}

// carrierOf is the first method the predicate accepts, nil for none.
func carrierOf(b *Bindings, match func(*subject.Method) bool) *subject.Method {
	for i := range b.Methods {
		if m := &b.Methods[i]; match(m) {
			return m
		}
	}
	return nil
}

// methodNamed resolves a driven action back to the declaration it came
// from, which is what carries the signature a defect is overridden at.
func methodNamed(b *Bindings, name string) *subject.Method {
	return carrierOf(b, func(m *subject.Method) bool { return m.Name == name })
}
