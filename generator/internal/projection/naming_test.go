// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/projection"
)

func TestOptionNamePolicy(t *testing.T) {
	t.Parallel()

	testkit.Equal(t, projection.OptionName("Log", "Append"), projection.Option("WithLogAppend"),
		"the option policy is With<Iface><Method>")
}

// fixtureCase is one fixture-accessor spelling.
type fixtureCase struct {
	name  string
	token string
	field string
	want  projection.Expr
}

func (c fixtureCase) Name() string { return c.name }

func TestEmittedSurfaceNames(t *testing.T) {
	t.Parallel()

	testkit.Equal(t, projection.HarnessName("Store"), "StoreHarness",
		"the per-implementation config literal's type")
	testkit.Equal(t, projection.VeneerName("Store"), "StoreSuite",
		"the exported entry value a consumer reads through")
	testkit.Equal(t, projection.ConfigName("Store"), "StoreConfig",
		"the run-config type")
}

//nolint:thelper // the case body is the test, not a helper; see core/lawid
func TestFixtureCallPolicy(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, []fixtureCase{
		{"the run's fixture, then the exported accessor", "fx", "entry", "fx.Entry()"},
		{"initialisms case the platform's way", "fx", "id", "fx.ID()"},
		{"an empty field degrades to the fixture itself", "fx", "", "fx"},
	}, func(t *testing.T, tc fixtureCase) {
		testkit.Equal(t, projection.FixtureCall(projection.Expr(tc.token), tc.field), tc.want,
			"the fixture draw's spelling has this one home")
	})
}

// The token is what every emitted identifier is qualified by, so it
// has to read as a name rather than as a run-together slug.
//
// Lower camel is the whole rule, and the multiword case is why: a
// plain lower-casing produces kvstorecheckindex, which compiles and
// which nobody wants to meet in a stack trace.
//
//nolint:thelper // the case body is the test, not a helper; see core/lawid
func TestTokenIsLowerCamel(t *testing.T) {
	t.Parallel()

	type tokenCase struct {
		name  string
		iface string
		want  string
	}

	testkit.TableTest(t, []tokenCase{
		{"a one-word interface lowers", "Log", "log"},
		{"a multiword interface stays readable", "KVStore", "kvStore"},
		{"an initialism keeps the platform's casing", "HTTPClient", "httpClient"},
	}, func(t *testing.T, tc tokenCase) {
		testkit.Equal(t, projection.Token(tc.iface), tc.want, tc.name)
	})
}

// The emitted identifiers compose from the token and nothing else.
//
// Pinned because these names are the generated file's whole surface: a
// consumer writes logCheckIndex.Append.Smoke(), and every one of the
// four spellings below has to agree with the others or the file does
// not compile.
func TestEmittedIdentifiersCompose(t *testing.T) {
	t.Parallel()

	tok := projection.Token("Log")

	t.Run("a method's name constant carries the token", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, projection.MethodConst(tok, "Append"), "logAppend",
			"the one home for the method's name")
	})

	t.Run("the index value reads, and its type takes the awkward name", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, projection.IndexVar(tok), "logCheckIndex", "what a consumer writes")
		testkit.Equal(t, projection.IndexType(tok), "logCheckIndexT", "what nobody writes")
	})

	// The qualifier constant is the one word a family-scoped ID is built
	// from, and the index file spells it once per family accessor. It
	// carries the token by the same rule as the method constants, so a
	// drift here would mint IDs no manifest row matches.
	t.Run("the qualifier constant carries the token too", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, projection.QualifierConst(tok), "logQualifier",
			"the one home for the interface's word inside a family ID")
	})

	t.Run("both scopes' groups spell one rule", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, projection.GroupType(tok, "Append"), "logAppendChecks",
			"a method group")
		testkit.Equal(t, projection.GroupType(tok, "Model"), "logModelChecks",
			"and a family group, by the same rule rather than the packs' second one")
	})
}

// The assertion's name is the packs', segment word included.
//
// The index and the assertion spell the same segment differently on
// purpose — `ix.Put.Deadline()` is a noun a consumer names, and
// `storeAssertPutHonoursDeadline` is the sentence it checks — so this
// pins the three that diverge alongside one that does not.
func TestAssertNameSpellsThePacksWords(t *testing.T) {
	t.Parallel()

	tok := projection.Token("Store")

	t.Run("a segment whose assertion reads as the index does", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, projection.AssertName(tok, "Get", suite.SegSmoke),
			"storeAssertGetSmoke", "the packs' own spelling")
	})

	t.Run("the three that read as sentences", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, projection.AssertName(tok, "Put", suite.SegDeadline),
			"storeAssertPutHonoursDeadline", "deadline is honoured, not merely had")
		testkit.Equal(t, projection.AssertName(tok, "Put", suite.SegNilContext),
			"storeAssertPutToleratesNilContext", "a nil context is tolerated")
		testkit.Equal(t, projection.AssertName(tok, "Get", suite.SegZeroValue),
			"storeAssertGetZeroOnError", "the zero rides the error")
	})

	t.Run("a segment no pack asserts falls back to the index's word", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, projection.AssertName(tok, "Get", suite.SegCancel),
			"storeAssertGetCancels", "which the packs also spell")
	})
}
