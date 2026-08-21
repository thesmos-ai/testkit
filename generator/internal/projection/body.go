// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection

import "go.thesmos.sh/testkit/engine/suite"

// BodyKind names a body variant; the value is the variant's template
// name, composed from the one dispatch prefix.
type BodyKind string

// BodyKindPrefix namespaces body templates in the dispatch table.
//
// The plugin's own name leads it because the backend parses every
// plugin's templates into one map: a kind is a {{define}} name in a
// shared namespace, so an unprefixed "body.smoke-survives" would
// collide with any other plugin that reached for the obvious word.
// Same rule the emitted node kinds follow.
const BodyKindPrefix = "suite.body."

// The body kinds, one per variant below.
const (
	KindSmokeSurvives   BodyKind = BodyKindPrefix + "smoke-survives"
	KindGuardedCall     BodyKind = BodyKindPrefix + "guarded-call"
	KindZeroOnMiss      BodyKind = BodyKindPrefix + "zero-on-miss"
	KindZeroOnCancel    BodyKind = BodyKindPrefix + "zero-on-cancel"
	KindRepeatProbe     BodyKind = BodyKindPrefix + "repeat-probe"
	KindReportsSentinel BodyKind = BodyKindPrefix + "reports-sentinel"
	KindAnswersZero     BodyKind = BodyKindPrefix + "answers-zero"
	KindHitProbe        BodyKind = BodyKindPrefix + "hit-probe"
	KindCountProbe      BodyKind = BodyKindPrefix + "count-probe"
	KindLawLeg          BodyKind = BodyKindPrefix + "law-leg"
	KindDifferentialLeg BodyKind = BodyKindPrefix + "differential-leg"
	KindSimLeg          BodyKind = BodyKindPrefix + "sim-leg"
	KindHookFires       BodyKind = BodyKindPrefix + "hook-fires"
	KindNonZeroAnswer   BodyKind = BodyKindPrefix + "non-zero-answer"
	KindPartnerAgrees   BodyKind = BodyKindPrefix + "partner-agrees"
	KindReadActRead     BodyKind = BodyKindPrefix + "read-act-read"
	KindWriteWriteRead  BodyKind = BodyKindPrefix + "write-write-read"
)

// CallPlan spells one method invocation a body makes: the method name
// and the rendered argument expressions, fixture draws included.
type CallPlan struct {
	Method string
	Args   []Expr
}

// The body variants. Each is exactly the data its template needs and
// nothing speculative — a variant grows a field the day a template
// renders it.

// SmokeSurvives asserts the call returns without panicking; a produced
// handle it opens is closed in the same body. One variant carries the
// three smoke shapes — plain, opener, borrower — because all three
// state one claim family and differ only in prologue and epilogue.
type SmokeSurvives struct {
	Call CallPlan
	// Borrow is the producing sibling called first when the smoked
	// method's input is pool-produced: its result feeds the smoked
	// call, and a failed borrow returns without judgment — the
	// producer's own smoke owns that path.
	Borrow CallPlan
	// CloseProduced names the produced handle's release method when
	// the smoked method answers one — the opener owns what it opens.
	CloseProduced string
}

// Guard names the engine primitive a [GuardedCall] delegates to.
//
// The values are the engine's own name constants, so the vocabulary
// keeps one home: a renamed primitive is a compile error in this
// package rather than a generated file calling a function that no
// longer exists, which a consumer would meet in their own build.
type Guard string

// The guards, from the engine vocabulary.
const (
	GuardCancelled  = Guard(suite.GuardCancelled)
	GuardDeadline   = Guard(suite.GuardDeadline)
	GuardNilContext = Guard(suite.GuardNilContext)

	// GuardNilArgument judges a nil VALUE argument rather than a nil
	// context. Which slot is nil is the emitted call's to spell, so the
	// two guards share this variant and differ in the primitive alone.
	GuardNilArgument = Guard(suite.GuardNilArgument)
)

// GuardedCall hands the call to an engine primitive that induces a
// hostile context and judges what comes back.
//
// One variant for the whole family because the emitted statements are
// one statement — the primitive's name, and a closure around the call.
// What "reports a cancelled context" MEANS lives in engine/suite, so
// the arms differ in an identifier and agree in everything else, and
// three template defines matching byte for byte apart from one literal
// are three places to fix the day the closure changes shape.
//
// The line this does not cross: a guard whose SETUP differs, rather
// than its name, is a different body. Nothing is induced here — the
// primitive does that — so there is no second shape hiding in the
// field.
type GuardedCall struct {
	Call  CallPlan
	Guard Guard
}

// ZeroOnMiss asserts the non-error results are zero when a draw that
// nothing seeded produces the declared miss sentinel.
//
// Split from [ZeroOnCancel] because the two induce their error
// differently and so are different statement sequences, not one body
// with a mode: this one draws the alternate member and skips when the
// subject answers anyway, that one cancels a context first. A single
// variant covering both was one variant covering three shapes, which is
// how the closed set stops meaning anything.
type ZeroOnMiss struct {
	// Call draws the ALTERNATE member — the error this check inspects
	// only happens for an input nothing wrote.
	Call CallPlan

	// Because is the phrase the failure gives for why the zero was
	// owed, carried for the reason [AnswersZero.Because] is: the
	// judgement fragment serves several claims and a reason written
	// for one reads as a lie under another.
	Because string

	// Pool is the config field a consumer seeds to make the miss a
	// miss, named in the skip so a run that proves nothing says what
	// would make it prove something.
	Pool string
}

// ZeroOnCancel asserts the non-error results are zero when a cancelled
// context produces the error.
//
// The form for a method whose inputs cannot miss — one that takes none,
// or one no sentinel declares a miss for. A cancelled context is the
// only error every context-taking method can be made to report.
type ZeroOnCancel struct {
	Call    CallPlan
	Because string
}

// RepeatProbe calls a method twice and judges the second: the first
// call is a precondition and fails the check outright, the second is
// the claim.
//
// The asymmetry is the whole shape and is why this is not a list of
// calls: a body that treated both the same would report a subject
// whose first Close failed as a subject that is not idempotent, which
// is a different fault with a different fix.
type RepeatProbe struct{ Call CallPlan }

// ReportsSentinel calls the method and requires the error it reports to
// be the declared one.
//
// Named for the sequence — one call, one errors.Is — which is what lets
// the reader miss, the order-after refusal and the stale-token write
// share it. Which error is owed, and why this call is one that owes it,
// belong to the deriving rule.
type ReportsSentinel struct {
	Call     CallPlan
	Sentinel Expr

	// Prologue is a call that must succeed before the judged one means
	// anything, empty where the claim needs no setup. It fatals rather
	// than erroring: a subject that refused the setup has a different
	// fault from one that failed to report the sentinel, and reporting
	// the second for the first sends a reader to the wrong method.
	//
	// A field rather than a second kind because the statements are the
	// same statements with one call in front, and the conditional write
	// needs exactly that — its conflict only exists once something is
	// there to conflict with.
	Prologue CallPlan
}

// AnswersZero calls the method and requires every value slot to be its
// own zero.
//
// Split from [ReportsSentinel] rather than carried as its empty-sentinel
// arm, which is what the two were: with a sentinel the claim is
// errors.Is against it, without one the claim is about the values and
// the call has to have succeeded first. Two statement sequences — and a
// kind rendering either is a kind whose name can only describe one.
type AnswersZero struct {
	Call CallPlan

	// Bound is a call whose answer supplies one of the judged call's
	// arguments, empty where every argument is drawn. The judged call
	// spells [ExprBound] in that slot.
	//
	// Because is the phrase the failure gives for WHY the answer was
	// owed to be zero — "for an input nothing supplied", "at the size
	// Len reports". Carried because the shape now serves two claims and
	// a reason written for one reads as a lie under the other.
	//
	// A field rather than a second kind, for the reason
	// [ReportsSentinel.Prologue] is one: the statements are the same
	// statements with a call in front. What differs from a prologue is
	// that this one's ANSWER is used — a positional read has no way to
	// ask for one past the end without first asking how many there are,
	// and a number invented from the type is the vacuity the sizer
	// exists to replace.
	Bound   CallPlan
	Because string
}

// HitProbe reads back what the run seeded and judges the answer
// against it.
//
// Only derivable where the interface seeds, which is what supplies
// something to read back.
type HitProbe struct{ Call CallPlan }

// CountProbe judges an aggregate against the size of what was seeded.
type CountProbe struct{ Call CallPlan }

// LawLeg delegates to legs.Law with the named engine laws; Laws also
// feeds the plan's Binds. Probes maps a law's probe name to its call,
// for multi-probe laws whose claim spans several methods.
type LawLeg struct {
	Laws   []Bind
	Probes map[string]CallPlan
	// Extra carries leg options (a history reset, a produced lift)
	// as rendered expressions.
	Extra []Expr
}

// DifferentialLeg delegates to legs.Differential over the derived
// reference and the action vocabulary.
type DifferentialLeg struct {
	// The model file owns actions and references; the suite file's
	// check only names the assert function the model file exports.
	AssertFunc string
}

// SimKind names a sim-tier leg; the values are the runtime's own
// segment constants, so the sim vocabulary keeps one home.
type SimKind string

// The sim kinds, from the engine vocabulary.
const (
	SimRecovery = SimKind(suite.SegRecovery)
	SimCrash    = SimKind(suite.SegCrash)
	SimFault    = SimKind(suite.SegFault)
)

// SimLeg is a recovery/crash/fault check body over the Recover seam.
type SimLeg struct {
	Kind SimKind
}

// HookFires installs a recording callback through the named registrar,
// calls the method, and requires the callback to have run.
//
// The one shape here whose observation is not a call on the subject. A
// registered callback is observable only from inside itself, so the
// body writes a closure over a local and reads that local afterwards —
// which is why no existing kind renders this and why it earned its own.
//
// The call between fatals. A hook that did not fire because the work
// never happened is a different fault from one the work forgot.
type HookFires struct {
	// Register installs the callback; its argument in the callback slot
	// is [ExprHook], the local the body declares just above.
	Register CallPlan

	// Call is the method whose work is claimed to run it.
	Call CallPlan

	// Must is the requirement the failure spells after the method's
	// name — "run what OnEvent registered".
	Must string
}

// NonZeroAnswer calls the method once and requires its first result to
// differ from that type's zero.
//
// The mirror of [AnswersZero] in meaning and not in statements: this
// one fatals when the call fails — a write that did not happen says
// nothing about what it would have answered — and judges the first slot
// alone, where the zero family judges every slot and judges no error.
//
// The weakest thing worth saying about a write that answers. What the
// answer SHOULD be is the subject's business: a store may return the
// value it was handed or one it stamped, and a check requiring either
// would fail the other. What no such write may do is hand back the
// zero, which tells the caller nothing and is exactly the outcome the
// shape exists to make observable.
type NonZeroAnswer struct {
	Call CallPlan

	// Must is the requirement the failure spells after the method's
	// name — "answer the state it kept". Two rules reach this shape and
	// they owe different things: a write owes the state it stored, a
	// subscriber owes something to receive on, and one prose for both
	// would be wrong for whichever it was not written for.
	Must string
}

// PartnerAgrees calls a partner and the subject on the same drawn
// value, and requires the two to agree about whether it is acceptable.
//
// Agreement rather than a verdict, because a suite has no invalid value
// to hand in: what counts as invalid is the subject's own business, and
// a fixture inventing one would be guessing at the very rule under
// test. What a caller CAN see is a disagreement — a value the validator
// refuses and the method takes, or the reverse — and that is exactly the
// bug the classification exists to catch.
//
// Both calls draw the same fixture field, which is what makes the
// comparison about the subject rather than about two different values.
type PartnerAgrees struct {
	// Call is the method whose acceptance is claimed; Partner is the
	// validator the directive named.
	Call, Partner CallPlan

	// PartnerBool says the partner answers a bool rather than an error,
	// so agreement is its yes against the subject's success. The zero
	// value compares two error channels.
	//
	// One sequence either way — two calls and one condition — which is
	// why the direction rides here rather than in a second kind that
	// would differ by an operator.
	PartnerBool bool

	// Must is the requirement the failure spells after the method's
	// name — "refuse exactly what Validate refuses".
	Must string
}

// ReadActRead reads a partner either side of the call and judges the
// two readings against each other.
//
// Named for the sequence it emits rather than for the classification
// that asked for it. That is the rule the whole set follows: what a
// classification MEANS is carried by the plan's class and claim, and a
// body says only what code comes out — so sideeffect, hooks read
// through a partner, and outbox share one template instead of minting
// three that differ in their prose.
//
// Distinct from [RepeatProbe], which judges the SAME method twice and
// can therefore say nothing about what the call left behind — an
// effect is only observable through something that reads it.
//
// Both readings must succeed before the comparison means anything, so a
// partner that fails fatals rather than reporting "no effect": a
// subject whose observer is broken is a different fault with a
// different fix.
type ReadActRead struct {
	// Call is the method whose effect is claimed.
	Call CallPlan

	// Observe is the partner the directive named, called before and
	// after with its own drawn arguments.
	Observe CallPlan

	// Unchanged says the two readings must MATCH. The zero value says
	// they must differ, which is what an effect means; the other
	// direction states that the call left the observation alone, which
	// is what a refused write claims.
	Unchanged bool

	// Must is the requirement the failure spells after the method's
	// name — "change what Total observes". Worded in claims.go beside
	// the claim it restates, so a claim and the failure reporting it
	// broken cannot drift into saying different things.
	Must string
}

// WriteWriteRead writes twice, then reads the first back and judges it
// against what that write carried.
//
// Named for the sequence rather than the classification, for the reason
// [ReadActRead] is: partition, if-absent and orderafter all emit these
// three calls and differ only in which argument the second write varies
// and in what the failure says. Which arguments vary is the deriving
// rule's decision and is argued where that rule lives.
//
// Three calls rather than two, and the third is the whole point. Two
// writes on their own pass against a subject that ignores the boundary
// entirely — which is why a rule needs a reader named as well as
// whatever it varies, and why one with only half of that refuses.
//
// Both writes fatal. A subject that refused either has a different
// fault from one whose write leaked, and reporting the second for the
// first sends a reader to the wrong method.
type WriteWriteRead struct {
	// First puts the value the read expects back; Second is the same
	// call with whatever the rule varies varied.
	First, Second CallPlan

	// Read is the partner the directive named, called at the FIRST
	// write's arguments. Reading the second would ask whether the
	// second write landed, which is the smoke check.
	Read CallPlan

	// Want is the payload the first write carried, which the read must
	// still answer.
	Want Expr

	// Must is the requirement the failure spells after the method's
	// name — "not let one partition reach another".
	Must string
}

// The strength of each body: what it examines before it passes.
//
// Written beside the variants rather than derived from their fields,
// because the answer is a reading of what the template emits and no
// field carries it. Each one is argued at the variant it belongs to,
// above.
//
// The three error-only answers are the ones worth stating plainly. A
// smoke calls and checks nothing came back wrong; a guarded call hands
// the error channel to a primitive; a repeat probe calls twice and
// judges the second call's error. None of them reads state, and a claim
// beside any of them that promises something about state is promising
// what its body does not check.

// Strength reports that the smoke judges the error channel alone.
func (SmokeSurvives) Strength() suite.Strength { return suite.StrengthErrorOnly }

// Strength reports that the guard judges the error channel alone.
func (GuardedCall) Strength() suite.Strength { return suite.StrengthErrorOnly }

// Strength reports that the repeat probe judges the second call's error
// and reads nothing back.
func (RepeatProbe) Strength() suite.Strength { return suite.StrengthErrorOnly }

// Strength reports that the sentinel check judges the error alone.
func (ReportsSentinel) Strength() suite.Strength { return suite.StrengthErrorOnly }

// Strength reports that the miss-induced zero check reads the returned
// values.
func (ZeroOnMiss) Strength() suite.Strength { return suite.StrengthObserved }

// Strength reports that the cancel-induced zero check reads the returned
// values.
func (ZeroOnCancel) Strength() suite.Strength { return suite.StrengthObserved }

// Strength reports that the zero-answer check reads every value slot.
func (AnswersZero) Strength() suite.Strength { return suite.StrengthObserved }

// Strength reports that the hit probe reads back what was seeded.
func (HitProbe) Strength() suite.Strength { return suite.StrengthObserved }

// Strength reports that the count probe judges the aggregate against
// what was seeded.
func (CountProbe) Strength() suite.Strength { return suite.StrengthObserved }

// Strength reports that the hook check reads the local its callback set.
func (HookFires) Strength() suite.Strength { return suite.StrengthObserved }

// Strength reports that the non-zero check reads the first result.
func (NonZeroAnswer) Strength() suite.Strength { return suite.StrengthObserved }

// Strength reports that the partner check compares two error channels
// and reads no state.
//
// It compares two calls, which looks like more than the single-call
// bodies above, and is not: what it compares is whether each call
// errored. Nothing is read back. It is also one-sided on any value the
// partner accepts — both agreeing proves the subject did not refuse too
// much, and cannot show it refused too little — which is exactly the
// kind of shortfall a claim is liable to paper over.
func (PartnerAgrees) Strength() suite.Strength { return suite.StrengthErrorOnly }

// Strength reports that the read-act-read sequence compares two readings
// taken either side of the call.
func (ReadActRead) Strength() suite.Strength { return suite.StrengthObserved }

// Strength reports that the write-write-read triple reads the first
// write back.
func (WriteWriteRead) Strength() suite.Strength { return suite.StrengthObserved }

// Strength reports that a law leg judges against the engine's laws.
func (LawLeg) Strength() suite.Strength { return suite.StrengthDifferential }

// Strength reports that the differential leg judges against a reference.
func (DifferentialLeg) Strength() suite.Strength { return suite.StrengthDifferential }

// Strength reports that a sim leg judges a crash against an oracle.
func (SimLeg) Strength() suite.Strength { return suite.StrengthDifferential }

// BodyKind names the template that renders the plain survives-smoke.
func (SmokeSurvives) BodyKind() BodyKind { return KindSmokeSurvives }

// BodyKind names the template that renders the guarded call.
func (GuardedCall) BodyKind() BodyKind { return KindGuardedCall }

// BodyKind names the template that renders the miss-induced zero check.
func (ZeroOnMiss) BodyKind() BodyKind { return KindZeroOnMiss }

// BodyKind names the template that renders the cancel-induced zero check.
func (ZeroOnCancel) BodyKind() BodyKind { return KindZeroOnCancel }

// BodyKind names the template that renders the repeat probe.
func (RepeatProbe) BodyKind() BodyKind { return KindRepeatProbe }

// BodyKind names the template that renders the miss probe.
func (ReportsSentinel) BodyKind() BodyKind { return KindReportsSentinel }

// BodyKind names the template that renders the zero-answer probe.
func (AnswersZero) BodyKind() BodyKind { return KindAnswersZero }

// BodyKind names the template that renders the seeded-hit probe.
func (HitProbe) BodyKind() BodyKind { return KindHitProbe }

// BodyKind names the template that renders the seeded-count probe.
func (CountProbe) BodyKind() BodyKind { return KindCountProbe }

// BodyKind names the template that renders the law leg.
func (LawLeg) BodyKind() BodyKind { return KindLawLeg }

// BodyKind names the template that renders the differential leg.
func (DifferentialLeg) BodyKind() BodyKind { return KindDifferentialLeg }

// BodyKind names the template that renders the sim leg.
func (SimLeg) BodyKind() BodyKind { return KindSimLeg }

// BodyKind names the template that renders the hook-fires sequence.
func (HookFires) BodyKind() BodyKind { return KindHookFires }

// BodyKind names the template that renders the non-zero answer check.
func (NonZeroAnswer) BodyKind() BodyKind { return KindNonZeroAnswer }

// BodyKind names the template that renders the partner agreement.
func (PartnerAgrees) BodyKind() BodyKind { return KindPartnerAgrees }

// BodyKind names the template that renders the read-act-read sequence.
func (ReadActRead) BodyKind() BodyKind { return KindReadActRead }

// BodyKind names the template that renders the write-write-read triple.
func (WriteWriteRead) BodyKind() BodyKind { return KindWriteWriteRead }

// BodyKinds enumerates every registered body variant. The template
// census holds this list and the embedded template set equal, so an
// unregistered variant or an orphaned template is a build failure, not
// a render error in a consumer's run.
func BodyKinds() []BodyKind {
	return []BodyKind{
		SmokeSurvives{}.BodyKind(),
		GuardedCall{}.BodyKind(),
		ZeroOnMiss{}.BodyKind(),
		ZeroOnCancel{}.BodyKind(),
		RepeatProbe{}.BodyKind(),
		ReportsSentinel{}.BodyKind(),
		AnswersZero{}.BodyKind(),
		HitProbe{}.BodyKind(),
		CountProbe{}.BodyKind(),
		LawLeg{}.BodyKind(),
		DifferentialLeg{}.BodyKind(),
		SimLeg{}.BodyKind(),
		HookFires{}.BodyKind(),
		NonZeroAnswer{}.BodyKind(),
		PartnerAgrees{}.BodyKind(),
		ReadActRead{}.BodyKind(),
		WriteWriteRead{}.BodyKind(),
	}
}
