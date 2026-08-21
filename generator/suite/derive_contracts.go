// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"

	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// The contract arm of derivation. Two jobs, and they are different
// jobs: the overrides below adjust a signature family where a contract
// changes what a method's own smoke must say, and [Contracts] derives
// the claims a contract licenses in its own right.

// contractRule derives one contract's claims from the method filling
// the role the table keys it under.
type contractRule func(f Iface, m subject.Method, call projection.CallPlan) ([]projection.CheckPlan, []Refusal)

// contractEntry is one tabled contract: the role a rule reads from, and
// the rule.
//
// The role is part of the key because a contract is a protocol rather
// than a property — a rule written against the wrong member calls the
// wrong method and asserts nothing.
type contractEntry struct {
	contract, role string
	rule           contractRule
}

// contractRules is the contracts this tier derives from. One absent
// here owes its coverage to another tier or is recorded in the census
// with the reason.
func contractRules() []contractEntry {
	return []contractEntry{
		{ContractIfAbsent, ContractIfAbsentRole, ifAbsentRule},
		{ContractIfMatch, ContractIfMatchRole, ifMatchRule},
		{ContractOutbox, ContractOutboxRole, outboxRule},
	}
}

// Contracts derives the claims a contract licenses on its own — the
// conditional writes, whose meaning is a relationship between members
// rather than a property of any one of them.
type Contracts struct{}

// Name implements [Deriver].
func (Contracts) Name() DeriverName { return DeriverContracts }

// Derive implements [Deriver].
//
// Silent about a contract with no rule, unlike [Stamps]: the census
// gate holds the contract registry to the union of tabled rules, laws
// and recorded entries, and a refusal per unruled contract would name
// the same gap again in every generated file that carries one.
func (Contracts) Derive(f Iface) ([]projection.CheckPlan, []Refusal) {
	var plans []projection.CheckPlan
	var refusals []Refusal
	for _, m := range f.Methods {
		if !fillsTabledRole(m) {
			continue
		}
		if r, refused := argsRefusal(DeriverContracts, f, m, "'s contract checks"); refused {
			refusals = append(refusals, r)
			continue
		}
		call := callOf(m)
		for _, e := range contractRules() {
			if !m.HasContractRole(e.contract, e.role) {
				continue
			}
			ruled, refused := e.rule(f, m, call)
			plans = append(plans, licensed(ruled, projection.AxisContract, e.contract)...)
			refusals = append(refusals, refused...)
		}
	}
	return plans, refusals
}

// fillsTabledRole reports whether any rule here would run for this
// method, so a draw nothing can supply is named only where it cost
// something.
func fillsTabledRole(m subject.Method) bool {
	for _, e := range contractRules() {
		if m.HasContractRole(e.contract, e.role) {
			return true
		}
	}
	return false
}

// ifAbsentRule derives the conditional write's refusal: the same value
// written twice, and the second reporting the declared conflict.
//
// The conflict sentinel is required and not guessed. Without it
// "refused" cannot be told from "broken" — a subject failing on a nil
// map would pass a check that only asked whether the second write
// errored — which is the fixture's own argument for declaring one.
func ifAbsentRule(f Iface, m subject.Method, call projection.CallPlan) ([]projection.CheckPlan, []Refusal) {
	refuse := func(why, remedy string) ([]projection.CheckPlan, []Refusal) {
		return nil, []Refusal{{
			Deriver: DeriverContracts,
			What:    m.Name + "'s conflict check",
			Why:     why,
			Remedy:  remedy,
		}}
	}
	conflict, declared := m.ContractParam(ContractIfAbsent, ContractIfAbsentConflict)
	if !declared || conflict == "" {
		return refuse(
			"no conflict sentinel names what a second write reports, so a subject failing "+
				"for its own reasons would pass for one that refuses duplicates",
			"name it with //testkit:contract if-absent role=writer conflict=Err…",
		)
	}
	if !m.ReturnsError() {
		return refuse(
			"the claim is that a duplicate is REPORTED, and this method has no error channel",
			"return an error from the writer, which is what the role means",
		)
	}
	return []projection.CheckPlan{{
		ID:    projection.IDPlan{Method: m.Name, Seg: vocab.SegConflict},
		Class: vocab.ClassConflict,
		Claim: ConflictClaim(m, conflict),
		Body: projection.ReportsSentinel{
			Prologue: call,
			Call:     call,
			Sentinel: projection.Expr(conflict),
		},
		Falsifiable: vocab.Proven(),
		Defect: projection.AnswersAnyway{
			Clause: projection.Clause{Text: TakesDuplicateClause(m)},
			Option: projection.OptionName(f.Name, m.Name),
		},
	}}, nil
}

// ifMatchRule derives the conditional write's agreement: the write
// lands exactly when its predicate says it may.
//
// The reasoning [validatesRule] runs on, over a bool instead of an
// error. A suite has no value it knows the predicate rejects — what it
// accepts is the subject's own rule — so what is checkable is that the
// two answers line up. A write landing where its own predicate said no,
// or refusing where it said yes, is the bug either way.
func ifMatchRule(f Iface, m subject.Method, call projection.CallPlan) ([]projection.CheckPlan, []Refusal) {
	refuse := func(why string) ([]projection.CheckPlan, []Refusal) {
		return nil, []Refusal{{
			Deriver: DeriverContracts,
			What:    m.Name + "'s match check",
			Why:     why,
			Remedy:  "name a predicate over the same input with //testkit:contract if-match match=…",
		}}
	}
	matchName := m.ContractPartner(ContractIfMatch, ContractIfMatchMatch)
	if matchName == "" {
		return refuse("the contract names no predicate the write is conditional on")
	}
	match := methodNamed(f, matchName)
	if match == nil {
		return refuse("its predicate " + matchName + " is not a method of this interface")
	}
	if !m.ReturnsError() || !answersBool(*match) {
		return refuse("agreement is between a predicate's yes and the write's success, and " +
			"one of the pair does not answer one")
	}
	if !sharesArgs(m, *match) {
		return refuse("its predicate draws a value the write does not, so the two would " +
			"answer about different inputs")
	}
	return []projection.CheckPlan{{
		ID:    projection.IDPlan{Method: m.Name, Seg: vocab.SegMatch},
		Class: vocab.ClassMatch,
		Claim: MatchClaim(m, match.Name),
		Body: projection.PartnerAgrees{
			Call:        call,
			Partner:     callOf(*match),
			PartnerBool: true,
			Must:        MatchRequirement(match.Name),
		},
		Falsifiable: vocab.Proven(),
		// The permissive double, not the refusing one. A double that
		// refuses everything agrees with a predicate saying no — and the
		// stub's predicate answers its zero, which for a bool IS no. The
		// two would agree by accident and the proof would pass having
		// planted nothing.
		//
		// The mirror of [validatesRule]'s choice, and opposite for the
		// same reason: there the partner answers an error, whose zero is
		// "valid", so the permissive double is the one that agrees.
		Defect: projection.AnswersAnyway{
			Clause: projection.Clause{Text: LandsRegardlessClause(m, match.Name)},
			Option: projection.OptionName(f.Name, m.Name),
		},
	}}, nil
}

// outboxRule derives what the subscriber owes a caller: a channel to
// receive on.
//
// The delivery claim is the model tier's, and this does not attempt it.
// "What was appended arrives" is a claim about a WAIT — a receive that
// blocks for the process's lifetime against a subject that never sends,
// or a bounded one, which is a clock the suite tier does not control.
// `eventually` is where that obligation lives and it binds a law.
//
// What is a caller's: Subscribe answered a channel at all. A receive on
// a nil channel blocks forever, and no timeout the caller writes around
// it helps — which makes a nil answer the one outcome a subscriber may
// not give, and the one this tier can see for free.
//
// The check lands on the subscribe partner rather than on the appending
// method carrying the directive. A contract is a protocol, and which
// member owes which half of it is exactly what the roles say.
func outboxRule(f Iface, m subject.Method, _ projection.CallPlan) ([]projection.CheckPlan, []Refusal) {
	refuse := func(why string) ([]projection.CheckPlan, []Refusal) {
		return nil, []Refusal{{
			Deriver: DeriverContracts,
			What:    m.Name + "'s subscriber check",
			Why:     why,
			Remedy:  "name the subscriber with //testkit:contract outbox role=append subscribe=…",
		}}
	}
	subName := m.ContractPartner(ContractOutbox, ContractOutboxPartner)
	if subName == "" {
		return refuse("the contract names no subscriber, so nothing says where an append arrives")
	}
	sub := methodNamed(f, subName)
	if sub == nil {
		return refuse("its subscriber " + subName + " is not a method of this interface")
	}
	if len(sub.ValueReturns()) == 0 {
		return refuse("its subscriber answers no value, so there is no channel to receive on")
	}
	if arg, field, missing := undeliverableArgs(f.Fixture, sub.ArgFields); missing {
		return refuse("its subscriber's " + arg + " argument needs a value " + field.Reason())
	}
	return []projection.CheckPlan{{
		ID:          projection.IDPlan{Method: sub.Name, Seg: vocab.SegAnswer},
		Class:       vocab.ClassAnswer,
		Claim:       SubscriberClaim(*sub),
		Body:        projection.NonZeroAnswer{Call: callOf(*sub), Must: SubscriberRequirement()},
		Falsifiable: vocab.Proven(),
		Defect: projection.AnswersAnyway{
			Clause: projection.Clause{Text: AnswersNoStreamClause(*sub)},
			Option: projection.OptionName(f.Name, sub.Name),
		},
	}}, nil
}

// answersBool reports whether the method's first result is a bool — the
// yes a predicate gives, as distinct from an error's no.
func answersBool(m subject.Method) bool {
	values := m.ValueReturns()
	return len(values) > 0 && values[0].Source != nil && golang.IsBool(values[0].Source)
}

// openerSmoke overrides the smoke for a producing method: the cursor
// contract's open role answers a handle the smoke must close, because
// the opener owns what it opens and a leaked handle in the suite's own
// smoke would be the harness teaching the leak. The produced type
// itself carries no suite directive, so its absence of smokes needs no
// rule.
//
// A stamped opener without a resolved close partner falls back to the
// plain smoke rather than refusing: the contract schema owns partner
// completeness, and eidos reports that gap at annotation time where
// the author can act on it.
func openerSmoke(f Iface, m subject.Method, call projection.CallPlan) (projection.CheckPlan, bool) {
	if !m.HasContractRole(ContractCursor, ContractCursorOpen) {
		return projection.CheckPlan{}, false
	}
	closeName := m.ContractPartner(ContractCursor, ContractCursorClose)
	if closeName == "" {
		return projection.CheckPlan{}, false
	}
	return projection.CheckPlan{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegSmoke},
		Class:       vocab.ClassSmoke,
		Claim:       OpenerSmokeClaim(m, ContractCursor),
		Body:        projection.SmokeSurvives{Call: call, CloseProduced: closeName},
		Falsifiable: vocab.Proven(),
		Defect: projection.StubPanic{
			Clause: projection.Clause{Text: PanicsClause(m)},
			Option: projection.OptionName(f.Name, m.Name),
		},
	}, true
}

// borrowSmoke overrides the smoke for the pool contract's put role:
// its input is pool-produced, nothing the fixture can derive, so the
// smoke borrows from the get sibling first and returns what it
// borrowed. Answers before the undeliverable-args refusal, because
// the produced draw is the borrow's to supply. Without a get sibling
// or a parameter taking the produced type there is nothing to borrow,
// and the ordinary refusal names the gap instead.
func borrowSmoke(f Iface, m subject.Method) (projection.CheckPlan, bool) {
	if !m.HasContractRole(ContractPool, ContractPoolPut) {
		return projection.CheckPlan{}, false
	}
	producer := roleMethod(f.Methods, ContractPool, ContractPoolGet)
	if producer == nil {
		return projection.CheckPlan{}, false
	}
	call, matched := borrowedCall(m, producedType(*producer))
	if !matched {
		return projection.CheckPlan{}, false
	}
	return projection.CheckPlan{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegSmoke},
		Class:       vocab.ClassSmoke,
		Claim:       BorrowSmokeClaim(m),
		Body:        projection.SmokeSurvives{Call: call, Borrow: callOf(*producer)},
		Falsifiable: vocab.Proven(),
		Defect: projection.StubPanic{
			Clause: projection.Clause{Text: PanicsClause(m)},
			Option: projection.OptionName(f.Name, m.Name),
		},
	}, true
}

// builtSmoke overrides the smoke for a method whose input its own
// sibling mints.
//
// `sample builder=NewInput` says the input space is too large to
// enumerate and names where a member comes from. A literal drawn from
// the type is a member of that space only by luck: a subject accepting
// tokens it issued, handles it opened or ids it assigned rejects
// everything a sampler can invent, and the smoke then reports a
// refusal as though the method were broken.
//
// The same seam [borrowSmoke] uses, reached from the mixin axis rather
// than the pool contract's roles — one shape, because "call the sibling
// that makes one, then pass it" is one sequence however it was
// declared.
//
// Falls through where the builder resolves to nothing or answers a type
// no parameter takes: the ordinary smoke and the ordinary refusal are
// better than a call this cannot spell.
func builtSmoke(f Iface, m subject.Method) (projection.CheckPlan, bool) {
	name, declared := m.MixinParam(MixinSample, MixinSampleParam)
	if !declared || name == "" {
		return projection.CheckPlan{}, false
	}
	builder := methodNamed(f, localName(name))
	if builder == nil {
		return projection.CheckPlan{}, false
	}
	call, matched := borrowedCall(m, producedType(*builder))
	if !matched {
		return projection.CheckPlan{}, false
	}
	return projection.CheckPlan{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegSmoke},
		Class:       vocab.ClassSmoke,
		Claim:       BuiltSmokeClaim(m, builder.Name),
		Body:        projection.SmokeSurvives{Call: call, Borrow: callOf(*builder)},
		Falsifiable: vocab.Proven(),
		Defect: projection.StubPanic{
			Clause: projection.Clause{Text: PanicsClause(m)},
			Option: projection.OptionName(f.Name, m.Name),
		},
	}, true
}

// roleMethod finds the sibling filling a contract role, nil when none
// does.
func roleMethod(methods []subject.Method, contract, role string) *subject.Method {
	for i := range methods {
		if methods[i].HasContractRole(contract, role) {
			return &methods[i]
		}
	}
	return nil
}

// producedType is the producer's non-error answer — what the borrow
// binds and the returning call passes back. Nil when the producer
// answers nothing, which no valid pool schema stamps.
func producedType(producer subject.Method) *node.TypeRef {
	values := producer.ValueReturns()
	if len(values) == 0 {
		return nil
	}
	return values[0].Source
}

// borrowedCall renders the returning method's invocation: the context
// first, the borrowed local where a parameter takes the produced
// type, the fixture draw otherwise. False when no parameter takes it.
func borrowedCall(m subject.Method, produced *node.TypeRef) (projection.CallPlan, bool) {
	var args []projection.Expr
	if m.TakesContext() {
		args = append(args, projection.ExprCtx)
	}
	matched := false
	for i, p := range m.CallArgs() {
		if sameNamed(p.Source, produced) {
			args = append(args, projection.ExprBorrowed)
			matched = true
			continue
		}
		if i >= len(m.ArgFields) {
			// ArgFields is derived per call argument, so the two agree
			// by construction. Where they somehow do not, the borrow
			// arm declines and the plain smoke stands: skipping the
			// argument instead emits a call with fewer arguments than
			// the method takes, which fails in the consumer's build
			// rather than in this run.
			return projection.CallPlan{}, false
		}
		args = append(args, projection.FixtureCall(projection.ExprFixture, m.ArgFields[i]))
	}
	return projection.CallPlan{Method: m.Name, Args: args}, matched
}

// sameNamed reports whether two refs name one declared type. Named
// refs only: the borrow correspondence is by declared type, and a
// composite cannot be the produced handle.
func sameNamed(a, b *node.TypeRef) bool {
	if a == nil || b == nil {
		return false
	}
	return a.TypeKind == node.TypeRefNamed && b.TypeKind == node.TypeRefNamed &&
		a.Name == b.Name && a.Package == b.Package
}
