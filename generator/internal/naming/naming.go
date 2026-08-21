// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package naming holds the conventions one generator's output
// establishes and another has to spell in order to reach it.
//
// A generated double is called `<Iface>Stub`, a hand-written seed
// function `<Type>Defaults`. Whoever emits the name owns the rule — but
// the harness generator's proofs construct the double, and its fixture
// looks for the seed, so a second generator has to spell both. Spelled
// as a literal there, a rename reaches one of the two and the other goes
// on naming something that no longer exists: a compile error in
// generated code the consumer cannot edit, attributed to the generator
// rather than to the rename.
//
// Here rather than exported from the plugin that emits it, for the
// reason [go.thesmos.sh/testkit/generator/internal/stamp] gives about
// the annotators: importing a generator to read one string also puts its
// constructor within reach, and a generator that can construct another
// generator can register a second copy of it. Reading a word costs a
// word.
package naming

// StubSuffix is the trailer on a generated double's type — `KVStub` for
// `KV`. The double generator emits it; the harness's proofs name it to
// build the deliberately broken implementation each check is tested
// against.
const StubSuffix = "Stub"

// CompanionSuffix is the trailer on the seed function declared beside a
// type — `UserDefaults()` for `User`.
//
// A convention rather than a declaration: the function is ordinary
// source in the consumer's own package, found by looking rather than by
// being told, and a package holding several types gets one companion
// each — which is why the name carries the type rather than being a bare
// `Defaults`. The builder generator resolves a struct's defaults through
// it and the harness generator draws a fixture value through it, so it
// belongs to neither.
const CompanionSuffix = "Defaults"

// The regions the harness generator hands out for another tier to
// contribute into, named for what each holds.
//
// A slot is reached by name, so the generator that hands the region out
// and the generator that fills it both have to spell it. Here rather
// than exported from the host for the reason the suffixes are: reaching
// a plugin for one string also puts its constructor in scope, and a
// misspelling on either side mints a second, unconstrained region under
// a near-miss name rather than failing.
//
// Named for the content and not for the mechanism. Every one of these is
// an injection point, so "the slot" distinguishes nothing; what a reader
// needs to know is which part of the generated file a contribution lands
// in and what shape it has to be.
const (
	// SlotHarnessFields is the harness struct's fields — the capability
	// doors a contributing tier's checks need a consumer to supply.
	SlotHarnessFields = "harness.fields"

	// SlotSuiteChecks is the run surface's check list — an expression per
	// contribution, each yielding a slice the surface appends.
	SlotSuiteChecks = "suite.checks"

	// SlotSuiteDecls is the file's package level, after everything the
	// harness generator declares.
	//
	// Where the function a SlotSuiteChecks expression names is emitted,
	// along with whatever it calls. The expression and the declaration
	// are one contribution in two regions, for the reason a field and its
	// lowering are: an expression naming a function nobody emitted is a
	// compile error over generated code.
	SlotSuiteDecls = "suite.decls"

	// SlotHarnessLowering is the body of the harness's Subject method,
	// after the runtime subject is built and before it is returned.
	//
	// A field and the line that carries it onto the runtime are one
	// contribution in two places: a tier that adds a field nothing lowers
	// has given the consumer somewhere to write a value that goes
	// nowhere, which is the defect the region pair exists to make
	// unrepresentable.
	SlotHarnessLowering = "harness.lowering"
)

// Slots is every region a generator hands out, so the host that answers
// them can be held to the list.
//
// A name here with no case in the host's switch mints a fresh, empty
// region: the contribution lands somewhere nobody reads and the file
// renders without it, which compiles. That is why the list exists rather
// than the constants alone.
func Slots() []string {
	return []string{SlotHarnessFields, SlotHarnessLowering, SlotSuiteChecks, SlotSuiteDecls}
}
