// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"

	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// The claim wording policy for the derived families, spelled from the
// corpus manifests. One function per family: a claim is derived text,
// and its one home is here — the corpus's historical wavering between
// "a derived entry" and "derived inputs" for one-parameter methods
// reconciles to the noun form, which says more. The noun is the
// parameter's own identifier, read from the shared signature
// projection.

// The supply adjectives a claim can speak: "derived" for inputs the
// fixture derives from pools, "seeded" on the seed-seam interface
// whose corpus the harness receives.
const (
	supplyDerived = "derived"
	supplySeeded  = "seeded"
)

// SmokeClaim words the smoke family by drawn arity: no draws →
// "survives a call"; one → "survives a call with a derived <noun>";
// several → "survives a call with derived inputs" — "seeded"
// replacing "derived" where the interface's corpus is seeded.
func SmokeClaim(m subject.Method, seeded bool) string {
	supply := supplyDerived
	if seeded {
		supply = supplySeeded
	}
	draws := m.CallArgs()
	switch len(draws) {
	case 0:
		return m.Name + " survives a call"
	case 1:
		return m.Name + " survives a call with a " + supply + " " + drawNoun(draws[0])
	default:
		return m.Name + " survives a call with " + supply + " inputs"
	}
}

// OpenerSmokeClaim words the producing method's smoke: the call
// survives and the handle it opens closes. The produced noun is the
// contract's own name — the vocabulary's one home for what the
// handle is called.
func OpenerSmokeClaim(m subject.Method, produced string) string {
	return m.Name + " survives a call and the " + produced + " it opens closes"
}

// BuiltSmokeClaim words the smoke of a method whose input its own
// sibling mints: the call survives an input that sibling made.
func BuiltSmokeClaim(m subject.Method, builder string) string {
	return m.Name + " survives a call with an input " + builder + " made"
}

// BorrowSmokeClaim words the returning method's smoke: the resource
// it returns was borrowed from the producing sibling. "resource" is
// the corpus's word for the borrowed thing; a second borrowing domain
// argues for deriving it before a rule invents one.
func BorrowSmokeClaim(m subject.Method) string {
	return m.Name + " survives returning a borrowed resource"
}

// CancelClaim words the cancel family.
func CancelClaim(m subject.Method) string {
	return m.Name + " reports a cancelled context as cancelled"
}

// DeadlineClaim words the deadline family.
func DeadlineClaim(m subject.Method) string {
	return m.Name + " reports an expired deadline as exceeded"
}

// NilCtxClaim words the nilcontext family.
func NilCtxClaim(m subject.Method) string {
	return m.Name + " returns an error rather than panicking on a nil context"
}

// ZeroOnErrorClaim words the zero family by the first value result's
// shape: "a nil channel" for channels, "the zero <Name>" for named
// types, "zero" for builtins — and for the synthesized signatures unit
// tests build without a source type, since a shape nobody declared has
// no name to speak. Empty for a method with no value result, which the
// deriver gates on before wording anything.
func ZeroOnErrorClaim(m subject.Method) string {
	values := m.ValueReturns()
	if len(values) == 0 {
		return ""
	}
	if len(values) > 1 {
		// The whole answer, because that is what the body judges. A
		// claim naming one slot of two is the understatement a subject
		// leaking its metadata slips through.
		return m.Name + " returns " + zeroNouns(values) + " alongside any error"
	}
	src := values[0].Source
	switch {
	case src != nil && golang.IsChannel(src):
		// The prose names the channel where the comparison only knows
		// it is nil-compared: a claim is read by someone deciding
		// whether it holds, and "nothing" would not tell them what.
		return m.Name + " returns a nil channel alongside any error"
	case ZeroShapeOf(m) == ZeroNil:
		return m.Name + " returns nothing alongside any error"
	case src != nil && src.Name != "" && !golang.IsPredeclared(src.Name):
		// Predeclared rather than IsBuiltin: the frontend records an
		// in-package named type with no package, exactly like "int",
		// and "the zero Value" needs the two told apart.
		return m.Name + " returns the zero " + src.Name + " alongside any error"
	default:
		return m.Name + " returns zero alongside any error"
	}
}

// ZeroShape is how a body compares a result against its zero.
type ZeroShape int

// Two shapes, and the split is comparability rather than spelling: a
// slice, map or func may only be compared against nil — `got != zero`
// does not compile for one — while every other type has a zero that can
// be declared and compared, predeclared types included.
const (
	// ZeroDeclared declares a zero of the result's own type and
	// compares against it: `var zero kv.Value`, then `got != zero`.
	// Right for named types, strings, bools and numbers alike.
	ZeroDeclared ZeroShape = iota

	// ZeroNil compares against nil, for the kinds that admit nothing
	// else.
	ZeroNil
)

// ZeroShapeOf classifies a method's first result.
//
// One home because the claim and the body it words are the same
// judgment about the same type: a claim promising "the zero Value"
// beside a body comparing against nil is the drift a single inventory
// exists to prevent.
func ZeroShapeOf(m subject.Method) ZeroShape {
	values := m.ValueReturns()
	if len(values) == 0 {
		return ZeroDeclared
	}
	src := values[0].Source
	if src == nil {
		return ZeroDeclared
	}
	switch src.TypeKind {
	case node.TypeRefSlice, node.TypeRefMap, node.TypeRefFunc, node.TypeRefPointer:
		// A pointer is comparable, so a declared zero would work — but
		// it has no name to declare one of, and nil is what the
		// language calls its zero anyway.
		return ZeroNil
	default:
		if golang.IsChannel(src) {
			// A channel arrives as a named ref with the frontend's own
			// stamp on it, never as a kind of its own.
			return ZeroNil
		}
		return ZeroDeclared
	}
}

// zeroNouns words what a multi-slot answer owes a zero of.
//
// The slots are named where naming them helps — "the zero Value and
// Meta" tells a reader which two results the check compares. Where a
// slot has no type name, or two share one, the names stop
// distinguishing anything and the claim says "every result" instead:
// "the zero int and int" is worse than saying nothing about which.
func zeroNouns(values []golang.Return) string {
	names := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, ret := range values {
		src := ret.Source
		if src == nil || src.Name == "" || seen[src.Name] {
			return "the zero for every result"
		}
		seen[src.Name] = true
		names = append(names, src.Name)
	}
	last := len(names) - 1
	return "the zero " + strings.Join(names[:last], ", ") + " and " + names[last]
}

// PartitionClaim words the isolation claim, naming the axis so a
// reader knows which parameter the boundary is drawn along.
func PartitionClaim(m subject.Method, axis string) string {
	return m.Name + " keeps one " + axis + " out of another"
}

// PartitionRequirement words what the failure says the method must do.
//
// Here rather than in the template because a claim and the failure
// reporting it broken are one sentence in two moods, and a wording
// policy split across two files is one that drifts: the row would go on
// saying the subject keeps partitions apart while the failure it emits
// asked for something else.
func PartitionRequirement(axis string) string {
	return "not let one " + axis + " reach another"
}

// SideEffectClaim words the named-pair claim: the call is observable
// through the partner the directive named.
//
// Names both members, because the claim is about the relationship and a
// reader deciding whether it holds needs to know where to look.
func SideEffectClaim(m subject.Method, observer string) string {
	return m.Name + " changes what " + observer + " observes"
}

// SideEffectRequirement words what the failure says the method must do
// — the imperative of [SideEffectClaim], for the reason
// [PartitionRequirement] is stated beside its claim.
func SideEffectRequirement(observer string) string {
	return "change what " + observer + " observes"
}

// The defect clauses: what a planted double DOES, in the grammar
// [projection.DefectName] wraps — "a Store whose Get panics".
//
// Beside the claims because a proof's subject line is prose about the
// same method, and a wording policy with two homes is one that drifts.
// One function per DEFECT rather than per claim: several claims are
// broken by the same planted statement, and what a report has to say is
// what the double does, not what the claim wanted.

// PanicsClause words the smoke family's double.
func PanicsClause(m subject.Method) string { return m.Name + " panics" }

// SwallowsContextClause words the cancel and deadline families' double.
func SwallowsContextClause(m subject.Method) string {
	return m.Name + " ignores the context it is handed"
}

// ForgivesNilContextClause words the nilcontext claim's double, whose
// stronger arm — returns an error — is broken by answering instead.
func ForgivesNilContextClause(m subject.Method) string {
	return m.Name + " forgives a nil context and answers"
}

// DropsWriteClause words the double that acknowledges a write and keeps
// none of it.
func DropsWriteClause(m subject.Method) string {
	return m.Name + " reports success and keeps nothing"
}

// AnswersMissClause words the double that answers for an input nothing
// supplied — the miss claim's, in both its arms.
func AnswersMissClause(m subject.Method) string {
	return m.Name + " answers for an input nothing wrote"
}

// EchoesBesideErrorClause words the zero-on-error family's double.
func EchoesBesideErrorClause(m subject.Method) string {
	return m.Name + " answers a believable value beside its error"
}

// ForgivesNilArgumentClause words the double that takes the nil and
// carries on.
func ForgivesNilArgumentClause(m subject.Method) string {
	return m.Name + " accepts a nil argument and answers"
}

// AnswersEarlyClause words the double that answers before its
// predecessor has run.
func AnswersEarlyClause(m subject.Method) string {
	return m.Name + " answers before its predecessor has run"
}

// AnswersTheZeroClause words the double that succeeds and hands back
// the zero.
func AnswersTheZeroClause(m subject.Method) string {
	return m.Name + " reports success and answers the zero"
}

// LandsRegardlessClause words the double that writes whatever its own
// predicate answered.
func LandsRegardlessClause(m subject.Method, match string) string {
	return m.Name + " lands whatever " + match + " says"
}

// AnswersNoStreamClause words the double that reports success and hands
// back nothing to receive on.
func AnswersNoStreamClause(sub subject.Method) string {
	return sub.Name + " reports success and answers no stream"
}

// AnswersPastTheEndClause words the double that answers for a position
// the collection does not hold.
func AnswersPastTheEndClause(m subject.Method) string {
	return m.Name + " answers for a position past the end"
}

// SkipsHooksClause words the double that does its work without running
// what was registered.
func SkipsHooksClause(m subject.Method) string {
	return m.Name + " reports success and runs no hook"
}

// AnswersTheZeroForEverySeedClause words the double that answers the
// zero for every key the run seeded.
func AnswersTheZeroForEverySeedClause(m subject.Method) string {
	return m.Name + " answers the zero for every key the run seeded"
}

// CountsNothingClause words the double that reports an empty aggregate
// over a seeded subject.
func CountsNothingClause(m subject.Method) string {
	return m.Name + " reports no entries however many the run seeded"
}

// TakesDuplicateClause words the double that accepts a write of what is
// already there.
func TakesDuplicateClause(m subject.Method) string {
	return m.Name + " accepts a duplicate as though it were new"
}

// RefusesEverythingClause words the double that reports an error for
// every call.
func RefusesEverythingClause(m subject.Method) string {
	return m.Name + " refuses everything it is handed"
}

// RepeatFailsClause words the idempotent claim's double.
func RepeatFailsClause(m subject.Method) string {
	return m.Name + " fails on the second call"
}

// localName cuts a resolved partner reference back to the identifier a
// call site spells.
//
// The annotator hands a partner back QUALIFIED, which is right for
// identity and wrong for a call: the generated body calls the method on
// the subject, where the package qualifier is not in scope.
func localName(ref string) string {
	if i := strings.LastIndex(ref, "."); i >= 0 {
		return ref[i+1:]
	}
	return ref
}

// NilArgumentClaim words the nil-safety claim about a value parameter,
// naming the argument so a reader knows which slot the nil goes in.
func NilArgumentClaim(m subject.Method, arg golang.Param) string {
	return m.Name + " reports a nil " + drawNoun(arg) + " rather than panicking"
}

// OrderAfterClaim words the ordering claim: the call refuses, with the
// declared sentinel, until its predecessor has run.
//
// Names all three — the call, what it waits for, and what being early
// reports — because a reader deciding whether the claim holds has to
// know which error counts as the refusal. A qualified sentinel speaks
// its bare name, as [MissClaim]'s does.
func OrderAfterClaim(m subject.Method, predecessor, sentinel string) string {
	return m.Name + " reports " + localName(sentinel) + " until " + predecessor + " has run"
}

// ValidatesClaim words the agreement claim: the method takes exactly
// what the validator takes.
//
// Narrower than the classification, deliberately. The mixin says
// invalid input is screened before any work runs; what a caller with no
// invalid value in hand can see is whether the two verdicts MATCH, and
// a suite that invented an invalid value would be guessing at the rule
// under test. The screening's other half — that a refused value left
// nothing behind — needs a reader the directive names no parameter for.
func ValidatesClaim(m subject.Method, validator string) string {
	return m.Name + " agrees with " + validator + " about the values this run draws"
}

// ValidatesRequirement words what the failure says the method must do —
// the imperative of [ValidatesClaim].
func ValidatesRequirement(validator string) string {
	return "refuse exactly what " + validator + " refuses"
}

// AnswerClaim words what a write that answers its stored state owes: an
// answer at all.
//
// Narrower than the shape, and deliberately. Whether the answer equals
// what was handed in or carries a stamp the store assigned is the
// subject's business — the shape exists BECAUSE stores assign stamps —
// and a claim picking either would fail the other. The zero is the one
// answer no such write may give.
func AnswerClaim(m subject.Method) string {
	return m.Name + " answers the state it kept rather than the zero"
}

// AnswerRequirement words what the failure says an answering write must
// do — the imperative of [AnswerClaim].
func AnswerRequirement() string { return "answer the state it kept" }

// SubscriberRequirement words what the failure says a subscriber must
// do — the imperative of [SubscriberClaim].
func SubscriberRequirement() string { return "answer a stream a caller can receive on" }

// SubscriberClaim words what an outbox subscriber owes a caller: a
// channel, not a nil one.
//
// Narrower than the contract, and the narrowing is the tier boundary.
// That an append ARRIVES is a claim about a wait, which needs a clock;
// that a subscriber answered something to wait on needs nothing but the
// call.
func SubscriberClaim(sub subject.Method) string {
	return sub.Name + " answers a stream a caller can receive on"
}

// ConflictClaim words the conditional write's refusal, naming the
// sentinel so a reader knows which error counts as the refusal. A
// qualified one speaks its bare name, as [MissClaim]'s does.
func ConflictClaim(m subject.Method, conflict string) string {
	return "a second " + m.Name + " of what is already there reports " + localName(conflict)
}

// MatchClaim words the conditional write's agreement with its own
// predicate.
//
// Narrower than the contract, for the reason [ValidatesClaim] is: a
// suite has no value it knows the predicate rejects, so what it can
// state is that the two answers line up.
func MatchClaim(m subject.Method, match string) string {
	return m.Name + " agrees with " + match + " about the values this run draws"
}

// MatchRequirement words what the failure says the method must do — the
// imperative of [MatchClaim].
func MatchRequirement(match string) string {
	return "land exactly when " + match + " says it may"
}

// The reasons a zero-judging failure gives for why the zero was owed,
// in the grammar the judgement fragment wraps: "Get must return the
// zero value <reason>".

// BecauseErred is the zero-on-error family's reason.
func BecauseErred() string { return "alongside an error" }

// BecauseUnsupplied is the reader miss's reason.
func BecauseUnsupplied() string { return "for an input nothing supplied" }

// BecausePastTheEnd is the positional read's, naming the sizer that
// said where the end was.
func BecausePastTheEnd(sizer string) string {
	return "at the size " + sizer + " reports"
}

// HooksClaim words the callback claim, naming the registrar so a reader
// knows where a hook is installed.
func HooksClaim(m subject.Method, register string) string {
	return m.Name + " runs what " + register + " registered"
}

// HooksRequirement words what the failure says the method must do — the
// imperative of [HooksClaim].
func HooksRequirement(register string) string {
	return "run what " + register + " registered"
}

// BoundClaim words the positional read's edge, naming the sizer so a
// reader knows where the bound came from.
func BoundClaim(m subject.Method, sizer string) string {
	return m.Name + " reports no element at the size " + sizer + " reports"
}

// IdempotentClaim words what the suite tier can state about the
// idempotent mixin: that a repeat is ACCEPTED.
//
// Narrower than the classification, deliberately, and for the reason
// [AccumulatesClaim] is narrower than its own. The mixin claims a repeat
// leaves the observable state unchanged, and "unchanged" needs something
// to read the state through — a reader the directive names no parameter
// for. The body behind this claim is a repeat probe: it calls twice and
// requires the second call to succeed, and it reads nothing back.
//
// It said "changes nothing" until the strength census put the two side by
// side. The claim promised state was compared; the body compared nothing;
// and the planted defect written for it fails the second call, so the
// proof agreed with the check and neither noticed. A claim wider than its
// body is the one defect this surface cannot catch for a consumer, and
// emitting one from the generator is worse than emitting none — the
// consumer has no way to know it was ever a promise nobody kept.
//
// Stating that the state is unchanged needs a reader, and that is the
// model tier's to make.
func IdempotentClaim(m subject.Method) string {
	return "a second " + m.Name + " after a clean one is accepted"
}

// AccumulatesClaim words what the suite tier can state about the
// accumulates mixin: that a repeat is ACCEPTED.
//
// Narrower than the classification, deliberately. The mixin claims N
// invocations have N observable effects, and observing them needs
// something to read the effect through — which the directive names no
// parameter for, and which no signature identifies. What one subject
// and a fixed sequence settle is the half a coalescing store gets
// wrong first: it refuses the second call, or answers it as a
// duplicate. The compounding half is the model tier's and waits on a
// law, recorded in the design doc's frontier.
//
// The mirror of [IdempotentClaim] over the same two calls: there the
// second must change nothing, here it must be taken.
func AccumulatesClaim(m subject.Method) string {
	return "a second " + m.Name + " is accepted rather than refused as a repeat"
}

// MissClaim words the reader miss by its answer shape and supply verb:
// the sentinel form where the declaration names one, the zero form
// otherwise. A qualified sentinel ("kv.ErrNotFound") speaks its bare
// name — claims read as prose, and the qualifier is the generated
// code's concern.
func MissClaim(m subject.Method, sentinel, verb string) string {
	noun := missNoun(m)
	if sentinel == "" {
		return m.Name + " reports zero for a " + noun + " nothing has " + verb
	}
	if i := strings.LastIndex(sentinel, "."); i >= 0 {
		sentinel = sentinel[i+1:]
	}
	return m.Name + " reports " + sentinel + " for a " + noun + " nothing " + verb
}

// HitClaim words the seeded hit.
func HitClaim(m subject.Method) string {
	return m.Name + " returns the seeded value for every seeded " + missNoun(m)
}

// CountClaim words the seeded aggregator.
func CountClaim(m subject.Method) string {
	return m.Name + " equals the number of seeded entries"
}

// missNoun is the word the reader claims call their probed input.
func missNoun(m subject.Method) string {
	if draws := m.CallArgs(); len(draws) > 0 {
		return drawNoun(draws[0])
	}
	return "input"
}

// drawNoun is the word a claim calls one drawn parameter, lower-cased —
// `Lookup(ctx, id Key)` draws "a seeded key", per the corpus.
//
// The word itself is [projection.DrawWord], which the fixture field is
// also spelled from: a claim naming one thing and a field holding
// another is the drift a single inventory exists to prevent. The
// composite-request form ("derived inputs" for a one-struct draw)
// needs the fixture's composed-field fact and arrives with the emitter
// wiring.
func drawNoun(p golang.Param) string {
	return strings.ToLower(projection.DrawWord(p))
}
