// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

// AdapterDebt registers every language-axis fixture without a model tier,
// each with the adapter limit that keeps it out. The reasons were real
// before this table existed — each fixture's package doc states its own —
// but they were prose the build could not read: the law debt has a
// register the gate enforces, and these adapter debts had comments. Both
// directions are enforced by name: a lang fixture that gains a model tier
// must delete its row, and a new lang fixture without one must argue its
// absence here or redden the census.
//
// Keys are the fixture's leaf directory under corpus/iface/lang.
//
//nolint:gochecknoglobals // a debt register, read-only, test-facing.
var AdapterDebt = map[string]string{
	"embedded":          "the flattened method set is the stub and suite axis's subject; a model tier would restate what those outputs already prove",
	"embeddedforeign":   "a foreign embed brings methods no fixture subject models; the axis proves the flattening, not a store",
	"function":          "free functions have no interface to derive a reference for; the axis proves the OnFunction walk",
	"genericbound":      "a named constraint is a reference the generator never loads, so no witness list can pin the property's types",
	"multireturn":       "more than two returns exceeds every driven closure shape the sequences can compare",
	"namedreturns":      "the axis proves the recorded-call field naming; its methods answer nothing a reference could disagree with",
	"nocontext":         "the driven closures forward a context the methods refuse to take",
	"partnernaming":     "the axis proves which relational checks the generator can write down; a model tier states laws about the subject and would say nothing about that",
	"receivercollision": "the axis proves receiver naming under collision; a model tier would add sequences to a fixture about identifiers",
	"seededreader":      "the axis proves the seed seam — a harness receiving its corpus because nothing on the interface writes; a model tier drives sequences through a writer, which is the one thing this fixture does not have",
	"roledtypes":        "the axis proves where a role can be WRITTEN — on a named type for a bare parameter, on a field for a request struct — which is a fact about the declaration shape; a model tier states laws about a subject and would say nothing about that",
	"variadic":          "a variadic tail has no single argument type for a pool to draw",
	"generic":           "the checks are generic with the interface and these sequences run at the concrete types witness= names; checks at concrete types cannot join a check set at type parameters",
}
