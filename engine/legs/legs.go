// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package legs holds the engine-facing leg idioms every generated
// conformance package shares: the oracle-or-derived reference pick, the
// law leg, the differential leg, the provenance-gated adversarial
// blend, and the vacuity note.
//
// It exists because of a doctrine and a duplication meeting head-on.
// The doctrine: package suite imports testing and clock and NOTHING
// else, so a consumer's non-test code composes with the vocabulary
// without pulling the model tier's dependency graph. The duplication:
// with suite unable to see the engine, every generated package
// re-emitted the same six idioms as private functions, and five copies
// of a semantics-bearing idiom drift where one cannot. This package is
// the sanctioned bridge: generated files import suite AND legs, legs
// imports the engine, and suite still does not.
package legs

import (
	"testing"

	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/model/crash"
	"go.thesmos.sh/testkit/engine/model/law"
	"go.thesmos.sh/testkit/engine/suite"
)

// CompatV1 is the compatibility witness for this package's leg
// contract, referenced once per generated model file:
//
//	var _ = legs.CompatV1
//
// A library the generated files ride can drift from the files a plugin
// version emitted — the same skew CompatV2 guards on suite. A breaking
// change here renames the witness, and every package generated against
// v1 stops compiling with the skew named.
func CompatV1() {}

// Reference picks the reference a model leg compares against and names
// the tier for the report: the run's declared oracle when one exists,
// the given derived reference otherwise. The twin is not reachable from
// here at all, which is the point — a second copy of the subject agrees
// with it about every bug they share, so the derived reference replaced
// the twin fallback for generated legs.
func Reference[S any](tb testing.TB, sub suite.Subject[S], derived func() S) (func() S, suite.Tier) {
	tb.Helper()
	if build, differential := sub.Reference(); differential {
		return func() S { return build(tb) }, suite.TierDifferential
	}
	return derived, suite.TierDerived
}

// NoteVacuity is the one home of the vacuous-outcome idiom: a law
// behind a precondition the run never met passed nothing, and the leg
// says so instead of joining the passes. The engine counts engagement
// in its census; this hands the count to the leg note the report reads.
func NoteVacuity[S any](tb testing.TB, sub suite.Subject[S], out model.Outcome) {
	tb.Helper()
	if out.Engaged() {
		// Engaged means AT LEAST one law reached a verdict — a bundle
		// where one law fires and four never engage is engaged, and
		// counting it a plain pass masks the four. The leg stays green
		// (something was asserted); the report carries which laws were
		// not.
		if names := out.Unengaged(); len(names) > 0 {
			sub.NoteUnengaged(names)
			tb.Logf("laws that never engaged on any draw: %v", names)
		}
		return
	}
	sub.Note(suite.ReasonVacuous)
	// Unengaged is NOT guaranteed non-empty here: a law whose Check was
	// never called has Ran == 0 and appears in neither census list —
	// the engine's Outcome doc owns that contract. Naming an empty list
	// would say "[]" and teach nothing.
	if names := out.Unengaged(); len(names) > 0 {
		tb.Logf("no law engaged on any draw: %v", names)
	} else {
		tb.Logf("no law reached a single check: the drawn sequences never invoked one")
	}
}

// Law is the one body every law leg shares: the given laws as the run's
// only oracle over the leg's action stream, vacuity reported through
// the leg note. sut is explicit rather than derived from the subject
// because a clocked leg builds its instances on a per-iteration clock;
// extra carries leg-specific options — a history reset, say — without a
// second leg shape.
//
// LawsOnly is the A1 split: with the differential armed, it would catch
// every defect first and make "can this law fail" unanswerable.
func Law[S any](
	tb testing.TB, sub suite.Subject[S], sut, ref func() S,
	actions []model.Action[S], laws []law.Law[S], extra ...model.Option[S],
) {
	tb.Helper()
	opts := make([]model.Option[S], 0, 3+len(laws)+len(extra))
	opts = append(opts,
		model.WithReference(ref),
		model.WithActions(actions...),
		model.WithLawsOnly[S](true),
	)
	for _, l := range laws {
		opts = append(opts, model.WithLaw(l))
	}
	opts = append(opts, extra...)
	NoteVacuity(tb, sub, model.Assert(tb, sut, opts...))
}

// Differential is the differential leg: random action sequences against
// the subject and the reference [Reference] picks, the tier noted for
// the report, extra options appended for legs that carry more — a
// history reset, say.
func Differential[S any](
	tb testing.TB, sub suite.Subject[S], derived func() S,
	actions []model.Action[S], extra ...model.Option[S],
) {
	tb.Helper()
	buildRef, tier := Reference(tb, sub, derived)
	sub.NoteTier(tier)
	opts := make([]model.Option[S], 0, 2+len(extra))
	opts = append(opts,
		model.WithReference(buildRef),
		model.WithActions(actions...),
	)
	opts = append(opts, extra...)
	model.Assert(tb, func() S { return sub.New(tb) }, opts...)
}

// Concurrent is the linearizability leg: workers driving one instance at
// once, the recorded history checked against the model this interface's
// shape selects.
//
// The reference [Reference] picks, and the tier noted for the report, as
// every other leg does. It is not what decides the claim — a
// linearizability verdict comes from the model searching the history for
// a serial order — but the run still builds a pair, and a leg that
// skipped the pick would report a tier nobody chose.
//
// The engine owns the interleaving, the property loop and the artifact a
// failing history is written to. This supplies the verbs and the model.
func Concurrent[S any](
	tb testing.TB, sub suite.Subject[S], derived func() S,
	cfg model.ConcurrentConfig[S], laws ...law.Law[S],
) {
	tb.Helper()
	buildRef, tier := Reference(tb, sub, derived)
	sub.NoteTier(tier)
	opts := make([]model.Option[S], 0, 2+len(laws))
	opts = append(opts,
		model.WithReference(buildRef),
		model.WithConcurrent(cfg),
	)
	// The laws a STEPLESS family's verdict comes from. A model that steps
	// nothing partitions nothing and decides nothing, and the runner
	// refuses that pair outright rather than passing a run that asserted
	// nothing — which is the check this argument exists to satisfy.
	for _, l := range laws {
		opts = append(opts, model.WithLaw(l))
	}
	model.Assert(tb, func() S { return sub.New(tb) }, opts...)
}

// Recover is the crash-recovery leg: a drawn schedule of writes, reads
// and crash points, every read judged against what the writes
// acknowledged.
//
// No reference is picked and none is wanted. Every other leg compares the
// subject against something that models it; this one compares the subject
// against its own acknowledgements, because a reference that never lost
// power has nothing to say about what survives losing it.
//
// [suite.Subject.Recover] is read without a nil guard on purpose: a row
// driving this declares [suite.NeedsRecover], so the runner has already
// refused a subject without one — by name, and naming the field that
// would arm it. A guard here would answer the same question a second
// time, in a worse place, and only for the rows that remembered it.
func Recover[S any, K comparable, V any](
	tb testing.TB, sub suite.Subject[S], sch crash.Schedule[S, K, V],
	wrap func(*model.T, S) S,
) {
	tb.Helper()
	sch.Rebuild = func(prior S) S { return sub.Recover(tb, prior) }
	model.Check(tb, func(rt *model.T) {
		crash.Run(rt, sub.New(tb), sch, wrap)
	})
}

// AsBuilt narrows an instance the run is holding back to the type the
// harness that built it declares.
//
// One field needs this and the rest never will. Every other field on a
// generated harness hands an instance OUT — a constructor returns the
// consumer's own type and the subject holds it as the interface — and
// [suite.Subject.Recover] is the one that takes one back in, because
// recovery is over the medium a particular instance left behind.
//
// It cannot fail for an instance the harness built: the only value that
// reaches it came from that harness's own constructor. The guard is for
// the case where that stops being true, where a bare assertion would
// panic naming two interface types and this names the harness instead.
func AsBuilt[T any](tb testing.TB, harness string, held any) T {
	tb.Helper()
	own, built := held.(T)
	if !built {
		var zero T
		tb.Fatalf("harness %q was handed a %T to recover over, and it builds %T; "+
			"every subject in one run comes from one harness", harness, held, zero)
		return zero
	}
	return own
}

// Blend is the provenance-gated adversarial widening: a DERIVED pool
// blends with the hostile half of the string space, and a pool the
// consumer RESTRICTED reaches every tier verbatim — a restricted pool
// is a statement about what the implementation accepts, and blending
// hostility past it would red correct code against inputs its owner
// ruled out.
func Blend[V any](derived bool, pool *model.Generator[V], hostile func(string) V) *model.Generator[V] {
	if !derived {
		return pool
	}
	return model.OneOf(pool, model.Map(model.AdversarialStrings(), hostile))
}
