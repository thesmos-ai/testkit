// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/lang/golang"

	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

// CheckRow is one row this tier contributes to the package's run
// surface — everything the template spells, worked out here.
//
// The same shape the harness generator's own rows take, because they end
// up in one []suite.Check and a consumer reading the report cannot tell
// which generator wrote which. A row that carried less would be a row
// the runner treats differently, and "the model tier's checks are second
// class" is not a claim anybody wants to defend.
type CheckRow struct {
	// Accessor is the index method naming this row —
	// `Model.TTLExpiry()`. The family is fixed; only the word varies.
	Accessor string

	// AssertName is the generated function the row's RunWith calls.
	AssertName string

	// ClassConst is the runtime class constant's identifier, for the
	// report's per-kind lines.
	ClassConst string

	// Claim is the sentence the row states, carried verbatim from the
	// plan. Not rephrased here: the plan's claim is what the census
	// measured and what the manifest pinned.
	Claim string

	// Proven says the row has a planted defect behind it, so the harness
	// may stamp it Proven rather than Argued.
	Proven bool

	// Needs are the capabilities the row demands of the harness — the
	// clock a clocked law advances, the induction a poison law provokes.
	//
	// Carried through rather than recomputed. The harness was projected
	// from these before this generator ran, so a second derivation could
	// only disagree with a field that already exists.
	Needs []projection.NeedPlan

	// Binds names the laws the row reports under, where any back it.
	Binds []projection.Bind
}

// CheckRows projects every row this tier renders out of the plans the
// harness generator queued for it.
//
// One row per plan, in plan order. Nothing is filtered here: a plan that
// reached [ModelRows] is one this tier owns, and dropping it silently
// would leave the index naming a check nothing emits.
func CheckRows(token string, plans []projection.CheckPlan) []CheckRow {
	out := make([]CheckRow, 0, len(plans))
	for _, p := range plans {
		out = append(out, CheckRow{
			Accessor:   rowAccessor(p.ID),
			AssertName: rowAssertName(token, p.ID),
			ClassConst: string(p.Class),
			Claim:      p.Claim,
			Proven:     p.Falsifiable.State == vocab.Proven().State,
			Needs:      p.Needs,
			Binds:      p.Binds,
		})
	}
	return out
}

// rowAccessor names the index method for a family-scoped row —
// `Model.TTLExpiry()`, spelled without its parentheses.
//
// The family is this tier's throughout, so the segment is the only word
// that varies. A method-scoped row cannot occur here: every plan this
// tier renders is about the interface as a whole, which is what "the
// claim needs sequences rather than a call" means.
func rowAccessor(id projection.IDPlan) string {
	return golang.ExportedName(id.Seg)
}

// rowAssertName is the generated assertion's identifier —
// `mixedAssertTTLExpiry`.
//
// The token and the segment, by the harness generator's own rule, so a
// reader who has learned one file's naming has learned both. It composes
// here rather than reading [projection.AssertName] because that policy
// takes a method and these rows have none.
func rowAssertName(token string, id projection.IDPlan) string {
	return token + "Assert" + golang.ExportedName(id.Seg)
}
