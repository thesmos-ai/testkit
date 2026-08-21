// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"embed"

	"go.thesmos.sh/eidos/sdk"
)

// GoPrimarySuffix is the per-source-basename trailer for the harness.
//
// `<source-basename>_suite.gen.go`, matching what stub and builder compose and
// what reference/layout.md documents.
const GoPrimarySuffix = "_suite.gen.go"

// GoTestSuffix is the trailer for the companion output, which drives every
// generated check against a subject that complies and one that violates.
//
// A suffix ending `_test.go` earns the external-test-package shift from Layout,
// which is what makes the companion reach the harness across a package boundary
// the way a consumer does.
const GoTestSuffix = "_suite.gen_test.go"

// GoTestOutputTag names the companion output. Routing overrides and CLI
// selection address an output by tag, so the two files can be routed apart.
const GoTestOutputTag = "test"

// GoIntegrationEnv is the variable a run sets to include integration-only
// checks.
//
// One name for every generated suite, so a consumer sets it once rather than
// per interface. `//testkit:mixin integrationonly` names no variable — the
// classification says the method reaches something outside the process, and
// which switch turns that on is a fact about how a team runs their tests.
//
// Unset is a skip rather than a pass, which is the whole point: a check that
// silently succeeded because its dependency was absent is a check that reports
// coverage it did not earn.
const GoIntegrationEnv = "TESTKIT_INTEGRATION"

// Module is testkit's own import path, which the generated harness references
// for its assertion helpers.
//
// Declared here rather than imported from a sibling generator: this plugin does
// not depend on any of them, and taking a constant from one would make it look
// as though it did.
const Module = "go.thesmos.sh/testkit"

// EngineModule is the module the vocabulary and the falsifiability
// harness live in, which is not [Module].
//
// Named because a caller assembling a build against the generated output
// has to require both — the emitted harness imports the root module's
// clock and the engine module's suite package, and a go.mod naming only
// the first fails to resolve the second with no clue that there were two.
const EngineModule = Module + "/engine"

// Vocab is the suite vocabulary's package — the ID grammar, the
// segment and family constants, and the Check the generated rows
// construct. Emitted code composes its identities from here rather
// than spelling slugs, so the grammar has one home across the
// generator and the runtime.
const Vocab = EngineModule + "/suite"

// ClockPkg is the controllable clock a clocked check advances. Named
// beside the vocabulary because the harness spells its test clock in
// type position, which `external` cannot build.
const ClockPkg = Module + "/clock"

// LawIDs is where a law's identifier is declared, for the index
// accessors that name one.
const LawIDs = Module + "/core/lawid"

// Prove is the falsifiability harness the companion drives its planted
// defects through.
//
// Its own package rather than part of [Vocab] because it needs the
// captive TB from the module root, which the vocabulary a consumer's
// non-test code composes with must not depend on. The companion is a
// test file and pays that import happily.
const Prove = Vocab + "/prove"

// The template tree, embedded through the recursive directory form rather than
// a `*.tmpl` glob.
//
// The glob reaches one level only, and this tree is nested by axis: one file
// per classification is the only arrangement in which seventy-two of them stay
// readable, and the backend's loader walks the filesystem depth-first, so the
// nesting costs nothing on the reading side.
//
//go:embed templates/golang
var goTemplatesFS embed.FS

// GoOutputs returns the Go output set: the harness and its companion.
//
// The empty tag is the primary and sits at index 0, which the pipeline
// validates at Build.
func GoOutputs() []sdk.Output {
	return []sdk.Output{
		{Tag: "", Suffix: GoPrimarySuffix},
		{Tag: GoTestOutputTag, Suffix: GoTestSuffix},
	}
}
