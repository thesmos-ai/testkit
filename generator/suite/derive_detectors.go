// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// The detector axis's rules: one per shape the annotator INFERS from a
// signature, as against the mixin axis's stamps an author writes.
//
// Fewer rules than the mixin axis and each serves several detectors: a
// reader, a lookup, a pointer-reader and a batch reader all owe the
// same miss, because what a detector settles is how the signature is
// shaped rather than what the author meant by it.

package suite

import (
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// missRule derives the miss, and the seeded hit beside it. The miss
// is reached by choosing an input that is not there, so a method
// taking nothing after its context offers nowhere to put one and the
// rule licenses nothing.
//
// The shape alone does not license it either. A codec's Encode is
// reader-shaped down to the return pair — one input, a value and an
// error — and nothing on the interface writes, so there is no input
// it has not been given: every draw is as valid as the canonical one
// and a check asserting the zero for the alternate asserts a
// falsehood. Either the declaration names what a miss reports, or the
// run has to be able to make one; without both the rule refuses, so
// the gap is named in the header rather than emitted as a claim.
func missRule(f Iface, m subject.Method, call projection.CallPlan) ([]projection.CheckPlan, []Refusal) {
	if !m.HasInput() {
		return nil, nil
	}
	sentinel, verb := missWording(f, m)
	if sentinel == "" && !f.supplies() {
		return nil, []Refusal{
			{
				Deriver: DeriverStamps,
				What:    m.Name + "'s miss check",
				Why: "no method here stores anything and nothing loads it with " +
					"starting data, so there is no input this check could ask " +
					"for and be sure was never stored",
				Remedy: "say what a lookup reports when it finds nothing, with " +
					"//testkit:mixin notfound sentinel=Err…, or write the check yourself",
			},
		}
	}
	plans := []projection.CheckPlan{{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegMiss},
		Class:       vocab.ClassReader,
		Claim:       MissClaim(m, sentinel, verb),
		Body:        missBody(f, m, sentinel),
		Falsifiable: vocab.Proven(),
		Defect:      missDefect(f, m, sentinel),
	}}
	if f.Corpus && !m.HasMixin(MixinTimeAware) {
		plans = append(plans, projection.CheckPlan{
			ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegHit},
			Class:       vocab.ClassReader,
			Claim:       HitClaim(m),
			Body:        projection.HitProbe{Call: hitCall(m)},
			Falsifiable: vocab.Proven(),
			// The zero rather than a live value. A double answering some
			// FIXED value only reds where the corpus holds a different
			// one, which makes the proof depend on the pool's contents;
			// the zero differs from every seeded value by construction,
			// because a sampled member is never a type's zero.
			Defect: projection.AnswersAnyway{
				Clause: projection.Clause{Text: AnswersTheZeroForEverySeedClause(m)},
				Option: projection.OptionName(f.Name, m.Name),
			},
		})
	}
	return plans, nil
}

// missBody picks the shape the declaration licenses: errors.Is where a
// sentinel names what a miss reports, the zero comparison where none
// does. The same split [missDefect] makes, for the same reason — what
// the claim is about decides both what the body judges and what a
// double has to do to break it.
func missBody(f Iface, m subject.Method, sentinel string) projection.Body {
	call := missCall(f, m)
	if sentinel != "" {
		return projection.ReportsSentinel{Call: call, Sentinel: projection.Expr(sentinel)}
	}
	return projection.AnswersZero{Call: call, Because: BecauseUnsupplied()}
}

// missDefect picks the double that breaks a miss claim.
//
// With a sentinel declared the claim is that error and nothing else, so
// a subject that merely ANSWERS has already broken it and a bare return
// says so. Without one the claim is about the value slot, and a double
// returning the zero would STATE the claim rather than break it — so
// that arm has to plant a live value, and where the result type yields
// none the row ships Argued.
func missDefect(f Iface, m subject.Method, sentinel string) projection.Defect {
	clause := projection.Clause{Text: AnswersMissClause(m)}
	option := projection.OptionName(f.Name, m.Name)
	if sentinel != "" {
		return projection.AnswersAnyway{Clause: clause, Option: option}
	}
	return projection.AnswersWithValue{Clause: clause, Option: option}
}

// answerRule derives what a write that answers its stored state owes:
// an answer that is not the zero.
//
// The detector's own reason for existing, restated as a check. It
// separates `(ctx, V) (V, error)` from `(ctx, V) error` precisely
// because the second loses the value the store actually kept — so a
// subject matching the first shape and answering the zero has the
// second shape's defect while advertising the first's.
//
// Nothing stronger is derivable. A store may answer what it was handed
// or one it stamped, and a check requiring either fails the other.
func answerRule(f Iface, m subject.Method, call projection.CallPlan) ([]projection.CheckPlan, []Refusal) {
	if len(m.ValueReturns()) == 0 {
		return nil, []Refusal{{
			Deriver: DeriverStamps,
			What:    m.Name + "'s answer check",
			Why:     "it answers no value, so there is no answer to hold to anything",
			Remedy:  "return the stored state beside the error, which is what the shape names",
		}}
	}
	return []projection.CheckPlan{{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegAnswer},
		Class:       vocab.ClassAnswer,
		Claim:       AnswerClaim(m),
		Body:        projection.NonZeroAnswer{Call: call, Must: AnswerRequirement()},
		Falsifiable: vocab.Proven(),
		Defect: projection.AnswersAnyway{
			Clause: projection.Clause{Text: AnswersTheZeroClause(m)},
			Option: projection.OptionName(f.Name, m.Name),
		},
	}}, nil
}

// countRule derives the seeded-aggregator equality. An aggregator on
// an interface that writes has no fixed number to equal — its count
// claims are the law catalogue's territory — and one on an interface
// nothing seeds has no number at all, so the rule licenses nothing in
// either case. Silent rather than refused: the count is the hit's
// companion and the miss beside it already names the gap.
func countRule(f Iface, m subject.Method, call projection.CallPlan) ([]projection.CheckPlan, []Refusal) {
	if !f.Corpus {
		return nil, nil
	}
	return []projection.CheckPlan{{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegCount},
		Class:       vocab.ClassReader,
		Claim:       CountClaim(m),
		Body:        projection.CountProbe{Call: call},
		Falsifiable: vocab.Proven(),
		// Zero against a corpus that has at least one entry — [CorpusOf]
		// refuses without both pools and a pool has members, so the
		// count under proof is never zero and the double is never
		// accidentally right.
		Defect: projection.AnswersAnyway{
			Clause: projection.Clause{Text: CountsNothingClause(m)},
			Option: projection.OptionName(f.Name, m.Name),
		},
	}}, nil
}

// hitCall is [callOf] with the drawn key replaced by the loop variable
// the hit body ranges the corpus with.
//
// Every seeded key, not the fixture's one. The fixture holds a member of
// the key pool and the corpus holds all of them, so a body drawing the
// fixture asks about the same entry once per iteration — which passes
// for a subject that kept the first thing it was given and dropped the
// rest, the exact failure a hit check is for.
func hitCall(m subject.Method) projection.CallPlan {
	call := callOf(m)
	for i, arg := range call.Args {
		if arg == projection.ExprCtx {
			continue
		}
		call.Args[i] = projection.ExprSeededKey
		break
	}
	return call
}
