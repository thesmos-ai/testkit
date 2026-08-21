// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"testing"

	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/model"
)

// TestCompanionSurface reaches the second emit value the plugin queues.
func TestCompanionSurface(t *testing.T) {
	t.Parallel()

	s := mixed(t)
	// Derived rather than read out of the emit graph: this tier queues
	// nothing, because a queued value renders into a file and it emits
	// none. See [model.CompanionFor].
	comp := model.CompanionFor(sdk.NewProvenance(model.Name), ifaceIn(t, s), bindingsOf(t, s))
	testkit.True(t, comp != nil, "the proof derives from the bindings")
	testkit.Equal(t, comp.Kind(), model.KindCompanion, "and renders as itself")
	testkit.Equal(t, comp.ModelPkg(), model.ModelPkg, "reaching the runner's package")
	comp.SetOutputPackages(map[string]string{"": "example.com/validates/validatestest"})
	testkit.Equal(t, comp.HarnessPkg, "example.com/validates/validatestest",
		"and the bindings through Layout's resolved route")
	comp.SetOutputPackages(map[string]string{})
	testkit.Equal(t, comp.HarnessPkg, "example.com/validates/validatestest",
		"which a partial later map does not clear")

	testkit.Equal(t, len(comp.Mutants), 2,
		"one kill-matrix row per driven method")
	testkit.Equal(t, comp.Mutants[0].Method, "Store", "the writer's mutant leads")
	testkit.Equal(t, comp.Mutants[1].Method, "Read", "the reader's follows")
	testkit.True(t, comp.Mutants[0].Sig != nil, "each with the override's signature")
	testkit.Equal(t, comp.ConcurrentName, "MixedModelConcurrent",
		"and the concurrent leg's proof rides along where the leg derives")
	testkit.Equal(t, comp.RootPkg(), "go.thesmos.sh/testkit",
		"the surrogate's import path reaches the template")
}
