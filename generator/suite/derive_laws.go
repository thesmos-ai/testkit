// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"slices"

	"go.thesmos.sh/testkit/core/lawid"
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// Laws plans the model tier's law rows for the inventory: which laws
// the interface earns, on which legs, under which claims — the lock
// and index rows, not the bindings. The field manifests, options and
// unbound refusal texts stay the model plugin's; the corpus locks
// hold the two derivations equal until the model plugin reads these
// plans.
//
// Registry-general by construction, and nothing law-specific lives
// here: selection is [tiers.Select] over the live catalogue plus the
// extra-rules table, leg and class come from [tiers.LegOf], wording
// from [lawid.ClaimOf] filled generically from the carriers' own
// stamps. An own-leg law without a wording, or a wording naming
// something no stamp supplies, refuses by name — the conformance
// corpus surfaces every such gap the day a fixture stamps the
// classification.
type Laws struct{}

// Name implements [Deriver].
func (Laws) Name() DeriverName { return DeriverLaws }

// Derive implements [Deriver].
func (Laws) Derive(f Iface) ([]projection.CheckPlan, []Refusal) {
	var plans []projection.CheckPlan
	var refusals []Refusal
	var bundle []projection.Bind

	for _, s := range selectLaws(f) {
		bind := projection.Bind{Law: s.Law, Probes: s.Probes}
		class, own := tiers.LegOf(s.Law)
		if !own {
			// A bundled law renders under the bundle's one claim; its
			// identity is the binds column's, so it needs no wording
			// of its own.
			bundle = append(bundle, bind)
			continue
		}
		template, worded := lawid.ClaimOf(s.Law)
		if !worded {
			refusals = append(refusals, Refusal{
				Deriver: DeriverLaws,
				What:    s.Law + " for " + f.Name,
				Why:     "the law rides its own leg but its claim is unworded",
				Remedy:  "word it in lawid's claim table",
			})
			continue
		}
		claim, err := template.Fill(fillsFor(f, s.carriers)...)
		if err != nil {
			refusals = append(refusals, Refusal{
				Deriver: DeriverLaws,
				What:    s.Law + " for " + f.Name,
				Why:     err.Error(),
				Remedy:  "declare the name the claim speaks on the selecting stamp",
			})
			continue
		}
		plans = append(plans, lawRow(f, class, claim, bind))
	}

	if slices.ContainsFunc(f.Methods, func(m subject.Method) bool { return m.HasMixin(MixinConcurrent) }) {
		// The one non-law leg row: linearizability runs the linearize
		// engine, not a law binding, so it has a segment instead of a
		// lawid and the suite's own wording policy speaks it.
		plan := projection.CheckPlan{
			ID: projection.IDPlan{
				Family:    vocab.FamilyModel,
				Qualifier: f.Qualifier,
				Seg:       vocab.SegLinearizable,
			},
			Class: vocab.ClassConcurrent,
			Claim: LinearizableClaim(),
			Body:  projection.LawLeg{},
		}
		defect, proven := observationDefect(f)
		plans = append(plans, proveOrArgue(plan, defect, proven))
	}

	if len(bundle) > 0 {
		plan := projection.CheckPlan{
			ID:    projection.IDPlan{Family: vocab.FamilyModel, Qualifier: f.Qualifier, Seg: vocab.SegLaws},
			Class: vocab.ClassLaws,
			Claim: BundleClaim(chainShaped(f)),
			Body:  projection.LawLeg{Laws: bundle},
			Binds: bundle,
		}
		defect, proven := observationDefect(f)
		plans = append(plans, proveOrArgue(plan, defect, proven))
	}
	return plans, refusals
}

// argueProofsPending is the interim falsifiability every law row
// carries: the per-family planted-defect rules are the proofs
// deriver's, and an underived Proven is the one thing worse than an
// argued row.
const argueProofsPending = "the planted-defect rule for this law family lands with the proofs deriver"

// lawRow builds one own-leg plan; the bind is spelled once and feeds
// both the body and the lock's binds column. The proof rules flip the
// row to Proven where one plants a defect; the residue stays Argued.
func lawRow(f Iface, class vocab.Class, claim string, bind projection.Bind) projection.CheckPlan {
	binds := []projection.Bind{bind}
	plan := projection.CheckPlan{
		ID:    projection.IDPlan{Family: vocab.FamilyModel, Qualifier: f.Qualifier, Seg: bind.Law},
		Class: class,
		Claim: claim,
		Body:  projection.LawLeg{Laws: binds},
		Binds: binds,
	}
	defect, proven := lawDefect(f, bind.Law)
	return proveOrArgue(plan, defect, proven)
}

// lawSelection is one law the interface earns: the carriers that
// selected it, and — for the mixin axis's multi-carrier stamps — the
// probe set those carriers arm.
type lawSelection struct {
	Law      string
	Probes   []string
	carriers []subject.Method
}

// selectLaws runs the tiers catalogue over every method's whole
// classification set — the same [tiers.Select] the model tier binds
// from, so the lock and the model file cannot disagree about what was
// earned — plus the extra-rules table for facts the classification
// axes cannot see. One row per law: a contract stamp riding every
// role method re-selects the same rule rather than duplicating rows.
//
// One exclusion, doctrine: a rule whose selecting mixin this
// package's own tables voice is the suite's claim, not a law row
// (kv's Close/idempotent has no model twin).
func selectLaws(f Iface) []lawSelection {
	var out []lawSelection
	index := map[string]int{}
	record := func(law string, m subject.Method, probe bool) {
		i, held := index[law]
		if !held {
			index[law] = len(out)
			out = append(out, lawSelection{Law: law})
			i = len(out) - 1
		}
		out[i].carriers = append(out[i].carriers, m)
		if probe {
			out[i].Probes = append(out[i].Probes, m.Name)
		}
	}

	for _, m := range f.Methods {
		classifications := m.Classifications()
		if len(classifications) == 0 {
			continue
		}
		params := subject.LawParams(f.Methods, m)
		for _, r := range tiers.Select(classifications, params) {
			if suiteTabled(r) {
				continue
			}
			record(r.Law, m, mixinSelected(r, m))
		}
	}
	for i := range out {
		// A single carrier probes implicitly; contract-selected laws
		// probe their roles, never their carriers, and record none.
		if len(out[i].Probes) < 2 {
			out[i].Probes = nil
		}
	}
	return out
}

// mixinSelected reports whether the rule reaches this carrier through
// a mixin — the axis whose multi-carrier stamps become probe sets.
func mixinSelected(r tiers.Rule, m subject.Method) bool {
	return slices.ContainsFunc(r.Needs, func(need string) bool {
		return slices.Contains(m.Mixins, need)
	})
}

// suiteTabled reports a rule whose selecting mixin this package's own
// stamp tables already voice — one tier owns each claim, and the
// census's tabled bucket is the authority.
func suiteTabled(r tiers.Rule) bool {
	rules := mixinRules()
	return slices.ContainsFunc(r.Needs, func(need string) bool {
		_, tabled := rules[need]
		return tabled
	})
}

// fillsFor resolves the claim placeholder vocabulary from the law's
// own carriers, first-stamped wins. Over-supplying is free — an
// absent placeholder ignores its pair — which is what keeps this
// generic: no law names its fills, the stamps do.
func fillsFor(f Iface, carriers []subject.Method) []string {
	pairs := []string{lawid.PlaceSubject, f.Token}
	seen := map[string]bool{lawid.PlaceSubject: true}
	set := func(place, v string) {
		if v != "" && !seen[place] {
			seen[place] = true
			pairs = append(pairs, place, v)
		}
	}
	for _, m := range carriers {
		if v, ok := m.MixinParam(MixinAfterClose, MixinAfterCloseClose); ok {
			set(lawid.PlaceClose, v)
		}
		if slices.Contains(m.Contracts, ContractCursor) {
			set(lawid.PlaceClose, m.ContractPartner(ContractCursor, ContractCursorClose))
			set(lawid.PlaceNext, m.ContractPartner(ContractCursor, ContractCursorNext))
			set(lawid.PlaceProduced, ContractCursor)
		}
	}
	return pairs
}

// chainShaped reports the append-and-replay protocol, whose bundle
// claim speaks "chain law".
func chainShaped(f Iface) bool {
	return slices.ContainsFunc(f.Methods, func(m subject.Method) bool {
		return slices.Contains(m.Contracts, ContractChain)
	})
}
