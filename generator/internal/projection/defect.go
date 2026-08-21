// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection

// DefectKind names a defect variant; the value is the variant's
// template name, composed from the one dispatch prefix.
type DefectKind string

// DefectKindPrefix namespaces defect templates in the dispatch table,
// under the plugin's own name for the reason [BodyKindPrefix] gives.
const DefectKindPrefix = "suite.defect."

// The defect kinds, one per variant below.
const (
	KindStubPanic        DefectKind = DefectKindPrefix + "stub-panic"
	KindAnswersAnyway    DefectKind = DefectKindPrefix + "answers-anyway"
	KindAnswersWithValue DefectKind = DefectKindPrefix + "answers-with-value"
	KindFreezeReturn     DefectKind = DefectKindPrefix + "freeze-return"
	KindFreshMedium      DefectKind = DefectKindPrefix + "fresh-medium"
	KindSentinelOnce     DefectKind = DefectKindPrefix + "sentinel-once"
	KindPartialOutlive   DefectKind = DefectKindPrefix + "partial-outlive"
	KindExceedBound      DefectKind = DefectKindPrefix + "exceed-bound"
	KindEchoBesideError  DefectKind = DefectKindPrefix + "echo-beside-error"
	KindSecondCallErrs   DefectKind = DefectKindPrefix + "second-call-errs"
	KindRefusesAlways    DefectKind = DefectKindPrefix + "refuses-always"
)

// Clause is what a planted defect DOES, as the proof's subject line
// spells it — "Get ignores the context it is handed".
//
// Carried by the variant rather than looked up from its kind, because
// several claims are broken by the SAME planted statement and differ
// only in what that statement means for them: a bare return is a
// swallowed context to one claim, a dropped write to another, and a
// refusal that never came to a third. A table keyed on the kind can
// spell one of those three.
//
// Empty is legal and falls back to the kind's own slug, which reads as
// machine text in a report and is the visible sign that a rule left its
// prose out.
type Clause struct{ Text string }

// DefectClause is the prose this defect's report spells.
func (c Clause) DefectClause() string { return c.Text }

// Clauser is a defect that carries its own report prose. Satisfied by
// embedding [Clause]; a variant that does not is reported by its kind.
type Clauser interface{ DefectClause() string }

// The defect variants: one per proofs rule in derivation-rules.md.
// Each is a small plan over the generated double (or a hand shape
// where the double cannot express the defect), and every Proven check
// carries exactly one.

// StubPanic is the smoke family's defect: the named method panics.
type StubPanic struct {
	Clause
	Option Option
}

// AnswersAnyway returns each result's zero and a nil error, planting
// success where the claim under proof requires a refusal.
//
// Named for the statement it emits — a bare return under named results
// — because that one statement breaks every claim whose subject owes an
// error: a swallowed context, a write acknowledged and dropped, a
// sentinel that never came. What it MEANS for the claim it breaks is
// the [Clause]'s to say, which is why four near-identical templates
// reconciled to this one.
type AnswersAnyway struct {
	Clause
	Option Option
}

// AnswersWithValue plants a live value in the first result slot, for a
// claim that a zero would state rather than break.
//
// Distinct from [AnswersAnyway] in its statements, not just its
// meaning: this one has to derive a value, and where the method's
// result type yields none the row ships Argued instead.
type AnswersWithValue struct {
	Clause
	Option Option
}

// FreezeReturn pins a monotonic return to a constant.
type FreezeReturn struct{ Option Option }

// FreshMedium recovers onto a new empty medium.
type FreshMedium struct{}

// SentinelOnce reports the sentinel once, then heals.
type SentinelOnce struct{ Sentinel Expr }

// PartialOutlive keeps exactly one stamped method alive after Close —
// the defect that forces multi-probe lifecycle laws to stay plural.
type PartialOutlive struct{ Option Option }

// ExceedBound reports an accounting number past the declared limit.
type ExceedBound struct{ Option Option }

// EchoBesideError returns a live value beside a non-nil error — the
// zero-on-error family's defect.
type EchoBesideError struct {
	Clause
	Option Option
}

// SecondCallErrs succeeds once and errors on the repeat — the
// idempotent claim's defect.
type SecondCallErrs struct {
	Clause
	Option Option
}

// RefusesAlways reports an error for every call, planting a refusal
// where the claim under proof requires the subject to accept.
//
// The mirror of [AnswersAnyway] and a different statement: an error
// assigned into its slot rather than a bare return. What makes it a
// defect is that a correct subject would have taken the value — so it
// breaks an agreement claim from the side a permissive double cannot,
// since a permissive double and a valid drawn value agree by accident.
type RefusesAlways struct {
	Clause
	Option Option
}

// DefectKind names the template that plants a panicking stub.
func (StubPanic) DefectKind() DefectKind { return KindStubPanic }

// DefectKind names the template that plants a bare answering return.
func (AnswersAnyway) DefectKind() DefectKind { return KindAnswersAnyway }

// DefectKind names the template that plants a live answering value.
func (AnswersWithValue) DefectKind() DefectKind { return KindAnswersWithValue }

// DefectKind names the template that plants a frozen return.
func (FreezeReturn) DefectKind() DefectKind { return KindFreezeReturn }

// DefectKind names the template that plants a fresh medium.
func (FreshMedium) DefectKind() DefectKind { return KindFreshMedium }

// DefectKind names the template that plants a sentinel reported once.
func (SentinelOnce) DefectKind() DefectKind { return KindSentinelOnce }

// DefectKind names the template that plants a partial that outlives its close.
func (PartialOutlive) DefectKind() DefectKind { return KindPartialOutlive }

// DefectKind names the template that plants an exceeded bound.
func (ExceedBound) DefectKind() DefectKind { return KindExceedBound }

// DefectKind names the template that plants an echo beside the error.
func (EchoBesideError) DefectKind() DefectKind { return KindEchoBesideError }

// DefectKind names the template that plants a second call that errors.
func (SecondCallErrs) DefectKind() DefectKind { return KindSecondCallErrs }

// DefectKind names the template that plants a blanket refusal.
func (RefusesAlways) DefectKind() DefectKind { return KindRefusesAlways }

// DefectKinds enumerates every registered defect variant, for the
// template census.
func DefectKinds() []DefectKind {
	return []DefectKind{
		StubPanic{}.DefectKind(),
		AnswersAnyway{}.DefectKind(),
		AnswersWithValue{}.DefectKind(),
		FreezeReturn{}.DefectKind(),
		FreshMedium{}.DefectKind(),
		SentinelOnce{}.DefectKind(),
		PartialOutlive{}.DefectKind(),
		ExceedBound{}.DefectKind(),
		EchoBesideError{}.DefectKind(),
		SecondCallErrs{}.DefectKind(),
		RefusesAlways{}.DefectKind(),
	}
}
