// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"strings"

	"go.thesmos.sh/eidos/sdk"

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
		witnesses, usable := modelWitnesses(ctx, iface)
		if !usable {
			continue
		}

		b, ok := bindingsOf(ctx, c, iface, &harness.Projection, harness.EntryName, harness.Inventory, witnesses)
		if !ok {
			continue
		}
		// The companion rides with the bindings rather than being optional:
		// an emission nothing proves is the hand-written probe this output
		// replaced, owed again by every armed package. The exceptions are the
		// references this plugin did not derive — a supplied constructor is
		// the consumer's to prove, and the twin floor is the subject's own
		// factory, which the contract run itself drives.
		queued := []sdk.EmitNode{b}
		if b.Reference.Derived() {
			queued = append(queued, companionOf(c, iface, b))
		}
		if err := sdk.QueueEmit(ctx.Store.Emit(), c, SlotName, iface, queued...); err != nil {
			return fmt.Errorf("%s: queue interface %q: %w", Name, iface.Name, err)
		}
	}
	return nil
}

func bindingsOf(
	ctx *sdk.GeneratorContext,
	c *sdk.Provenance,
	iface *sdk.Interface,
	harness *subject.Projection,
	entry string,
	inv projection.Inventory,
	witnesses []sdk.Ref,
) (*Bindings, bool) {
	harness, witnessQ := witnessedHarness(harness, iface, witnesses)
	b := &Bindings{
		BaseEmit:       sdk.EmitBase(c, iface),
		Subject:        harness.Subject,
		OptionName:     harness.IfaceName + "Model",
		PropertyName:   harness.IfaceName + "ModelProperty",
		OptionTypeName: harness.IfaceName + "ModelOption",
		ConfigName:     strings.ToLower(harness.IfaceName[:1]) + harness.IfaceName[1:] + "ModelConfig",
		EntryName:      entry,
		Rows:           ModelRows(inv),
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
	return b, true
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
