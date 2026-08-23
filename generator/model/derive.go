// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"strings"

	"go.thesmos.sh/eidos/sdk"

	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
	"go.thesmos.sh/testkit/generator/suite"
)

// Generate queues one set of bindings per interface carrying the directive.
func (*Plugin) Generate(ctx *sdk.GeneratorContext) error {
	c := sdk.NewProvenance(Name)
	// The lookup names the harness generator's emit value, because that
	// is what the store holds. Nothing below this line does: each
	// derivation takes the projection the harness embeds, so a change to
	// the harness's own shape reaches exactly one line of this tier.
	harnesses := sdk.PendingByOrigin[*suite.Contract](ctx.Store.Emit())
	// The falsification companion, queued beside each harness. A row this
	// tier stamps Proven owes a defect there, and the parity gate holds
	// the two to each other at run time.
	proofs := sdk.PendingByOrigin[*suite.Proofs](ctx.Store.Emit())
	// The property alias is a package-level type, and a package can hold
	// more than one interface carrying the directive — the corpus has
	// three. Declared once per output package, by whichever interface
	// reaches it first: a second declaration is a compile error over
	// generated code the consumer may not edit, and naming it per
	// interface would make a consumer spell which interface's property
	// state they meant in a signature that already says.
	aliased := map[string]bool{}

	for _, iface := range ctx.Reader.Interfaces().Slice() {
		if !iface.HasPositiveDirective(DirectiveName) {
			continue
		}
		harness, hosted := harnesses[sdk.Node(iface)]
		if !hosted {
			// Every generated identifier comes from the projection, so there
			// is nothing to bind onto. Asked and impossible.
			ctx.Diag.Errorf(iface.Pos(),
				"%s: interface %q carries //testkit:%s but no harness exists for it; "+
					"add //testkit:%s",
				Name, iface.Name, DirectiveName, suite.DirectiveName)
			continue
		}
		b, ok := BindingsFor(ctx, c, iface, harness)
		if !ok {
			continue
		}
		// The field a clocked check needs, and the line that carries it
		// onto the runtime subject: one contribution in two regions,
		// because a field nothing lowers is somewhere for a consumer to
		// write a value that goes nowhere.
		//
		// The harness generator renders both without reading them. It
		// emits the surface; which capabilities exist is a fact about the
		// checks written here.
		fields, lowerings := doorsFor(b, harness.SubjectType())
		for _, f := range fields {
			if err := harness.HarnessFields().Append(f, c.Provenance(Name+".door")); err != nil {
				return fmt.Errorf("%s: contribute a door for %q: %w", Name, iface.Name, err)
			}
		}
		for _, l := range lowerings {
			if err := harness.HarnessLowering().Append(l, c.Provenance(Name+".lowering")); err != nil {
				return fmt.Errorf("%s: lower a door for %q: %w", Name, iface.Name, err)
			}
		}

		// The rows, and the declarations they name. Three contributions
		// and one order: the expression is appended to the run surface,
		// the function it calls is declared beside it, and the bodies
		// that function's rows run — the pools, the reference, the
		// actions — are declared beside that.
		//
		// All three or none. An expression naming a function nobody
		// declared, or a row whose RunWith calls a body that is not
		// there, is a compile error over generated code the consumer
		// cannot edit.
		if why := declineReason(b, harness); len(why) > 0 {
			note := &Declined{BaseEmit: sdk.EmitBase(c, iface), Iface: iface.Name, Why: why}
			if err := harness.Decls().Append(note, c.Provenance(Name+".declined")); err != nil {
				return fmt.Errorf("%s: record the refusal for %q: %w", Name, iface.Name, err)
			}
			continue
		}
		// Before the rows are shaped: this settles which of them carry
		// evidence, and a row it downgrades must not render as Proven.
		plantDefects(ctx, b, proofs[sdk.Node(iface)])

		call := rowCallFor(c, iface, b, harness)
		if call == nil {
			// The tier ran, planned nothing, and would otherwise say so
			// nowhere: the row call is what carries every declaration,
			// including the header naming what did not bind. Meanwhile
			// the harness file tells a reader this interface is "checked
			// by testkit's model tier", which is then untrue with no
			// trace. The note is the honest state.
			note := &Declined{
				BaseEmit: sdk.EmitBase(c, iface),
				Iface:    iface.Name,
				Why:      nothingBoundWhy(b),
			}
			if err := harness.Decls().Append(note, c.Provenance(Name+".declined")); err != nil {
				return fmt.Errorf("%s: record the empty plan for %q: %w", Name, iface.Name, err)
			}
			continue
		}
		if err := harness.Rows().Append(call, c.Provenance(Name+".rows")); err != nil {
			return fmt.Errorf("%s: contribute the rows for %q: %w", Name, iface.Name, err)
		}
		decls := []sdk.EmitNode{
			&Compat{BaseEmit: sdk.EmitBase(c, iface), LegsPkg: LegsPkg},
			rowDeclFor(c, iface, b, harness), b,
		}
		for _, d := range decls {
			if err := harness.Decls().Append(d, c.Provenance(Name+".decls")); err != nil {
				return fmt.Errorf("%s: declare the rows for %q: %w", Name, iface.Name, err)
			}
		}

		// The property surface a consumer writes their own drawn-input
		// checks through: the alias they name in a signature, the row
		// fields they set, and the dispatch that turns one into a body
		// the runner calls.
		//
		// Last, and beside the declarations rather than before them. A
		// sugared field draws from this tier's own pools, and those are
		// declared by the emit two lines above — so a run that got this
		// far and no further would offer a consumer a field whose draw
		// names a function nobody emitted.
		//
		// Three regions and one contribution, appended together because
		// any two without the third is worse than none: a field with no
		// dispatch is a body somebody writes and nothing calls, and the
		// check it belongs to reports green.
		alias, propFields, dispatch := propFor(b, harness)
		parts := []struct {
			slot *sdk.Slot
			node sdk.EmitNode
			what string
		}{
			{harness.CheckFields(), propFields, "fields"},
			{harness.CheckBodies(), dispatch, "dispatch"},
		}
		if pkg := iface.Package; !aliased[pkg] {
			aliased[pkg] = true
			parts = append(parts, struct {
				slot *sdk.Slot
				node sdk.EmitNode
				what string
			}{harness.Decls(), alias, "alias"})
		}
		for _, part := range parts {
			if err := part.slot.Append(part.node, c.Provenance(Name+".prop")); err != nil {
				return fmt.Errorf("%s: contribute the property %s for %q: %w",
					Name, part.what, iface.Name, err)
			}
		}
	}
	return nil
}

// plantDefects writes the evidence for every row this tier stamped
// Proven into the falsification companion beside the harness.
//
// A row that stamped the claim and cannot spell the defect loses the
// stamp here rather than shipping one: the parity gate refuses a Proven
// check with nothing planted for it, and it reports that against the
// generated package — where a reader would take it for a fault in their
// own code rather than in this generator.
//
// A missing companion is not an error. It means the harness is generic,
// where a Go test function cannot name the types a defect would be built
// at; the rows for such an interface never reached this far anyway.
func plantDefects(ctx *sdk.GeneratorContext, b *Bindings, proofs *suite.Proofs) {
	if proofs == nil {
		return
	}
	for _, o := range b.overrides {
		plan, at := planAt(b, o.ID)
		if plan == nil {
			continue
		}
		// A nil method is not a missing one. Most defects are a method
		// override and name the method they go through; a few are facts
		// about the SUBJECT — a rebuild onto an empty medium, a clock
		// that is accepted and ignored — and there is no method to name.
		// The view's spellings all tolerate an absent signature, and the
		// template for such a defect reaches for none of them.
		var over subject.Method
		if o.Method != nil {
			over = *o.Method
		}
		if proofs.Plant(ctx.Reader, over, *plan, rowAccessor(o.ID)) {
			continue
		}
		// The rule reached it and this run cannot write it out. The row
		// drops to Argued and says which of the two gaps it met.
		b.Rows[at].Falsifiable = vocab.Argued(unspellable)
		b.Rows[at].Defect = nil
	}
}

// planAt is the planned row under this identity, and where it sits.
func planAt(b *Bindings, id projection.IDPlan) (*projection.CheckPlan, int) {
	for i := range b.Rows {
		if b.Rows[i].ID == id {
			return &b.Rows[i], i
		}
	}
	return nil, 0
}

// declineReason is why this tier's rows cannot land on the harness it
// derived against, empty where they can.
//
// Two of them, both about the file rather than about the claims. A generic
// harness is generic all the way through — its run surface, its fixture
// and its checks all carry the interface's type parameters — and this
// tier's derivation is pinned at the witnesses the source named, so the
// rows would be checks at concrete types appended to a set of checks at
// parameters. There is no spelling of that. The second is narrower: the
// pools read the harness's sample inputs, and a harness that derives none
// has nothing to hand them.
//
// Returned rather than diagnosed, because a refusal a reader has to run
// the generator to hear about is one most readers never hear. It renders
// into the generated file where the rows would have been.
func declineReason(b *Bindings, harness *suite.Contract) []string {
	switch {
	case len(harness.TypeParams) > 0:
		return []string{
			"this interface is generic and its checks are generic with it,",
			"while these sequences run at the concrete types " + WitnessKey + "= names.",
			"Checks at concrete types cannot join a check set at type parameters.",
		}
	case b.NeedsFixture() && !harness.DrawsFixture:
		return []string{
			"the sequences draw sample inputs and this harness derives none,",
			"so there is nothing to draw from. The header above names what",
			"each parameter is waiting on.",
		}
	}
	return nil
}

// nothingBoundWhy words an empty plan for the generated header.
//
// Each refusal the derivation already recorded, and a closing sentence
// where it recorded none — because "no rows" with no reasons beside it
// is the state that reads as an oversight and is usually a shape this
// tier has no claim for.
func nothingBoundWhy(b *Bindings) []string {
	why := []string{
		"no claim this tier knows how to state reached this interface,",
		"so it contributes no checks. Each reason below is one it tried:",
	}
	for _, u := range b.Unbound {
		why = append(why, "  "+u.Method+" — "+u.Reason)
	}
	if len(b.Unbound) == 0 {
		why = append(why,
			"  nothing was tried: no law's classification is stamped here, and",
			"  no reference derives from these methods, so there is no sequence",
			"  claim to make. A directive naming a partner method is what opens one.")
	}
	return why
}

// rowCallFor is this tier's contribution to the run surface, nil where
// it planned no row.
//
// Nil rather than an empty call, because an expression yielding nothing
// is a line in the generated file that says this tier ran and found
// nothing to say — which reads as coverage. The header is where an
// absence belongs.
func rowCallFor(
	c *sdk.Provenance, iface *sdk.Interface, b *Bindings, harness *suite.Contract,
) *RowCall {
	if len(b.Rows) == 0 {
		return nil
	}
	fixture := ""
	if drawsFixture(b, harness) {
		fixture = fixtureIdent
	}
	return &RowCall{
		BaseEmit: sdk.EmitBase(c, iface),
		Func:     b.RowsFuncName(),
		Fixture:  fixture,
		Plans:    b.Rows,
	}
}

// BindingsFor derives one interface's model tier, false where it
// reported why it could not.
//
// Exported because the derivation is the tier's substance and nothing
// queues it any more. This generator emits no file: it contributes into
// the harness the other generator emits, so a consumer reads one
// generated file per interface and a claim's rows sit beside the rows
// they are compared against. A queued emit value renders into a file by
// construction, so the way to emit none is to queue none — and then the
// only way to reach the derivation, here or from a test, is to call it.
// It resolves the witnesses too, so a caller reaches the derivation the
// one way. A generic interface with no witness stamp, or a partial one,
// is refused here with the diagnostic that names the gap.
func BindingsFor(
	ctx *sdk.GeneratorContext,
	c *sdk.Provenance,
	iface *sdk.Interface,
	harness *suite.Contract,
) (*Bindings, bool) {
	witnesses, usable := modelWitnesses(ctx, iface)
	if !usable {
		return nil, false
	}
	b, ok := bindingsOf(ctx, c, iface, &harness.Projection, harness.EntryName, witnesses)
	if !ok {
		return nil, false
	}
	// How the harness file spells the things this tier lands beside. Taken
	// rather than derived: a qualified spelling of the subject compiles
	// beside the local one, so a second derivation puts two names for one
	// type in one file and nothing complains.
	// After the pools are derived and while the harness's config plans are
	// still in hand: which pools a run can replace is the harness's fact,
	// and drawing from them rather than from the fixture's pair is this
	// tier's.
	poolFields(ctx, b, harness.Pools)
	b.SubjectSpelling = harness.SubjectType()
	b.FixtureTypeName = harness.Fixture.TypeName + harness.TypeArgs
	b.VeneerVar = projection.VeneerName(iface.Name)
	b.Legs = legsFor(b, harness)
	return b, true
}

func bindingsOf(
	ctx *sdk.GeneratorContext,
	c *sdk.Provenance,
	iface *sdk.Interface,
	harness *subject.Projection,
	entry string,
	witnesses []sdk.Ref,
) (*Bindings, bool) {
	harness, witnessQ := witnessedHarness(harness, iface, witnesses)
	b := &Bindings{
		BaseEmit:       sdk.EmitBase(c, iface),
		Subject:        harness.Subject,
		Methods:        harness.Methods,
		OptionName:     harness.IfaceName + "Model",
		PropertyName:   harness.IfaceName + "ModelProperty",
		OptionTypeName: harness.IfaceName + "ModelOption",
		ConfigName:     strings.ToLower(harness.IfaceName[:1]) + harness.IfaceName[1:] + "ModelConfig",
		EntryName:      entry,
		FixtureCtor:    harness.Fixture.CtorName,
		Witnesses:      witnesses,
		witnessQ:       witnessQ,
	}

	// Classification first, actions second: the values pool is one local, and
	// which writer feeds it decides which writers may draw from it — a second
	// writer taking a different type would draw values no signature accepts.
	partners := partnerMethods(iface)
	var keyed, composite, collector, keyFallback, valueFallback *subject.Method
	var writers []*subject.Method
	for i := range harness.Methods {
		m := &harness.Methods[i]
		if _, partner := partners[m.Name]; partner {
			continue
		}
		switch pseudoShape(m) {
		case shapeReader:
			keyed = m
		case shapeWriter, shapeAnsweringWriter:
			writers = append(writers, m)
		case shapeCompositeWriter:
			composite = m
		case tiers.ShapeCollector:
			collector = m
		case "readernoerror", "readerwithbool", "pointerreader", "multireader",
			"lookup", "batchreader":
			// Key-drawing shapes no oracle reads through: they cannot select
			// a store, but where nothing else supplies the keys pool, their
			// first argument's fixture pair is it.
			if keyFallback == nil {
				keyFallback = m
			}
		case "mutator":
			if valueFallback == nil {
				valueFallback = m
			}
		}
	}
	valued := feederOf(b, keyed, collector, writers)
	valueSrc := valueSourceOf(valued, composite, valueFallback)

	valueQ := ""
	if valueSrc != nil {
		valueQ, _ = b.valueQOf(valueSrc)
	}
	for i := range harness.Methods {
		m := &harness.Methods[i]
		if role, partner := partners[m.Name]; partner {
			b.Skipped = append(b.Skipped, Skip{Method: m.Name, Reason: role})
			continue
		}
		a, skip := actionOf(ctx, b, m)
		if skip != "" {
			b.Skipped = append(b.Skipped, Skip{Method: m.Name, Reason: skip})
			continue
		}
		// Every shape drawing from the values pool is measured against the
		// one method that types it, whatever shape that method is. The pool
		// is a single local of a single type, so a second drawer taking
		// another type reads values no signature of its accepts — and the
		// only place that surfaces is the Go compiler, over generated code
		// the consumer did not write. Refusing here trades a driven method
		// for a named line in the header.
		if a.Pool == poolValues && m != valueSrc {
			if q, _ := b.valueQOf(m); q != valueQ {
				b.Skipped = append(b.Skipped, Skip{
					Method: m.Name,
					Reason: "takes " + spelling(q) + " where the values pool draws " + spelling(valueQ),
				})
				continue
			}
		}
		b.Actions = append(b.Actions, a)
	}

	if len(b.Actions) == 0 {
		ctx.Diag.Errorf(iface.Pos(),
			"%s: no method of %q maps to an action, so the sequences would drive "+
				"nothing; the header lists what each method is waiting on",
			Name, iface.Name)
		return nil, false
	}

	if !referenceOf(ctx, iface, harness, b, keyed, valued, composite, collector, partners) {
		return nil, false
	}
	// The sequences drive only what the oracle models: an action on a method
	// the derived adapter holds inert compares the subject against a body
	// answering zeros, and fails every correct implementation at the first
	// observable answer.
	if b.Reference.Derived() {
		inert := map[string]string{}
		for _, am := range b.Adapter {
			if am.Op == "" {
				inert[am.Sig.Name] = am.Reason
			}
		}
		kept := b.Actions[:0]
		for _, a := range b.Actions {
			if reason, is := inert[a.Method]; is {
				b.Skipped = append(b.Skipped, Skip{
					Method: a.Method,
					Reason: "the derived reference holds it inert — " + reason,
				})
				continue
			}
			kept = append(kept, a)
		}
		b.Actions = kept
	}
	// A keyed contract's roles draw keys — an acquire's argument is the
	// lease's key — and the actions were composed before the derivation
	// could say so.
	for _, a := range b.Actions {
		if b.contractKeyedRoles[a.Method] && a.Shape == shapeWriter {
			a.Pool = poolKeys
		}
	}
	// The declaration's miss identity, armed on every error-answering
	// reader: the actions were composed before the reference resolved it,
	// and the identity agreed on here is the same one the derived oracle's
	// constructor consumes — a subject answering a private error where the
	// declaration stamped a sentinel stops reading as agreement.
	if sym := b.Reference.MissSym; sym != nil {
		for _, a := range b.Actions {
			switch a.Shape {
			case shapeReader, shapeMultiReader, shapeBatchReader:
				a.Sentinel = sym
			}
		}
	}
	contractActionsOf(b, harness)
	// The oracle derivation sees only the canonical reader and writer; the
	// pools serve every drawing action, so their sources widen to the
	// fallbacks where the canonical shapes are absent. The value side was
	// resolved before the actions were built, because the mismatch guard
	// above measures against it.
	keySrc := keyed
	if keySrc == nil {
		keySrc = keyFallback
	}
	if keySrc == nil {
		keySrc = b.contractKeySrc
	}
	genFunc, _ := directiveValue(iface, GenKey)
	if strings.Contains(genFunc, ".") {
		ctx.Diag.Errorf(iface.Pos(),
			"%s: %s=%q on %q carries a qualifier; name a generator constructor "+
				"in the routed output package",
			Name, GenKey, genFunc, iface.Name)
		return nil, false
	}
	poolsOf(ctx, b, harness, keySrc, valueSrc, composite, genFunc)
	lawsOf(b, harness, partners, keyed)
	saturationOf(b, harness)
	concurrentOf(b, harness, keyed, valued)
	simOf(b, keyed, valued)
	b.FaultSym = faultSymOf(iface.Package, valued)
	// Last, because a row is planned from what bound: the laws decide
	// which legs report, and the reference decides whether there is a
	// differential at all.
	b.Rows = PlanRows(b)
	return b, true
}

// simOf wires the crash-recovery pair: the write whose acknowledgement
// the medium owes across a rebuild, and the read that collects the debt.
//
// Only the wiring. Whether the pair can carry the claim is
// [simLegReason]'s question, kept apart for the reason the concurrent
// leg keeps it apart: a shape is CLASSIFIED here, and whether a leg over
// it can be written is a second question whose answer a reader of the
// generated header is owed in words.
//
// The pair is the interface's own reader and writer — the same two the
// sequential actions drive — so the crash schedule and the ordinary
// legs cannot disagree about what a key is or what a write installs.
func simOf(b *Bindings, keyed, valued *subject.Method) {
	if keyed == nil || valued == nil {
		return
	}
	for _, a := range b.Actions {
		switch a.Method {
		case keyed.Name:
			b.SimReader = a
		case valued.Name:
			b.SimWriter = a
		}
	}
	if b.SimReader == nil || b.SimWriter == nil {
		// Half a pair cannot state the claim: a write with no read owes a
		// debt nothing can collect, and a read with no write collects one
		// nothing incurred.
		b.SimReader, b.SimWriter = nil, nil
	}
}

// concurrentOf wires the linearizability leg where the map derivation holds
// unrefined: the Porcupine keyed-store model speaks reader and writer over
// one key at a time, and a claim that changes what a read means — the sticky
// pin — is a different model, not a different wiring. The leg reuses the
// sequential actions, so both legs draw from the same pools and spell the
// same closures; concurrency that never collides checks nothing, which is
// the mistake the shared pools exist to rule out.
//
// A keyless fold is not here on purpose: its state is one accumulation, so
// no partition derives, and a commutative or associative fold is
// order-insensitive by its own claim — linearizability over an operation
// whose order is unobservable checks close to nothing, and the claims that
// do bite are already bound as sequential laws.
func concurrentOf(b *Bindings, harness *subject.Projection, keyed, valued *subject.Method) {
	// The lease leg: acquire and release over the shared keys pool, checked
	// against the lease-table model — the same op vocabulary the model
	// switches on, and the same lenient release the oracle speaks.
	if b.concAcquireName != "" && b.UsesKeys() {
		for _, a := range b.Actions {
			switch a.Method {
			case b.concAcquireName:
				b.ConcAcquire = a
			case b.concReleaseName:
				b.ConcRelease = a
			}
		}
		if b.ConcAcquire != nil && b.ConcRelease != nil {
			b.ConcFamily = concFamilyLease
			return
		}
		b.ConcAcquire, b.ConcRelease = nil, nil
	}
	// The session leg: the same reader/writer interleaving, checked by the
	// per-client laws over the multi-client trace rather than by Porcupine —
	// a store-assigned version defeats the KV model's value equality, so the
	// model stays stepless and the laws carry the run.
	if b.Session != nil && keyed != nil && valued != nil {
		for _, a := range b.Actions {
			switch a.Method {
			case keyed.Name:
				b.ConcReader = a
			case valued.Name:
				b.ConcWriter = a
			}
		}
		if b.ConcReader != nil && b.ConcWriter != nil {
			b.ConcFamily = concFamilySession
			return
		}
		b.ConcReader, b.ConcWriter = nil, nil
	}
	// The cas leg: the version-guarded write against the cell model, in the
	// live oracle's own dialect — stamp is seen+1, an empty cell matches
	// only the zero version. Only the shipped VersionedCell derives it,
	// because the model matches the stamped mismatch identity the same
	// constructor consumes.
	if b.Reference.Oracle == OracleContract && b.Reference.ContractStore == "VersionedCell" &&
		b.Reference.VersionField != "" {

		var w, r *Action
		for _, a := range b.Actions {
			switch a.Shape {
			case shapeCASWriter:
				w = a
			case shapeAggregator:
				r = a
			}
		}
		if w != nil && r != nil && w.Pool != "" {
			b.ConcWriter, b.ConcReader = w, r
			b.ConcFamily = concFamilyCAS
		}
		return
	}
	// The append leg: offset-answering appends into the one shared history.
	// The monotonic-offsets law states the claim per client; this leg states
	// it across them, which is where a torn append hides.
	if a, entry := appendActionOf(b, harness); a != nil {
		b.ConcWriter = a
		b.ConcEntry = entry
		b.ConcFamily = concFamilyAppend
		return
	}
	if b.Reference.Oracle != OracleMap || !b.Reference.Derived() || b.Reference.Pins || keyed == nil || valued == nil {
		return
	}
	for _, a := range b.Actions {
		switch a.Method {
		case keyed.Name:
			b.ConcReader = a
		case valued.Name:
			b.ConcWriter = a
		}
	}
	if b.ConcReader == nil || b.ConcWriter == nil {
		// Half a pair drives nothing Porcupine can order; the leg derives
		// whole or not at all.
		b.ConcReader, b.ConcWriter = nil, nil
		return
	}
	b.ConcFamily = concFamilyKV
}
