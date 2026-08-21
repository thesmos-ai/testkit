// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"strings"
	"text/template"

	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/projection"
)

// renderFuncs is the rewrite's template function map, contributed to
// the backend's merged map through the plugin builder's Funcs seam —
// which is what lets the body templates live in templates/golang
// beside the plugin's structural ones: the backend parses every .tmpl
// in the plugin's FS with one merged map, so the functions must be in
// it. Registered bare; the sdk prefixes them under the plugin's name,
// and the templates call the prefixed form (suite_callExpr).
func renderFuncs() template.FuncMap {
	return template.FuncMap{
		"callExpr":          callExpr,
		"methodConst":       projection.MethodConst,
		"qualifierConst":    projection.QualifierConst,
		"harnessName":       projection.HarnessName,
		"subjectType":       subjectType,
		"withParam":         withParam,
		"checksName":        projection.ChecksName,
		"rowsName":          projection.RowsName,
		"rowName":           projection.RowName,
		"methodsVar":        projection.MethodsVar,
		"defectTypeName":    projection.DefectTypeName,
		"brokenName":        projection.BrokenName,
		"suiteName":         projection.SuiteName,
		"veneerVar":         projection.VeneerName,
		"configName":        projection.ConfigName,
		"defaultConfigName": projection.DefaultConfigName,
		"newFixtureName":    projection.NewFixtureName,
		"corpusName":        projection.CorpusName,
		"invariantsName":    projection.InvariantsTestName,
		"lockPath":          projection.LockPath,
		"limitConst":        projection.LimitConst,
		"missKeyName":       projection.MissKeyName,
		"corpusTypeName":    projection.CorpusTypeName,
		"fixtureIdent":      func() string { return string(projection.ExprFixture) },
		"docsIdent":         func() string { return string(projection.ExprDocs) },
		"runName":           projection.RunName,
		"proveName":         projection.ProveName,
		"runOptName":        projection.RunOptName,
		"runConfigName":     projection.RunConfigName,
		"dropOptName":       projection.DropOptName,
		"withoutName":       projection.WithoutName,
		"exportedWithout":   projection.ExportedWithoutName,
		"exportedSuiteOf":   projection.ExportedSuiteName,
		"veneerName":        projection.VeneerTypeName,
		"indexPathName":     projection.IndexPathName,
		"dropHintName":      projection.DropHintName,
		"indexVar":          projection.IndexVar,
		"indexType":         projection.IndexType,
		"groupType":         projection.GroupType,
		"borrowedIdent":     func() string { return string(projection.ExprBorrowed) },
		"producedIdent":     func() string { return string(projection.ExprProduced) },
	}
}

// bodyView is one check body's rendering context: the concrete body
// variant plus the facts the emitted text needs that no projection
// carries — the subject receiver's ident, the check-name constant
// the failure attributes to, and the two return shapes only the
// method's signature knows. The emitter builds one per check as it
// walks the inventory beside the methods.
type bodyView struct {
	Recv  string
	Check string

	// Vocab is the package the engine primitives live in, carried so a
	// body naming one registers the import through the canonical helper
	// rather than spelling a qualifier some other section happened to
	// have brought in.
	Vocab string

	// Discard drops a call's results where the body only asks whether
	// the call returned: "_ =" for one result, "_, _ =" for two.
	Discard string

	// ErrBind binds the error a context body inspects — "_, err :="
	// where a value precedes the error — and is EMPTY where the error
	// is the only result, which the packs return directly.
	//
	// Empty rather than "err :=" because a bind whose only use is the
	// next line is a local the reader has to follow: `return s.Put(ctx,
	// fx.Put())` says the whole thing. The two arms are the validated
	// spelling, and the difference between them is exactly the
	// difference between the signatures.
	ErrBind string

	// Seeds says the emitted assert function takes the run's corpus,
	// which the two seeded probes judge whole rather than drawing one
	// member from.
	Seeds bool

	// Draws says the emitted assert function takes the run's fixture,
	// which it does wherever the method has a drawable argument — the
	// packs' own rule, `storeAssertLenSmoke(tb, s)` beside
	// `storeAssertGetSmoke(tb, s, fx)`.
	//
	// Read from the method rather than from the body's arguments, and
	// so an over-approximation: the borrow arm substitutes the borrowed
	// local for a drawn one and may take the fixture without reading
	// it. That direction is free — an unused parameter compiles — while
	// the other is a body naming a local nothing declared.
	Draws bool

	// ObserveMethod is the partner a body calls beside the subject —
	// the observer a pair reads through, the reader an isolation check
	// asks, the validator an agreement compares against. Carried for the
	// messages that have to say WHICH method was involved. Empty on
	// every body that names no partner.
	ObserveMethod string

	// HookParams and HookReturns spell the callback a hooks body
	// declares: parameters blanked, results named so a bare return
	// answers each slot's zero without this file naming a type it may
	// not be able to spell. RegisterDiscard drops the registrar's own
	// results where it has any.
	//
	// Emit types rather than a projection shape, because the emitted
	// closure is a Go signature and the backend is what renders one.
	HookParams      []*sdk.EmitParam
	HookReturns     []*sdk.EmitReturn
	RegisterDiscard string

	// Guard is the engine primitive a guarded body delegates to, as the
	// emitted call spells it. A plain string rather than the typed
	// [projection.Guard] because the template hands it to the backend's
	// reference helper, which takes the identifier as text.
	Guard string

	// Method is the name the failure messages spell. The check constant
	// beside it is what the engine primitives take; a message is prose
	// and says the method the way the source declares it.
	Method string

	// ValueBind binds a call's results where the body judges the first
	// of them — "got, err :=", widening by one blank per extra result.
	ValueBind string

	// Pool is the config field the miss body's skip tells a consumer to
	// seed, empty on every other body.
	Pool string

	// NeedsCtx says the body's calls take a context, so it declares one;
	// HasErr says the method reports failure at all, which decides
	// whether a body can judge an error or only a value.
	NeedsCtx bool
	HasErr   bool

	// ValueDiscard blanks the results after the first, for a body that
	// judges one value from a method reporting no error.
	ValueDiscard string

	// ErrStmt binds the error inside an if-statement's init — `err :=`,
	// or `_, err :=` past each value. Never empty where the method
	// reports one, because a body judging the error in the condition
	// has nowhere else to bind it.
	ErrStmt string

	// Sentinel is the error a declared miss reports, resolved to a
	// reference so the body naming it registers the import. Nil where
	// the declaration names none, which is a different body.
	Sentinel *sdk.Expr

	// Zeros is every value result a zero-judging body holds to its own
	// zero, in declaration order.
	//
	// A list rather than one slot because the claim is about the whole
	// answer: a read returning a value beside metadata that zeroes the
	// first and leaks the second has told a caller the read failed and
	// handed them state anyway, and a body judging the first slot
	// renders identically to one judging both.
	Zeros []zeroSlot

	// ZeroBind binds every value slot plus the error, for a body that
	// judges them all; ZeroBindNoErr is the same without the error, for
	// a method that reports none.
	ZeroBind      string
	ZeroBindNoErr string

	Body projection.Body
}

// zeroSlot is one value result and how its zero is spelled.
type zeroSlot struct {
	// Bind is the identifier this slot binds to — "got" for the first,
	// numbered past it, so a single-slot body reads as it always did.
	// Zero is the local its declared zero binds to, numbered the same
	// way so the two read as a pair.
	Bind string
	Zero string

	// Nil says the slot compares against nil rather than a declared
	// zero, which is comparability rather than spelling.
	Nil bool

	// Type is the reference a declared zero is declared of, Word the
	// bare word for a predeclared one. At most one is set.
	Type *sdk.Expr
	Word string

	// Label names this slot in a failure message where there is more
	// than one, empty where the method answers a single value and the
	// message has nothing to disambiguate.
	Label string
}

// callExpr spells one invocation on the subject receiver, the args
// already rendered by the derivation.
func callExpr(recv string, c projection.CallPlan) string {
	args := make([]string, len(c.Args))
	for i, a := range c.Args {
		args[i] = string(a)
	}
	return recv + "." + c.Method + "(" + strings.Join(args, ", ") + ")"
}

// zeroJudge is the zero-comparison fragment's context: how the result's
// zero is spelled, what the body was asking about, and whether it has an
// error to print.
//
// The last two are the whole reason this type exists. The fragment used
// to read [bodyView.HasErr] — does the METHOD return an error — when the
// question is what THIS body caught. They come apart in both directions:
// a zero-on-error body has definitely caught one, and a miss body on a
// method that reports errors has an err in scope it cannot claim
// anything about, since a miss reporting a failure is exactly what the
// check permits.
type zeroJudge struct {
	Method string

	// Slots is every value result under judgement, in declaration
	// order.
	Slots []zeroSlot

	// Because is the phrase the failure gives for why the zero was owed
	// — "alongside an error", "for an input nothing supplied", "at the
	// size Len reports".
	//
	// A phrase rather than a flag. It was a bool selecting between two
	// hard-coded reasons, which held while two claims reached this
	// fragment; a third arrived and the choice stopped being binary.
	Because string

	// ShowErr says an err is bound and worth printing beside the
	// mismatch. Separate from Erred: a miss on an erroring method binds
	// one without the error being the claim.
	ShowErr bool
}

// ZeroJudge pairs this body's zero shape with why the zero was owed.
//
// A method rather than a template function because the dot inside a
// body is the emit node, which embeds this view — so promotion hands
// every body the same call without the funcmap carrying an entry that
// would have to be told which of the two shapes it was given.
func (v bodyView) ZeroJudge(because string, showErr bool) zeroJudge {
	return zeroJudge{
		Method:  v.Method,
		Slots:   v.Zeros,
		Because: because,
		ShowErr: showErr,
	}
}

// subjectType spells the interface as the generated declarations name
// it: the alias for a concrete interface, the qualified reference plus
// its arguments for a generic one.
//
// A generic interface has no witnessed instantiation to alias — `type
// Store = generic.Store` names a type that does not exist without its
// arguments — so the file spells what the alias was shorthand for. The
// packs never reach this: every validated interface is concrete.
func subjectType(c *Contract) string {
	if len(c.TypeParams) == 0 {
		return c.IfaceName
	}
	return c.IfaceName + c.TypeArgs
}

// withParam appends one parameter to a rendered type-parameter list.
//
// The harness declares the subject's own parameters ahead of T, because
// T is constrained by the interface and Go admits no forward reference:
// `StoreHarness[K comparable, V any, T Store[K, V]]`. The list itself
// is the backend's to render — only it knows how to spell a
// constraint — so this takes what it rendered and adds to it, which
// keeps the composition here and the spelling there.
func withParam(rendered, extra string) string {
	if rendered == "" {
		return "[" + extra + "]"
	}
	return strings.TrimSuffix(rendered, "]") + ", " + extra + "]"
}
