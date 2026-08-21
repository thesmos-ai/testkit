// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

// The claims this tier's own rows state.
//
// Worded here because they are claims about SEQUENCES, and no
// classification states one: a law says what holds, and these say that
// it holds over sequences nobody wrote. A per-law claim comes from
// lawid's table instead, where the law's own wording lives.
const (
	lawsClaim = "every bound law holds over random operation sequences"
	refClaim  = "every operation sequence leaves the subject agreeing with the reference"
)

// unproven is why neither row claims to have been shown able to fail.
//
// Argued rather than Proven, and not a formality: the parity gate refuses
// the Proven stamp without a planted defect beside it, and the defects
// this package plants break one method of a stub at a time. A sequence
// claim needs a subject that is wrong over a history — the saturation
// prover's job, which runs against a surface this tier does not emit yet.
// Claiming proof without the evidence is refused in both directions, so
// the honest record is the argument.
const unproven = "no defect is planted for it: the proofs beside this file break one " +
	"method at a time, and a claim about sequences needs a subject that is wrong " +
	"over a history"

// PlanRows is the checks this tier owns for one interface.
//
// Planned here rather than by the harness generator. That generator
// planned them once, because the harness's capability fields were
// projected from the plans and a clocked law's row was what opened the
// clock — so the rows and the field had to be worked out together. They
// no longer do: this tier contributes the field itself, so it can plan
// the row that needs it.
//
// One row per leg, not per law. A law with a leg of its own reports
// under it; the rest ride the shared sequences under one claim, because
// what a reader wants to know is which oracle failed, and every law on
// the shared leg failed the same way.
func PlanRows(b *Bindings) []projection.CheckPlan {
	var out []projection.CheckPlan

	if bundle := b.bundledLaws(); len(bundle) > 0 {
		out = append(out, projection.CheckPlan{
			ID:          projection.IDPlan{Family: vocab.FamilyModel, Qualifier: b.qualifier(), Seg: vocab.SegLaws},
			Class:       vocab.ClassLaws,
			Claim:       lawsClaim,
			Body:        projection.LawLeg{Laws: bundle},
			Binds:       bundle,
			Falsifiable: vocab.Argued(unproven),
		})
	}

	// The differential is the strongest oracle this tier has, and it runs
	// with no law registered so nothing competes with it: a disagreement
	// is what ends the run.
	if b.Reference.Derived() || b.Reference.Supplied() {
		out = append(out, projection.CheckPlan{
			ID: projection.IDPlan{
				Family: vocab.FamilyModel, Qualifier: b.qualifier(), Seg: vocab.SegDifferential,
			},
			Class:       vocab.ClassDifferential,
			Claim:       refClaim,
			Body:        projection.DifferentialLeg{},
			Falsifiable: vocab.Argued(unproven),
		})
	}
	return out
}

// qualifier is the interface's word inside a family-scoped identity.
//
// Composed through the same policy the harness generator's index spells
// it with, because the two have to agree: the index names the row and
// the row reports under the name, and a slug derived twice diverges the
// moment an interface name has two words.
func (b *Bindings) qualifier() string { return projection.IDQualifier(b.IfaceName) }

// bundledLaws are the laws riding the shared sequences, as the row's
// Binds column spells them.
//
// Read off [Bindings.LegLaws] rather than filtered again here, so the row
// names exactly the laws the leg beside it registers. Two derivations of
// one set is how a manifest comes to promise a law nothing runs.
func (b *Bindings) bundledLaws() []projection.Bind {
	legLaws := b.LegLaws()
	out := make([]projection.Bind, 0, len(legLaws))
	for _, l := range legLaws {
		out = append(out, projection.Bind{Law: l.ID})
	}
	return out
}
