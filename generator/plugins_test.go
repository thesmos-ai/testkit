// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"maps"
	"testing"
	"text/template"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/model"
	"go.thesmos.sh/testkit/generator/suite"
)

// The set is what a binary registers, and every structural fault in it is one
// the pipeline already rejects at Build: a duplicate plugin name, two plugins
// declaring one directive schema, two providers of one capability in a bucket,
// an emit major nothing can render, a malformed output table. Asserting those
// here by hand would be a second, weaker copy of a check eidos runs anyway —
// and the hand-written one this replaces asserted only that each role was
// filled at least once, which no real fault looks like.
//
// RunSetSuite fills the roles the set does not claim, so it reports what is
// wrong with the set rather than what the set is not, and runs the per-plugin
// conformance suite over each member on the way past.
func TestAllIsAWorkingPluginSet(t *testing.T) {
	t.Parallel()

	plugintest.RunSetSuite(t, generator.All()...)
}

// Every generator must appear in the full set. A generator registered in
// Generators but dropped from All builds and tests cleanly, and silently never
// runs.
func TestAllContainsEveryGenerator(t *testing.T) {
	t.Parallel()

	all := make(map[string]struct{})
	for _, p := range generator.All() {
		all[p.Name()] = struct{}{}
	}

	gens := generator.Generators()
	if len(gens) == 0 {
		t.Fatal("no generators are registered at all")
	}
	for _, g := range gens {
		if _, ok := all[g.Name()]; !ok {
			t.Errorf("generator %q is registered but absent from All", g.Name())
		}
	}
}

// Every function a shipped template calls has to be one somebody in the run
// provides — the plugin's own funcmap, or the backend's reserved set.
//
// [plugintest.RunSuite] does not ask this. It parses each template with every
// unresolved name stubbed, deliberately, so that it judges syntax alone; a
// template calling a function nobody registers therefore parses, ships, and
// fails midway through Render in the consumer's build, naming the merged tree
// rather than the file.
//
// Asserted over the set rather than beside each plugin, because it is one
// question with one answer and five copies would be five things to remember
// when a sixth generator arrives.
func TestEveryTemplateFuncResolves(t *testing.T) {
	t.Parallel()

	for _, p := range generator.All() {
		t.Run(p.Name(), func(t *testing.T) {
			t.Parallel()
			plugintest.AssertTemplateFuncsResolve(t, p, reservedFuncs(), golang.Language)
		})
	}
}

// reservedFuncs is everything the Go backend brings to a template, assembled
// from the three places it is reachable from.
//
// The assertion parses each template with this map, and parsing resolves names
// without calling bodies — so a placeholder binds as well as the real function,
// and a variadic one binds against every call shape a template can write.
//
// All three sources are authoritative. plugintest exports the canonical
// reserved names and the overrideable ones the backend registers, and
// lang/golang exports the shared Go conventions layered on top. The
// overrideable list was hand-kept here until eidos grew an accessor for it;
// what that cost was a check that went red for a correct template every time
// the backend added a helper.
func reservedFuncs() template.FuncMap {
	out := template.FuncMap{}
	for _, name := range plugintest.ReservedTemplateFuncNames() {
		out[name] = placeholderFunc
	}
	for _, name := range plugintest.OverrideableTemplateFuncNames() {
		out[name] = placeholderFunc
	}
	maps.Copy(out, golang.FuncMap())
	return out
}

// placeholderFunc stands in for a backend helper whose body this check never
// calls. Variadic and untyped so it binds against any call a template writes:
// what is under test is whether the *name* resolves.
func placeholderFunc(...any) any { return nil }

// The shape plugin is three annotators. Registering fewer is silent in a way
// no coverage gate can see — the classifier stamps either way — so the count
// is asserted here, where a companion dropped from the set is a failure rather
// than a class of enforcement that quietly stops running.
func TestAnnotatorsCarryEveryShapeCompanion(t *testing.T) {
	t.Parallel()

	want := generator.Annotator().Annotators()
	got := generator.Annotators()

	names := make(map[string]struct{}, len(got))
	for _, a := range got {
		names[a.Name()] = struct{}{}
	}
	for _, w := range want {
		if _, ok := names[w.Name()]; !ok {
			t.Errorf("annotator %q is a shape companion but absent from Annotators", w.Name())
		}
	}
}

// The annotator is configured here so the CLI and the conformance gate cannot
// disagree about which classifications exist. An unconfigured one stamps
// nothing, which would read as an empty corpus rather than as a wiring fault.
func TestAnnotatorIsConfigured(t *testing.T) {
	t.Parallel()

	// The role itself is enforced by the return type, so what is left to check
	// is that one was configured and that it satisfies the interface the set
	// registers it under.
	a := generator.Annotator()
	if a == nil {
		t.Fatal("no annotator is configured")
	}
	var _ sdk.Annotator = a
}

// Every declared check-body shape is rendered by exactly one tier.
//
// The closed set is the projection's; who spells a template for each
// member is the tiers'. Neither generator can state this — each knows
// only its own answer — so it is stated here, where the plugin set is
// composed.
//
// Both directions are failures and they fail differently. A kind NO tier
// renders is a row the harness generator plans, files under Withheld and
// nothing ever emits: the capability it declared still reaches the
// harness, so a consumer gets a field to fill for a check that does not
// exist. A kind BOTH render is one identity emitted twice, which the
// manifest catches only if the two spellings differ.
func TestEveryBodyKindRendersInExactlyOneTier(t *testing.T) {
	t.Parallel()

	bySuite := suite.RenderedBodyKinds()
	byModel := map[projection.BodyKind]bool{}
	for _, k := range model.ModelRowKinds() {
		byModel[k] = true
	}

	for _, k := range projection.BodyKinds() {
		switch {
		case bySuite[k] && byModel[k]:
			t.Errorf("both tiers render %s, so one plan emits two checks under one identity", k)
		case !bySuite[k] && !byModel[k]:
			t.Errorf("no tier renders %s, so the rows planned for it are emitted nowhere", k)
		}
	}
}

// The two sets together are the whole declared set and nothing more.
//
// The count is what catches a kind a tier claims and the projection does
// not declare — which reads as coverage and renders against nothing.
func TestTheTwoTiersAccountForTheWholeSet(t *testing.T) {
	t.Parallel()

	testkit.Equal(t,
		len(suite.RenderedBodyKinds())+len(model.ModelRowKinds()),
		len(projection.BodyKinds()),
		"a tier claiming a kind the projection does not declare renders against nothing, "+
			"and a kind neither claims is planned and dropped")
}
