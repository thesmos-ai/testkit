// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/lifecycleafterclose"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/ttl"

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

	mixinTTL     = ttl.Name
	mixinTTLRead = ttl.ParamRead
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
	defect, over, why := rule(b, l)
	if why != "" {
		return nil, nil, false, why
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
	// NoRule is the honest residue: nothing in the table reaches this claim.
	//
	// Two things wear this reason and it does not pretend to tell them
	// apart. Some claims are domain composites no rule could reach from
	// shape and stamps alone — the cursor's hand-built handles, the lease's
	// lying accounting. Others are reachable and simply have no rule yet.
	// Fifty-nine distinct laws carry it, and sorting them would be a
	// classification this generator has not earned; saying which is which
	// wrongly is worse than saying neither.
	NoRule = "no rule in this generator plants a defect for this claim — either " +
		"nothing reaches it from shape and stamps alone, or nobody has written " +
		"the rule; the defect is yours to write and this row claims no proof"

	// RuleDeclined is the other half: a rule exists and this declaration
	// did not give it what it needs — a method it names, a stamp it
	// reads, or a reference to be correct against.
	RuleDeclined = "a rule for this claim exists and this declaration does not supply " +
		"what it needs: the method it plants through, the stamp it reads, or a " +
		"derived reference for the defect to be right about everything else"
)

// lawDefectRule plants one law's defect from the interface's own stamps,
// or says why it cannot.
//
// The reason is the rule's own and reaches the row's Argued line verbatim.
// [RuleDeclined] is the catch-all for a rule with nothing more specific to
// say; a rule that knows exactly what was missing says that instead,
// because a reader sent to the general answer for a particular gap loses
// the time twice.
type lawDefectRule func(b *Bindings, l *LawBinding) (projection.Defect, *subject.Method, string)

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
		lawid.TTLExpiry:                ttlKeepsAnswering,
		lawid.CountEqualsReference:     countMiscounts,
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
func carrierDoesNothing(b *Bindings, l *LawBinding) (projection.Defect, *subject.Method, string) {
	m := methodNamed(b, l.carrier.Name)
	if m == nil {
		return nil, nil, RuleDeclined
	}
	return projection.AnswersAnyway{
		Clause: projection.Clause{Text: suite.DropsWriteClause(*m)},
		Option: projection.OptionName(b.IfaceName, m.Name),
	}, m, ""
}

// carrierRefusesItsRepeat plants the first call landing and the second
// refusing — the one shape an idempotence claim can see.
//
// A total defect cannot state it: a method that always fails breaks the
// first call too, which is a different claim, and one that always
// succeeds is what the law expects.
func carrierRefusesItsRepeat(b *Bindings, l *LawBinding) (projection.Defect, *subject.Method, string) {
	m := methodNamed(b, l.carrier.Name)
	if m == nil {
		return nil, nil, RuleDeclined
	}
	return projection.SecondCallErrs{
		Clause: projection.Clause{Text: m.Name + " refuses its repeat"},
		Option: projection.OptionName(b.IfaceName, m.Name),
	}, m, ""
}

// differentialDefect is the rule that proves the reference comparison: a
// write acknowledged and dropped.
//
// It reddens because the reference kept what the subject threw away, and
// the differential compares the two after every call. That holds for any
// interface with a writer, whatever it writes, which is what makes this
// mechanical.
//
// Except where the read is evicting, which inverts it — see the rule's
// first arm, and [action.EvictingReader] for why a miss cannot be a
// divergence there.
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
func differentialDefect(b *Bindings) (projection.Defect, *subject.Method, string) {
	if b.Reference.Twin() {
		// No planted defect can redden a twin comparison, and the reason is
		// structural rather than a gap in this table: the reference is the
		// subject's own factory, so a proof run builds BOTH sides from the
		// defect. Whatever it does, it does on both, and they agree.
		//
		// The row is still worth emitting — it is what drives the actions on
		// a fixture that binds no law, and two instances that disagree have
		// found nondeterminism or shared state. It just cannot claim a
		// proof, and says so.
		return nil, nil, "the reference is the subject's own factory, so a proof " +
			"run builds both sides from the defect and they agree however broken " +
			"it is; what this row can catch is nondeterminism, which no planted " +
			"defect exhibits"
	}
	if b.Delivery != nil && b.Delivery.PermitsLoss() {
		// This rule's whole statement is a write acknowledged and dropped,
		// and at-most-once is the one guarantee that PERMITS it: the
		// comparison lets a subject deliver less than the reference on
		// purpose, so the double slips through and the row would claim a
		// proof that cannot fail. The corpus caught it the run after the
		// delivery actions landed.
		//
		// What would redden this mode is a publisher that delivers a
		// message twice, and no stub option states that: a double answers
		// its own return and cannot call the subject a second time.
		return nil, nil, "the defect this rule plants is a write acknowledged " +
			"and dropped, which at-most-once permits; the duplicate that would " +
			"break it is not a statement a generated double can make"
	}
	if m := methodNamed(b, b.EvictingRead); b.EvictingRead != "" && m != nil {
		// Where the read is compared one way, a dropped write is invisible
		// and this rule would ship a proof that proves nothing: a subject
		// keeping nothing answers a miss for every key, and a subject miss
		// is what eviction looks like. The corpus caught it on the first
		// run — the row claimed Proven and the dropping double sailed
		// through.
		//
		// The direction the comparison CAN see is the other one: a hit the
		// reference cannot explain. A read answering for everything invents
		// one on the first key nothing wrote, which is the strongest claim
		// an asymmetric comparison makes and the only one worth proving
		// here.
		return projection.AnswersWithValue{
			Clause: projection.Clause{Text: m.Name + " answers for a key nothing wrote"},
			Option: projection.OptionName(b.IfaceName, m.Name),
		}, m, ""
	}
	m := writerCarrier(b)
	if m == nil {
		return nil, nil, RuleDeclined
	}
	return projection.AnswersAnyway{
		// The harness generator's own wording for this double, because it
		// plants the same one: two sentences for one planted statement
		// read as two different defects in a report.
		Clause: projection.Clause{Text: suite.DropsWriteClause(*m)},
		Option: projection.OptionName(b.IfaceName, m.Name),
	}, m, ""
}

// afterCloseOutlives plants the after-close defect: one stamped method
// keeps working past Close.
//
// Exactly one. A double ignoring Close entirely reddens a law probing
// any method and proves nothing about whether the probe set is wide
// enough, which is what the claim is really about.
func afterCloseOutlives(b *Bindings, _ *LawBinding) (projection.Defect, *subject.Method, string) {
	m := carrierOf(b, func(m *subject.Method) bool { return m.HasMixin(mixinAfterClose) })
	if m == nil {
		return nil, nil, RuleDeclined
	}
	return projection.PartialOutlive{Option: projection.OptionName(b.IfaceName, m.Name)}, m, ""
}

// poisonHeals plants the un-sticky poison the law forbids: the subject
// reports the stamped sentinel once and then heals.
//
// The sentinel is the same declaration that licensed the law, so the
// defect cannot break the claim by naming a different one.
func poisonHeals(b *Bindings, _ *LawBinding) (projection.Defect, *subject.Method, string) {
	for i := range b.Methods {
		m := &b.Methods[i]
		v, stamped := m.MixinParam(mixinAfterClose, mixinAfterCloseSentinel)
		if !stamped || v == "" {
			continue
		}
		return projection.SentinelOnce{
			Clause:   projection.Clause{Text: m.Name + " reports it once and heals"},
			Sentinel: projection.Expr(v),
		}, m, ""
	}
	// The other road to the same law. `poisonable induce=` stamps the
	// PROBE, and the law asks only that its answer stay non-nil once the
	// induction has taken — so the defect reports something once and
	// heals, and what it reports does not matter.
	if m := poisonProbe(b); m != nil {
		return projection.SentinelOnce{
			Clause: projection.Clause{Text: m.Name + " reports the state once and heals"},
		}, m, ""
	}
	return nil, nil, RuleDeclined
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
func poisonBornFailing(b *Bindings, _ *LawBinding) (projection.Defect, *subject.Method, string) {
	m := poisonProbe(b)
	if m == nil {
		return nil, nil, RuleDeclined
	}
	return projection.RefusesAlways{
		Clause: projection.Clause{Text: m.Name + " reports poison on a subject nothing touched"},
		Option: projection.OptionName(b.IfaceName, m.Name),
	}, m, ""
}

// poisonAnswersTwoWays plants a probe whose two consecutive reads
// disagree.
//
// Read purity rather than stickiness: the law calls the probe twice and
// compares. A probe that answers clean once and poisoned after breaks
// the pair without ever being induced, which is what keeps this defect
// off the other two claims.
func poisonAnswersTwoWays(b *Bindings, _ *LawBinding) (projection.Defect, *subject.Method, string) {
	m := poisonProbe(b)
	if m == nil {
		return nil, nil, RuleDeclined
	}
	return projection.SecondCallErrs{
		Clause: projection.Clause{Text: m.Name + " answers differently the second time"},
		Option: projection.OptionName(b.IfaceName, m.Name),
	}, m, ""
}

// ttlKeepsAnswering plants a store whose entries never lapse: the read
// answers whatever it is asked, including a key whose lifetime the clock
// has already run past.
//
// The read half rather than the write, and the choice is what makes the
// defect isolate the claim. The law puts, reads, advances and reads
// again, and only the last of those four states the claim — so a defect
// on the read breaks that one and leaves the first three alone. A defect
// on the write would have to store something the later read still finds,
// which a double with no medium behind it cannot do: it would break the
// pre-advance read instead, and the red would name a claim about
// storing rather than one about expiring.
//
// The method the stamp NAMES rather than the one carrying it. Both
// corpus fixtures declare the mixin on their own reader, which is the
// common shape and exactly why reading the carrier would look right
// until an interface stamped it somewhere else.
func ttlKeepsAnswering(b *Bindings, _ *LawBinding) (projection.Defect, *subject.Method, string) {
	m := roleNamed(b, mixinTTL, mixinTTLRead)
	if m == nil {
		return nil, nil, RuleDeclined
	}
	return projection.AnswersWithValue{
		Clause: projection.Clause{Text: m.Name + " answers past the lifetime it was given"},
		Option: projection.OptionName(b.IfaceName, m.Name),
	}, m, ""
}

// roleNamed is the method a mixin's callable parameter names, nil where
// nothing on this interface carries the stamp or where the name it
// carries is not a method here.
//
// Through [golang.LocalName], because a sibling callable arrives
// qualified — the resolver answers with the name as identity spells it,
// and a method is looked up by the name the interface declares.
func roleNamed(b *Bindings, mixin, param string) *subject.Method {
	for i := range b.Methods {
		name, stamped := b.Methods[i].MixinParam(mixin, param)
		if !stamped || name == "" {
			continue
		}
		return methodNamed(b, golang.LocalName(name))
	}
	return nil
}

// countMiscounts plants an accounting that answers nothing: the count
// reports zero however much the subject is holding.
//
// Only against a DERIVED reference, and the refusal is the interesting
// half. This law compares two counts, and where the reference is the
// subject's own factory both sides are built from the same defect — the
// planted count answers zero on the left and zero on the right, the law
// finds them equal, and a row claiming Proven would rest on a proof that
// cannot fail. Nothing else about the defect would look wrong.
//
// So a count over a twin stays Argued and says which of the two it is
// waiting on. Twenty-three corpus rows sit there, and not one of them is
// waiting on a rule nobody has written.
func countMiscounts(b *Bindings, l *LawBinding) (projection.Defect, *subject.Method, string) {
	m := methodNamed(b, l.carrier.Name)
	if m == nil {
		return nil, nil, RuleDeclined
	}
	if !b.Reference.Derived() {
		return nil, nil, "this claim compares the subject's count against the " +
			"reference's, and the reference here is the subject's own factory — so " +
			"a planted miscount lands on both sides and the two agree; it needs a " +
			"derived reference to be wrong against"
	}
	return projection.AnswersAnyway{
		Clause: projection.Clause{Text: m.Name + " answers zero whatever is held"},
		Option: projection.OptionName(b.IfaceName, m.Name),
	}, m, ""
}

// appenderFreezes plants the frozen position: every append lands and
// every one answers the same offset.
func appenderFreezes(b *Bindings, l *LawBinding) (projection.Defect, *subject.Method, string) {
	// The law's own carrier, not a driven writer: this law appends
	// through a closure of its own, and the corpus's appender fixture
	// drives nothing but a reader. A defect over a method the law never
	// calls is one it cannot notice.
	m := methodNamed(b, l.carrier.Name)
	if m == nil {
		return nil, nil, RuleDeclined
	}
	return projection.FreezeReturn{Option: projection.OptionName(b.IfaceName, m.Name)}, m, ""
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
	if b.Delivery != nil {
		// A publisher's write is driven by the delivery set rather than by
		// an action of its own, and it is still the write this rule plants
		// through: the reference delivers what it was published, so a
		// publish that reports success and keeps nothing shows up at the
		// first comparison of what each side handed its subscriber.
		return methodNamed(b, b.Delivery.Publish)
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
