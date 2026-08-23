// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"

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
	// Group is the index field this row is named under, and Accessor the
	// method on it — together `Model.TTLExpiry()`.
	//
	// Per row rather than per declaration. The index groups by the ID's
	// family, and this tier no longer plans into one family: the crash
	// row reports under sim because it judges the subject against its own
	// acknowledgements across a seam nothing else in the run crosses, and
	// a declaration naming one group for all its rows would look every
	// row up under whichever family happened to be written here.
	Group, Accessor string

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

	// Needs are the capability doors this row demands of the harness, as
	// the literal spells them.
	//
	// A list rather than one constructor, because a law can read a clock
	// and a supplied fact at once, and a row naming only the first is one
	// the runner admits against a subject that cannot serve it.
	Needs []NeedCap

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
	return checkRows(token, plans, false)
}

// checkRows is [CheckRows] told whether this run's reference is a twin.
//
// A twin is a second instance of the SUBJECT, so a row comparing the two
// is not judging against anything outside the implementation — it is
// asking the implementation to agree with itself, which catches
// nondeterminism and hidden shared state and nothing else. Reporting
// that at differential strength credits it with a comparison it did not
// make.
func checkRows(token string, plans []projection.CheckPlan, twin bool) []CheckRow {
	out := make([]CheckRow, 0, len(plans))
	for _, p := range plans {
		out = append(out, CheckRow{
			Group:         golang.ExportedName(p.ID.Family),
			Accessor:      rowAccessor(p.ID),
			AssertName:    rowAssertName(token, p.ID),
			ClassConst:    classConst(p.Class),
			StrengthConst: strengthConst(p.Body, twin),
			Claim:         p.Claim,
			Proven:        p.Falsifiable.State == vocab.Proven().State,
			Argument:      p.Falsifiable.Why,
			Needs:         needsCaps(p.Needs),
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

// needsCaps spells every door a row demands of the harness, empty for a
// row that demands nothing.
//
// Every one, in plan order. A law can read a clock and a
// consumer-supplied fact at once — the windowed law does — and a row
// naming only the first is one the runner admits against a subject that
// cannot serve it, so the check fails inside the body naming a nil
// closure rather than at the gate naming the field that arms it.
func needsCaps(needs []projection.NeedPlan) []NeedCap {
	out := make([]NeedCap, 0, len(needs))
	for _, n := range needs {
		if name, named := capConsts[n.Capability]; named {
			out = append(out, NeedCap{Const: name, Sym: n.Sym})
			continue
		}
		out = append(out, NeedCap{Name: string(n.Capability), Sym: n.Sym})
	}
	return out
}

// capConsts is the runtime's identifier for each door it names.
//
// The doors with a dedicated Subject field, which is the same thing: a
// door the runtime answers from a field of its own has a constant
// declaring it, and a door only a given interface's checks know does
// not. A generated file spelling the string where a constant exists is a
// second home for the vocabulary, and the two drift the moment one is
// reworded.
//
//nolint:gochecknoglobals // a vocabulary table, read-only after init.
var capConsts = map[vocab.Capability]string{
	vocab.CapClock:   "CapClock",
	vocab.CapInduce:  "CapInduce",
	vocab.CapRecover: "CapRecover",
}

// NeedCap is one door in a row's Needs literal: the runtime constant the
// vocabulary declares it under, or its bare name where it declares none.
//
// The constant where one exists, because a door the runtime names has a
// home for its spelling — and a generated file repeating the literal is
// where that home stops being single.
type NeedCap struct {
	Const, Name string

	// Sym is the value the door is answered with, where it takes one: an
	// induce door names the sentinel whose state the check needs, and the
	// runner matches a subject's trigger on that identity. Nil for a door
	// whose presence is the whole answer, which renders as nil.
	Sym *sdk.Expr
}

// strengthConst is the runtime's identifier for how far a body looks.
//
// A switch rather than a table, and the default is the weakest of the
// three: a body whose strength nobody decided is reported as looking at
// the least, which is the reading that cannot flatter it.
func strengthConst(body projection.Body, twin bool) string {
	if body == nil {
		return "StrengthErrorOnly"
	}
	switch body.Strength() {
	case vocab.StrengthDifferential:
		if twin {
			// The comparison is against a second copy of the subject, so
			// it observes a value and judges it against nothing the
			// subject does not already believe.
			return "StrengthObserved"
		}
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
// `mixedAssertExpiry`.
//
// The token and the row's own index accessor, by the harness generator's
// own rule, so a reader who has learned one file's naming has learned
// both. The accessor rather than the segment, because a law's segment is
// its AUTO- identifier and `mixedAssertAUTOTTLEXPIRY` is what spelling it
// directly produces. It composes here rather than reading
// [projection.AssertName] because that policy takes a method and these
// rows have none.
func rowAssertName(token string, id projection.IDPlan) string {
	return token + "Assert" + rowAccessor(id)
}
