// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import "testing"

// Row is one check a consumer wrote, in the form every generated
// authoring struct reduces to.
//
// The generated struct keeps its own flat fields, because a consumer
// writes a literal and an embedded type would make them name it. What
// lives here is the meaning of each field, once, so a wording fix is one
// edit rather than a regeneration of every package. The fields a
// generated struct adds beyond these are the ones that name its own
// interface's types, and are documented where they are declared.
type Row[S, F any] struct {
	// Method groups this check under one of the interface's methods, so it
	// is named <Method>/<Name> in output and reruns with the rest of that
	// method's checks. Empty for a check that is not about one method in
	// particular.
	Method string

	// Name identifies this check, in output and when dropping it. It wants
	// to be stable: it is what a colleague's Without call refers to.
	Name string

	// Claim is one sentence saying exactly what this check proves. It is
	// recorded in the manifest and read out in the report.
	//
	// Only what the body actually establishes. "a second write leaves the
	// value unchanged" promises that something reads the value back; a
	// body that checks only the write's error should say that instead.
	// Nothing here can catch a claim wider than its check, and a claim
	// nobody verifies is worse than none.
	Claim string

	// Set exactly one of these.
	//
	// Run is handed one fresh instance, which suits nearly every check.
	// RunWith is handed something that can build instances, for a claim
	// one instance cannot state on its own: build two and compare them,
	// move the clock forward, put one into a failure state.
	//
	// Both are handed the run's sample inputs as well, so a check written
	// by hand draws from the values the generated ones draw from — and
	// sees an override to those values the same way they do.
	//
	// Every input in them comes as a pair: a value, and a second that
	// differs from it. Both are needed for a check to mean anything —
	// looking up a key that was just stored proves nothing on its own
	// unless there is also a key that was never stored. A parameter whose
	// type has no value that can be written down is left at its zero, the
	// checks that needed it were not emitted rather than run against
	// something meaningless, and the accessor for it says so where you
	// read it.
	Run     func(tb testing.TB, s S, fx F)
	RunWith func(tb testing.TB, sub Subject[S], fx F)

	// Class groups this check in the report's summary. Optional; a check
	// written by hand is grouped as hand-written by default.
	Class Class

	// Needs says what this check requires of an implementation beyond
	// being constructed — a clock it can move, a failure it can induce.
	//
	// An implementation that cannot supply it fails this check by name,
	// with the field to fill in named in the message. It never skips: a
	// check that quietly skipped for want of wiring would look exactly
	// like one that passed.
	Needs Caps

	// Proven reports that the row's ProvenBy was set: an implementation
	// built to break this check and nothing else. Setting it makes two
	// statements at once — that the check can fail, and here is the proof
	// — and the run's Prove entry point fails unless the broken
	// implementation really does turn this check red.
	//
	// The fact rather than the value, because there is no one type here
	// that could hold it. A generated defect is lowered by its own
	// Subject method, and what that method takes varies with the
	// interface: a reader-only one is handed the corpus the run seeded it
	// from. Binding needs to know whether a proof was offered, and the
	// lowering stays where the types are.
	//
	// ProvenReason, if set, is text the failure must contain. Use it so a
	// broken implementation that fails for some unrelated reason stops
	// counting as evidence.
	//
	// Argued is for when no such implementation can be built: record why,
	// in a sentence. Set at most one of Proven and Argued. Setting
	// neither is allowed and the report says so — the check is unproven,
	// which is honest, and different from proven.
	Proven       bool
	ProvenReason string
	Argued       string
}

// Binding is a row part-way to a [Check], holding what the last step needs
// to refuse a row that set two bodies or none.
//
// A builder rather than one call, because the bodies a row may set are not
// all known here: a generated struct adds its own — a property whose
// inputs are drawn, one scoped to a method that names its argument's type
// — and those close over types this package cannot see. The counting and
// the refusal are the same wherever a body came from, so they live here
// and the bodies are handed in.
type Binding[S any] struct {
	check   Check[S]
	row     string
	fields  string
	methods NameSet
	bodies  int
	scoped  bool
	proven  bool
	reason  string
	argued  string
	err     error
}

// BindRow starts a binding from a row's invariant half: the two bodies
// every generated struct declares, and the reporting fields around them.
func BindRow[S, F any](r Row[S, F], fx F, methods NameSet) *Binding[S] {
	b := &Binding[S]{
		check:   Check[S]{Claim: r.Claim, Class: r.Class, Needs: r.Needs},
		row:     r.Name,
		fields:  "Run, RunWith",
		methods: methods,
		proven:  r.Proven,
		reason:  r.ProvenReason,
		argued:  r.Argued,
	}
	if b.check.Class == "" {
		b.check.Class = ClassHandWritten
	}
	if run := r.Run; run != nil {
		b.Scoped("Run", r.Method, func(tb testing.TB, s S) { run(tb, s, fx) })
	}
	if with := r.RunWith; with != nil {
		b.Fixed(HandRowID(r.Name), func(tb testing.TB, sub Subject[S]) { with(tb, sub, fx) })
	}
	return b
}

// Scoped records a body handed one instance and reading the row's Method,
// naming the check <Method>/<Name>. field is the struct field it came
// from, for the refusal a row setting two bodies gets.
func (b *Binding[S]) Scoped(field, method string, run func(testing.TB, S)) *Binding[S] {
	b.scope(field, method)
	b.check.Run, b.check.RunWith = run, nil
	return b
}

// ScopedWith is [Binding.Scoped] for a body handed the subject rather than
// one instance.
func (b *Binding[S]) ScopedWith(
	field, method string, run func(testing.TB, Subject[S]),
) *Binding[S] {
	b.scope(field, method)
	b.check.Run, b.check.RunWith = nil, run
	return b
}

// Fixed records a body that names its own check, for one whose scope is
// settled by the field it was written in rather than by Method.
func (b *Binding[S]) Fixed(id ID, run func(testing.TB, Subject[S])) *Binding[S] {
	b.bodies++
	b.check.ID = id
	b.check.Run, b.check.RunWith = nil, run
	return b
}

// scope counts a Method-reading body and resolves its identity.
func (b *Binding[S]) scope(field, method string) {
	b.scoped = true
	b.bodies++
	id, err := RowID(field, method, b.row, b.methods)
	if err != nil && b.err == nil {
		b.err = err
	}
	b.check.ID = id
}

// Offers widens the field list a two-bodies refusal names, for the fields
// a generated struct adds beyond the two above.
func (b *Binding[S]) Offers(fields string) *Binding[S] {
	b.fields += ", " + fields
	return b
}

// Seal finishes the check, or reports what is wrong with the row: no body,
// two bodies, a Method the interface does not declare, a Method on a body
// that fixes its own scope, or a proof claimed both ways.
func (b *Binding[S]) Seal(method string) (Check[S], error) {
	if b.err != nil {
		return b.check, b.err
	}
	if err := OneBody(b.row, b.bodies, b.fields); err != nil {
		return b.check, err
	}
	if err := methodScoped(b.row, method, b.scoped); err != nil {
		return b.check, err
	}
	falsifiable, err := Falsify(b.row, b.proven, b.argued)
	if err != nil {
		return b.check, err
	}
	b.check.Falsifiable = falsifiable
	return b.check, nil
}
