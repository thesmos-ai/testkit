// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection

import (
	"strings"

	"go.thesmos.sh/eidos/core/naming"
	"go.thesmos.sh/eidos/lang/golang"

	"go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// Expr is a rendered Go expression destined for a template hole. The
// type exists so a template argument cannot be confused with prose,
// and so the well-known spellings below have one home.
type Expr string

// ExprCtx is the context argument every ctx-taking call receives; the
// emitted body names its context ctx, and this constant is the only
// place that decision is spelled.
const ExprCtx Expr = "ctx"

// ExprNil is the literal a nil-argument body spells in the slot it is
// testing. Here rather than in the rule because it is a spelling, and
// every spelling the emitted bodies share has one home.
const ExprNil Expr = "nil"

// ExprBound is the local a bounded probe binds its sizer's answer to,
// and the argument the judged call then spells. One name, here, for the
// reason [ExprCtx] is here.
const ExprBound Expr = "bound"

// ExprHook is the local a hooks body binds its recording callback to,
// and the argument the registration call then spells.
const ExprHook Expr = "hook"

// ExprBorrowed is the local a borrow-first smoke binds the producing
// sibling's answer to; the returning call's args reference it where
// the parameter takes the produced type.
const ExprBorrowed Expr = "borrowed"

// ExprProduced is the local an opener smoke binds its answered handle
// to before closing it — the body's one name for what the opener
// owns.
const ExprProduced Expr = "produced"

// The emitted-surface suffixes, composed only through the policy
// functions below so each generated identifier's spelling has one
// home.
const (
	harnessSuffix         = "Harness"
	withoutSuffixExported = "Without"
	suiteOfSuffix         = "SuiteOf"
	veneerSuffix          = "Suite"
	configSuffix          = "Config"
)

// HarnessName is the generated harness type's identifier — the config
// literal a consumer writes per implementation.
func HarnessName(iface string) string { return iface + harnessSuffix }

// VeneerName is the generated veneer's identifier — the exported
// entry value a consumer's test file reads checks and runs through.
func VeneerName(iface string) string { return iface + veneerSuffix }

// ConfigName is the generated run-config type's identifier.
func ConfigName(iface string) string { return iface + configSuffix }

// poolSuffix trails every drawn-pool config field; the config doc
// promises "fields ending in Pool are drawn from", and this is where
// that promise is spelled.
const poolSuffix = "Pool"

// PoolFieldName is a role's config field, derived from the stamped
// field's own exported name — kv's Key and Value fields open KeyPool
// and ValuePool.
func PoolFieldName(field string) string { return golang.ExportedName(field) + poolSuffix }

// Option is a generated stub option's name ("WithLogAppend"),
// constructed only through [OptionName] so the naming policy has one
// home.
type Option string

// OptionName spells the per-method construction option the stub
// plugin emits: With<Iface><Method>.
func OptionName(iface, method string) Option {
	return Option("With" + iface + method)
}

// StubCtorName spells the double's constructor — `NewCalculatorStub`.
//
// The suffix is a parameter, and that is the whole difference from
// [OptionName] above: `With<Iface><Method>` is composed from the
// interface whatever the double is called, while the constructor is
// named after the type, and what the type is called follows the stub
// plugin's `suffix` option. Taking it keeps this data model from
// importing a plugin to read a default out of it — the caller has the
// plugin in scope and this does not.
func StubCtorName(iface, suffix string) string {
	return golang.ConstructorName(iface + suffix)
}

// AssertName is the generated assertion's identifier —
// `storeAssertGetHonoursDeadline`.
//
// The word for the segment is the assertion's, not the index's, and the
// two genuinely differ: the index reads as a noun a consumer names
// (`ix.Put.Deadline()`) while the assertion reads as the sentence it
// checks (`…HonoursDeadline`). Transcribed from the packs, which spell
// every one of them.
func AssertName(token, method, seg string) string {
	return token + assertInfix + golang.ExportedName(method) + assertWord(seg)
}

// assertWord is the segment as an assertion reads it, falling back to
// the index's word where the packs never spelled one — a segment with
// no assertion in any pack has no validated sentence, and the index's
// noun is the honest stand-in until one exists.
func assertWord(seg string) string {
	if w, ok := assertWords()[seg]; ok {
		return w
	}
	name, _ := segAccessor(seg)
	return name
}

// assertWords are the segments whose assertion reads differently from
// their index entry.
func assertWords() map[string]string {
	return map[string]string{
		suite.SegDeadline:   "HonoursDeadline",
		suite.SegNilContext: "ToleratesNilContext",
		suite.SegZeroValue:  "ZeroOnError",
	}
}

// assertInfix separates the subject from what is asserted about it.
const assertInfix = "Assert"

// DrawWord is the word a drawn parameter is known by: the named type's
// own word where the source declares one, and the parameter's
// identifier otherwise.
//
// The type rather than the parameter, because a fixture holds one value
// per thing drawn and the thing is the type: `Put(ctx, v Value)` and
// `Get(ctx, key Key)` draw a Value and a Key, whichever letters the
// author happened to name the parameters. A predeclared type says
// nothing — every `string` would collide with every other — so there
// the parameter's own identifier is the only word available.
//
// One home because the claim text and the fixture field are the same
// word cased differently: "a seeded key" and Key. They were derived
// separately, and only the claim side had the rule.
func DrawWord(p golang.Param) string { return subject.DrawWord(p) }

// MissKeyCall is the emitted call for the key outside the corpus.
func MissKeyCall(token string) Expr { return Expr(MissKeyName(token) + "()") }

// ExprSeededKey is the loop variable a hit probe calls with.
//
// The ranged key rather than a fixture draw, which is the difference
// between judging the whole corpus and judging one entry of it: a hit
// body drawing fx.Key() asks the same question len(docs) times and
// passes for a subject that remembers only what it was handed first.
const ExprSeededKey Expr = "k"

// ExprDocs is the local a seeded body reads the run's corpus through.
// The hit and count probes judge the WHOLE set rather than one drawn
// member, so they take it beside the fixture rather than through it.
const ExprDocs Expr = "docs"

// ExprFixture is the local a body reads its draws through. The
// generated assert function takes the fixture by this name wherever any
// of its calls draws, and never where none does.
const ExprFixture Expr = "fx"

// FixtureCall spells one drawn field as the body reads it — `fx.Value`
// for ("fx", "value").
//
// Through the fixture rather than a package-level accessor, because the
// value has to be the RUN's: a package function returning a literal
// reaches no check that draws.
//
// A call rather than a field read, which is the packs' spelling and the
// one that survives the value's source changing: a fixture backed by a
// literal today and by a draw from the run's pools tomorrow reads the
// same at every site, and there are hundreds of them.
func FixtureCall(recv Expr, field string) Expr {
	if field == "" {
		return recv
	}
	return recv + "." + Expr(golang.ExportedName(field)) + "()"
}

// otherSuffix names the companion accessor. Spelled here as well as in
// the fixture projection because this package is the naming policy's
// home and cannot import back into the deriver.
const otherSuffix = "Other"

// FixtureCallOther draws the companion member — the second, different
// value every field carries — for a body that has to tell two draws
// apart.
func FixtureCallOther(recv Expr, field string) Expr {
	return FixtureCall(recv, field+otherSuffix)
}

// Token is the interface's qualifier in every generated identifier —
// "log" for Log, "kvStore" for KVStore.
//
// Lower camel rather than a plain lower-casing, because the token
// prefixes identifiers a human reads (logAppend, kvStoreCheckIndex)
// and "kvstorecheckindex" is not a name anybody wants in a stack
// trace. The casing engine is the platform's own, initialisms
// included, so the token agrees with what every other plugin emits.
func Token(iface string) string { return naming.Camel(iface) }

// IDQualifier is the interface's word inside a family-scoped check ID —
// "log", "paginated-reader".
//
// A slug rather than [Token], because the two qualify different things
// and the ID grammar admits only a-z, 0-9 and '-'. Token names Go
// declarations and reads as lower camel; an ID is what a lock file row
// and a Without() call are written against, and "paginatedReader"
// there is refused by the grammar rather than merely ugly. Every
// validated pack has a single-word interface, which is why one word
// served both jobs until the corpus asked.
func IDQualifier(iface string) string { return naming.Kebab(iface) }

// MethodConst is the generated constant holding a method's name —
// `logAppend = "Append"`.
//
// One home per name: the index accessors, the check bodies and the
// failure messages all spell the method, and a literal repeated across
// three emitted sections is three chances to rename two of them.
func MethodConst(token, method string) string {
	return token + golang.ExportedName(method)
}

// The run surface's identifiers, composed here rather than in the
// template so the half-dozen names that have to agree with each other
// agree by construction: a veneer naming an index its own file does not
// declare is a compile error a consumer meets, not one a run does.
// [HarnessName] and [VeneerName] are the two a consumer writes; these
// are the machinery those hang off.

// DefaultConfigName is the constructor for what this run derived.
func DefaultConfigName(token string) string { return token + defaultConfigSuffix }

// defaultConfigSuffix names the derived config apart from the type a
// consumer declares one of.
const defaultConfigSuffix = "DefaultConfig"

// ChecksName is the builder holding every check this run derived.
func ChecksName(token string) string { return token + checksBuilderSuffix }

// checksBuilderSuffix follows the packs' majority spelling, which names
// the tier the checks come from rather than the family — one builder
// per tier, paired with the model tier's own.
const checksBuilderSuffix = "SignatureChecks"

// RunName is the entry point a consumer calls to run the suite.
func RunName(iface string) string { return runPrefix + iface }

// ProveName is the entry point that runs a check set against a
// deliberately broken subject.
func ProveName(iface string) string { return provePrefix + iface }

// RunOptName is the interface every run option satisfies.
func RunOptName(iface string) string { return iface + runOptSuffix }

// RunConfigName is what the run options accumulate into.
func RunConfigName(token string) string { return token + runConfigSuffix }

// DropOptName is the option that declines checks by identity.
func DropOptName(token string) string { return token + dropOptSuffix }

// WithoutName is the constructor a consumer calls to decline them.
func WithoutName(token string) string { return token + withoutSuffix }

// ExportedWithoutName is the same constructor under an exported name,
// for a generic subject.
//
// The veneer carries Without for a concrete interface and cannot for a
// generic one: a method introduces no type parameters, and there is no
// one instantiation to fix them at. Every _test.go is an external test
// package, so leaving the drop unexported leaves a generic consumer no
// way to decline a check at all.
func ExportedWithoutName(iface string) string { return iface + withoutSuffixExported }

// ExportedSuiteName is the assembler under an exported name, for the
// same reason [ExportedWithoutName] exists.
func ExportedSuiteName(iface string) string { return iface + suiteOfSuffix }

// VeneerTypeName is the veneer's type; [VeneerName] is the value a
// consumer reads it through.
func VeneerTypeName(token string) string { return token + veneerTypeSuffix }

// IndexPathName maps every emitted ID to the path that drops it.
func IndexPathName(token string) string { return token + indexPathSuffix }

// DropHintName is the reporter that turns a dropped ID into that path.
func DropHintName(token string) string { return token + dropHintSuffix }

// SuiteName is the assembler that returns the checks as data —
// `calculatorSuite`.
//
// Unexported, with [VeneerName]'s Suite method as the way in. A caller
// outside the package composing suites is doing tooling, and tooling
// reaches one name rather than one per interface. The same word as
// [VeneerName], because they are the same noun read through the two
// qualifiers — which is why it is not spelled twice.
func SuiteName(token string) string { return token + veneerSuffix }

// RowsName is the consumer's own check table — `StoreChecks`.
func RowsName(iface string) string { return iface + checksSuffix }

// RowName is one row of that table — `StoreCheck`.
//
// The same trailing word as a per-method extension point and not the
// same type: that one is per method, this one is per interface, which
// is the difference between "an assertion about Get" and "a row you
// write".
func RowName(iface string) string { return iface + checkSuffix }

// MethodsVar is the method-name set a row's Method field is checked
// against — `storeMethods`.
//
// Emitted because a misspelled method name starts with a capital too,
// so the ID grammar alone would file a check under a method that does
// not exist and report it under a path nothing drops.
func MethodsVar(token string) string { return token + methodsSuffix }

// DefectTypeName is what a row's planted defect must satisfy —
// `StoreDefect`.
func DefectTypeName(iface string) string { return iface + defectSuffix }

// BrokenName is the sugar that names one — `BrokenStore`.
func BrokenName(iface string) string { return brokenPrefix + iface }

// The consumer seam's fixed words.
const (
	checkSuffix   = "Check"
	methodsSuffix = "Methods"
	defectSuffix  = "Defect"
	brokenPrefix  = "Broken"
)

// NewFixtureName is the constructor that draws a fixture from a config.
func NewFixtureName(token string) string { return token + newFixtureSuffix }

// LimitConst is the constant a declared bound is emitted under —
// `mixedCapacity`.
//
// A constant rather than the literal at each use, because the number
// reaches three places: the harness constructors receive it, the bounded
// law enforces it, and a consumer's own policy check needs the same one.
// Three homes for one declared number is three chances to drift.
func LimitConst(token string) string { return token + limitSuffix }

const limitSuffix = "Capacity"

// InvariantsTestName is the generated package's self-check.
//
// Per interface rather than per package, which is where the validated
// packs put it. Our emission is per interface — a package holding three
// suites queues three of these — and one shared function would need a
// coordination point no emit has. Named for its subject, so three
// suites give three tests rather than a redeclaration.
func InvariantsTestName(iface string) string {
	return golang.TestFuncName(iface, invariantsSuffix)
}

// LockPath is the manifest this interface's rows are pinned in.
//
// Per interface for the reason above. The packs write one checks.lock
// per package, which reads better in review; that arrives when something
// owns the package rather than the declaration.
func LockPath(token string) string { return token + ".checks.lock" }

const invariantsSuffix = "Invariants"

// CorpusName is the seeded corpus builder — `catalogCorpus`.
func CorpusName(token string) string { return token + corpusSuffix }

// CorpusTypeName is the seeded corpus's own type — `CatalogCorpus`.
//
// A named alias rather than the map spelled at each use: both halves
// render through the backend, and a template composing `map[` around two
// renderType calls would spell the type once per appearance — five
// times, in the harness fields, the Subject parameter and the builder.
func CorpusTypeName(iface string) string { return iface + corpusSuffix }

// MissKeyName is the key outside it — `catalogMissKey`.
func MissKeyName(token string) string { return token + missKeySuffix }

const (
	newFixtureSuffix = "NewFixture"
	corpusSuffix     = "Corpus"
	missKeySuffix    = "MissKey"
)

// ProofsName is the companion's defect map — `calculatorProofs`.
func ProofsName(token string) string { return token + proofsSuffix }

// ProofsTestName is the test the companion runs the proofs from.
//
// Through [golang.TestFuncName] rather than composed here: what a Go
// test function is called is a Go convention, and eidos states it once.
func ProofsTestName(iface string) string {
	return golang.TestFuncName(iface, proofsSuffix)
}

// DefectName words a planted defect for the report — "a Calculator whose
// Add panics", the subject line [prove.One] takes.
//
// Composed here rather than in the templates because each defect
// template spells its own clause and they must all read as one sentence;
// the clause is the parameter, the sentence is the rule.
//
// The article follows the interface's first letter, which is a spelling
// rule and not a heuristic worth apologising for: these strings are read
// in failure output beside real ones, and "a AnsweringWriter" is the kind
// of wrongness that makes a reader distrust the rest of the line.
func DefectName(iface, clause string) string {
	return article(iface) + " " + iface + " whose " + clause
}

// article picks "a" or "an" from the leading sound, approximated by the
// leading letter.
//
// Vowel letters only. The exceptions English has — a European, an hour —
// turn on pronunciation, which no rule over an identifier can reach, and
// a table of them would be a list of words nobody names an interface.
func article(word string) string {
	if word == "" {
		return "a"
	}
	if strings.ContainsRune("AEIOUaeiou", rune(word[0])) {
		return "an"
	}
	return "a"
}

// The run surface's fixed words.
const (
	runPrefix        = "Run"
	provePrefix      = "Prove"
	runOptSuffix     = "RunOpt"
	runConfigSuffix  = "RunConfig"
	dropOptSuffix    = "DropOpt"
	withoutSuffix    = "Without"
	veneerTypeSuffix = "Veneer"
	indexPathSuffix  = "IndexPath"
	dropHintSuffix   = "DropHint"
	proofsSuffix     = "Proofs"
)

// QualifierConst is the generated constant holding the interface's word
// inside a family-scoped ID — `logQualifier = "log"`.
//
// A constant for the same reason the method names are: the qualifier is
// spelled once per family accessor, and a literal repeated per accessor
// is a rename that compiles after changing some of them. The hand
// written packs spell it inline; this is the rule they were written
// before.
func QualifierConst(token string) string { return token + qualifierSuffix }

// IndexVar is the generated index value a consumer reaches through —
// `logCheckIndex`.
func IndexVar(token string) string { return token + indexSuffix }

// IndexType is the index value's type — `logCheckIndexT`.
//
// The suffix exists because the value carries the readable name: a
// consumer writes `logCheckIndex.Append.Smoke()` and never writes the
// type, so the type takes the awkward one.
func IndexType(token string) string { return IndexVar(token) + indexTypeSuffix }

// GroupType is one index member's type — `logAppendChecks` for a
// method group, `logModelChecks` for a family's.
//
// Uniform across both scopes, which is a decision the hand-written
// packs did not have to make: they spell the family group's type
// `<token>ModelIndex` and the method groups' `<token><Method>Checks`.
// A generator emits one rule, and the group is a group of checks
// whichever scope named it.
func GroupType(token, field string) string {
	return token + golang.ExportedName(field) + checksSuffix
}

// The emitted index's fixed words.
const (
	qualifierSuffix = "Qualifier"
	indexSuffix     = "CheckIndex"
	indexTypeSuffix = "T"
	checksSuffix    = "Checks"
)
