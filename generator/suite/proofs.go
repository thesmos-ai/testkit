// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"slices"
	"sort"
	"strconv"
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"

	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/naming"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// KindProofs is the emit kind for the falsification companion, which is
// also its template's name.
const KindProofs sdk.Kind = "suite.proofs"

// Proofs is the companion file: one planted defect per check the run
// stamped Proven.
//
// A separate emit rather than a section of [Contract] because it lands in
// a separate output — the `_test.go` the layout shifts into the external
// test package — and an emit is routed as a whole. Everything it needs
// from the harness it carries a copy of; reaching back through a pointer
// would put a cycle in a graph that gets walked.
type Proofs struct {
	sdk.BaseEmit
	subject.Subject

	// Pkg is the import path of the primary output, which is where every
	// symbol this file names is declared.
	//
	// The companion is always the external test package of the harness,
	// so every reference is qualified and there is no routing under which
	// a bare name would be right. Provisional during Generate and
	// corrected by [Proofs.SetOutputPackages], exactly as the double's
	// companion does it.
	Pkg string

	// Token qualifies the identifiers this file declares for itself.
	Token string

	// SourcePkg is the declaration's own package, where a stamped
	// sentinel is resolved from.
	//
	// Not [Proofs.Pkg]: that one is repointed at wherever Layout routed
	// the harness, and a sentinel the source declares lives where the
	// source does whatever the output moved to.
	SourcePkg string

	// Vocab and Prove are the packages the emitted calls come from,
	// carried rather than spelled in the template because an import path
	// built inside a template is one the backend cannot register.
	Vocab, Prove string

	// Fixture is the derived input set, whose constructor the proofs bind
	// the suite to — the same one a default run binds to, so a proof
	// cannot pass against inputs no run uses. DrawsFixture says whether
	// the assembler takes one at all, which follows the harness's own
	// answer rather than being decided again here.
	Fixture      subject.Fixture
	DrawsFixture bool

	// SeedsCorpus says the assembler takes the run's corpus too, and
	// CorpusFunc is the builder that makes one — both mirroring the
	// harness's own answer rather than deciding it again.
	SeedsCorpus bool
	CorpusFunc  string

	// Pools are the config's drawn fields, for the provenance regression.
	//
	// The one property of this surface nothing else can see: a pool the
	// consumer did not narrow has to keep its adversarial arm, and a pool
	// they did narrow has to reach every tier verbatim. Both directions,
	// because getting either wrong is silent — the same values are drawn,
	// the same rows report, and only the strength of the check moves.
	Pools []projection.PoolPlan

	// Defects are the planted defects, shared with the run surface that
	// renders them.
	//
	// A pointer to one value rather than a slice in each, because a
	// contributing tier plants into it AFTER both emits are queued: a copy
	// taken at queue time would hold this generator's own rows and none of
	// the model tier's, and the parity gate would then fail naming every
	// model row that claims Proven.
	//
	// Not a pointer back to the other emit — that would put a cycle in a
	// graph that gets walked. Both hold the leaf.
	Defects *PlantedDefects

	// Unproven names the checks a deriver stamped Proven and this file
	// has no defect template for.
	//
	// The row is emitted Argued instead, so the parity gate stays quiet;
	// this is what keeps that downgrade from reading as a claim nothing
	// could ever falsify. Non-empty wherever the defect census is behind
	// the body census, which is where the rewrite currently stands.
	Unproven []string

	// Generic reports that the subject is parameterised, which is the one
	// case where a companion cannot be written: a Go test function takes
	// no type parameters, so there is nothing to instantiate the defects
	// at. It gets a note in place of its proofs rather than no file, the
	// same contract the double's companion keeps.
	Generic bool
}

// PlantedDefects is the defect map the run surface renders and the
// companion beside it lists.
//
// A named type rather than a bare slice pointer so both templates read
// one field name, and so [Proofs.Plant] has somewhere to append that
// every holder sees.
type PlantedDefects struct{ Rows []*ProofEmit }

// Kind returns [KindProofs].
func (*Proofs) Kind() sdk.Kind { return KindProofs }

// SetOutputPackages repoints [Proofs.Pkg] at wherever Layout routed the
// harness.
//
// An empty path for the primary tag means the target resolved without a
// derivable import path. The provisional value is left rather than
// blanked: a wrong package is a compile error naming the symbol, a bare
// name silently binds to whatever else is in scope.
//
// Each defect is repointed too. A defect renders standalone through the
// backend's dispatch and holds no path back to this node, so the
// correction has to reach every one of them rather than only the file
// they sit in.
func (p *Proofs) SetOutputPackages(byTag map[string]string) {
	path, ok := sdk.PrimaryPackage(byTag)
	if !ok {
		return
	}
	p.Pkg = path
	if p.Defects == nil {
		return
	}
	for _, d := range p.Defects.Rows {
		d.Pkg = path
	}
}

// VeneerVar is the exported value the proofs reach the check index
// through.
func (p *Proofs) VeneerVar() string { return projection.VeneerName(p.IfaceName) }

// ProveFunc is the entry point that drives the defects, which this file
// names in place of the test it used to run itself.
func (p *Proofs) ProveFunc() string { return projection.ProveName(p.IfaceName) }

// ProofEmit is one planted defect as its template renders it.
//
// The node the backend dispatches on, mirroring [CheckEmit]: its kind IS
// the defect variant's template name, so a variant nothing spells fails
// the render by name rather than emitting a proofs map with a hole in it
// — which the parity gate would report against the generated file
// instead of against the generator.
type ProofEmit struct {
	sdk.BaseEmit
	defectView

	// Plan is the check this defect is the evidence for. The map key
	// reads its identity; the view beside it says how the defect is
	// spelled.
	Plan projection.CheckPlan

	accessor string
}

// Kind returns the planted defect's variant, which is its template's
// name.
func (p *ProofEmit) Kind() sdk.Kind { return sdk.Kind(p.Plan.Defect.DefectKind()) }

// Group is the index member the check sits under, so the map key is
// written through the same tree a consumer drops it through.
//
// The method for a method-scoped check and the family for one scoped to
// the interface as a whole, which is the same rule [projection.IndexOf]
// groups by — a defect keyed through a member the index does not have
// names nothing, and compiles nowhere.
func (p *ProofEmit) Group() string {
	if p.Plan.ID.Method != "" {
		return golang.ExportedName(p.Plan.ID.Method)
	}
	return golang.ExportedName(p.Plan.ID.Family)
}

// Accessor is the check's entry point within that member.
func (p *ProofEmit) Accessor() string { return p.accessor }

// defectView is one planted defect's rendering context: the names the
// generated double is reached by, the signature the override is declared
// at, and the prose the report reads.
//
// Every field is a fact about the METHOD or the generated package, which
// is why they are resolved where both are still in hand rather than
// carried through the projection — the same split [bodyView] makes.
type defectView struct {
	// Pkg is the harness package, which every generated name below is
	// qualified by; Prove and Vocab are the runtime packages the defect
	// itself is built from.
	//
	// Carried per defect rather than read off the file, because a defect
	// is dispatched standalone — the backend renders it by its own kind,
	// with no path back to the emit that holds it.
	Pkg, Prove, Vocab string

	// Subject is the interface as this file names it, and Method the
	// method the defect overrides.
	Subject, Method string

	// Ctor is the double's constructor and Option the one-method
	// override the defect is planted through — `NewCalculatorStub`,
	// `WithCalculatorAdd`. Names rather than references: the template
	// qualifies them against Pkg through the backend, which is what
	// registers the import.
	Ctor, Option string

	// DefectName is the subject line the report prints for this defect.
	DefectName string

	// EchoMessage, PanicMessage and RepeatMessage are the planted
	// failures' own text.
	// Prose belonging to the defect rather than to any claim, so it has
	// its home here and not in a runtime constant.
	PanicMessage, RepeatMessage, EchoMessage string

	// RefusalMessage is the blanket refusal a [projection.RefusesAlways]
	// double reports, beside the three above for the same reason.
	RefusalMessage string

	// NeedsClock marks a defect standing in for a clocked check.
	NeedsClock bool

	// ForwardParams is the override's parameter list with every position
	// named, and ForwardArgs the call that passes them on — the two
	// halves a delegating defect needs. Ref, DelegateTo and Mutate are
	// its other three: what to put behind the double, the option that
	// does it, and the one statement that makes it wrong.
	ForwardParams   []*sdk.EmitParam
	ForwardArgs     string
	Ref, DelegateTo string
	Mutate          string

	// HealsMessage is what the healing defect reports where the
	// declaration stamps no sentinel for it to borrow.
	HealsMessage string

	// Echo is a live value of the method's first result, for the two
	// defects that must ANSWER something a correct subject would not.
	//
	// Derived through the same sampler the fixture uses, so a defect and
	// a check draw values from one rule. Its Text is empty where no
	// sample could be derived — a func result, a channel, a type this
	// run never read — and that empties the whole defect: a body that
	// cannot name a live value cannot plant the wrongness it is for, so
	// the row stays Argued rather than shipping evidence that is really
	// the zero it was supposed to contradict.
	Echo golang.Sample

	// Sentinel is the error a healing defect reports before it heals —
	// the same identity the declaration stamped, so the defect cannot
	// break the claim by naming a different one. Nil for every other
	// variant.
	Sentinel *sdk.Expr

	// ReasonConst is the identifier the substring the red must contain is
	// declared under, empty where this run has none to quote.
	//
	// The identifier rather than the text: quoting the message would let
	// a reworded primitive silently weaken every proof that reads it,
	// while naming the constant makes the same reword a compile error in
	// the generated file. Empty for a family whose failure prose is
	// authored in a body template, which has no constant to name yet —
	// a weaker proof, and one the generated comment says out loud.
	ReasonConst string

	// AnonParams is the override's parameter list with every name blanked
	// — the defect bodies read no argument, and a named parameter nothing
	// reads is a lint finding in a file nobody can edit.
	AnonParams []*sdk.EmitParam

	// AnonReturns is the result list with every name dropped, for a body
	// that never returns.
	AnonReturns []*sdk.EmitReturn

	// NamedReturns is the same list with a name on every slot, for a body
	// that answers: a bare `return` then yields each slot's zero without
	// this file having to spell a zero of a type it may not be able to
	// name. ErrSlot is the identifier the error among them binds to, and
	// ValueSlot the first result beside it — what an answering defect
	// assigns its live value to.
	NamedReturns       []*sdk.EmitReturn
	ErrSlot, ValueSlot string

	// FlagSlot is the presence flag a keyed read answers beside its value,
	// empty where the signature has none. See [flagSlot] for why a defect
	// answering anyway has to fill it.
	FlagSlot string
}

// errLocal is the identifier a defect body assigns its planted error to.
//
// Fixed rather than derived: it names the error slot of a result list
// whose other slots are `r0`, `r1`, …, so no signature can collide with
// it, and a body that reads the same in every generated file is one a
// reviewer can scan.
const errLocal = "err"

// defectRendered is the defect variants whose templates exist today.
//
// The mirror of [RenderedBodyKinds]: a check whose body renders and whose defect
// does not is emitted Argued, so the two censuses decide together what a
// row claims about itself.
func defectRendered() map[projection.DefectKind]bool {
	return map[projection.DefectKind]bool{
		projection.KindStubPanic:        true,
		projection.KindAnswersAnyway:    true,
		projection.KindAnswersWithValue: true,
		projection.KindSecondCallErrs:   true,
		projection.KindRefusesAlways:    true,
		projection.KindEchoBesideError:  true,
		projection.KindPartialOutlive:   true,
		projection.KindSentinelOnce:     true,
		projection.KindDelegated:        true,
		projection.KindFreshMedium:      true,
		projection.KindFreezeReturn:     true,
	}
}

// spellsDefect reports whether this run can write the defect out.
//
// A template is necessary and not sufficient. Two variants have to
// ANSWER a live value, and whether one can be derived is a
// property of the METHOD's result type rather than of the variant — so
// renderability is asked per check rather than read off the census.
// Answering wrongly in the permissive direction ships a defect that
// returns the zero it was meant to contradict, which is a proof that
// passes while proving the opposite.
func spellsDefect(kind projection.DefectKind, view defectView) bool {
	if !defectRendered()[kind] {
		return false
	}
	switch kind {
	case projection.KindEchoBesideError, projection.KindAnswersWithValue:
		// OK() rather than a text test. A sample no Ref-and-Text pair
		// can spell — a func literal, a make, a constructor call —
		// carries its expression instead and leaves Text empty, so
		// asking about the text alone would withhold a defect this run
		// can perfectly well write.
		return view.Echo.OK()
	default:
		return true
	}
}

// proofsOf pairs every check the run can render with the defect that
// proves it, and reports which of them found none.
//
// Driven off the emitted checks rather than the whole inventory: a proof
// for a check the harness does not emit is a defect naming nothing, which
// the parity gate fails on — so the two sets are derived from one walk
// instead of two that could disagree.
func proofsOf(
	base sdk.BaseEmit, pkg string, r golang.Resolver, iface Iface, checks []*CheckEmit,
) ([]*ProofEmit, []string) {
	byName := make(map[string]subject.Method, len(iface.Methods))
	for _, m := range iface.Methods {
		byName[m.Name] = m
	}

	out := make([]*ProofEmit, 0, len(checks))
	var unproven []string
	for _, c := range checks {
		m, found := byName[c.Plan.ID.Method]
		if !found {
			continue
		}
		if !c.Proven {
			// Stamped Proven by its deriver and downgraded for want of a
			// template. Named rather than dropped: the row says Argued
			// and this says why, so a reader can tell a claim nothing
			// can falsify from one nobody has spelled yet.
			unproven = append(unproven, m.Name+"/"+c.Plan.ID.Seg)
			continue
		}
		view := defectViewOf(pkg, iface.Package, r, iface.Name, m, c.Plan)
		if !spellsDefect(c.Plan.Defect.DefectKind(), view) {
			// The template exists and this METHOD defeats it: no live
			// value of its result type could be derived, so the defect
			// would answer the very zero it was meant to contradict.
			// The row loses its stamp here rather than shipping one, and
			// says which of the two downgrades it met — "nobody wrote the
			// template" sends a reader to this generator, "your result
			// type yields no value" sends them to their own signature.
			c.Proven = false
			c.sampleless = true
			unproven = append(unproven, m.Name+"/"+c.Plan.ID.Seg)
			continue
		}
		out = append(out, &ProofEmit{
			BaseEmit:   base,
			defectView: view,
			Plan:       c.Plan,
			accessor:   c.Accessor(),
		})
	}
	sort.Strings(unproven)
	return out, unproven
}

// Plant records one contributed row's planted defect in this file, false
// where this run cannot write it out.
//
// The seam a tier that owns rows of its own reaches: it derived the row
// and the rule that breaks it, and everything else — how a double is
// named, what an override is spelled at, which variants have templates —
// is this file's. A contributor building the defect itself would be a
// second place that has to know all of that, and the first sign of a
// disagreement would be a proofs map that does not compile.
//
// False rather than a silent drop, because the row's own stamp has to
// move with it: the parity gate refuses a check claiming Proven with no
// defect beside it as firmly as the reverse.
func (p *Proofs) Plant(
	r golang.Resolver, m subject.Method, plan projection.CheckPlan, accessor string,
) bool {
	if plan.Defect == nil {
		return false
	}
	view := defectViewOf(p.Pkg, p.SourcePkg, r, p.IfaceName, m, plan)
	if !spellsDefect(plan.Defect.DefectKind(), view) {
		return false
	}
	if p.Defects == nil {
		p.Defects = &PlantedDefects{}
	}
	p.Defects.Rows = append(p.Defects.Rows, &ProofEmit{
		BaseEmit:   p.BaseEmit,
		defectView: view,
		Plan:       plan,
		accessor:   accessor,
	})
	return true
}

// defectViewOf spells one planted defect against the method it overrides.
func defectViewOf(
	pkg, srcPkg string, r golang.Resolver,
	ifaceName string, m subject.Method, plan projection.CheckPlan,
) defectView {
	sig := m.Sig
	reason, _ := vocab.RedConst(plan.ID.Seg)
	// A defect scoped to the SUBJECT rather than to a method arrives with
	// no method at all — a rebuild onto an empty medium overrides
	// nothing. Method embeds its signature by pointer and promotes Name
	// through it, so every read below has to go through this rather than
	// through m.
	var name string
	var echo golang.Sample
	if sig != nil {
		name = m.Name
		echo, _ = echoSample(m, r)
	}
	var sentinel *sdk.Expr
	var ref, delegateTo, mutate string
	if d, ok := plan.Defect.(projection.DelegatedOverride); ok {
		ref, delegateTo, mutate = string(d.Ref), string(d.DelegateTo), d.Mutate
	}
	if d, ok := plan.Defect.(projection.FreshMedium); ok {
		ref = string(d.Ref)
	}
	if heals, ok := plan.Defect.(projection.SentinelOnce); ok && heals.Sentinel != "" {
		// The declaration's own package, because the stamp names the
		// sentinel as the source spells it: bare where the interface
		// declares it, qualified where it does not.
		sentinel = sentinelRef(srcPkg, string(heals.Sentinel))
	}
	return defectView{
		Sentinel:      sentinel,
		Echo:          echo,
		ValueSlot:     valueSlot(sig),
		FlagSlot:      flagSlot(sig),
		Pkg:           pkg,
		Prove:         Prove,
		Vocab:         Vocab,
		Subject:       ifaceName,
		Method:        name,
		Ctor:          projection.StubCtorName(ifaceName, naming.StubSuffix),
		Option:        string(projection.OptionName(ifaceName, name)),
		DefectName:    projection.DefectName(ifaceName, defectClause(name, plan.Defect)),
		PanicMessage:  plantedPrefix + name + " panics",
		RepeatMessage: plantedPrefix + name + " refuses its repeat",
		EchoMessage:   plantedPrefix + name + " refused with a believable value",
		RefusalMessage: plantedPrefix + name +
			" refuses everything it is handed",
		// What the healing defect reports where the declaration stamps
		// no sentinel. The law it breaks asks only that the answer stay
		// non-nil once the state is reached, so the identity is free —
		// but the message still says the state is planted, because a
		// reader meeting it in a failure has to know it was put there.
		HealsMessage: plantedPrefix + name + " reports the state once and heals",
		ReasonConst:  reason,
		// A clocked check refuses a subject with no OnClock, and a defect
		// refused for wiring reds without saying anything about the
		// claim. The defects this tier plants never read the clock —
		// that is usually the planted statement — so they accept one and
		// ignore it.
		NeedsClock: slices.ContainsFunc(plan.Needs, func(n projection.NeedPlan) bool {
			return n.Capability == vocab.CapClock
		}),
		AnonParams:    anonParams(sig),
		ForwardParams: forwardParams(sig),
		ForwardArgs:   forwardArgs(sig),
		Ref:           ref,
		DelegateTo:    delegateTo,
		Mutate:        mutate,
		AnonReturns:   anonReturns(sig),
		NamedReturns:  namedReturns(sig),
		ErrSlot:       errLocal,
	}
}

// echoSample derives a live value of the method's first result — the
// one an answering defect returns where a correct subject would not.
//
// Through the fixture's own sampler, so the value a defect plants and
// the values a check draws come from one rule rather than two that
// could disagree about what this type looks like. The ALTERNATE member
// deliberately: a defect returning the same value the fixture seeds
// would be indistinguishable from a subject that answered correctly.
// The sample is withheld where it spells the result's own zero, which
// is where a planted answer would BE the claim rather than break it —
// a predicate answering bool has exactly two values and one of them is
// what the check demands. The row ships Argued and says so.
func echoSample(m subject.Method, r golang.Resolver) (golang.Sample, bool) {
	src := firstValueSource(m)
	if src == nil {
		return golang.Sample{}, false
	}
	_, alternate := derivedPair(src, projection.DrawWord(golang.Param{Source: src}), r)
	if !alternate.OK() || spellsZero(alternate, src) {
		// OK() rather than a text test, for the reason [spellsDefect]
		// asks the same way: a sample no Ref-and-Text pair can spell
		// carries its expression instead and leaves Text empty, and a
		// gate reading the text alone refuses a value this run can
		// perfectly well write — before the renderability check
		// downstream ever sees it.
		//
		// spellsZero stays a text test on purpose: it only answers for a
		// predeclared type, and a predeclared type always has a literal.
		return golang.Sample{}, false
	}
	return alternate, true
}

// zeroLiterals are the texts a sampler writes that ARE a zero value.
// Keyed by the predeclared type they belong to, because "0" is the zero
// of int and a perfectly good non-zero for nothing else.
func zeroLiterals() map[string]string {
	return map[string]string{
		"bool":   "false",
		"string": `""`,
		"int":    "0", "int8": "0", "int16": "0", "int32": "0", "int64": "0",
		"uint": "0", "uint8": "0", "uint16": "0", "uint32": "0", "uint64": "0",
		"uintptr": "0",
		"float32": "0", "float64": "0",
		"byte": "0", "rune": "0",
	}
}

// spellsZero reports that this sample is the zero of the type it was
// derived for, so a defect planting it would assert the claim instead
// of violating it.
//
// Only predeclared types are answered. A struct sample carries its
// fields in the text, and one whose every field happened to be its own
// zero is a sampler that produced nothing — a different fault, and one
// the fixture's own OK flag already reports.
func spellsZero(s golang.Sample, src *node.TypeRef) bool {
	if src == nil || src.Name == "" || !golang.IsPredeclared(src.Name) {
		return false
	}
	zero, known := zeroLiterals()[src.Name]
	return known && s.Text == zero
}

// valueSlot is the identifier the first result binds to under named
// returns, empty for a method that answers nothing but an error.
func valueSlot(sig *golang.Sig) string {
	if sig == nil {
		return ""
	}
	for _, ret := range sig.Returns {
		if !ret.Error {
			return ret.Local
		}
	}
	return ""
}

// flagSlot is the presence flag a read answers beside its value — the
// `bool` of `Get(ctx, K) (V, bool)` — empty for a signature with none.
//
// A defect answering where a correct subject answers nothing has to fill
// this as well as the value. On a reader with an error channel the value
// alone is the whole statement: nil error plus a live value IS the
// invented answer. Here the value slot says nothing on its own — a caller
// reads the flag and ignores what came with it — so a defect that set
// only the value would state the claim it was built to break.
//
// The first bool AFTER the value slot, rather than the last return or a
// fixed position. A page read answers `(items, next, more, error)` and
// its flag is third; a keyed read answers `(V, bool)` and its flag is
// second. Skipping the value slot is what keeps a reader whose VALUE is a
// bool — `Enabled(ctx, K) (bool, error)` — from having its own answer
// overwritten with true, which would state the claim rather than break
// it. A signature with no bool past the value has no flag, and the
// value alone is the whole planted statement.
func flagSlot(sig *golang.Sig) string {
	if sig == nil {
		return ""
	}
	seenValue := false
	for _, ret := range sig.Returns {
		if ret.Error {
			continue
		}
		if !seenValue {
			seenValue = true
			continue
		}
		if ret.Source != nil && golang.IsBool(ret.Source) {
			return ret.Local
		}
	}
	return ""
}

// plantedPrefix opens every planted failure's own text.
//
// One word, at the front, because these strings surface in a run's output
// beside real failures and the reader's first question is which they are
// looking at.
const plantedPrefix = "planted: "

// defectClause words what one planted defect does, in the grammar
// [projection.DefectName] wraps.
//
// Read off the variant, which carries the prose its deriving rule chose.
// A table keyed on the kind cannot serve: several claims are broken by
// the same planted statement — a bare return under named results — and
// what a report has to say about it differs with the claim. The wording
// itself lives in claims.go beside the claims, which is the one home for
// derived prose.
//
// A variant carrying none falls back to its kind, which reads as a slug
// and is the visible sign that a rule left its wording out.
func defectClause(method string, d projection.Defect) string {
	if c, carries := d.(projection.Clauser); carries && c.DefectClause() != "" {
		return c.DefectClause()
	}
	return method + " " + strings.TrimPrefix(string(d.DefectKind()), projection.DefectKindPrefix)
}

// anonParams is the override's parameters with every name blanked.
//
// All blank rather than some: Go's grammar forbids a list that mixes
// named and unnamed entries, and the backend's renderParams reports that
// as an error rather than emitting it.
func anonParams(sig *golang.Sig) []*sdk.EmitParam {
	if sig == nil {
		return nil
	}
	out := make([]*sdk.EmitParam, 0, len(sig.Params))
	for _, p := range sig.Params {
		out = append(out, &sdk.EmitParam{Name: "_", Type: p.Type, Variadic: p.Variadic})
	}
	return out
}

// forwardParams is the parameter list with every position named, for an
// override that has to refer to what it was handed.
//
// The delegating defect needs both halves: a statement altering one
// argument, and a call forwarding the rest. A blanked list serves the
// defects that ignore their arguments and cannot serve these.
func forwardParams(sig *golang.Sig) []*sdk.EmitParam {
	if sig == nil {
		return nil
	}
	out := make([]*sdk.EmitParam, 0, len(sig.Params))
	for i, p := range sig.Params {
		out = append(out, &sdk.EmitParam{Name: forwardName(i, p), Type: p.Type, Variadic: p.Variadic})
	}
	return out
}

// forwardArgs is [forwardParams]'s call site — the same names in the
// same order, with a variadic tail spread.
func forwardArgs(sig *golang.Sig) string {
	if sig == nil {
		return ""
	}
	names := make([]string, 0, len(sig.Params))
	for i, p := range sig.Params {
		name := forwardName(i, p)
		if p.Variadic {
			name += "..."
		}
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

// forwardName is one parameter's local, its own where the declaration
// gave it one and a positional otherwise.
//
// A declaration may name none of its parameters — `Put(context.Context,
// Value)` is legal — and a forwarding body has to call them something.
func forwardName(i int, p golang.Param) string {
	if p.Name != "" && p.Name != "_" {
		return p.Name
	}
	return "a" + strconv.Itoa(i)
}

// anonReturns is the result list with no names, for a body that panics
// and therefore never returns.
func anonReturns(sig *golang.Sig) []*sdk.EmitReturn {
	if sig == nil {
		return nil
	}
	out := make([]*sdk.EmitReturn, 0, len(sig.Returns))
	for _, r := range sig.Returns {
		out = append(out, &sdk.EmitReturn{Type: r.Type})
	}
	return out
}

// namedReturns is the result list with a name on every slot, which is
// what lets a defect body answer without spelling a zero.
//
// A bare `return` under named results yields each slot's zero value,
// whatever its type — so a defect that has to answer success needs no
// literal, no sample, and no import for a type it might not be able to
// name. The error slot takes [errLocal] so a body can plant a failure
// into it by name; the rest take the identifier the signature projection
// already chose for them.
func namedReturns(sig *golang.Sig) []*sdk.EmitReturn {
	if sig == nil {
		return nil
	}
	out := make([]*sdk.EmitReturn, 0, len(sig.Returns))
	for _, r := range sig.Returns {
		name := r.Local
		if r.Error {
			name = errLocal
		}
		out = append(out, &sdk.EmitReturn{Name: name, Type: r.Type})
	}
	return out
}
