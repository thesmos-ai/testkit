// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/subject"
)

// Companion is the generated proof of the emission, routed to the test
// output: the derived reference passes its own tier, and every inert body
// answers zero values rather than panicking.
//
// It certifies what the generator produced, not any consumer's subject —
// which is why it is generated beside the bindings instead of asked of the
// package that arms them.
type Companion struct {
	sdk.BaseEmit
	subject.Subject

	// PropertyName, RefCtorName and ReferenceOptionName reach the bindings
	// across the package boundary; FixtureCtor supplies the inert calls'
	// arguments. HarnessPkg qualifies all four — see [Companion.SetOutputPackages].
	PropertyName, RefCtorName, ReferenceOptionName, FixtureCtor string
	HarnessPkg                                                  string

	// Saturated marks a harness that emitted the saturation prover, which
	// the companion then holds to the derived reference — the proof ships
	// with the derivation.
	Saturated bool

	// ConcurrentName is the concurrent leg's runner, empty where none
	// derives. The companion holds the leg to the derived reference: a
	// mutex-guarded store is linearizable, so a red run is the wiring's own.
	ConcurrentName string

	// Mutants is the kill matrix: one row per driven method, each a
	// reference whose one method answers zeros and forwards nothing. The
	// property must fail every row — a mutant that survives means that
	// method's participation in the run checks nothing, which is a hole in
	// this derivation rather than in any consumer's subject.
	Mutants []Mutant

	// LowerIface prefixes the mutant type names.
	LowerIface string
}

// Kind returns [KindCompanion].
func (*Companion) Kind() sdk.Kind { return KindCompanion }

// ModelPkg surfaces the runner's import path to the template.
func (*Companion) ModelPkg() string { return ModelPkg }

// SetOutputPackages records where Layout routed the bindings. The companion
// lands in the external test package and reaches every generated identifier
// through that qualifier; the provisional value is the source package, whose
// failure mode is a compile error naming the symbol rather than a bare name
// silently binding to whatever is in scope.
//
// Layout may pass a partial map — a run that recorded routing errors reaches
// dispatch with tags missing — so absence keeps the provisional value.
func (c *Companion) SetOutputPackages(byTag map[string]string) {
	if path := byTag[""]; path != "" {
		c.HarnessPkg = path
	}
}

// RootPkg surfaces the runtime module's import path to the template, which
// reaches the failure surrogate through it.
func (*Companion) RootPkg() string { return RootPkg }

// Mutant is one row of the companion's kill matrix.
type Mutant struct {
	// Method is the one method the mutant makes inert; Sig spells the
	// override.
	Method string
	Sig    *golang.Sig
}

// CompanionFor derives the companion from the bindings it proves.
//
// Exported for the reason [BindingsFor] is: nothing queues it now, so a
// call is the only way to reach it. It has no output to render into
// either — the proofs for this tier's rows belong in the companion the
// harness generator emits, beside the proofs for the rows they sit with,
// and that is a contribution this tier does not make yet.
func CompanionFor(c *sdk.Provenance, iface *sdk.Interface, b *Bindings) *Companion {
	comp := &Companion{
		BaseEmit:            sdk.EmitBaseTagged(sdk.EmitBase(c, iface), GoTestOutputTag),
		Subject:             b.Subject,
		PropertyName:        b.PropertyName,
		RefCtorName:         b.Reference.CtorName,
		ReferenceOptionName: b.IfaceName + "ModelReference",
		FixtureCtor:         b.FixtureCtor,
		HarnessPkg:          iface.Package,
	}
	if b.Concurrent() {
		comp.ConcurrentName = b.IfaceName + "ModelConcurrent"
	}
	comp.Saturated = len(b.SatLaws) > 0
	comp.LowerIface = strings.ToLower(b.IfaceName[:1]) + b.IfaceName[1:]
	// One kill-matrix row per driven method: the coherence rule already
	// guarantees each has a live adapter op, so its inertness is observable —
	// by the comparison that reads it, or by the read that follows it.
	sigs := map[string]*golang.Sig{}
	for _, am := range b.Adapter {
		sigs[am.Sig.Name] = am.Sig
	}
	for _, a := range b.Actions {
		comp.Mutants = append(comp.Mutants, Mutant{Method: a.Method, Sig: sigs[a.Method]})
	}
	return comp
}
