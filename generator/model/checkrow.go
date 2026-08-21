// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/lang/golang"

	"go.thesmos.sh/testkit/core/lawid"
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
	// report's per-kind lines, and StrengthConst how far the row looks
	// before it passes.
	//
	// Read off the plan's body rather than stamped here: strength is a
	// fact about what the body examines, and the body is what the plan
	// carries. Both legs judge against something outside the subject,
	// which is the strongest of the three — and saying so is what stops a
	// reader crediting the row for more or less than it does.
	ClassConst, StrengthConst string

	// Claim is the sentence the row states, carried verbatim from the
	// plan. Not rephrased here: the plan's claim is what the census
	// measured and what the manifest pinned.
	Claim string

	// Proven says the row has a planted defect behind it, so the harness
	// may stamp it Proven rather than Argued. Argument is why not, where
	// it is not: claiming proof without the evidence is refused in both
	// directions, so an unproven row states its case.
	Proven   bool
	Argument string

	// Needs are the capabilities the row demands of the harness — the
	// clock a clocked law advances, the induction a poison law provokes.
	//
	// Carried through rather than recomputed. The harness was projected
	// from these before this generator ran, so a second derivation could
	// only disagree with a field that already exists.
	Needs []projection.NeedPlan

	// Binds names the laws the row reports under, where any back it, each
	// as the lawid identifier that declares it. A generated file naming a
	// law as a literal drifts the moment the catalogue rewords one.
	Binds []string
}

// CheckRows projects every row this tier renders out of the plans the
// harness generator queued for it.
//
// One row per plan, in plan order. Nothing is filtered here: a plan that
// reached [Rows] is one this tier owns, and dropping it silently
// would leave the index naming a check nothing emits.
func CheckRows(token string, plans []projection.CheckPlan) []CheckRow {
	out := make([]CheckRow, 0, len(plans))
	for _, p := range plans {
		out = append(out, CheckRow{
			Accessor:      rowAccessor(p.ID),
			AssertName:    rowAssertName(token, p.ID),
			ClassConst:    classConst(p.Class),
			StrengthConst: strengthConst(p.Body),
			Claim:         p.Claim,
			Proven:        p.Falsifiable.State == vocab.Proven().State,
			Argument:      p.Falsifiable.Why,
			Needs:         p.Needs,
			Binds:         lawConsts(p.Binds),
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
	acc, err := projection.AccessorOf(id)
	if err != nil {
		return golang.ExportedName(id.Seg)
	}
	return acc.Name
}

// strengthConst is the runtime's identifier for how far a body looks.
//
// A switch rather than a table, and the default is the weakest of the
// three: a body whose strength nobody decided is reported as looking at
// the least, which is the reading that cannot flatter it.
func strengthConst(body projection.Body) string {
	if body == nil {
		return "StrengthErrorOnly"
	}
	switch body.Strength() {
	case vocab.StrengthDifferential:
		return "StrengthDifferential"
	case vocab.StrengthObserved:
		return "StrengthObserved"
	default:
		return "StrengthErrorOnly"
	}
}

// lawConsts spells each bound law through the identifier lawid declares
// it under, dropping any the catalogue does not know.
//
// Dropped rather than spelled as a literal: a law nobody declares is one
// the manifest cannot be held to, and a string in a generated file that
// nothing checks is how a renamed law goes on reading as bound.
func lawConsts(binds []projection.Bind) []string {
	out := make([]string, 0, len(binds))
	for _, b := range binds {
		if name, ok := lawid.ConstOf(b.Law); ok {
			out = append(out, name)
		}
	}
	return out
}

// classConst is the runtime's identifier for a class, so a generated row
// names the constant rather than spelling its value.
//
// A class rendered as its value is a qualified reference to something
// that is not a declaration — `suite.model/laws` — which does not parse.
// The vocabulary has one home and this asks it.
func classConst(c vocab.Class) string {
	if name, ok := vocab.ClassConst(c); ok {
		return name
	}
	return ""
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
