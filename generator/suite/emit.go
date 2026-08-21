// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"sort"
	"strconv"
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"

	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// CheckEmit is one derived check as its template renders it.
//
// The node the backend dispatches on: its kind IS the body variant's
// template name, so an unrendered variant fails by name rather than
// emitting nothing. The plan says what the check asserts; the fields
// beside it say how this file spells it, and every one of them is a
// fact about the METHOD rather than about the check — which is why
// they are computed here, where the method is still in hand, instead
// of being carried through the projection.
type CheckEmit struct {
	sdk.BaseEmit
	bodyView

	// Plan is the derived check itself: identity, class, claim, binds.
	// The rows read it; the body templates read the view above.
	Plan projection.CheckPlan

	// Proven says this row claims to have been shown able to fail,
	// which it does exactly when the proofs companion beside it can
	// spell the defect its plan carries.
	//
	// Not simply the plan's own stamp. A deriver proves a claim by
	// naming the defect that breaks it, and this file proves it by
	// EMITTING that defect — so a plan stamped Proven whose variant no
	// template renders would ship a claim with no evidence, and the
	// parity gate would report it against the generated companion
	// rather than against the generator that left it out.
	Proven bool

	// provable records whether this run could have planted evidence,
	// which decides which of the arguments a downgraded row gives.
	provable bool

	// sampleless records that the defect's template exists and this
	// method's result type defeated it — set by the proofs walk, which
	// is where the sampler runs.
	sampleless bool

	// The spellings the row needs, resolved once here rather than
	// composed in the template: a row naming an accessor its own index
	// does not declare is a compile error a consumer meets.
	accessor, assertName, classConst string
}

// Argument is what a downgraded row says about itself, empty where the
// row is Proven and needs none.
//
// The Argued stamp demands a reason, and the honest one here is not a
// fact about the claim — it is that this generator has not spelled the
// defect yet. Saying so keeps a reader from reading "argued" as "nothing
// could falsify this", which is what the state means everywhere else.
func (c *CheckEmit) Argument() string {
	switch {
	case c.Proven:
		return ""
	case c.Plan.Defect == nil:
		// The deriver's own argument, which is a fact about the claim
		// and belongs to it rather than to this file's coverage.
		return c.Plan.Falsifiable.Why
	case !c.provable:
		return "a planted defect for a generic subject has to be built at " +
			"concrete types and a Go test function cannot name them, so nothing " +
			"has driven this claim"
	case c.sampleless:
		return "the " +
			strings.TrimPrefix(string(c.Plan.Defect.DefectKind()), projection.DefectKindPrefix) +
			" defect has to answer a live value and no sample of this " +
			"method's result could be derived, so this run plants no evidence for the claim"
	default:
		return "no defect template spells " +
			strings.TrimPrefix(string(c.Plan.Defect.DefectKind()), projection.DefectKindPrefix) +
			" yet, so this run plants no evidence for the claim"
	}
}

// Group is the index member this check sits under, so a row names it
// through the same tree a consumer drops it through.
func (c *CheckEmit) Group() string { return golang.ExportedName(c.Plan.ID.Method) }

// Accessor is this check's entry point within that member.
func (c *CheckEmit) Accessor() string { return c.accessor }

// AssertName is the identifier of the function carrying this check.
func (c *CheckEmit) AssertName() string { return c.assertName }

// ClassConst is the engine identifier the check's class is declared
// under, so the row names the class rather than repeating its slug.
func (c *CheckEmit) ClassConst() string { return c.classConst }

// StrengthConst is the engine identifier for how far this check looks
// before passing, so the generated file names the constant rather than
// repeating the string it stands for.
func (c *CheckEmit) StrengthConst() string {
	switch c.Plan.Body.Strength() {
	case vocab.StrengthDifferential:
		return "StrengthDifferential"
	case vocab.StrengthObserved:
		return "StrengthObserved"
	default:
		return "StrengthErrorOnly"
	}
}

// Kind returns the plan's body variant, which is its template's name.
func (c *CheckEmit) Kind() sdk.Kind { return sdk.Kind(c.Plan.Body.BodyKind()) }

// RenderedBodyKinds is the body variants whose templates exist today.
//
// A body with no template fails the backend's dispatch by name, which
// is the guard working — so the rows carry only what can be rendered,
// and [WithheldBodies] names the rest in the generated file rather than
// letting a reader infer coverage from silence.
// RenderedBodyKinds is the set of body shapes THIS tier spells a
// template for.
//
// Exported so the composition root can hold the two tiers' answers to
// the whole declared set: a kind no tier renders is a row planned and
// emitted nowhere, and a kind both render is one check emitted twice
// under one identity. Neither is visible from inside either generator.
func RenderedBodyKinds() map[projection.BodyKind]bool {
	return map[projection.BodyKind]bool{
		projection.KindSmokeSurvives:   true,
		projection.KindGuardedCall:     true,
		projection.KindZeroOnMiss:      true,
		projection.KindZeroOnCancel:    true,
		projection.KindRepeatProbe:     true,
		projection.KindReportsSentinel: true,
		projection.KindAnswersZero:     true,
		projection.KindHitProbe:        true,
		projection.KindCountProbe:      true,
		projection.KindHookFires:       true,
		projection.KindNonZeroAnswer:   true,
		projection.KindPartnerAgrees:   true,
		projection.KindReadActRead:     true,
		projection.KindWriteWriteRead:  true,
	}
}

// checkEmitsOf pairs every plan with the method it is about.
//
// The pairing is the seam the whole emission turns on. A plan names its
// method as a string, deliberately — the projection is unit-testable
// without the pipeline and holds no source node — so the facts a body
// needs from the signature are resolved here, once per check, against
// the method set the run already projected.
//
// A plan whose method is not in the set is dropped rather than emitted
// half-spelled: it can only come from a deriver naming something the
// interface does not declare, and a call to a method that is not there
// fails in the consumer's build rather than in this run. Family-scoped
// plans name no method and carry no body of ours, so they are not here
// at all.
//
// provable says this run can plant evidence at all, which a generic
// subject cannot: a Go test function takes no type parameters, so its
// companion carries a note in place of proofs and no row it stamps
// Proven would have anything behind it.
//
// seeded says a corpus exists to judge. The two seeded probes read back
// what a run put in, and an interface with a writer populates itself
// through the surface under test rather than through a corpus — so they
// are derived from the reader shape and rendered only where there is
// something for them to have been seeded WITH.
func checkEmitsOf(
	base sdk.BaseEmit, iface Iface, inv projection.Inventory, provable, seeded bool,
) []*CheckEmit {
	byName := make(map[string]subject.Method, len(iface.Methods))
	for _, m := range iface.Methods {
		byName[m.Name] = m
	}

	out := make([]*CheckEmit, 0, len(inv.Checks))
	for _, plan := range inv.Checks {
		if plan.ID.Method == "" {
			continue
		}
		m, found := byName[plan.ID.Method]
		if !found {
			continue
		}
		if !RenderedBodyKinds()[plan.Body.BodyKind()] {
			continue
		}
		if !seeded && seedsCorpusBody(plan.Body) {
			continue
		}
		acc, err := projection.AccessorOf(plan.ID)
		if err != nil {
			// The index refused to name it, so a row naming it would
			// spell a method the index does not declare. IndexOf
			// reports the same refusal by name; this drops quietly
			// rather than saying it twice.
			continue
		}
		class, named := vocab.ClassConst(plan.Class)
		if !named {
			continue
		}
		view := viewOf(iface, m)
		view.Body = plan.Body
		view.Seeds = seedsCorpusBody(plan.Body)
		if miss, ok := plan.Body.(projection.ZeroOnMiss); ok {
			view.Pool = miss.Pool
		}
		if hook, ok := plan.Body.(projection.HookFires); ok {
			// The callback's own signature, which only the registrar's
			// parameter carries — the projection names the registrar and
			// this resolves what it takes.
			if reg, found := byName[hook.Register.Method]; found {
				view.ObserveMethod = reg.Name
				view.HookParams, view.HookReturns = hookSignature(reg)
				view.RegisterDiscard = discardOf(reg)
			}
		}
		if guarded, ok := plan.Body.(projection.GuardedCall); ok {
			view.Guard = string(guarded.Guard)
		}
		if triple, ok := plan.Body.(projection.WriteWriteRead); ok {
			// The reader's name, for the message that has to say which
			// method failed to answer.
			view.ObserveMethod = triple.Read.Method
		}
		if agree, ok := plan.Body.(projection.PartnerAgrees); ok {
			// The validator's own name, for the message that has to
			// report which verdict disagreed with which.
			view.ObserveMethod = agree.Partner.Method
		}
		if pair, ok := plan.Body.(projection.ReadActRead); ok {
			// The partner's own name, for the two messages that report
			// an unreadable observer: "must be readable" without saying
			// WHICH method sends a reader to the wrong one.
			view.ObserveMethod = pair.Observe.Method
		}
		if probe, ok := plan.Body.(projection.ReportsSentinel); ok {
			view.Sentinel = sentinelRef(iface, string(probe.Sentinel))
		}
		out = append(out, &CheckEmit{
			BaseEmit: base,
			bodyView: view,
			Plan:     plan,
			provable: provable,
			// Provisional: a defect variant with a template. Whether this
			// run can actually WRITE it depends on the method's result
			// type for two of them, which only the view knows — proofsOf
			// downgrades the row where it cannot.
			Proven:     provable && plan.Defect != nil && defectRendered()[plan.Defect.DefectKind()],
			accessor:   acc.Name,
			assertName: projection.AssertName(iface.Token, plan.ID.Method, plan.ID.Seg),
			classConst: class,
		})
	}
	return out
}

// viewOf spells the facts a body needs from one method's signature.
func viewOf(iface Iface, m subject.Method) bodyView {
	return bodyView{
		Recv:          receiverIdent(iface),
		Vocab:         Vocab,
		Check:         projection.MethodConst(iface.Token, m.Name),
		Discard:       discardOf(m),
		ErrBind:       errBindOf(m),
		Draws:         len(m.ArgFields) > 0,
		Method:        m.Name,
		ValueBind:     valueBindOf(m),
		ErrStmt:       errStmtOf(m),
		ValueDiscard:  valueDiscardOf(m),
		NeedsCtx:      m.TakesContext(),
		HasErr:        m.ReturnsError(),
		Zeros:         zeroSlotsOf(m),
		ZeroBind:      zeroBindOf(m, true),
		ZeroBindNoErr: zeroBindOf(m, false),
	}
}

// receiverIdent is the local a body calls the subject through.
//
// The interface's own initial, which is what the packs spell — `l Log`,
// `s Store`, `p Pool`. Short because it appears in every call of every
// body and names something the signature beside it already declares.
func receiverIdent(iface Iface) string {
	if iface.Token == "" {
		return "subject"
	}
	return iface.Token[:1]
}

// discardOf drops a call's results where the body only asks whether the
// call returned: one blank per result.
func discardOf(m subject.Method) string {
	n := len(m.Returns)
	if n == 0 {
		return ""
	}
	return strings.Repeat("_, ", n-1) + "_ ="
}

// errBindOf binds the error a body inspects, and is empty where the
// error is the only result — which the packs return directly rather
// than binding to a local used once on the next line.
func errBindOf(m subject.Method) string {
	values := len(m.ValueReturns())
	if values == 0 {
		return ""
	}
	return strings.Repeat("_, ", values) + "err :="
}

// withheldBodies names the variants this inventory derived and no
// template renders yet, sorted, each once.
//
// Emitted into the header rather than logged: a consumer reading a
// short check list deserves to know whether the run derived little or
// spelled little, and those are different problems with different
// owners.
func withheldBodies(inv projection.Inventory) []string {
	seen := map[projection.BodyKind]bool{}
	var out []string
	for _, c := range inv.Checks {
		if c.ID.Method == "" || c.Body == nil {
			continue
		}
		kind := c.Body.BodyKind()
		if RenderedBodyKinds()[kind] || seen[kind] {
			continue
		}
		seen[kind] = true
		out = append(out, strings.TrimPrefix(string(kind), projection.BodyKindPrefix))
	}
	sort.Strings(out)
	return out
}

// stampsUsed reports which of the two row constructors the table binds.
//
// Asked rather than binding both: an alias nothing calls is a compile
// error in a file a consumer cannot edit, and both ends of the split are
// reachable — an interface whose every derived check carries a spelled
// defect needs no Argued, and one whose checks are all waiting on a
// defect template needs no Proven.
func stampsUsed(checks []*CheckEmit) (proven, argued bool) {
	for _, c := range checks {
		if c.Proven {
			proven = true
		} else {
			argued = true
		}
	}
	return proven, argued
}

// seedsCorpusBody reports the two variants that judge the seeded set
// whole rather than drawing one member from the fixture.
func seedsCorpusBody(b projection.Body) bool {
	switch b.(type) {
	case projection.HitProbe, projection.CountProbe:
		return true
	default:
		return false
	}
}

// seedsCorpus reports whether any check reads the run's corpus, which
// decides whether the builder takes one — the rows close over it, so a
// body that judges the seeded set cannot reach a corpus the builder was
// never handed.
func seedsCorpus(checks []*CheckEmit) bool {
	for _, c := range checks {
		if c.Seeds {
			return true
		}
	}
	return false
}

// drawsFixture reports whether any check reads the run's fixture, which
// is what decides the builder's own parameter: the rows are closures
// over it, so one that draws cannot reach a fixture the builder was
// never handed.
func drawsFixture(checks []*CheckEmit) bool {
	for _, c := range checks {
		if c.Draws {
			return true
		}
	}
	return false
}

// valueBindOf binds a call's results where the body judges the first of
// them: `got, err :=`, one blank per result between.
func valueBindOf(m subject.Method) string {
	values := len(m.ValueReturns())
	if values == 0 {
		return ""
	}
	return "got" + strings.Repeat(", _", values-1) + ", err :="
}

// valueDiscardOf blanks the results after the first, for a body judging
// one value from a method that reports no error.
//
// Empty where the method answers nothing: no body judges a result that
// does not exist, and the count would otherwise go negative.
func valueDiscardOf(m subject.Method) string {
	if len(m.Returns) < 2 {
		return ""
	}
	return strings.Repeat(", _", len(m.Returns)-1)
}

// sentinelRef resolves a declared sentinel to the reference a body
// names it through.
//
// The annotator hands the parameter back QUALIFIED — which is right for
// identity and wrong for a call site, as [subject.Method.MixinParam] says — so
// the qualifier is split back off and handed to the backend, which is
// what registers the import. A bare name means the interface's own
// package, which is where the declaration wrote it.
func sentinelRef(iface Iface, declared string) *sdk.Expr {
	pkg, symbol := iface.Package, declared
	if i := strings.LastIndex(declared, "."); i >= 0 {
		pkg, symbol = declared[:i], declared[i+1:]
	}
	return sdk.NewExternal(pkg, symbol)
}

// errStmtOf binds the error inside an if-statement's init, where a body
// judges it in the condition rather than after it.
func errStmtOf(m subject.Method) string {
	return strings.Repeat("_, ", len(m.ValueReturns())) + "err :="
}

// zeroSlotsOf spells every value result a zero-judging body holds to
// its own zero.
//
// Every one, not the first: a read answering a value beside metadata
// can zero one and leak the other, and a caller who was told the read
// failed has been handed state anyway. The two bodies render
// identically, so only a subject that leaks a later slot tells them
// apart — which is why this list exists rather than a single shape.
func zeroSlotsOf(m subject.Method) []zeroSlot {
	values := m.ValueReturns()
	out := make([]zeroSlot, 0, len(values))
	for i, ret := range values {
		src := ret.Source
		slot := zeroSlot{Bind: zeroBindIdent("got", i), Zero: zeroBindIdent("zero", i), Nil: zeroIsNil(src)}
		switch {
		case slot.Nil:
			// nil needs no type spelled.
		case src == nil || src.Name == "":
			// Nothing to declare a zero of; the body cannot judge it.
			continue
		case golang.IsPredeclared(src.Name):
			slot.Word = src.Name
		default:
			slot.Type = sdk.NewExternal(src.Package, src.Name)
		}
		if len(values) > 1 {
			slot.Label = zeroSlotLabel(i, src)
		}
		out = append(out, slot)
	}
	return out
}

// zeroBindIdent names one slot's local under the given stem. The first
// keeps the bare name every single-value body has always used, so
// widening this to a list left the common output byte for byte where it
// was; the rest are numbered from two, as a reader counting slots would.
func zeroBindIdent(stem string, i int) string {
	if i == 0 {
		return stem
	}
	return stem + strconv.Itoa(i+1)
}

// zeroSlotLabel names a slot in a failure message, by its own type
// where it has a name and by position where it does not.
func zeroSlotLabel(i int, src *node.TypeRef) string {
	if src != nil && src.Name != "" {
		return src.Name
	}
	return "result " + strconv.Itoa(i+1)
}

// zeroIsNil reports the comparability split: a slice, map, func or
// pointer has no name to declare a zero of, and nil is what the
// language calls its zero anyway.
func zeroIsNil(src *node.TypeRef) bool {
	if src == nil {
		return false
	}
	switch src.TypeKind {
	case node.TypeRefSlice, node.TypeRefMap, node.TypeRefFunc, node.TypeRefPointer:
		return true
	default:
		// A channel arrives as a named ref with the frontend's own stamp
		// on it, never as a kind of its own.
		return golang.IsChannel(src)
	}
}

// zeroBindOf binds every value slot, and the error beside them when the
// method reports one.
//
// Distinct from [valueBindOf], which blanks past the first: the seeded
// probes judge one answer against one seeded value, and binding a slot
// they never read would not compile.
func zeroBindOf(m subject.Method, withErr bool) string {
	slots := zeroSlotsOf(m)
	if len(slots) == 0 {
		return ""
	}
	binds := make([]string, 0, len(m.Returns))
	next := 0
	for _, ret := range m.Returns {
		if ret.Error {
			continue
		}
		if next < len(slots) && slots[next].Bind == zeroBindIdent("got", next) {
			binds = append(binds, slots[next].Bind)
			next++
			continue
		}
		binds = append(binds, "_")
	}
	if withErr {
		binds = append(binds, "err")
	}
	return strings.Join(binds, ", ") + " :="
}

// hookSignature spells the callback the registrar takes: its parameters
// blanked and its results named.
//
// Blank parameters because the recording closure reads none of them —
// what it records is that it ran at all. Named results because a bare
// return then answers every slot's zero, whatever the callback's
// signature, without this generator having to name a type it may not be
// able to spell.
func hookSignature(register subject.Method) ([]*sdk.EmitParam, []*sdk.EmitReturn) {
	fn := callbackParam(register)
	if fn == nil {
		return nil, nil
	}
	params := make([]*sdk.EmitParam, 0, len(fn.FuncParams))
	for _, p := range fn.FuncParams {
		params = append(params, &sdk.EmitParam{Name: "_", Type: golang.FromNode(p)})
	}
	returns := make([]*sdk.EmitReturn, 0, len(fn.FuncReturns))
	for i, r := range fn.FuncReturns {
		returns = append(returns, &sdk.EmitReturn{
			Name: "r" + strconv.Itoa(i),
			Type: golang.FromNode(r),
		})
	}
	return params, returns
}

// firstValueSource is the result a zero-on-error body judges.
func firstValueSource(m subject.Method) *node.TypeRef {
	values := m.ValueReturns()
	if len(values) == 0 {
		return nil
	}
	return values[0].Source
}

// emittedIDs renders every identity this run declares, sorted.
//
// Sorted because the listing it feeds is read by a human diffing two
// generations, and derivation order is an implementation detail that
// would make an unrelated reordering look like a change in coverage.
// An unrenderable plan is reported rather than skipped: the listing
// claims to be every ID, and one silently short is worse than none.
func emittedIDs(inv projection.Inventory) ([]string, error) {
	out := make([]string, 0, len(inv.Checks))
	for _, c := range inv.Checks {
		id, err := c.ID.Render()
		if err != nil {
			return nil, err
		}
		out = append(out, string(id))
	}
	sort.Strings(out)
	return out, nil
}

// emittedPlans is the plan behind every check this run renders.
//
// The seam between "what the derivers licensed" and "what this file
// contains". Three projections read it — the index, the ID listing and
// the lock — and each was reading the inventory instead, which is the
// wider set: every one of them named checks the file does not emit.
func emittedPlans(checks []*CheckEmit) []projection.CheckPlan {
	out := make([]projection.CheckPlan, 0, len(checks))
	for _, c := range checks {
		out = append(out, c.Plan)
	}
	return out
}
