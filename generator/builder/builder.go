// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"fmt"
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"
	sdkgolang "go.thesmos.sh/eidos/sdk/golang"

	"go.thesmos.sh/testkit/generator/internal/naming"
	"go.thesmos.sh/testkit/generator/internal/source"
	"go.thesmos.sh/testkit/generator/internal/stamp"
)

// Name is the plugin's stable identifier.
const Name = "builder"

// Capability is the label the plugin advertises so a downstream consumer can
// declare a documentary dependency on builder generation.
const Capability = "builder"

// DirectiveName is the bare directive name — without the `//testkit:` prefix —
// the plugin reads from source structs.
const DirectiveName sdk.DirectiveName = "builder"

// CompanionKey names the seeding function explicitly, for one that does not
// follow the convention or does not live beside the struct:
//
//	//testkit:builder defaults=example.com/seed.UserDefaults
//
// The full-path notation matters: a companion in another package would
// otherwise need an import written only for this directive, which does not
// compile.
const CompanionKey = "defaults"

// CompanionSuffix forms the name of the seeding function a constructor calls:
// `User` is seeded from `UserDefaults()`.
//
// Convention rather than declaration. The function is an ordinary declaration
// in the source package, so it is found by looking rather than by being told,
// and a package holding several types gets one companion each — which is why
// the name carries the type rather than being a bare `Defaults`.
const CompanionSuffix = naming.CompanionSuffix

// SkipTag is the struct-tag key excluding a field from the builder:
//
//	Internal string `builder:"-"`
//
// For a field a test should never set but which cannot be unexported —
// something a neighbouring package reads directly. Any value other than `-` is
// rejected, so a typo is reported rather than silently keeping the setter.
const SkipTag = "builder"

// SkipValue is the only value [SkipTag] accepts.
const SkipValue = "-"

// SlotName is the [sdk.EmitFile] slot the builders land in. `top` renders
// between the package clause and the first core decl, which is where a
// template-rendered block of whole declarations belongs.
const SlotName = "top"

// KindBuilder and KindBuilderTests are the plugin-defined emit kinds. The
// backend resolves a template by the kind's string value, so each constant
// doubles as the name the matching template defines.
const (
	KindBuilder      sdk.Kind = "builder.type"
	KindBuilderTests sdk.Kind = "builder.test"
)

// Version composes into the pipeline's plugin fingerprint. Bump it on any
// change to what this plugin emits — the projection or the templates alike.
//
// A constant rather than a digest of the templates: the version renders into
// every generated file's `Plugins:` header, so a content-derived one would
// churn the header of every output in every consuming repository on any
// template edit.
const Version = "1.4.0"

// Suffix is the trailer appended to the source type's name to form the
// builder's identifier.
const Suffix = "Builder"

// Plugin is the builder generator. The zero value is unusable; go through
// [New], which is where the declaration is frozen.
type Plugin struct{ *sdkgolang.Base }

// New returns a fresh plugin instance.
//
// The foundation bucket is where a builder belongs: it is a base a later
// generator may decorate, so it exists before composition runs. Nothing is
// required — the plugin reads source structs and depends on no other plugin's
// contribution — so no capability is declared on that side.
//
// The templates register no helper of their own. Everything they call is
// either a backend builtin — `renderType`, `renderTypeParams`, `renderExpr`,
// `external` — or one of [golang.AllFuncMap]'s entries, which the base merges
// under this plugin's own prefix. The testkit import paths a template used to
// resolve through a local function are carried on the emit value instead; see
// [RuntimePaths].
//
// # Failure mode
//
// [sdkgolang.Builder.Build] panics on a declaration the pipeline cannot serve.
// That fires here, inside New, so it lands on the first test that constructs
// the plugin rather than on the first run that renders a short file.
func New() *Plugin {
	return &Plugin{Base: sdkgolang.NewGenerator(Name, goTemplates, GoOutputs()...).
		Version(Version).
		Priority(sdk.GeneratorFoundation).
		Provides(Capability).
		Directives(directives()...).
		Build()}
}

// directives declares the `//testkit:builder` schema.
//
// The directive takes no positional argument: a builder exists exactly where
// one is declared, so deleting the line is the suppression and a negated form
// would have nothing to act on.
func directives() []sdk.DirectiveSchema {
	return []sdk.DirectiveSchema{
		sdk.NewDirective(DirectiveName).
			Describe(
				"Generates a fluent builder for the annotated struct, plus a " +
					"companion test file exercising it. Takes no positional " +
					"argument. A `<Type>Defaults()` function in the same package " +
					"seeds the constructor, and per-field //testkit:default " +
					"directives override it. The negated form is rejected — a builder exists only where " +
					"declared, so removing the directive is the suppression.",
			).
			AllowedKeys(CompanionKey).
			On(sdk.NodeKindStruct).
			DenyNegation().
			Build(),
	}
}

// Shape classifies a field by what setter it owes, which depends on the
// field's type rather than on its name.
//
// It is testkit's vocabulary rather than Go's: eidos answers what a type *is*
// — [golang.IsSlice], [golang.IsByteSlice], [golang.IsMap],
// [golang.IsEmptyStruct] — and this names what a builder therefore owes it.
type Shape string

// The field shapes. Scalar is the zero value, so a projection that never
// classified reads as "one plain setter" rather than as an unhandled case.
const (
	// Scalar owes one replacing setter. Defined types land here and keep
	// their own type: a `Weekday int` setter taking `int` loses what the
	// declaration was for.
	Scalar Shape = ""

	// Slice owes a variadic replacing setter and an appending one.
	Slice Shape = "slice"

	// Bytes owes a `[]byte` setter and a string-accepting one, so a caller
	// with a string need not convert at every call site.
	Bytes Shape = "bytes"

	// Map owes a replacing setter, a single-entry setter, and a merging one.
	Map Shape = "map"

	// Set owes the same three, with the value parameter gone. A map to the
	// empty struct carries its whole meaning in its keys, so a setter asking
	// for the value asks the caller for the one thing they cannot vary.
	//
	// The value has to be an anonymous `struct{}`. A named one — `type unit
	// struct{}`, `map[string]unit` — arrives as a reference into a package this
	// generator never read, so its emptiness is not knowable here and the field
	// takes the ordinary map shape.
	Set Shape = "set"

	// Chan owes a plain setter, and a check comparing identity: a freshly made
	// channel is distinct from any the constructor could have seeded, so one
	// value proves what a comparable type needs two for.
	Chan Shape = "chan"

	// Error owes a plain setter, and a check matching by identity. An error is
	// an interface, so no value of it can be written down — but it is a builtin
	// interface with a universal constructor, which is what separates it from
	// [io.Reader] and the rest.
	Error Shape = "error"

	// Func owes a plain setter, and a check that a function arrived at all. A
	// func is not comparable, so there is nothing else to assert — but a setter
	// assigning nothing leaves nil, which is what the check catches.
	Func Shape = "func"

	// Pointer owes a setter taking the pointee by value and addressing it. A
	// pointer field distinguishes unset from zero, and the caller who wants to
	// say "set" should not have to produce an address to say it. Clearing the
	// field, or pointing two values at one address, goes through Mutate.
	Pointer Shape = "pointer"
)

// Sample is [golang.Sample] — the literal a generated check writes together
// with the type it is written against.
//
// Aliased rather than restated: the shape is eidos's, the derivation is
// [golang.SampleRefFor], and a local copy of a value type whose fields the
// templates read by name is exactly the drift this migration removed
// elsewhere.
type Sample = golang.Sample

// Field is one rendered struct field.
type Field struct {
	// Name is the field identifier, which is also what the setter is named
	// after — `Username` gives `WithUsername`.
	Name string

	// Type is the field's declared type. Named types arrive named: an alias is
	// its underlying type by the time the frontend records it, which is what
	// makes `Bytes = []byte` take the byte-slice setter rather than one of its
	// own.
	Type sdk.Ref

	Shape Shape

	// Elem is a slice's element type or a map's value type, nil otherwise.
	Elem sdk.Ref

	// Key is a map's key type, nil otherwise.
	Key sdk.Ref

	// Default is the field's declared default as Go source, empty when it
	// declared none. It renders straight into the constructor's literal.
	Default string

	// DefaultRef qualifies a default naming a symbol in another package, nil
	// when the default is a plain literal. A rendered file has to register the
	// import, which only a reference carries — text cannot.
	DefaultRef *sdk.Expr

	// Sample and Alternate are two distinct values of whatever the field's
	// setter takes, empty when its type admits none. See [Sample] for why the
	// checks need a pair rather than one value, and [resolver] for how far
	// "admits none" reaches.
	Sample    Sample
	Alternate Sample

	// Returns are a func field's return types, for the literal a check hands
	// its setter. Empty for every other shape.
	Returns []sdk.Ref
}

// Copies reports whether the field owns storage a clone must not share.
func (f Field) Copies() bool {
	return f.Shape == Slice || f.Shape == Bytes || f.Shape == Map || f.Shape == Set
}

// Builder is the emit value rendered into the primary output.
type Builder struct {
	sdk.BaseEmit

	// TypeName is the builder's identifier — `<Type>Builder`.
	TypeName string

	// SourceName is the struct's own identifier, which names the constructors:
	// `NewUser`, `NewUserFrom`.
	SourceName string

	// ValueRef qualifies the struct the builder constructs. A builder routed
	// into its own package cannot reach it unqualified, and where the two share
	// a package the backend renders it bare.
	ValueRef *sdk.Expr

	// TypeParams is the source struct's type-parameter list in declaration
	// form, empty for a plain struct.
	TypeParams []*sdk.EmitTypeParam

	// TypeArgs is the same list in use position — `[K, V]`, or empty.
	TypeArgs string

	Fields []Field

	// Companion qualifies the seeding function the constructor calls, nil when
	// the package declares none. It lives in the source package, so a builder
	// routed elsewhere cannot reach it unqualified.
	Companion *sdk.Expr
}

// Seeded reports whether any field declares a default, which is what decides
// whether the constructor builds a literal or an empty builder.
func (b *Builder) Seeded() bool {
	for i := range b.Fields {
		if b.Fields[i].Default != "" {
			return true
		}
	}
	return false
}

// Kind returns [KindBuilder].
func (*Builder) Kind() sdk.Kind { return KindBuilder }

// ZeroDefault reports whether the declared default is the field's zero value,
// where no check can tell a constructor that seeded it from one that did not.
//
// `//testkit:defaults 0` on an int and `nil` on a pointer are legitimate
// declarations — the author is saying the zero is deliberate — and the
// generated check for them asserted `0 == 0`. It passed for a constructor
// that ignored the directive entirely, which is the one thing it was there to
// notice.
//
// Sound in the direction that matters. A spelling this misses keeps a
// tautological check, which is the state everything was in before; a spelling
// it wrongly matches is impossible, because every literal listed here is the
// zero of whatever type accepts it.
func (f Field) ZeroDefault() bool {
	switch strings.TrimSpace(f.Default) {
	case "0", "nil", `""`, "false", "0.0":
		return true
	default:
		return false
	}
}

// Seedable returns the fields a generated check can set to a named value.
//
// The seed for the round-trip checks, and the reason they assert anything.
// Both used to build their seed with `var seed T` and compare it against
// itself, so `From(zero).Build() == zero` passed against a constructor that
// dropped every field it was given — the exact defect the round trip exists
// to catch. Set through the setters rather than a struct literal, because a
// pointer field needs an addressable value and the setter already takes one.
//
// Scalar shapes only, and the restriction is about what the *setter* accepts
// rather than what the sample is. A map field has a sample and a setter taking
// a map, a slice field a variadic one, a set field an entry at a time — so
// handing any of them the sample is a type error, which is what the corpus
// said the first time this was written without the guard. Each is driven
// through the shape it owns by its own per-field check.
func (t *Tests) Seedable() []Field {
	out := make([]Field, 0, len(t.Fields))
	for _, f := range t.Fields {
		if f.Shape == Scalar && f.Sample.OK() {
			out = append(out, f)
		}
	}
	return out
}

// Tests is the emit value rendered into the tagged test output.
//
// The companion lands in the external test package of wherever the builder was
// routed, so it reaches neither the builder nor the struct unqualified. The
// struct's package is known during Generate; the builder's is not decided until
// Layout, which is why [Tests] implements [sdk.OutputPackageSetter].
type Tests struct {
	sdk.BaseEmit
	RuntimePaths

	// TypeName is the builder's identifier, which names the generated check.
	TypeName string

	// SourceName is the struct's own identifier, which names the constructors
	// the check calls.
	SourceName string

	// CtorRef qualifies the builder's constructor. Set during Generate against
	// the source package as a provisional value, then corrected once routing
	// resolves — a wrong package is a compile error naming the symbol, while a
	// bare name silently binds to whatever else is in scope.
	CtorRef *sdk.Expr

	// FromRef qualifies the seeding constructor, which lives beside it.
	FromRef *sdk.Expr

	// ValueRef qualifies the struct the builder constructs.
	ValueRef *sdk.Expr

	TypeArgs   string
	TypeParams []*sdk.EmitTypeParam

	// Witnesses are the concrete types the entry points instantiate at, empty
	// for a plain struct and for one whose constraints admit none — the latter
	// gets a note in place of its checks.
	Witnesses []sdk.Ref

	Fields []Field

	// Seeded mirrors [Builder.Seeded], so the constructor's check asserts what
	// the constructor actually does rather than what it usually does.
	Seeded bool

	// Companion mirrors [Builder.Companion]. With one and no field defaults
	// the check compares the constructed value against the companion's own
	// return, which is exact — anything weaker would pass against a
	// constructor that called something else.
	Companion *sdk.Expr
}

// Generic reports that the struct is parameterised and no witness could be
// found, which is the one case where no check can be written: a Go test
// function cannot take type parameters, so there is nothing to instantiate at.
func (t *Tests) Generic() bool {
	return len(t.TypeParams) > 0 && len(t.Witnesses) == 0
}

// Copies reports whether any field owns storage a clone must not share, which
// is what decides whether the independence check is emitted at all.
func (t *Tests) Copies() bool {
	for i := range t.Fields {
		if t.Fields[i].Copies() {
			return true
		}
	}
	return false
}

// Kind returns [KindBuilderTests].
func (*Tests) Kind() sdk.Kind { return KindBuilderTests }

// SetOutputPackages repoints the references at wherever Layout routed the
// builder.
func (t *Tests) SetOutputPackages(byTag map[string]string) {
	path, ok := sdk.PrimaryPackage(byTag)
	if !ok {
		return
	}
	t.CtorRef = sdk.NewExternal(path, "New"+t.SourceName)
	t.FromRef = sdk.NewExternal(path, "New"+t.SourceName+"From")
}

// Generate walks every source struct carrying `//testkit:builder` and queues
// one [Builder] against the primary output and one [Tests] against the tagged
// test output.
//
// A struct with no exported fields is skipped with a positioned diagnostic: a
// builder with no setters configures nothing, and emitting the shell would hide
// a declaration that cannot do what it says.
func (*Plugin) Generate(ctx *sdk.GeneratorContext) error {
	c := sdk.NewProvenance(Name)
	for s := range ctx.Reader.Structs().All() {
		if !s.HasPositiveDirective(DirectiveName) {
			continue
		}
		fields := fieldsOf(ctx, s)
		if len(fields) == 0 {
			ctx.Diag.Errorf(s.Pos(),
				"%s: struct %q carries //testkit:%s but has no exported fields to set",
				Name, s.QName(), DirectiveName)
			continue
		}

		value := &Builder{
			BaseEmit:   sdk.EmitBase(c, s),
			TypeName:   s.Name + Suffix,
			SourceName: s.Name,
			ValueRef:   sdk.NewExternal(s.Package, s.Name),
			TypeParams: golang.TypeParams(s),
			TypeArgs:   golang.TypeArgs(s),
			Fields:     fields,
			Companion:  companionOf(ctx, s),
		}
		w := golang.Witnesses(s.TypeParams)

		// Queued in one call rather than two: the pair differs only in its emit
		// kind and output tag, and a second append is where the two would drift.
		if err := sdk.QueueEmit(ctx.Store.Emit(), c, SlotName, s, value, &Tests{
			BaseEmit:     sdk.EmitBaseTagged(value.BaseEmit, GoTestOutputTag),
			RuntimePaths: GoRuntime(),
			TypeName:     value.TypeName,
			SourceName:   s.Name,
			CtorRef:      sdk.NewExternal(s.Package, "New"+s.Name),
			FromRef:      sdk.NewExternal(s.Package, "New"+s.Name+"From"),
			ValueRef:     sdk.NewExternal(s.Package, s.Name),
			TypeArgs:     golang.WitnessUse(s.TypeParams),
			TypeParams:   value.TypeParams,
			Fields:       substituted(fields, s.TypeParams, w),
			Seeded:       value.Seeded(),
			Companion:    value.Companion,
			Witnesses:    w,
		}); err != nil {
			// Wrapped even though the queue names the plugin and the slot: what
			// it cannot name is which declaration the run was on when it failed,
			// and that is the only part a reader needs to find the source line.
			return fmt.Errorf("%s: queue struct %q: %w", Name, s.Name, err)
		}
	}
	return nil
}

// substituted rewrites each field's type with the witnesses the checks
// instantiate at, so a companion can be an ordinary non-generic test function.
//
// A Go test function cannot take type parameters, so a check naming `T` in a
// field position would not compile. Rewriting the projection is enough here
// because a builder's checks name types and nothing else — unlike a double's,
// which also name the subject's own methods.
//
// Returns fields unchanged when there is nothing to substitute, so the
// non-generic path allocates nothing: [golang.WitnessBindings] answers nil for
// a non-generic declaration and for lists that disagree in length, which is the
// case a partial rewrite would leave naming a parameter no longer in scope.
func substituted(fields []Field, params []*sdk.TypeParam, witnesses []sdk.Ref) []Field {
	by := golang.WitnessBindings(params, witnesses)
	if by == nil {
		return fields
	}
	out := make([]Field, len(fields))
	for i, f := range fields {
		f.Type = golang.SubstituteTypeParams(f.Type, by)
		f.Elem = golang.SubstituteTypeParams(f.Elem, by)
		f.Key = golang.SubstituteTypeParams(f.Key, by)
		// Upgraded rather than overwritten: substitution only ever resolves a
		// type parameter into something more concrete, so it can add a pair
		// where none was derivable and must never clear one that was.
		if sample, alternate := samplesOfRef(sampleSource(f), f.Name); sample.OK() {
			f.Sample, f.Alternate = sample, alternate
		}
		out[i] = f
	}
	return out
}

// companionOf finds the seeding function for s, or nil when none applies.
//
// A `defaults=` key names one explicitly, in either notation [source.Resolve]
// accepts, which is what lets a companion live in another package — including
// one imported only for this directive. Absent the key, the convention applies:
// a `<Type>Defaults()` beside the struct.
//
// The last declaration wins, matching `//testkit:default`. [sdk.Struct.Directive]
// is first-wins and answers a different question — whether the directive is
// there at all — and two tie-break rules for two directives in one repo is a
// difference nobody can predict from the outside.
//
// The signature is checked rather than only the name: a `UserDefaults` taking
// arguments, or returning something else, is a different function that happens
// to collide, and calling it would emit a constructor that does not compile.
func companionOf(ctx *sdk.GeneratorContext, s *sdk.Struct) *sdk.Expr {
	if dir := sdk.Last(s.Directives(), DirectiveName); dir != nil && dir.KV[CompanionKey] != "" {
		// The qualifier form resolves against the imports of the file that
		// declared the struct, so the file is what the resolver needs. Passing
		// no file at all only ever resolved the full-path form.
		pkgNode, _ := ctx.Reader.PackageAt(s.Package)
		pkg, symbol, err := source.Resolve(
			golang.FileOf(pkgNode, s), dir.KV[CompanionKey],
		)
		if err != nil {
			ctx.Diag.Errorf(s.Pos(), "%s: %s on %s: %v", Name, CompanionKey, s.Name, err)
			return nil
		}
		if pkg == "" {
			pkg = s.Package
		}
		return sdk.NewExternal(pkg, symbol)
	}
	return source.Companion(ctx, s.Package, s.Name, CompanionSuffix)
}

// fieldsOf lifts every field a builder can set.
//
// Unexported fields are skipped: a builder in another package cannot name them,
// and one in the same package would offer a setter the type's own invariants
// were written to prevent.
func fieldsOf(ctx *sdk.GeneratorContext, s *sdk.Struct) []Field {
	// The reader is the resolver: [store.Reader] implements [golang.Resolver],
	// so the sample walk needs nothing built for it.
	rv := ctx.Reader
	out := make([]Field, 0, len(s.Fields)+len(s.Embeds))
	// An embedded type is a field named after itself, and the frontend records
	// it apart from the declared ones. It takes a single setter for the whole
	// value rather than promoting the fields inside it: a struct literal sets
	// an embedded value as a unit, and promoting would offer two ways to write
	// the same thing that disagree about whether the embedded value is set.
	for _, e := range s.Embeds {
		name, pointer := golang.EmbedIdent(e)
		if name == "" || !golang.IsExported(name) {
			continue
		}
		field := Field{Name: name, Type: golang.FromNode(e.Type)}
		if pointer {
			// Embedded by pointer takes the same setter a pointer field does:
			// the promoted fields are reachable only once the pointer is
			// non-nil, so a setter demanding an address makes every caller
			// allocate before it can set anything.
			field.Shape = Pointer
			field.Elem = golang.FromNode(golang.EmbedTarget(e))
		}
		field.Sample, field.Alternate = golang.SampleRefFor(golang.EmbedTarget(e), name, rv)
		out = append(out, field)
	}
	for _, f := range golang.ExportedFields(s) {
		if skipped(ctx, s, f) {
			continue
		}
		if f.Type == nil {
			// A field whose type the run could not record. A setter needs one
			// to declare its parameter, and there is nothing to put there — the
			// template renders the reference unconditionally, so keeping the
			// field fails the whole run with a message naming a template line
			// rather than the declaration that caused it.
			ctx.Diag.Warnf(f.Pos(),
				"%s: %s.%s has no recorded type, so no setter is generated for it",
				Name, s.Name, f.Name)
			continue
		}
		field := Field{
			Name:    f.Name,
			Type:    golang.FieldType(f),
			Default: stamp.DefaultOf(f.Meta()),
		}
		if pkg := stamp.DefaultPackage(f.Meta()); pkg != "" {
			field.DefaultRef = sdk.NewExternal(pkg, field.Default)
		}
		classify(rv, &field, f.Type)
		out = append(out, field)
	}
	return out
}

// skipped reports whether the field opted out of a setter.
func skipped(ctx *sdk.GeneratorContext, s *sdk.Struct, f *sdk.Field) bool {
	raw, ok := golang.TagValue(f.Tag, SkipTag)
	if !ok {
		return false
	}
	if raw != SkipValue {
		ctx.Diag.Errorf(f.Pos(),
			"%s: %s.%s carries %s:%q; the only value that excludes a field is %q",
			Name, s.Name, f.Name, SkipTag, raw, SkipValue)
		return false
	}
	return true
}

// classify records the shape the field's setter follows, and the values its
// check sets it to.
//
// Every question asked here is eidos's; what is decided here is which setter
// testkit owes the answer.
func classify(rv golang.Resolver, field *Field, t *sdk.TypeRef) {
	// The cycle guard lives in [golang.SampleRefFor] now, bounded by its own
	// recursion budget, so nothing here threads a visited set through.
	switch {
	case t.IsFunc():
		field.Shape = Func
		field.Returns = golang.RefsOf(t.FuncReturns)
	case golang.IsChannel(t):
		// Every channel, not only the bidirectional ones. A directional
		// field used to fall through to the default arm, where the
		// sampler refused and the check was dropped; the sampler answers
		// now, and the default arm compares a value against a SECOND
		// evaluation of the same expression — two distinct channels,
		// never equal, a check that cannot pass.
		//
		// The sample is what the local binds, rather than a make built
		// from the field's own reference: `make(<-chan T)` is not legal
		// Go, and the direction lives in the reference's stamp where a
		// render cannot drop it. What the sampler answers is the
		// bidirectional form, which assigns to either direction.
		field.Shape = Chan
		field.Sample, field.Alternate = golang.SampleRefFor(t, field.Name, rv)
	case golang.IsError(t):
		field.Shape = Error
	case golang.IsByteSliceAny(t):
		// The `Any` spelling rather than [golang.IsByteSlice]: `[]uint8` is the
		// same type as `[]byte`, an author may have written either, and a field
		// spelled the second way owes the same string-accepting setter.
		field.Shape = Bytes
	case golang.IsSlice(t):
		field.Shape = Slice
		field.Elem = golang.ElemType(t)
	case golang.IsMap(t) && golang.IsEmptyStruct(golang.MapValue(t)):
		// Ahead of the map arm: a set is a map, and the narrower reading wins.
		// Elem stays nil — there is no value type worth carrying when every
		// value is the same one.
		field.Shape = Set
		field.Key = golang.MapKeyType(t)
		field.Sample, field.Alternate = golang.SampleRefFor(golang.MapKey(t), field.Name, rv)
	case golang.IsMap(t):
		field.Shape = Map
		field.Key = golang.MapKeyType(t)
		field.Elem = golang.MapValType(t)
	case t.IsPointer():
		field.Shape = Pointer
		field.Elem = golang.ElemType(t)
		field.Sample, field.Alternate = golang.SampleRefFor(golang.PointerElem(t), field.Name, rv)
	default:
		field.Sample, field.Alternate = golang.SampleRefFor(t, field.Name, rv)
	}
}

// samplesOfRef derives the pair from an emit reference, which is what a
// field's type has become by the time witnesses are substituted into it.
//
// Builtins only: a witness is always one, and this runs after the source types
// are gone, so there is nothing left to resolve a named type against.
func samplesOfRef(r sdk.Ref, fieldName string) (sample, alternate Sample) {
	b, ok := r.(*sdk.BuiltinRef)
	if !ok {
		return Sample{}, Sample{}
	}
	text, alt := golang.SampleValues(b.Name, fieldName)
	return Sample{Text: text}, Sample{Text: alt}
}

// sampleSource returns the reference a field's pair is derived from, which is
// whatever its setter actually takes: the pointee for a pointer, the key for a
// set, and the field's own type otherwise.
func sampleSource(f Field) sdk.Ref {
	switch f.Shape {
	case Pointer:
		return f.Elem
	case Set:
		return f.Key
	default:
		return f.Type
	}
}
