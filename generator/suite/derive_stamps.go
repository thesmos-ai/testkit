// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/aggregator"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/answeringwriter"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/batchreader"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/lookup"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/multireader"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/pointerreader"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/reader"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/readernoerror"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/readerwithbool"

	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// stampRule derives one stamp's checks. The tables below key rules by
// the upstream registry's own name constants — the derive layer never
// respells the vocabulary, and never enumerates it either: an unknown
// mixin refuses loudly, a law-backed one is recognized through tiers
// (the one home of law-backed-ness), and the conformance census holds
// table ∪ tiers ∪ refusals equal to the registry.
//
// A rule answers with refusals as well as plans, because a stamp can
// be recognized and still be unstateable on the interface carrying it:
// the reader shape is a signature, and a signature does not say
// whether anything can supply the input a miss withholds.
type stampRule func(f Iface, m subject.Method, call projection.CallPlan) ([]projection.CheckPlan, []Refusal)

// mixinRules is the mixin-axis derivation table: one row per attached
// classification this deriver speaks. Adding a row is the whole cost
// of covering a new deterministic mixin.
//
// The effect axis carries two positions and both are tabled:
// idempotent's repeat changes nothing, accumulates' repeat is taken.
// Neither is the other's negation — a callable carrying neither has
// not been considered, and only a stamped position is a contract.
func mixinRules() map[string]stampRule {
	return map[string]stampRule{
		MixinIdempotent:  idempotentRule,
		MixinAccumulates: accumulatesRule,
		MixinSideEffect:  sideEffectRule,
		MixinPartition:   partitionRule,
		MixinNilSafe:     nilSafeRule,
		MixinOrderAfter:  orderAfterRule,
		MixinValidates:   validatesRule,
		MixinIndexed:     indexedRule,
		MixinHooks:       hooksRule,
	}
}

// detectorRules is the detector-axis derivation table, keyed by the
// shape the annotator stamped. Only the reader family so far: the
// writer detectors are the seed derivation's input, the teardown
// shapes are the signature deriver's, and the remaining shapes join
// with the contracts deriver — the detector census arms when that
// deriver lands.
func detectorRules() map[string]stampRule {
	return map[string]stampRule{
		reader.Name:          missRule,
		readernoerror.Name:   missRule,
		readerwithbool.Name:  missRule,
		lookup.Name:          missRule,
		pointerreader.Name:   missRule,
		multireader.Name:     missRule,
		batchreader.Name:     missRule,
		aggregator.Name:      countRule,
		answeringwriter.Name: answerRule,
	}
}

// Stamps derives the deterministic stamp families — the claims a
// probe or two settles without the property engine. Law-backed stamps
// (ttl, bounded, lifecycleafterclose, …) are recognized through the
// tiers catalogue and left to the laws deriver; a mixin neither
// tabled nor law-backed refuses with the census framing, because an
// uncovered classification must be a named gap, never a silent one.
type Stamps struct{}

// Name implements [Deriver].
func (Stamps) Name() DeriverName { return DeriverStamps }

// Derive implements [Deriver].
func (Stamps) Derive(f Iface) ([]projection.CheckPlan, []Refusal) {
	var plans []projection.CheckPlan
	var refusals []Refusal
	mixins := mixinRules()
	detectors := detectorRules()

	for _, m := range f.Methods {
		detected := m.Shape()
		if len(m.Mixins) == 0 && detected == "" {
			continue
		}
		drawn := true
		if r, refused := argsRefusal(DeriverStamps, f, m, "'s stamp checks"); refused {
			refusals = append(refusals, r)
			drawn = false
		}
		call := callOf(m)
		for _, name := range m.Mixins {
			switch rule, tabled := mixins[name]; {
			case tabled:
				if !drawn && !spellsOwnArgs()[name] {
					// The call above names fixture accessors this run
					// has no values for. One refusal already says so;
					// a rule spelling that call would emit a body a
					// consumer cannot compile.
					continue
				}
				ruled, refused := rule(f, m, call)
				plans = append(plans, licensed(ruled, projection.AxisMixin, name)...)
				refusals = append(refusals,
					licensedRefusals(refused, projection.AxisMixin, name)...)
				// A rule here covers ONE of the classification's
				// obligations, and a law-backed classification carries
				// another (docs/adr/0028). Emitting the row and saying
				// nothing about the rest reads as a file that covered
				// the directive.
				refusals = append(refusals, elsewhereFor(name, m.Name)...)
			case len(tiers.LawsFor(name)) > 0:
				// The model tier's: the laws deriver binds it through
				// the tiers catalogue. Named rather than passed over in
				// silence — the consumer wrote the directive, and a file
				// that never mentions it again reads as one that checked it.
				refusals = append(refusals, Refusal{
					Deriver:   DeriverStamps,
					What:      name + " on " + m.Name,
					Why:       "stating it needs sequences of calls judged against a reference, which this file does not build",
					Remedy:    "nothing to fix — it is checked by the model tier, not here",
					Licensed:  projection.Licence{Axis: projection.AxisMixin, Name: name},
					Elsewhere: true,
				})
			case legStamps()[name]:
				// Claimed by the laws deriver, which covers it with a
				// leg rather than a law binding — so tiers.LawsFor is
				// empty for it and the case above cannot see the
				// coverage. A third state the census needs: not "no
				// rule", but "somebody else's rule".
			case documentedStamps()[name]:
				// Owes documentation rather than a check, by upstream
				// ruling. Recognised here so the census stops reporting
				// a gap against a classification whose own docblock says
				// it licenses nothing falsifiable.
			case consumedStamps()[name]:
				// An input to another rule rather than a claim of its
				// own. It owes no check because the check it feeds is
				// already derived — and refusing it would report a gap
				// against a stamp that closed one.
			default:
				refusals = append(refusals, Refusal{
					Deriver: DeriverStamps,
					What:    name + " on " + m.Name,
					Why: "testkit has no rule for deriving a check from this " +
						"declaration yet, so nothing here states it",
					Remedy: "write the check yourself, or report the gap — this " +
						"one is testkit's to close, not yours",
					Licensed:    projection.Licence{Axis: projection.AxisMixin, Name: name},
					Unaccounted: true,
				})
			}
		}
		if rule, tabled := detectors[detected]; tabled && drawn {
			ruled, refused := rule(f, m, call)
			plans = append(plans, licensed(ruled, projection.AxisDetector, detected)...)
			refusals = append(refusals,
				licensedRefusals(refused, projection.AxisDetector, detected)...)
		}
	}
	return plans, refusals
}

// licensed stamps every plan a rule answered with the classification
// that dispatched to it.
//
// At the dispatch rather than in the rule bodies, and that is the whole
// reason it can be trusted: the rules tables are KEYED by the
// classification, so the stamp and the dispatch read the same value and
// a rule reachable under two names is stamped correctly under each.
// Thirty-five plan literals stamping themselves would be thirty-five
// chances to name the wrong one — and missRule, which seven detector
// shapes share, could not have named any single one.
func licensed(
	plans []projection.CheckPlan, axis, name string,
) []projection.CheckPlan {
	for i := range plans {
		plans[i].Licensed = projection.Licence{Axis: axis, Name: name}
	}
	return plans
}

// Accounting reports how this deriver accounts for a classification it
// emits no check from, and whether it accounts for it at all.
//
// The three tables below are the deriver's answer to "why is there no
// check for this", and until now that answer only existed as a `case`
// arm with a comment. The conformance census asks the same question
// from outside and could not read them, so every classification these
// tables settle had to be settled a second time by hand in
// [gate.UnevidencedClassifications] — two registers for one fact, with
// nothing holding them level.
//
// Exported so the census reads the deriver's own accounting instead of
// a copy of it. A classification that moves between tables changes its
// reason here and the census follows on the next build.
func Accounting(name string) (string, bool) {
	switch {
	case legStamps()[name]:
		return "covered by a leg the laws deriver emits, which has a segment " +
			"rather than a lawid and so names nothing in the tiers catalogue", true
	case documentedStamps()[name]:
		return "licenses no falsifiable claim by upstream ruling — it declares " +
			"what something MEANS, which a reader needs and no check can drive", true
	case consumedStamps()[name]:
		return "an input another derivation reads rather than a claim of its " +
			"own; the check it feeds is derived under the rule that reads it", true
	default:
		return "", false
	}
}

// licensedRefusals stamps every refusal a rule answered with the
// classification that dispatched to it, as [licensed] does for plans.
//
// At the dispatch for the reason given there, and the shared rules are
// what make it necessary rather than tidy: missRule refuses for seven
// detector shapes and its refusal text names none of them, so a census
// reading the refusal alone could not say which shape went uncovered.
func licensedRefusals(rs []Refusal, axis, name string) []Refusal {
	for i := range rs {
		rs[i].Licensed = projection.Licence{Axis: axis, Name: name}
	}
	return rs
}

// spellsOwnArgs is the rules that write at least one argument
// themselves and so survive a draw the fixture cannot supply.
//
// The nil-argument check is the case that forced this: the very slot
// the fixture had no value for is the slot it spells nil, so refusing
// it for want of that value refuses the check on exactly the interfaces
// it exists for — a pointer parameter is both what makes nil
// expressible and what a literal sampler declines to invent.
//
// Each such rule still has to answer for the arguments it does NOT
// spell; the deriver cannot know which those are, so the checking is
// the rule's.
func spellsOwnArgs() map[string]bool {
	return map[string]bool{MixinNilSafe: true}
}

// The past-tense supply verbs a writer-fed miss claim speaks: the
// simple past beside a sentinel, the participle after the zero form's
// "has". The seed-seam interface speaks "seeded" in both shapes. A
// domain wording the defaults cannot reach — the corpus's "counted" —
// waits on a directive home, recorded in the design doc's frontier.
const (
	missVerbWrote   = "wrote"
	missVerbWritten = "written"
)

// missWording derives what a miss claim speaks: the declared sentinel
// where one is stamped — the ttl declaration's notfound param is the
// one stamped home today — and the supply verb from how the subject
// is populated.
//
// Keyed on Corpus rather than on seeded(): the two agree on every
// interface that has a key and a payload to zip, and diverge on one
// that writes nothing and has no pools either — a codec — where
// seeded() is vacuously true and no run seeds anything. "Nothing has
// seeded" is not what a transform's miss would mean.
func missWording(f Iface, m subject.Method) (sentinel, verb string) {
	sentinel, _ = subject.MissSentinel(m)
	switch {
	case f.Corpus:
		verb = supplySeeded
	case sentinel == "":
		verb = missVerbWritten
	default:
		verb = missVerbWrote
	}
	return sentinel, verb
}

// legStamps are the classifications the laws deriver covers with a LEG
// rather than with a law binding.
//
// A leg runs an engine other than the property runner — linearizability
// runs the linearize engine — so it has a segment instead of a lawid and
// nothing in the tiers catalogue names it. That makes it invisible to
// the law-backed case above, which asks tiers.LawsFor and gets nothing,
// and the classification was reported as an uncovered gap while a row
// for it was being emitted two derivers away.
func legStamps() map[string]bool {
	return map[string]bool{
		// ClassConcurrent is model/concurrent and [Laws.Derive] emits
		// the linearizable leg wherever this is stamped.
		MixinConcurrent: true,
	}
}

// documentedStamps are the classifications that license no falsifiable
// claim at all, by upstream ruling rather than by our omission.
//
// `scope` names what an axis MEANS — request, session, tenant — which
// a human and a grouping consumer read and no check needs. The
// isolation it describes is `partition`'s, which names the observer
// too; the two compose on one callable, the naming form beside the
// checkable one, as `idempotent` and `accumulates` sit on the effect
// axis.
//
// `errors` marks a callable's error returns as part of its contract
// rather than "shouldn't happen". That changes how a reader treats
// them and is worth declaring; it is not a claim any tier can drive.
// Which sentinel answers which condition is a separate declaration —
// `notfound sentinel=` and the siblings named as corpora ask for them
// — and encoding the mapping in one value would be a graph the
// resolver cannot check.
func documentedStamps() map[string]bool {
	return map[string]bool{
		MixinErrors: true,
		MixinScope:  true,
	}
}

// consumedStamps are the classifications another derivation READS
// rather than deriving a check from.
//
// The category the census had no name for. A stamp usually states a
// claim and owes a check; these state an identity some other rule needs
// — and a stamp that closed a gap being reported AS a gap is the
// refusal list saying the opposite of what happened.
func consumedStamps() map[string]bool {
	return map[string]bool{
		// Names WHAT a miss reports. The claim that a miss IS reported
		// belongs to the reader shape, whose rule reads this to choose
		// the sentinel arm of the body over the zero arm.
		MixinNotFound: true,

		// Names where a member of an input space too large to enumerate
		// comes from. Spent by the smoke, which borrows from the builder
		// rather than drawing a literal — and the bare form, with no
		// builder, states what this tier does by construction: it draws
		// one value per role and has no exhaustive mode to be told not
		// to use.
		MixinSample: true,

		// Declares that the answer moves with the clock. Spent by
		// withholding the seeded probes: a hit compares an answer
		// against what the run put in, and a count against how many —
		// both of which a subject may legitimately change between the
		// seeding and the read when time is an input. Controlling the
		// clock so those hold again is the model tier's; what this tier
		// owes is not to assert them.
		MixinTimeAware: true,
	}
}

// elsewhereFor names the obligations another tier holds for a
// classification this deriver already emitted a row for.
//
// The two are not alternatives. A rule here settles what a fixed call
// sequence can settle; a law settles what needs generated ones. Where a
// classification carries both, the header has to say so — a file
// listing `idempotent` among its checks and nothing else has told the
// reader that a repeat was tried once, not that N calls equal one.
//
// Empty for a classification the suite tier covers alone, which is the
// ordinary case and wants no note.
func elsewhereFor(name, method string) []Refusal {
	obligations := tiers.ObligationsFor(name)
	out := make([]Refusal, 0, len(obligations))
	for _, ob := range obligations {
		tier, _ := ob.Tier()
		out = append(out, Refusal{
			Deriver: DeriverStamps,
			What:    name + " on " + method + ", for call sequences this file does not write",
			Why: "the check above runs one fixed sequence, and your directive claims " +
				"the same of every other",
			Remedy:     "nothing to fix — it is checked by the " + string(tier) + " tier",
			Obligation: ob,
			Licensed:   projection.Licence{Axis: projection.AxisMixin, Name: name},
			Elsewhere:  true,
		})
	}
	return out
}
