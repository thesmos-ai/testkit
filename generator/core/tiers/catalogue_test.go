// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers_test

import (
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/core/tiers"
)

// TestEveryRuleNamesALiveClassification is the half of the gate that does not
// need the law package.
//
// A classification is a string eidos owns, and a rule naming one that no
// longer exists selects nothing — for exactly one law, silently, with every
// other rule still working. This is the failure the old ownership table
// suffered for forty-seven laws before anybody noticed.
func TestEveryRuleNamesALiveClassification(t *testing.T) {
	t.Parallel()

	live := liveClassifications()
	for _, r := range tiers.Rules() {
		for _, need := range r.Needs {
			testkit.True(t, slices.Contains(live, need),
				r.Law+" needs `"+need+"`, which eidos registers")
		}
	}
}

// TestEveryRuleNamesADeclaredLaw holds the other end of the selection.
//
// Weaker than the conformance gate's version, which reflects over the law
// types themselves — this only proves the identifier is one the vocabulary
// declares, not that a law reports it. Worth having anyway: it fails in the
// module that owns the rule, where the fix is.
func TestEveryRuleNamesADeclaredLaw(t *testing.T) {
	t.Parallel()

	declared := lawid.All()
	for _, r := range tiers.Rules() {
		testkit.True(t, slices.Contains(declared, r.Law),
			r.Law+" is a declared identifier")
	}
}

// TestEveryFieldSourceIsResolvable holds each manifest entry's From to the
// vocabulary its Kind defines.
//
// The gate that stops a manifest from naming a role no contract declares. A
// binding reading `cursor.closed` when the contract's roles are `next` and
// `close` would fill the field with nothing, and a law with a nil closure
// asserts nothing while reporting as bound.
func TestEveryFieldSourceIsResolvable(t *testing.T) {
	t.Parallel()

	roles, params := contractVocabulary()
	mixinParams := mixinVocabulary()

	for _, r := range tiers.Rules() {
		testkit.True(t, len(r.Fields) > 0, r.Law+" names the fields it fills")

		for _, f := range r.Fields {
			label := r.Law + "." + f.Name

			switch f.Kind {
			case tiers.KindTrace, tiers.KindDefault:
				testkit.Equal(t, f.From, "",
					label+" names no source: the runner or the law fills it")

			case tiers.KindConstant:
				testkit.True(t, knownParam(f.From, params, mixinParams),
					label+" reads `"+f.From+"`, a parameter some classification declares")

			case tiers.KindRole, tiers.KindMethodName:
				// Both resolve a role: one to call it, the other to spell
				// its name, and a name resolved any other way could differ
				// from the method the closure beside it actually calls.
				testkit.True(t, knownRole(f.From, roles, mixinParams),
					label+" calls `"+f.From+"`, a role or partner some classification declares")

			case tiers.KindGenerator, tiers.KindHandle, tiers.KindSupplied:
				testkit.True(t, f.From != "",
					label+" names which "+string(f.Kind)+" fills it")

			default:
				t.Fatalf("%s: unknown field kind %q", label, f.Kind)
			}
		}
	}
}

// TestEveryConditionReadsADeclaredParameter holds the When clauses to the same
// standard as the manifests.
//
// A condition on a parameter nothing declares never matches, so its law never
// binds — and the symptom is a law quietly absent from every generated file
// rather than an error anywhere.
func TestEveryConditionReadsADeclaredParameter(t *testing.T) {
	t.Parallel()

	_, params := contractVocabulary()
	mixinParams := mixinVocabulary()

	for _, r := range tiers.Rules() {
		for _, c := range r.When {
			testkit.True(t, knownParam(c.Param, params, mixinParams),
				r.Law+" conditions on `"+c.Param+"`, a declared parameter")
			testkit.False(t, c.Absent && (c.Equals != "" || c.NotEquals != ""),
				r.Law+" does not require a parameter to be both absent and valued")
		}
	}
}

// TestSuppliedFieldsAreNamedForTheOptionThatArmsThem keeps the skip message
// actionable.
//
// A law reporting "skipped: supply " is worse than one that never bound. The
// From is what the generated header prints, so it has to read as something a
// consumer can act on rather than as a field name echoed back at them.
func TestSuppliedFieldsAreNamedForTheOptionThatArmsThem(t *testing.T) {
	t.Parallel()

	for _, r := range tiers.Rules() {
		for _, f := range r.Fields {
			if f.Kind != tiers.KindSupplied {
				continue
			}
			testkit.True(t, f.From != "" && f.From == strings.ToLower(f.From),
				r.Law+"."+f.Name+" names its option in the lower-case form a directive uses")
		}
	}
}

// liveClassifications is every name eidos registers, across all three axes.
//
// One namespace, because a method carries them in one namespace: its detector
// shape, its mixins and its contract roles all reach [tiers.Select] as one set.
func liveClassifications() []string {
	out := make([]string, 0, len(detectors.All())+len(mixins.All())+len(contracts.All()))
	for _, d := range detectors.All() {
		out = append(out, d.Name)
	}
	for _, m := range mixins.All() {
		out = append(out, m.Name)
	}
	for _, c := range contracts.All() {
		out = append(out, c.Name)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// contractVocabulary returns every declared `<contract>.<role>` and every
// contract parameter stamp key.
func contractVocabulary() (roles, params []string) {
	roles = make([]string, 0, 2*len(contracts.All()))
	params = make([]string, 0, len(contracts.All()))
	for _, c := range contracts.All() {
		for _, role := range c.Roles {
			roles = append(roles, c.Name+"."+role)
		}
		for _, p := range c.Params {
			params = append(params, shape.ContractParamKey(c.Name, p.Key).Name())
		}
	}
	slices.Sort(roles)
	slices.Sort(params)
	return roles, params
}

// mixinVocabulary returns every declared `<mixin>.<param>` in both spellings:
// the bare form a role reference uses, and the stamp key a constant reads.
func mixinVocabulary() map[string]string {
	out := map[string]string{}
	for _, m := range mixins.All() {
		for _, p := range m.Params {
			out[m.Name+"."+p.Key] = shape.MixinParamKey(m.Name, p.Key).Name()
		}
	}
	return out
}

// knownParam reports whether from names a parameter some classification
// declares, in either the contract or the mixin spelling.
func knownParam(from string, contractParams []string, mixinParams map[string]string) bool {
	if slices.Contains(contractParams, from) {
		return true
	}
	for _, key := range mixinParams {
		if key == from {
			return true
		}
	}
	return false
}

// knownRole reports whether from names something a generated call can reach:
// the stamped method itself, a cluster member of a given shape, a contract
// role, or the sibling a mixin parameter names.
func knownRole(from string, contractRoles []string, mixinParams map[string]string) bool {
	if from == "self" || strings.HasPrefix(from, "family.") {
		return true
	}
	if slices.Contains(contractRoles, from) {
		return true
	}
	_, isMixinPartner := mixinParams[from]
	return isMixinPartner
}
