// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// The mixin axis's rules: one per classification an author attaches by
// hand, each answering what that stamp licenses a caller to check.
//
// Split from the deriver that runs them because the two change for
// different reasons. The loop in derive_stamps.go changes when the
// derivation's SHAPE moves — a new census state, a new refusal policy;
// a rule here changes when one classification's meaning does, which is
// an upstream event and touches nothing else.

package suite

import (
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"

	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// idempotentRule probes the repeat: two clean calls, the second
// changing nothing.
func idempotentRule(f Iface, m subject.Method, call projection.CallPlan) ([]projection.CheckPlan, []Refusal) {
	return []projection.CheckPlan{{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegIdempotent},
		Class:       vocab.ClassIdempotent,
		Claim:       IdempotentClaim(m),
		Body:        projection.RepeatProbe{Call: call},
		Falsifiable: vocab.Proven(),
		Defect: projection.SecondCallErrs{
			Clause: projection.Clause{Text: RepeatFailsClause(m)},
			Option: projection.OptionName(f.Name, m.Name),
		},
	}}, nil
}

// accumulatesRule probes the repeat from the other side: two calls, and
// the second taken rather than refused.
//
// The same body as [idempotentRule] and a different claim, which is
// what the effect axis's two positions come to at this tier — a
// coalescing store and a compounding one diverge in what they LEAVE,
// and nothing here can read that. What they also diverge in, and what
// one subject can settle, is whether the second call is accepted at
// all: a store that deduplicated by refusing is wrong for this mixin
// and right for its sibling.
func accumulatesRule(f Iface, m subject.Method, call projection.CallPlan) ([]projection.CheckPlan, []Refusal) {
	return []projection.CheckPlan{{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegAccumulates},
		Class:       vocab.ClassAccumulates,
		Claim:       AccumulatesClaim(m),
		Body:        projection.RepeatProbe{Call: call},
		Falsifiable: vocab.Proven(),
		Defect:      projection.SecondCallErrs{Option: projection.OptionName(f.Name, m.Name)},
	}}, nil
}

// sideEffectRule reads the partner the directive names either side of
// the call, and requires the two readings to differ.
//
// The first of the named-pair family. `observe=` is what makes the
// mixin checkable at all: without it the classification says only THAT
// there is an effect, which is not a claim a test can drive — an
// effect nothing can observe is indistinguishable from none.
//
// Refuses rather than guessing where the partner is absent or takes
// draws this fixture cannot supply. A rule that fell back to some other
// reader would be asserting a relationship the author did not declare.
func sideEffectRule(f Iface, m subject.Method, call projection.CallPlan) ([]projection.CheckPlan, []Refusal) {
	refuse := func(why string) ([]projection.CheckPlan, []Refusal) {
		return nil, []Refusal{{
			Deriver: DeriverStamps,
			What:    m.Name + "'s side-effect check",
			Why:     why,
			Remedy: "name a readable partner with //testkit:mixin sideeffect observe=…, on a " +
				"line of its own — a directive naming several mixins takes no parameters, " +
				"because the owner of one would be a guess",
		}}
	}
	name, declared := m.MixinParam(MixinSideEffect, MixinSideEffectParam)
	if !declared || name == "" {
		return refuse("the mixin names no partner to observe the effect through")
	}
	observer := methodNamed(f, localName(name))
	if observer == nil {
		return refuse("its observe partner " + localName(name) + " is not a method of this interface")
	}
	if _, _, missing := undeliverableArgs(f.Fixture, observer.ArgFields); missing {
		return refuse("its observe partner draws a value no literal can be written for")
	}
	return []projection.CheckPlan{{
		ID:    projection.IDPlan{Method: m.Name, Seg: vocab.SegSideEffect},
		Class: vocab.ClassSideEffect,
		Claim: SideEffectClaim(m, observer.Name),
		Body: projection.ReadActRead{
			Call:    call,
			Observe: callOf(*observer),
			Must:    SideEffectRequirement(observer.Name),
		},
		Falsifiable: vocab.Proven(),
		Defect: projection.AnswersAnyway{
			Clause: projection.Clause{Text: DropsWriteClause(m)},
			Option: projection.OptionName(f.Name, m.Name),
		},
	}}, nil
}

// partitionRule writes twice varying the axis, then reads the first
// back and requires it unchanged.
//
// The argument split is the classification's own, spelled in its
// docblock: a parameter the reader ALSO takes is identity and is held
// fixed, one only the writer takes is payload and is varied, and the
// axis is varied. Holding the payload would let a subject that
// clobbers across the boundary pass by writing the same value twice;
// varying the identity would write two different keys, which any
// implementation survives.
//
// Both params are optional upstream, so the bare form classifies and
// derives nothing — refused by name rather than guessed at, because a
// rule inventing either half would assert a boundary the author never
// declared.
func partitionRule(f Iface, m subject.Method, call projection.CallPlan) ([]projection.CheckPlan, []Refusal) {
	refuse := func(why string) ([]projection.CheckPlan, []Refusal) {
		return nil, []Refusal{{
			Deriver: DeriverStamps,
			What:    m.Name + "'s isolation check",
			Why:     why,
			Remedy:  "name both halves with //testkit:mixin partition read=… axis=…",
		}}
	}
	axis, hasAxis := m.MixinParam(MixinPartition, MixinPartitionAxis)
	readName, hasRead := m.MixinParam(MixinPartition, MixinPartitionRead)
	if !hasAxis || axis == "" {
		return refuse("no axis names the parameter to vary, so two writes land on two keys " +
			"and pass against a subject that ignores the boundary")
	}
	if !hasRead || readName == "" {
		return refuse("no read partner names what observes the boundary, so nothing reads " +
			"the first partition back")
	}
	reader := methodNamed(f, localName(readName))
	if reader == nil {
		return refuse("its read partner " + localName(readName) + " is not a method of this interface")
	}
	plan, ok := isolationPlan(m, *reader, axis)
	if !ok {
		return refuse("the writer's payload and the reader's answer are different types, " +
			"so what was written cannot be compared with what comes back")
	}
	return []projection.CheckPlan{{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegPartition},
		Class:       vocab.ClassPartition,
		Claim:       PartitionClaim(m, axis),
		Body:        plan,
		Falsifiable: vocab.Proven(),
		Defect: projection.AnswersAnyway{
			Clause: projection.Clause{Text: DropsWriteClause(m)},
			Option: projection.OptionName(f.Name, m.Name),
		},
	}}, nil
}

// isolationPlan builds the three calls, classifying each of the
// writer's draws by whether the reader takes it too.
//
// The axis is matched by PARAMETER name, which is what the upstream
// validator guarantees both halves spell identically — so the two
// calls can be lined up by name rather than by position, and a writer
// whose parameters are ordered differently from its reader still pairs.
func isolationPlan(writer, reader subject.Method, axis string) (projection.WriteWriteRead, bool) {
	shared := make(map[string]bool, len(reader.ArgFields))
	for _, field := range reader.ArgFields {
		shared[field] = true
	}
	axisField := fieldForParam(writer, axis)
	if axisField == "" || !shared[axisField] {
		return projection.WriteWriteRead{}, false
	}

	var payload string
	first := projection.CallPlan{Method: writer.Name}
	second := projection.CallPlan{Method: writer.Name}
	if writer.TakesContext() {
		first.Args = append(first.Args, projection.ExprCtx)
		second.Args = append(second.Args, projection.ExprCtx)
	}
	for _, field := range writer.ArgFields {
		first.Args = append(first.Args, projection.FixtureCall(projection.ExprFixture, field))
		switch {
		case field == axisField, !shared[field]:
			// The axis and the payload both vary: the first so the two
			// writes reach different partitions, the second so the read
			// can tell which one answered.
			second.Args = append(second.Args, projection.FixtureCallOther(projection.ExprFixture, field))
			if field != axisField {
				payload = field
			}
		default:
			second.Args = append(second.Args, projection.FixtureCall(projection.ExprFixture, field))
		}
	}
	if payload == "" || !sameNamed(payloadSource(writer, payload), firstValueSource(reader)) {
		return projection.WriteWriteRead{}, false
	}

	read := projection.CallPlan{Method: reader.Name}
	if reader.TakesContext() {
		read.Args = append(read.Args, projection.ExprCtx)
	}
	for _, field := range reader.ArgFields {
		read.Args = append(read.Args, projection.FixtureCall(projection.ExprFixture, field))
	}
	return projection.WriteWriteRead{
		First:  first,
		Second: second,
		Read:   read,
		Want:   projection.FixtureCall(projection.ExprFixture, payload),
		Must:   PartitionRequirement(axis),
	}, true
}

// fieldForParam maps a parameter name onto the fixture field its draw
// lands in, empty where nothing matched.
//
// By name rather than by position: two parameters of different types
// may share a name across the pair, and the fixture keys on both — so
// the field is what the two calls actually agree on.
func fieldForParam(m subject.Method, param string) string {
	for i, p := range m.CallArgs() {
		if p.Name == param && i < len(m.ArgFields) {
			return m.ArgFields[i]
		}
	}
	return ""
}

// payloadSource is the declared type of the draw landing in one field,
// which the isolation rule compares against the reader's answer.
func payloadSource(m subject.Method, field string) *node.TypeRef {
	for i, f := range m.ArgFields {
		if f == field && i < len(m.CallArgs()) {
			return m.CallArgs()[i].Source
		}
	}
	return nil
}

// hooksRule derives the callback claim: what the registrar installed
// runs when the method does its work.
//
// The registrar is required and not guessed. `register=` names it
// because nothing in a signature says which method takes a callback for
// which other method's benefit — a subject may have several — and a
// rule picking one would install a hook the call was never going to
// run.
//
// The callback's own results are answered by a bare return under named
// results, so a hook that reports something is as installable as one
// that reports nothing.
func hooksRule(f Iface, m subject.Method, call projection.CallPlan) ([]projection.CheckPlan, []Refusal) {
	refuse := func(why string) ([]projection.CheckPlan, []Refusal) {
		return nil, []Refusal{{
			Deriver: DeriverStamps,
			What:    m.Name + "'s hook check",
			Why:     why,
			Remedy:  "name the registrar with //testkit:mixin hooks register=…",
		}}
	}
	name, declared := m.MixinParam(MixinHooks, MixinHooksParam)
	if !declared || name == "" {
		return refuse("the mixin names no registrar, so there is no way to install a hook " +
			"and nothing to observe the call through")
	}
	register := methodNamed(f, localName(name))
	if register == nil {
		return refuse("its registrar " + localName(name) + " is not a method of this interface")
	}
	slot, takesCallback := callbackArg(*register)
	if !takesCallback {
		return refuse("its registrar " + register.Name + " takes no function, so nothing " +
			"can be registered through it")
	}
	if arg, field, missing := undeliverableArgs(f.Fixture, otherArgFields(*register, slot)); missing {
		return refuse("its registrar's " + arg + " argument needs a value " + field.Reason())
	}
	return []projection.CheckPlan{{
		ID:    projection.IDPlan{Method: m.Name, Seg: vocab.SegHooks},
		Class: vocab.ClassHooks,
		Claim: HooksClaim(m, register.Name),
		Body: projection.HookFires{
			Register: hookCall(*register, slot),
			Call:     call,
			Must:     HooksRequirement(register.Name),
		},
		Falsifiable: vocab.Proven(),
		Defect: projection.AnswersAnyway{
			Clause: projection.Clause{Text: SkipsHooksClause(m)},
			Option: projection.OptionName(f.Name, m.Name),
		},
	}}, nil
}

// callbackArg is the registrar's first function-typed parameter, and
// its position among the drawn ones.
func callbackArg(register subject.Method) (int, bool) {
	for i, p := range register.CallArgs() {
		if p.Variadic || p.Source == nil {
			continue
		}
		if p.Source.TypeKind == node.TypeRefFunc {
			return i, true
		}
	}
	return 0, false
}

// callbackParam is the registrar's function-typed parameter, nil where
// it takes none.
func callbackParam(register subject.Method) *node.TypeRef {
	slot, found := callbackArg(register)
	if !found {
		return nil
	}
	return register.CallArgs()[slot].Source
}

// hookCall is [callOf] on the registrar with the callback slot spelled
// as the local the body declares.
func hookCall(register subject.Method, slot int) projection.CallPlan {
	call := callOf(register)
	if register.TakesContext() {
		slot++
	}
	call.Args[slot] = projection.ExprHook
	return call
}

// indexedRule derives the positional read's edge: at the size the
// declared sizer reports, there is no element.
//
// The classification carries two obligations and this is the one a
// caller can discharge. The other — that an index handed to the subject
// is INSIDE the collection — is a supply obligation, not a check: the
// bound is a fact about the seeded subject at run time, and a consumer
// meets it by seeding the config pool the fixture draws from. `by=`
// tells them which method reports it. Deriving a bound-respecting draw
// here would mean every check on the method calling the sizer first,
// which is a different change and one the pool already answers.
//
// The sizer's answer is read rather than invented, which is the whole
// point: an integer derived from its type is a magnitude, and a
// magnitude used as a position is out of range for every collection
// smaller than it — a broken harness rather than a failed claim.
func indexedRule(f Iface, m subject.Method, _ projection.CallPlan) ([]projection.CheckPlan, []Refusal) {
	refuse := func(why string) ([]projection.CheckPlan, []Refusal) {
		return nil, []Refusal{{
			Deriver: DeriverStamps,
			What:    m.Name + "'s bound check",
			Why:     why,
			Remedy:  "name the sizer with //testkit:mixin indexed by=…",
		}}
	}
	sizerName, declared := m.MixinParam(MixinIndexed, MixinIndexedBy)
	if !declared || sizerName == "" {
		return refuse("no sizer names what bounds the positions, so nothing says where the " +
			"end is and any index this run chose would be a guess")
	}
	sizer := methodNamed(f, localName(sizerName))
	if sizer == nil {
		return refuse("its sizer " + localName(sizerName) + " is not a method of this interface")
	}
	if !answersInteger(*sizer) {
		return refuse("its sizer answers no integer, so there is no size to ask past")
	}
	slot, positional := integerArg(m)
	if !positional {
		return refuse("it takes no integer argument, so there is no position to bound")
	}
	if len(m.ValueReturns()) == 0 {
		return refuse("it answers no value, so there is nothing to hold to the zero past the end")
	}
	return []projection.CheckPlan{{
		ID:    projection.IDPlan{Method: m.Name, Seg: vocab.SegBound},
		Class: vocab.ClassBound,
		Claim: BoundClaim(m, sizer.Name),
		Body: projection.AnswersZero{
			Call:    boundedCall(m, slot),
			Bound:   callOf(*sizer),
			Because: BecausePastTheEnd(sizer.Name),
		},
		Falsifiable: vocab.Proven(),
		Defect: projection.AnswersWithValue{
			Clause: projection.Clause{Text: AnswersPastTheEndClause(m)},
			Option: projection.OptionName(f.Name, m.Name),
		},
	}}, nil
}

// answersInteger reports whether the method's first result is an
// integer — a count, as distinct from whatever else a sibling answers.
func answersInteger(m subject.Method) bool {
	values := m.ValueReturns()
	return len(values) > 0 && values[0].Source != nil && golang.IsInteger(values[0].Source)
}

// integerArg is the first drawn argument that could hold a position.
func integerArg(m subject.Method) (int, bool) {
	for i, p := range m.CallArgs() {
		if p.Variadic || p.Source == nil {
			continue
		}
		if golang.IsInteger(p.Source) {
			return i, true
		}
	}
	return 0, false
}

// boundedCall is [callOf] with the positional argument spelled as the
// sizer's answer rather than a drawn value.
func boundedCall(m subject.Method, slot int) projection.CallPlan {
	call := callOf(m)
	if m.TakesContext() {
		slot++
	}
	call.Args[slot] = projection.ExprBound
	return call
}

// nilSafeRule derives the nil-argument guard: the call with nil in one
// slot returns an error rather than panicking.
//
// The mixin takes no parameter and says only that nil inputs are
// tolerated, so WHICH slot to nil is read off the signature. A method
// whose every argument is a value type has no nil to be handed, which
// makes the claim unstateable rather than false — refused by name, so
// the header says so rather than the coverage list going quietly short.
func nilSafeRule(f Iface, m subject.Method, _ projection.CallPlan) ([]projection.CheckPlan, []Refusal) {
	refuse := func(why string) ([]projection.CheckPlan, []Refusal) {
		return nil, []Refusal{{
			Deriver: DeriverStamps,
			What:    m.Name + "'s nil-argument check",
			Why:     why,
			Remedy:  "take one argument that can be nil — a pointer, a slice, a map or an interface",
		}}
	}
	if !m.ReturnsError() {
		return refuse("the claim is that a nil argument is REPORTED, and this method " +
			"has no error channel to report it through")
	}
	slot, arg, nilable := nilableArg(m)
	if !nilable {
		return refuse("no argument can hold nil, so there is no nil to hand it")
	}
	if other, field, missing := undeliverableArgs(f.Fixture, otherArgFields(m, slot)); missing {
		// This rule supplies the nil slot and nothing else, so a draw
		// missing elsewhere still leaves the call unspellable.
		return refuse("its " + other + " argument needs a value " + field.Reason())
	}
	return []projection.CheckPlan{{
		ID:    projection.IDPlan{Method: m.Name, Seg: vocab.SegNilArgument},
		Class: vocab.ClassNilArgument,
		Claim: NilArgumentClaim(m, arg),
		Body: projection.GuardedCall{
			Call:  nilArgumentCall(m, slot),
			Guard: projection.GuardNilArgument,
		},
		Falsifiable: vocab.Proven(),
		Defect: projection.AnswersAnyway{
			Clause: projection.Clause{Text: ForgivesNilArgumentClause(m)},
			Option: projection.OptionName(f.Name, m.Name),
		},
	}}, nil
}

// nilableArg is the first argument a nil can be handed to, and its
// position among the drawn ones.
//
// The variadic parameter is skipped. `f(nil)` on a `...T` passes a nil
// ELEMENT where T is nilable and an empty list where it is not, so a
// check built on it would mean different things for different
// signatures — and one that means two things proves neither.
func nilableArg(m subject.Method) (int, golang.Param, bool) {
	for i, p := range m.CallArgs() {
		if p.Variadic || p.Source == nil {
			continue
		}
		if golang.Nilable(p.Source) {
			return i, p, true
		}
	}
	return 0, golang.Param{}, false
}

// nilArgumentCall is [callOf] with the drawn argument at slot spelled
// nil, every other draw left as it was.
//
// One nil rather than all of them: the claim is that a nil is handled,
// and a call with everything nil cannot say which slot the subject
// tripped over.
func nilArgumentCall(m subject.Method, slot int) projection.CallPlan {
	call := callOf(m)
	if m.TakesContext() {
		slot++
	}
	call.Args[slot] = projection.ExprNil
	return call
}

// otherArgFields is the method's fixture fields with the nil'd slot
// left out — the draws a nil-argument body still has to make.
func otherArgFields(m subject.Method, slot int) []string {
	out := make([]string, 0, len(m.ArgFields))
	for i, field := range m.ArgFields {
		if i == slot {
			continue
		}
		out = append(out, field)
	}
	return out
}

// orderAfterRule derives the ordering refusal: called before its
// predecessor, the method reports the declared unready sentinel.
//
// Both parameters are needed and neither is guessed. Without the
// predecessor nothing says what the call is early FOR; without the
// sentinel "refused" cannot be told from "broken", and an
// implementation failing on a nil map would pass for one that honours
// the ordering.
//
// The predecessor is resolved but not called: the check runs against a
// subject the harness has just built, and what makes the call early is
// that nothing has run yet. Naming it is how a misspelling fails at the
// directive rather than in a consumer's build.
func orderAfterRule(f Iface, m subject.Method, call projection.CallPlan) ([]projection.CheckPlan, []Refusal) {
	refuse := func(why string) ([]projection.CheckPlan, []Refusal) {
		return nil, []Refusal{{
			Deriver: DeriverStamps,
			What:    m.Name + "'s ordering check",
			Why:     why,
			Remedy:  "name both halves with //testkit:mixin orderafter fn=… unready=…",
		}}
	}
	fnName, hasFn := m.MixinParam(MixinOrderAfter, MixinOrderAfterParam)
	if !hasFn || fnName == "" {
		return refuse("no predecessor names what has to have run, so nothing says " +
			"what the call would be early for")
	}
	predecessor := methodNamed(f, localName(fnName))
	if predecessor == nil {
		return refuse("its predecessor " + localName(fnName) + " is not a method of this interface")
	}
	unready, hasUnready := m.MixinParam(MixinOrderAfter, MixinOrderAfterUnready)
	if !hasUnready || unready == "" {
		return refuse("no unready sentinel names what being early reports, so an " +
			"implementation failing for its own reasons would pass for one that waits")
	}
	if !m.ReturnsError() {
		return refuse("the claim is that being early is REPORTED, and this method " +
			"has no error channel to report it through")
	}
	return []projection.CheckPlan{{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegOrderAfter},
		Class:       vocab.ClassOrderAfter,
		Claim:       OrderAfterClaim(m, predecessor.Name, unready),
		Body:        projection.ReportsSentinel{Call: call, Sentinel: projection.Expr(unready)},
		Falsifiable: vocab.Proven(),
		Defect: projection.AnswersAnyway{
			Clause: projection.Clause{Text: AnswersEarlyClause(m)},
			Option: projection.OptionName(f.Name, m.Name),
		},
	}}, nil
}

// validatesRule derives the agreement: the method and its named
// validator, handed the same drawn value, reach the same verdict.
//
// The suite tier cannot state the classification whole. "Invalid input
// is screened" needs an invalid value, and what counts as invalid is
// the subject's own rule — a fixture inventing one would be guessing at
// the thing under test, and would fail a correct subject the day its
// rule changed. Agreement needs no such guess and catches the same bug
// from either side: a value the validator refuses and the method takes,
// or one the validator passes and the method refuses.
//
// The validator has to take an argument the subject takes too. Two
// methods drawing different fixture fields would reach verdicts about
// different values, and a disagreement between those says nothing.
func validatesRule(f Iface, m subject.Method, call projection.CallPlan) ([]projection.CheckPlan, []Refusal) {
	refuse := func(why string) ([]projection.CheckPlan, []Refusal) {
		return nil, []Refusal{{
			Deriver: DeriverStamps,
			What:    m.Name + "'s validation check",
			Why:     why,
			Remedy:  "name a validator over the same input with //testkit:mixin validates fn=…",
		}}
	}
	fnName, declared := m.MixinParam(MixinValidates, MixinValidatesParam)
	if !declared || fnName == "" {
		return refuse("the mixin names no validator to agree with")
	}
	validator := methodNamed(f, localName(fnName))
	if validator == nil {
		return refuse("its validator " + localName(fnName) + " is not a method of this interface")
	}
	if !validator.ReturnsError() || !m.ReturnsError() {
		return refuse("agreement is between two verdicts, and one of the pair has no " +
			"error channel to give one through")
	}
	if !sharesArgs(m, *validator) {
		return refuse("its validator draws a value the method does not, so the two would " +
			"reach verdicts about different inputs")
	}
	return []projection.CheckPlan{{
		ID:    projection.IDPlan{Method: m.Name, Seg: vocab.SegValidates},
		Class: vocab.ClassValidates,
		Claim: ValidatesClaim(m, validator.Name),
		Body: projection.PartnerAgrees{
			Call:    call,
			Partner: callOf(*validator),
			Must:    ValidatesRequirement(validator.Name),
		},
		Falsifiable: vocab.Proven(),
		Defect: projection.RefusesAlways{
			Clause: projection.Clause{Text: RefusesEverythingClause(m)},
			Option: projection.OptionName(f.Name, m.Name),
		},
	}}, nil
}
