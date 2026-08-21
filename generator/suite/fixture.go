// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/answeringwriter"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/compositewriter"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/multiargwriter"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/writer"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/causal"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/naming"
	"go.thesmos.sh/testkit/generator/internal/source"
	"go.thesmos.sh/testkit/generator/internal/subject"
	"go.thesmos.sh/testkit/generator/suite/projection"
)

// fixtureOf derives one input per distinct parameter across the method set.
//
// Which parameters share a field, and how a name two types contest is spelled,
// is [subject.GroupParams]. What is decided here is what each group is filled with: the
// type's `<Type>Defaults()` where the source declares one, the composed parts of
// a struct, and the derived pair otherwise.
func fixtureOf(
	ctx *sdk.GeneratorContext, iface *sdk.Interface, methods []subject.Method,
	pools []projection.PoolPlan,
) subject.Fixture {
	f := subject.Fixture{
		TypeName: iface.Name + "Fixture",
		CtorName: "Default" + iface.Name + "Fixture",
	}
	defer func() { bindPools(&f, pools) }()
	f.Groups = subject.GroupParams(methods)
	for _, g := range f.Groups {
		// Both derivations run for every field, including a composed one
		// whose whole-value Sample the template never reaches. Skipping
		// it there looks free and is not: sampleFor answers the pair, and
		// a composed field still renders Other — the miss value a reader
		// check needs. Splitting the pair to save one resolve at build
		// time would buy a few microseconds for a seam where the two
		// values stop being derived together.
		sample, other := sampleFor(g.Param, ctx.Reader)
		f.Fields = append(f.Fields, subject.FixtureField{
			Name:      g.Name,
			Type:      g.Param.Type,
			Variadic:  g.Param.Variadic,
			Sample:    sample,
			Other:     other,
			Parts:     partsFor(g.Param, ctx.Reader, admitsFresh(methods, g.Method)),
			Companion: companionFor(ctx, g.Param.Source),
		})
	}
	return f
}

// The mixins whose claim is an admission precondition on the method's own
// input: the call is refused unless something the value names already
// happened.
//
// `causal` is the whole list today. Its claim is that an entry naming its
// causes lands only after they do, so a derived value carrying causes names
// entries no fresh subject holds — and the seed, which is the first thing a
// generated harness runs, fails before any check does. The subject is correct
// and the fixture is wrong, which is the worst way round.
//
// A table rather than a signature test because nothing in the signature says
// it. eidos's causal directive carries one parameter, `version`, and no stamp
// names the member that holds the dependencies — so the claim is the only
// evidence the derivation has that this input is not self-contained.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var admissionMixins = map[string]string{
	causal.Name: "the causal claim admits an entry only once its causes have landed",
}

// admitsFresh reports whether a value derived for this parameter can be handed
// to a subject holding no state yet.
//
// False where the method introducing the parameter claims an admission
// precondition over it. The composed value then drops the parts that could
// express one — see [partsFor].
func admitsFresh(methods []subject.Method, method string) bool {
	for _, m := range methods {
		if m.Name != method {
			continue
		}
		for _, name := range m.Mixins {
			if _, constrained := admissionMixins[name]; constrained {
				return false
			}
		}
	}
	return true
}

// sampleFor derives one parameter's pair of values.
//
// For a struct it sets *every* exported field it can, which is where this
// departs from [golang.SampleRefFor] — and deliberately. That function sets the
// first settable field and says why: a sample exists to be distinguishable, and
// one field achieves that while staying readable. Right for a builder, where
// the value is handed to a setter and read straight back.
//
// A conformance suite asks a different question. An implementation that
// silently drops a field passes every check built from a sample that never set
// it, and two values differing in one field are indistinguishable to a subject
// keyed on another. Discrimination is the whole point here, and readability is
// the thing to trade for it.
//
// The per-field values are still eidos's — only the policy of how many to set
// is testkit's, and that is a testing decision rather than a fact about Go.
// Nesting is left to [golang.SampleRefFor], so a struct inside a struct gets
// eidos's one-field form: the field that matters is the parameter's own.
func sampleFor(p golang.Param, r golang.Resolver) (sample, alternate golang.Sample) {
	return derivedPair(p.Source, p.Name, r)
}

// The bare-integer pair a conformance fixture draws.
//
// eidos samples an integer as 42 and 7, which is right for a builder — a
// magnitude set and read back — and wrong here. A generated suite hands its
// fixture values to the subject, and the integers a Go interface takes are
// mostly positions and sizes: `Less(i, j int)` against the five-element slice
// the harness seeded is an index panic, not a failed claim, and the consumer
// reads it as the tool being broken.
//
// Small enough to be a valid index into anything the fixtures build, and
// non-zero on purpose: a sample equal to the zero value cannot tell a subject
// that stored the field from one that dropped it, which is the vacuity the
// builder seeds already suffer from. 1 and 2 are the smallest pair that is
// both in-range and discriminating.
//
// Bare builtins only. A defined type over an integer — a `Weekday`, a
// `Priority` — is a domain of its own, and eidos spells it `Weekday(42)`; the
// value is not an index into anything and narrowing it would only make the
// sample less distinguishable.
//
// This is the interim. The real answer is an index-domain vocabulary that
// says "this parameter is bounded by Len()", so the fixture derives a value
// the subject's own state admits rather than one small enough to usually fit.
const (
	smallIntSample    = "1"
	smallIntAlternate = "2"
)

// derivedPair is [golang.SampleRefFor] under testkit's integer policy — the
// one derivation both the parameter and the struct-field paths draw from, so
// a field and the parameter carrying it cannot disagree about what an int is.
func derivedPair(
	t *sdk.TypeRef, name string, r golang.Resolver,
) (sample, alternate golang.Sample) {
	sample, alternate = golang.SampleRefFor(t, name, r)
	if !golang.IsInteger(t) {
		return sample, alternate
	}
	sample.Text = smallIntSample
	alternate.Text = smallIntAlternate
	return sample, alternate
}

// partsFor composes a struct parameter's value field by field.
//
// Every exported field it can, which is where this departs from
// [golang.SampleRefFor] — and deliberately. That function sets the first
// settable field and says why: a sample exists to be distinguishable, and one
// field achieves that while staying readable. Right for a builder, where the
// value is handed to a setter and read straight back.
//
// A conformance suite asks a different question. An implementation that
// silently drops a field passes every check built from a sample that never set
// it, and two values differing in one field are indistinguishable to a subject
// keyed on another. Discrimination is the point here, and readability is the
// thing to trade for it.
//
// The per-field values are still eidos's — only the policy of how many to set
// is testkit's, and that is a testing decision rather than a fact about Go.
// Each is carried as a [golang.Sample] rather than as text, so a field whose own
// type is a struct keeps the reference the backend needs to spell it.
//
// admissible false leaves every collection-typed field at its zero. The
// method claims a precondition over this value — see [admissionMixins] — and
// a collection is the only shape a precondition can be written in: a set of
// things that must already exist. Derived, it names things that do not, and
// the fresh subject the harness seeds correctly refuses the whole value.
// Zeroing costs the discrimination those fields would have carried, which is
// the cheaper half of the trade: a check that compares less still runs, and a
// seed that cannot land runs nothing at all.
func partsFor(p golang.Param, r golang.Resolver, admissible bool) []subject.FixturePart {
	decl, resolved := r.Resolve(p.Source)
	s, ok := decl.(*sdk.Struct)
	if !resolved || !ok {
		return nil
	}

	var parts []subject.FixturePart
	for _, f := range golang.ExportedFields(s) {
		if !admissible && f.Type != nil && (f.Type.IsSlice() || f.Type.IsMap()) {
			continue
		}
		inner, innerAlt := derivedPair(f.Type, f.Name, r)
		if !inner.OK() {
			// A field no literal can be written for is left at its zero rather
			// than losing the whole sample: the fields around it still
			// discriminate, and refusing here would drop every check the
			// parameter feeds.
			continue
		}
		parts = append(parts, subject.FixturePart{Name: f.Name, Sample: inner, Other: innerAlt})
	}
	return parts
}

// companionFor returns a call to the type's `<Type>Defaults()` function, or nil
// when the source declares none.
//
// The one place a team already writes down "here is a valid instance of this
// type". A derived sample is plausible strings, and a type with real validation
// — an identifier that must be a UUID, an address that must hold an `@` —
// accepts only some of them. Deriving one anyway means the first run of a
// correct implementation is a wall of failures that are the fixture's fault,
// which is how a suite gets switched off.
//
// Found by looking rather than by being told, which is [naming.CompanionSuffix]'s
// own rule — the function is an ordinary declaration and the convention is
// shared, so a package that wrote one for its builder gets this for free.
//
// The signature is checked rather than only the name: a `PayloadDefaults`
// taking arguments, or returning something else, is a different function that
// happens to collide, and calling it would emit a fixture that does not compile.
func companionFor(ctx *sdk.GeneratorContext, t *sdk.TypeRef) *sdk.Expr {
	if t == nil || t.Name == "" {
		return nil
	}
	return source.Companion(ctx, t.Package, t.Name, naming.CompanionSuffix)
}

// seedOf names the writer a harness populates its subject through, or nil when
// the interface declares none.
//
// A reader over an empty subject asserts nothing, so something has to write
// first — and for any interface carrying a writer, that something is the
// interface itself. Nothing is asked of the consumer: the shape annotator has
// already said which method writes, and the fixture already holds a value of
// what it takes.
//
// A read-only interface over external state has no writer, and is the case a
// consumer's own seed exists for.
//
// The first writer in method-set order rather than a choice between several.
// Two writers over one value type is a shape this cannot resolve — and where
// they differ, an author who cares supplies a seed rather than being asked
// which is meant.
func seedOf(f subject.Fixture, methods []subject.Method) (*Seed, string) {
	var (
		mute        []string
		undelivered []string
	)
	for _, m := range methods {
		if !writesSomething(m) {
			continue
		}
		if !m.ReturnsError() {
			// A write that cannot report its own failure cannot seed: the
			// checks after it would assert against whatever state a silent
			// failure left.
			mute = append(mute, m.Name)
			continue
		}
		// The method already carries this: Generate derives ArgFields
		// per method before anything reads them, and deriving them a
		// second time here is a second chance to disagree about which
		// field a parameter draws from.
		args := m.ArgFields
		if _, _, undeliverable := undeliverableArgs(f, args); undeliverable {
			undelivered = append(undelivered, m.Name)
			continue
		}
		return &Seed{Method: m, Args: args, AnswersState: answersState(m)}, ""
	}
	return nil, whyUnseeded(mute, undelivered)
}

// whyUnseeded names which of seedOf's three exits was taken.
//
// Three reasons, not one, because the fix differs for each: a mute writer
// wants an error return, an undeliverable argument wants a fixture, and an
// interface with no writer at all wants the consumer's own seed. "No seed was
// derived" sends the reader to look for all three.
//
// Order is deliberate. A method that got as far as its arguments is the
// closest to being usable, so it is named first — that is the one worth
// fixing.
func whyUnseeded(mute, undelivered []string) string {
	switch {
	case len(undelivered) > 0:
		return "the arguments " + strings.Join(undelivered, ", ") +
			" takes cannot be written as literals; supply them through the fixture"
	case len(mute) > 0:
		return strings.Join(mute, ", ") + " writes but reports no error, " +
			"so a failed seed would leave every check after it asserting against an empty subject"
	default:
		return "no method here is classified as a write"
	}
}

// writesSomething reports whether the shape annotator classified this method as
// a write.
//
// Three detectors rather than one. They differ only in arity — `writer` takes a
// single non-context argument, `compositewriter` two, `multiargwriter` three or
// more — and the seed passes whatever the method declares, so arity is not
// something it has to know. Keying on `writer` alone left a `Put(ctx, key, v)`
// interface unable to seed itself, which is the ordinary keyed store.
//
// `answeringwriter` writes and answers the stored state; the seed discards
// the answer and keeps the error, which is all a seed reports.
//
// `mutator` is deliberately absent even though it writes: it returns nothing, so
// a seed through one cannot report its own failure, and a seed that fails
// silently leaves every check after it asserting against an empty subject. That
// exclusion is [Seed]'s error return restated, and [seedOf]'s ReturnsError guard
// would refuse it anyway.
func writesSomething(m subject.Method) bool {
	switch m.Shape() {
	case writer.Name, compositewriter.Name, multiargwriter.Name, answeringwriter.Name:
		return true
	default:
		return false
	}
}

// answersState reports whether the seed's writer answers the stored state
// beside its error, so the derived seed discards the value it does not read.
func answersState(m subject.Method) bool {
	return shape.Get(m.Source.Meta()) == answeringwriter.Name
}

// Seed is the write a harness populates each fresh subject with.
type Seed struct {
	// Method is the writer the seed calls.
	Method subject.Method

	// Args names the fixture fields it is handed.
	Args []string

	// AnswersState marks an answering writer: the seed discards the stored
	// state it does not read and keeps the error, which is all it reports.
	AnswersState bool
}

// undeliverableArgs names the first argument the fixture cannot supply, with
// the field it would have come from so a caller can say why.
func undeliverableArgs(f subject.Fixture, args []string) (string, subject.FixtureField, bool) {
	for _, name := range args {
		field, found := f.Field(strings.TrimSuffix(name, subject.OtherSuffix))
		if !found || !field.OK() {
			return name, field, true
		}
	}
	return "", subject.FixtureField{}, false
}

// bindPools points every fixture value at the config field its role
// opened, leaving the unroled ones on their derived literals.
//
// The match is by name and is exact. A pool is named for the declaration
// that carries the role — `<Name>Pool` — and a fixture value is named
// for the same declaration, so the correspondence is one both sides
// already computed rather than a third mapping to keep in step. A value
// with no matching pool keeps its literal, which is every value on an
// interface that stamps no role at all.
func bindPools(f *subject.Fixture, pools []projection.PoolPlan) {
	if len(pools) == 0 {
		return
	}
	byField := make(map[string]string, len(pools))
	for _, p := range pools {
		byField[p.Field] = p.Field
	}
	for i := range f.Fields {
		if pool, roled := byField[projection.PoolFieldName(f.Fields[i].Name)]; roled {
			f.Fields[i].Pool = pool
		}
		for j := range f.Fields[i].Parts {
			if pool, roled := byField[projection.PoolFieldName(f.Fields[i].Parts[j].Name)]; roled {
				f.Fields[i].Parts[j].Pool = pool
			}
		}
	}
}
