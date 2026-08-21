// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"embed"

	"go.thesmos.sh/eidos/sdk"
)

// GoPrimarySuffix is the per-source-basename trailer for the bindings.
//
// `<source-basename>_model.gen.go`, beside the harness rather than inside it.
// A shared suffix would merge the two plugins' output into one file, and they
// have to be separately deletable: removing `//testkit:model` removes the
// emission, the file, and the engine dependency together.
const GoPrimarySuffix = "_model.gen.go"

// GoTestSuffix is the trailer for the companion, which proves the emission:
// the derived reference passes its own tier, and its inert bodies answer
// rather than panic.
//
// Generated rather than left to a hand-written probe beside each fixture,
// because the surface it drives exists in every armed package and a probe
// someone has to remember to write is a probe most packages will not have.
// The `_test.go` ending earns Layout's external-test-package shift, so the
// companion reaches the bindings the way a consumer does — which is why the
// reference constructor is exported.
const GoTestSuffix = "_model.gen_test.go"

// GoTestOutputTag names the companion output, so routing overrides and CLI
// selection can address the two files apart.
const GoTestOutputTag = "test"

// EngineModule is the module the bindings import, and the reason this plugin
// is directive-gated rather than triggered by classifications alone.
//
// It carries rapid behind it. A classification line must not add that to a
// consumer's build with no way to decline; the directive is the exit.
const EngineModule = "go.thesmos.sh/testkit/engine"

// The packages the generated bindings reach into, composed here so the emit
// values carry qualified expressions and the templates ask rather than spell.
const (
	// ModelPkg is the runner: Property, Check, SampledFrom, the options.
	ModelPkg = EngineModule + "/model"

	// VocabPkg is the runtime suite vocabulary, whose Subject this tier
	// lowers a contributed harness field onto.
	VocabPkg = EngineModule + "/suite"

	actionPkg = ModelPkg + "/action"

	// RefPkg holds the reference implementations the derived adapter wraps,
	// and LawPkg the law catalogue the registry instantiates from.
	RefPkg = ModelPkg + "/ref"
	LawPkg = ModelPkg + "/law"

	// TimeawarePkg is where the clock-shaped laws live, boxed apart because
	// their checks read time; ClockPkg is the deterministic clock the
	// ModelClocked option threads through the factory and every Advance.
	TimeawarePkg = ModelPkg + "/timeaware"
	ClockPkg     = "go.thesmos.sh/testkit/clock"

	// LinearizePkg holds the Porcupine models and the concurrent action
	// helpers the derived concurrent leg wires.
	LinearizePkg = ModelPkg + "/linearize"

	// HistoryPkg is the per-iteration append log the no-drops law reads
	// and the recording action fills.
	HistoryPkg = ModelPkg + "/history"

	// RootPkg is the runtime module — the kill matrix's failure surrogate
	// lives there.
	RootPkg = "go.thesmos.sh/testkit"
)

// The template tree, embedded through the recursive directory form — the
// action bodies are one file per shape, mirroring how the harness keeps its
// checks readable.
//
//go:embed templates/golang
var goTemplatesFS embed.FS

// GoOutputs returns the Go output set: the bindings and the companion that
// proves them.
//
// The empty tag is the primary and sits at index 0, which the pipeline
// validates at Build.
func GoOutputs() []sdk.Output {
	return []sdk.Output{
		{Tag: "", Suffix: GoPrimarySuffix},
		{Tag: GoTestOutputTag, Suffix: GoTestSuffix},
	}
}
