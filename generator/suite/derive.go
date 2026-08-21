// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"slices"

	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// DeriverName identifies a deriver in the registry and in refusal
// attributions.
type DeriverName string

// The deriver names, one per rule family.
const (
	// DeriverSignature derives the per-method families: smoke, cancel,
	// deadline, nilcontext, zero-on-error.
	DeriverSignature DeriverName = "signature"

	// DeriverStamps derives the deterministic stamp families: the
	// mixin axis (idempotent) and the detector axis (the reader
	// miss/hit/count set).
	DeriverStamps DeriverName = "stamps"

	// DeriverContracts derives the claims a contract stamp licenses that
	// no mixin or detector does. Its own axis rather than a table inside
	// [DeriverStamps] because a contract is a protocol over several
	// methods and a rule keys on a ROLE, where every stamps rule keys on
	// the method it is written on.
	DeriverContracts DeriverName = "contracts"

	// DeriverLaws plans the model tier's law rows — which laws tiers
	// selects, on which legs, under which claims.
	DeriverLaws DeriverName = "laws"

	// DeriverDifferential plans the model tier's reference-comparison
	// row, worded by the derived reference's kind.
	DeriverDifferential DeriverName = "differential"
)

// Deriver is one rule family: the interface's projections in, plans
// and refusals out. A deriver returns every check its rules license
// and a refusal for every check its rules reach but cannot complete —
// the two lists together are the family's whole answer, and silence
// is not in the vocabulary.
type Deriver interface {
	Name() DeriverName
	Derive(f Iface) ([]projection.CheckPlan, []Refusal)
}

// Refusal is a check the rules reached and could not derive: what it
// would have asserted, why it cannot, and the consumer action that
// closes the gap. Refusals render into the generated header — a claim
// the reader cannot see refused reads as a claim the run checks.
type Refusal struct {
	Deriver DeriverName
	What    string
	Why     string
	Remedy  string

	// Elsewhere marks a classification this file does not check because
	// another part of testkit owns it, rather than because deriving it
	// failed.
	//
	// Not a gap in the sense the others are, and it is listed for the
	// opposite reason: the consumer stamped the directive and would
	// otherwise read a file that never mentions it again. Whether the tier
	// that owns it ran is not this generator's to know — what it can say,
	// and what stays true either way, is that the claim is not made here.
	Elsewhere bool

	// Obligation names WHICH claim of the classification this refusal is
	// about, where the classification carries more than one.
	//
	// Empty for a refusal about the whole of one. Set where a rule here
	// covered part and another tier holds the rest, which is the case
	// docs/adr/0028 exists for: the census reads it to tell "nothing
	// checks this" from "this half is checked and that half is over
	// there", and those are different findings.
	Obligation tiers.Obligation

	// Unaccounted marks the one refusal that is not an argument: the
	// deriver reached a classification with no rule, no law binding and
	// no entry in [Accounting], and said so.
	//
	// The distinction the census turns on. A derivation refusal states a
	// fact about THIS interface — nothing here writes, no sentinel is
	// declared — and naming it closes the question honestly. This one
	// states that nobody has decided what the classification owes, which
	// is the gap the census exists to report; counting it as an argument
	// would let a vocabulary grow uncovered as long as it grew loudly.
	Unaccounted bool

	// Licensed names the classification whose check was refused, empty
	// for a refusal the shape alone reached.
	//
	// The field [projection.CheckPlan.Licensed] is, and it exists for
	// the same census. A classification whose rule always refuses emits
	// no check, so a census reading emitted checks alone cannot tell it
	// from one no rule ever reached — and those are opposite findings.
	// The first is a gap the generator already argued in the header; the
	// second is a gap nobody has looked at.
	Licensed projection.Licence
}

// Registry returns the derivers in derivation order. Closed like the
// projection's variant sets: the conformance census holds this list
// and the rule tables equal, so a family added to derivation-rules.md
// without a deriver is a build failure.
func Registry() []Deriver {
	return []Deriver{
		Signature{},
		Stamps{},
		Contracts{},
	}
}

// argsRefusal folds a method whose draws the fixture cannot supply
// into the one refusal its whole derived family set shares, false
// when every draw has a supplier.
func argsRefusal(d DeriverName, f Iface, m subject.Method, what string) (Refusal, bool) {
	arg, field, missing := undeliverableArgs(f.Fixture, m.ArgFields)
	if !missing {
		return Refusal{}, false
	}
	return Refusal{
		Deriver: d,
		What:    m.Name + what,
		Why:     "its " + arg + " argument needs a value " + field.Reason(),
		Remedy: "stamp the type with //testkit:role and //testkit:default so " +
			projection.ConfigName(f.Name) + " can supply one, or write the check " +
			"yourself as a " + projection.RowsName(f.Name) + " entry",
	}, true
}

// callOf renders the method's invocation: the context first when the
// method takes one, then every draw through the fixture policy.
func callOf(m subject.Method) projection.CallPlan {
	var args []projection.Expr
	if m.TakesContext() {
		args = append(args, projection.ExprCtx)
	}
	for _, field := range m.ArgFields {
		args = append(args, projection.FixtureCall(projection.ExprFixture, field))
	}
	return projection.CallPlan{Method: m.Name, Args: args}
}

// InventoryOf runs every deriver over the interface and folds their
// answers into the one inventory every artifact projects from.
//
// The registry's order is the emission order, and it is preserved: the
// generated file's sections read in the order the rules are written
// down, so a reader following derivation-rules.md top to bottom is
// following the output too.
//
// Refusals accumulate beside the plans rather than short-circuiting.
// A deriver refusing one method's family says nothing about the next
// deriver's answer, and a run that stopped at the first refusal would
// report one gap where the header owes the reader all of them.
func InventoryOf(f Iface) (projection.Inventory, []Refusal) {
	inv := projection.Inventory{Iface: f.Name, Token: f.Token}
	var refusals []Refusal
	for _, d := range Registry() {
		plans, refused := d.Derive(f)
		inv.Checks = append(inv.Checks, plans...)
		refusals = append(refusals, refused...)
	}
	return inv, refusals
}

// derivedSeeded reports the seed-seam interface: nothing on it can
// write, so a run cannot populate it through the surface under test and
// the harness has to receive the corpus instead.
//
// The exported half of [Iface.seeded], taken over a method slice because
// the shell asks before it has built an Iface.
func derivedSeeded(methods []subject.Method) bool {
	return !slices.ContainsFunc(methods, writesSomething)
}

// declaredLimit reports the bound any method declares, empty where none
// does.
//
// The first one found. An interface stamping two different bounds has
// declared two ceilings for one subject, which is a contradiction its
// author has to settle — and taking either silently would build every
// harness at a capacity half the declarations disagree with.
func declaredLimit(methods []subject.Method) string {
	for _, m := range methods {
		if v, declared := m.MixinParam(MixinBounded, MixinBoundedLimit); declared && v != "" {
			return v
		}
	}
	return ""
}

// methodNamed finds a sibling by name, nil where the interface declares
// none — which is the author naming a partner that is not there, and a
// refusal rather than a compile error in a consumer's file.
func methodNamed(f Iface, name string) *subject.Method {
	for i := range f.Methods {
		if f.Methods[i].Name == name {
			return &f.Methods[i]
		}
	}
	return nil
}

// sharesArgs reports whether every value the partner draws is one the
// subject draws too.
//
// By fixture field, which is what the two calls actually agree on: the
// same field yields the same value in both, so a partner drawing only
// the subject's fields is asking about the subject's input.
func sharesArgs(m, partner subject.Method) bool {
	mine := make(map[string]bool, len(m.ArgFields))
	for _, field := range m.ArgFields {
		mine[field] = true
	}
	if len(partner.ArgFields) == 0 {
		return false
	}
	for _, field := range partner.ArgFields {
		if !mine[field] {
			return false
		}
	}
	return true
}

// Iface is one interface as the derivers read it: the projections the
// plugin already computes — [subject.Method], [subject.Fixture] — plus the directive
// facts no projection carries. It exists so derivation is
// unit-testable without the eidos pipeline, and it invents nothing:
// every field is either an incumbent projection or a directive the
// plugin shell reads once.
type Iface struct {
	// Name is the interface's exported name ("Log"); Token its Go
	// identifier qualifier ("log", "paginatedReader"), which every
	// emitted declaration is named from.
	Name  string
	Token string

	// Package is the interface's own import path, which a body naming
	// something declared beside it — a miss sentinel — resolves
	// against.
	Package string

	// Qualifier is the interface's word inside a family-scoped ID
	// ("log", "paginated-reader"). Slug rather than identifier: the
	// grammar admits a-z, 0-9 and '-' only, so the two qualifiers
	// diverge the moment an interface name has two words.
	Qualifier string

	Methods []subject.Method

	// Fixture is the derived input set; a draw it cannot deliver turns
	// a method's derived families into one refusal.
	Fixture subject.Fixture

	// Corpus reports that this run seeds every subject from a derived
	// corpus, which changes what a miss draws.
	//
	// The fixture's alternate is a second SEEDED key once a corpus
	// exists — the zip takes every member of the key pool — so a miss
	// body drawing it hits, and passes while asserting the opposite of
	// its claim. Where this holds, the miss draws the key deliberately
	// left out instead.
	Corpus bool
}

// seeded reports the seed-seam interface: nothing on it can write, so
// the harness receives its corpus from the pools and the derived
// claims speak "seeded" rather than "derived".
func (f Iface) seeded() bool {
	return !slices.ContainsFunc(f.Methods, writesSomething)
}

// supplies reports that something on this run can put an input where a
// read will find it — a writer to call, or a corpus the harness is
// handed. It is what makes "an input nothing supplied" name a real case
// rather than every input the method takes.
//
// Distinct from seeded, which asks the opposite question and answers
// yes for an interface with no state at all: a codec writes nothing,
// so it is seeded() and supplies() nothing.
func (f Iface) supplies() bool {
	return f.Corpus || slices.ContainsFunc(f.Methods, writesSomething)
}
