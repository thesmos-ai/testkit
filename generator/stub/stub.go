// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/deprecated"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/orderafter"
	"go.thesmos.sh/eidos/sdk"
	sdkgolang "go.thesmos.sh/eidos/sdk/golang"

	"go.thesmos.sh/testkit/generator/internal/naming"
	"go.thesmos.sh/testkit/generator/internal/source"
)

// Name is the plugin's stable identifier.
const Name = "stub"

// Capability is the label the plugin advertises so a downstream
// consumer can declare a documentary dependency on stub generation.
const Capability = "stub"

// DirectiveName is the bare directive name — without the `//testkit:`
// prefix — the plugin reads from source interfaces.
const DirectiveName sdk.DirectiveName = "stub"

// SlotName is the [sdk.EmitFile] slot both emit values append into.
// `top` renders between the package clause and the first core decl,
// which is where a template-rendered block of whole declarations
// belongs.
const SlotName = "top"

// KindStub and KindStubTests are the plugin-defined emit kinds. The
// backend resolves a template by the kind's string value, so each
// constant doubles as the name the matching template defines.
const (
	KindStub      sdk.Kind = "stub.double"
	KindStubTests sdk.Kind = "stub.test"
)

// Version composes into the pipeline's plugin fingerprint, which frontends
// fold into their cache keys — so a change here invalidates a warm cache
// populated when this plugin behaved differently. A plugin declaring no
// version contributes an empty string and can never invalidate anything,
// which is a silent staleness bug waiting for its first behavioural change.
//
// Bump it on any change to what this plugin emits — the Go projection or the
// templates alike.
//
// It is deliberately a constant rather than a digest of the templates, even
// though a digest would invalidate automatically. The version renders into
// every generated file's `Plugins:` header, so a content-derived one would
// churn the header of every output in every consuming repository on any
// template edit, and a golden diff would stop isolating what actually changed
// in the output. Stability in the header is worth the discipline; during
// development `--no-cache` covers the gap.
const Version = "1.4.0"

// WitnessKey is the directive key naming the concrete types a generic
// double's companion is instantiated at, in type-parameter order —
// `//testkit:stub witness=int,string`.
//
// Needed only where the constraint is one the generator cannot reason about.
// `any` and `comparable` are satisfied by anything and by every basic type
// respectively, so those are derived; a named constraint like
// `constraints.Ordered` is a reference into a package the generator never
// loaded, and guessing at its type set would produce a companion that fails to
// compile for a reason the author could not have predicted.
const WitnessKey = "witness"

// DefaultSuffix is the trailer appended to the source interface's
// name to form the stub type's identifier.
const DefaultSuffix = naming.StubSuffix

// Mixin names this plugin reads, taken from the packages that declare them.
//
// The vocabulary is the shape annotator's and testkit registers it whole, so a
// name spelled here as a literal would be a copy no rename could reach — and
// the failure would be silent rather than loud. A moved name would simply stop
// matching: every order guard would vanish from every generated double, the
// compile would still succeed, and the generated suite would still pass,
// because the check that would have caught it is the one that went missing.
const (
	// MixinDeprecated marks a method whose use should be reported.
	MixinDeprecated = deprecated.Name

	// MixinOrderAfter marks a method that may only be called once its
	// prerequisite has been.
	MixinOrderAfter = orderafter.Name
)

// MixinOrderAfterParam is the key carrying the prerequisite method's name, as
// in `//testkit:mixin orderafter fn=Prepare`.
const MixinOrderAfterParam = orderafter.ParamFn

// Options carries the plugin's user-tunable settings.
//
// Recording is not optional: it is what the suite, bench, and model tiers
// read to assert on interactions rather than only on return values. A
// toggle would let a consumer silently disable every tier above.
type Options struct {
	// Suffix overrides the stub type's name suffix. Empty falls back
	// to [DefaultSuffix].
	Suffix string `eidos:"suffix,default=Stub"`
}

// Plugin is the stub generator. The zero value is unusable; go
// through [New] so the embedded holder binds to the options field.
//
// The embedded base answers every declaration method — name, version,
// priority, provides, requires, directives, and the per-language
// output, template and funcmap dispatch. Written out per plugin those
// drift: testkit carried sixteen hand-written copies of the language
// dispatch, and two of them tested the backend's language marker
// against a local constant rather than the backend's own — plugins
// that emitted nothing, with no diagnostic, because the string did not
// match.
type Plugin struct {
	*sdkgolang.Base
	*sdk.Holder[Options]
	opts Options
}

// New returns a fresh plugin instance with the options holder bound.
//
// The templates register no helper of their own. Everything they call is
// either a backend builtin — `renderType`, `renderExpr`, `renderTypeParams`,
// `external`, `camel` — or one of [golang.AllFuncMap]'s signature helpers,
// which the base merges under this plugin's own prefix. The testkit import
// paths a template used to resolve through a local function are carried on
// the emit value instead; see [RuntimePaths].
//
// # Failure mode
//
// Build panics on a declaration the pipeline cannot serve — a missing
// output suffix, a duplicate output tag, a template tree that is not
// there. Every one is a mistake in this function rather than in a
// consumer's source, so it fires on the first construction in any
// test rather than on a run that renders a short file and explains
// why in no output at all.
func New() *Plugin {
	p := &Plugin{Base: sdkgolang.NewGenerator(Name, goTemplates, GoOutputs()...).
		Version(Version).
		// The foundation bucket: the double is a base type other
		// generators decorate, so it must exist before composition and
		// cross-cutting plugins run.
		Priority(sdk.GeneratorFoundation).
		Provides(Capability).
		// Nothing is required: the plugin reads source interfaces and
		// depends on no other plugin's contribution.
		Directives(directives()...).
		Build()}
	p.Holder = sdk.BindOptions(&p.opts)
	return p
}

// directives declares the `//testkit:stub` schema.
//
// The directive takes no positional argument and denies negation: a
// stub exists exactly where one is declared, so deleting the line is
// the suppression and a negated form would have nothing to act on.
func directives() []sdk.DirectiveSchema {
	return []sdk.DirectiveSchema{
		sdk.NewDirective(DirectiveName).
			Describe(
				"Generates a recording test double for the annotated interface, " +
					"plus a companion test file proving the double satisfies it. " +
					"Optional `witness=<types>` names the concrete types the " +
					"compile-time guard instantiates a generic interface at, " +
					"comma-separated; without it a generic interface gets the " +
					"double but no guard. The negated form is rejected — a stub " +
					"exists only where declared, so removing the directive is the " +
					"suppression.",
			).
			On(sdk.NodeKindInterface).
			// One key, declared. The Describe text said "takes no arguments"
			// while the generator read witness= three hundred lines below, and
			// nothing could catch the contradiction: an empty AllowedKeys means
			// unrestricted, so the directive accepted witness= and every typo
			// of it alike.
			AllowedKeys(WitnessKey).
			DenyNegation().
			Build(),
	}
}

// suffix returns the configured stub-type suffix, or the documented
// default when unset.
func (p *Plugin) suffix() string {
	if p.opts.Suffix != "" {
		return p.opts.Suffix
	}
	return DefaultSuffix
}

// Method is one rendered interface method.
//
// The signature itself — the method's name, its parameters, its return slots
// with their recorded-call fields and capture locals, and the named-return
// decision — comes from [golang.Sig], because every Go generator projects the
// same source the same way and four independent implementations had already
// disagreed about it. What this type adds is the naming convention specific to
// a double.
type Method struct {
	// Sig is the source signature in rendered form, embedded so `.Name`,
	// `.Params`, `.Returns`, `.NamedReturns` and `.Source` promote onto the
	// method and the templates can hand the whole projection to the shared
	// list helpers.
	//
	// [golang.Sig.Source] is the escape hatch a contributing generator reads
	// the method's own metadata through. Carried rather than re-derived
	// because after resolution the method set is not the interface's declared
	// methods, and a contributor walking the declarations would miss
	// everything an embedded interface added.
	//
	// [golang.Sig.NamedReturns] decides whether the generated signature
	// declares its return names. See [golang.NamedReturnsUsable] for why that
	// is all-or-nothing rather than per-return.
	*golang.Sig

	// CallType is the identifier of the per-method recorded-call
	// struct — `<Iface><Method>Call`.
	CallType string

	// StubType is the identifier of the per-method configuration point —
	// `<Iface><Method>Stub`. It embeds the runtime's MethodStub, which is
	// what supplies recording, fault injection, latency, and gates.
	StubType string

	// ReturnType is the identifier of the fixed-return holder —
	// `<Iface><Method>Return`.
	//
	// Exported because the generated companion lives in the external test
	// package and names it when boxing a call's answer, and because a
	// consumer writing their own assertions against a recorded answer needs
	// the same. Returns remains how it is configured.
	ReturnType string

	// OnField is the field on the aggregate double that exposes this
	// method's configuration — `On<Method>`.
	OnField string

	// Sequence classifies the method's return as a range-over-func sequence.
	// Its zero value reads as "not one", so a template branches on
	// [golang.Sequence.Kind] with no separate flag beside it.
	//
	// A sequence-returning method gains Yields helpers, because building one by
	// hand in every test is a closure a caller should not have to write.
	Sequence golang.Sequence

	// OrderAfter is the prerequisite method this one may only follow, taken
	// from the orderafter mixin's `fn` parameter. Empty when unconstrained.
	OrderAfter string

	// From names the embedded interface that contributed this method, empty
	// for one the source declared directly.
	//
	// Carried so the generated field says where it came from: a resolved
	// method set reads as if every method were declared on the interface, and
	// a double that grows because an embedded interface gained a method would
	// otherwise offer nothing to explain the change.
	From string

	// Mixins are the opt-in behavioural laws the source attached through
	// `//testkit:mixin <name>`, in the order they were written.
	//
	// Read from the annotator rather than declared as directives of this
	// plugin's own: the mixin vocabulary belongs to the shape annotator, the
	// corpus gate measures coverage against it, and a second declaration here
	// would be free to drift from it.
	Mixins []string
}

// Pin is one return position's literal for the check that Returns is honoured.
type Pin struct {
	// Type is the return's type, for the `var` declaration that carries the
	// literal.
	Type sdk.Ref

	// Text is a value distinguishable from the zero, empty where none can be
	// written for this type.
	Text string
}

// Pins returns one entry per result, each carrying a value the check can tell
// apart from an unconfigured double.
//
// The whole of what "answers with the value pinned by Returns" is worth. That
// check declared `var want T`, handed the zero to Returns and asserted the
// call answered it — which an implementation ignoring Returns entirely also
// does, because an unconfigured double answers the zero by design. Seven
// hundred and six generated sites asserted it, and none of the four hundred
// and sixty-two Returns calls passed a value that was not the zero.
//
// The runtime helper had already refused this exact check, in those words,
// and delegated it here on the grounds that a distinguishable value needs the
// type — which is true, and is what this derives. [stub.Behaviour] asserts the
// honest half beside it: an unconfigured double answers the zero.
//
// Empty Text is not a failure. A return whose type admits no literal this
// generator can write keeps the zero and the check says so rather than
// pretending; see the template's suppression.
func (m Method) Pins() []Pin {
	out := make([]Pin, 0, len(m.Returns))
	for i := range m.Returns {
		p := Pin{Type: m.Returns[i].Type}
		if b, ok := m.Returns[i].Type.(*sdk.BuiltinRef); ok {
			p.Text, _ = golang.SampleValues(b.Name, m.Name)
		}
		out = append(out, p)
	}
	return out
}

// ArgPins is [Method.Pins] over the parameters a test calls with.
//
// The same derivation, one field over, and the reason it is a second method
// rather than a parameterised one: the two answer different questions and only
// one of them may skip a slot. A result that admits no literal is left at its
// zero and [Method.Pinnable] declines the whole check; a *parameter* that
// admits none still has to be passed, so its zero is written and the call goes
// ahead.
//
// A variadic tail takes no literal either: the declaration spells a slice and
// a scalar sample does not assign to one. Left at its zero for the same
// reason, which the recorded-call check then compares against — the residue
// `gate.VacuityDebt` keeps a ceiling on.
func (m Method) ArgPins() []Pin {
	out := make([]Pin, 0, len(m.Params))
	for i := range m.Params {
		p := Pin{Type: m.Params[i].Type}
		b, builtin := m.Params[i].Type.(*sdk.BuiltinRef)
		if builtin && !m.Params[i].Variadic {
			p.Text, _ = golang.SampleValues(b.Name, m.Params[i].Name)
		}
		out = append(out, p)
	}
	return out
}

// Pinnable reports whether any result admits a distinguishable literal.
//
// One is enough: a call answering several results is wrong if any position
// comes back unconfigured, so the check earns its name as soon as a single
// position can be told apart.
func (m Method) Pinnable() bool {
	for _, p := range m.Pins() {
		if p.Text != "" {
			return true
		}
	}
	return false
}

// PinTargets names the left-hand side of the pinned call: `got<i>` where the
// slot admits a literal, `_` where it does not.
//
// [Method.Pinnable] answers per method — one slot with a literal earns the
// check — and the slots beside it were being asserted anyway, at their zero,
// against a double configured with that same zero. Both sides the zero, which
// passes for a double that honoured Returns and for one that ignored it. That
// is the defect the whole check exists to catch, surviving inside the check.
//
// So the unpinnable slots are blanked rather than compared, and the generated
// comment names them. A slot dropped in silence would be the other half of the
// same mistake.
func (m Method) PinTargets() []string {
	pins := m.Pins()
	out := make([]string, 0, len(pins))
	for i, p := range pins {
		if p.Text == "" {
			out = append(out, "_")
			continue
		}
		out = append(out, "got"+strconv.Itoa(i))
	}
	return out
}

// Unpinned names the result slots no literal can be written for, for the
// comment that says why the check is silent about them.
func (m Method) Unpinned() []string {
	var out []string
	for i, p := range m.Pins() {
		if p.Text == "" {
			out = append(out, "result "+strconv.Itoa(i)+" ("+golang.QName(m.Returns[i].Source)+")")
		}
	}
	return out
}

// HasMixin reports whether the source attached the named mixin, which is how
// the template decides whether to emit the configuration that mixin implies.
func (m Method) HasMixin(name string) bool { return slices.Contains(m.Mixins, name) }

// Deprecated reports whether the source marked this method deprecated.
//
// Answered here rather than by a template asking [Method.HasMixin] for a name
// spelled as a template literal. A literal in a template is the one copy of an
// upstream name that neither a rename nor a compiler can reach, and the mixin
// vocabulary belongs to the shape annotator.
func (m Method) Deprecated() bool { return m.HasMixin(MixinDeprecated) }

// HasResults and ErrReturn are promoted from the embedded [golang.Sig].
// HasResults decides whether a fixed-return holder and its Returns setter are
// emitted at all; ErrReturn names the slot fault injection stamps the injected
// error onto before recording, found by flag rather than by position because a
// signature returning `(error, bool)` is unusual and legal.

// Stub is the emit value rendered into the primary output.
type Stub struct {
	sdk.BaseEmit
	RuntimePaths

	// TypeName is the stub struct's identifier — `<Iface><Suffix>`.
	TypeName string

	// CtorName and DelegateToName are the double's constructor and the option
	// that wraps a real implementation.
	//
	// Fields rather than something each reader composes from TypeName. The
	// templates spell them, and so does any generator building a call to
	// them — the conformance harness runs itself a second time through the
	// double, and two derivations of one identifier are two chances to name a
	// symbol this plugin never emitted.
	CtorName, DelegateToName string

	// IfaceName is the source interface's identifier, used in the
	// generated doc comments.
	IfaceName string

	// IfaceRef qualifies the source interface for DelegateTo's parameter.
	//
	// A double routed into its own package through `out=` no longer shares a
	// package with the interface it doubles, so the reference has to carry
	// one. Where the two do share a package the backend renders it bare.
	IfaceRef *sdk.Expr

	Methods []Method

	// TypeParams is the source interface's type-parameter list, in the form
	// `renderTypeParams` spells as `[K comparable, V any]`. Empty for a
	// non-generic interface, where the helper renders nothing.
	TypeParams []*sdk.EmitTypeParam

	// TypeArgs is the same list in use position — `[K, V]`, or empty. Every
	// generated identifier that names one of the double's own types has to
	// carry it, since a generic type cannot be referenced bare.
	TypeArgs string

	// Witnesses are the concrete types the compile-time guard instantiates a
	// generic double at, in parameter order. Empty for a non-generic double,
	// and for one whose constraints admit no witness — the latter gets no
	// guard, because there is no way to name the types it would hold at.
	Witnesses []sdk.Ref
}

// Generic reports whether the double is parameterised, which is what decides
// whether a companion can be generated for it at all — a Go test function
// cannot take type parameters, so the checks have no way to name the types the
// double is instantiated at.
func (s *Stub) Generic() bool { return len(s.TypeParams) > 0 }

// Ordered reports whether any method carries an order constraint, which is
// what decides whether the double allocates a tracker at all.
func (s *Stub) Ordered() bool {
	// Indexed rather than ranged by value: Method is wide enough that copying
	// one per iteration is measurable, and nothing here needs a copy.
	for i := range s.Methods {
		if s.Methods[i].OrderAfter != "" {
			return true
		}
	}
	return false
}

// Kind returns [KindStub].
func (*Stub) Kind() sdk.Kind { return KindStub }

// Tests is the emit value rendered into the tagged test output.
//
// The companion always lands in an external test package — the
// framework appends `_test` to whatever package the primary output
// resolved to — so it can never reach either type it exercises
// unqualified. Both are carried as [sdk.NewExternal] expressions and
// the backend registers the qualifying imports.
//
// The two references resolve from different places, and the
// difference is the whole reason [Tests] implements
// [sdk.OutputPackageSetter]:
//
//   - IfaceRef names the source interface, which is hand-written and
//     stays where the author put it. Its package is known during
//     Generate.
//   - StubRef names the stub this plugin generates, which follows
//     `out=` / `pkg=` routing. Its package is not decided until
//     Layout, so it is filled in by [Tests.SetOutputPackages].
type Tests struct {
	sdk.BaseEmit
	RuntimePaths

	TypeName  string
	IfaceName string

	// IfaceRef qualifies the source interface. Set during Generate.
	IfaceRef *sdk.Expr

	// CtorRef qualifies the double's constructor, which lives beside it and
	// therefore follows the same routing.
	CtorRef *sdk.Expr

	// StubRef qualifies the generated stub. Set during Generate
	// against the source package as a provisional value, then
	// corrected by [Tests.SetOutputPackages] once routing resolves.
	// The provisional value is what a run without a Layout phase —
	// a direct generator unit test — observes.
	StubRef *sdk.Expr

	Methods []Method

	// TypeArgs is the type-parameter list in use position — `[K, V]` — which
	// is what the generic check helpers instantiate the double at. Distinct
	// from Witnesses: inside a helper the double is named at the helper's own
	// parameters, and only the entry point substitutes concrete types.
	TypeArgs string

	// TypeParams is the source interface's type-parameter list in declaration
	// form. The checks live in generic helpers carrying it, because a Go test
	// function cannot take type parameters and an entry point therefore has to
	// instantiate rather than parameterise.
	TypeParams []*sdk.EmitTypeParam

	// Witnesses are the concrete types each entry point instantiates at, in
	// parameter order. Empty for a non-generic double.
	//
	// References rather than strings: a witness declared in the source package
	// is not reachable unqualified from the external test package, exactly as a
	// sentinel is not.
	Witnesses []sdk.Ref

	// Generic reports that the double is parameterised and no witness could be
	// found for it, which is the one case where a companion cannot be written.
	// It gets a note in place of its checks rather than no file at all: the
	// absence has to be visible to a reader who expected one, and an empty file
	// would read as a generator that failed silently.
	Generic bool
}

// Kind returns [KindStubTests].
func (*Tests) Kind() sdk.Kind { return KindStubTests }

// SetOutputPackages repoints [Tests.StubRef] at wherever Layout
// routed the primary output.
//
// The companion is always the external test package of the primary
// (`<pkg>_test`), so the reference is always qualified — there is no
// routing under which the stub and its test share a package, and
// therefore no case where the correct rendering is a bare name.
//
// An empty path for the primary tag means the Target resolved
// without a derivable import path, which centralised routing does.
// The provisional source-package reference is left in place rather
// than replaced with an unqualified name: a wrong package is a
// compile error naming the symbol, while a bare name silently binds
// to whatever else is in scope.
func (t *Tests) SetOutputPackages(byTag map[string]string) {
	path, ok := sdk.PrimaryPackage(byTag)
	if !ok {
		return
	}
	t.StubRef = sdk.NewExternal(path, t.TypeName)
	t.CtorRef = sdk.NewExternal(path, "New"+t.TypeName)
}

// Generate walks every source interface carrying `//testkit:stub` and
// queues one [Stub] against the primary output and one [Tests]
// against the tagged test output. The Layout phase resolves each
// contribution's target; both land beside the source interface by
// default and follow directive / config / CLI overrides otherwise.
//
// Interfaces without the directive are skipped silently. An
// annotated interface with no methods is skipped with a positioned
// diagnostic — a double with no behaviour to stand in for is
// certainly a mistake, and emitting an empty struct would hide it.
func (p *Plugin) Generate(ctx *sdk.GeneratorContext) error {
	c := sdk.NewProvenance(Name)
	for _, iface := range ctx.Reader.Interfaces().Slice() {
		if !iface.HasPositiveDirective(DirectiveName) {
			continue
		}
		if golang.IsConstraintInterface(iface) {
			// A constraint declares a type set, not a method-set contract, so
			// there is no behaviour to stand in for. Declined here rather than
			// left to the embed walk: a term like `~MyInt` is a Named ref and
			// indistinguishable from an unloaded interface by shape alone, so
			// the walk would report it as an embed the run failed to resolve —
			// naming something the author never wrote. This reads the Go
			// frontend's own stamp, which is the only answer that knows.
			ctx.Diag.Errorf(iface.Pos(),
				"%s: interface %q carries //testkit:%s but is a generic constraint, not a "+
					"method-set contract; there is nothing to double",
				Name, iface.QName(), DirectiveName)
			continue
		}
		typeName := iface.Name + p.suffix()
		set, complete := source.MethodSet(ctx, iface, Name, doubleConsequence)
		if !complete {
			// Nothing is emitted for an interface whose method set could not be
			// completed. A double missing a method does not satisfy the
			// interface it doubles, so it cannot be passed anywhere that
			// interface is expected — which is the whole of what a double is
			// for. Recording faithfully is worth nothing if nothing can accept
			// it, so the diagnostic raised during resolution stands alone rather
			// than accompanying an artefact that cannot be used.
			continue
		}
		methods := methodsOf(iface, set)

		// Measured after projection rather than on the source method set: an
		// interface whose every method is integration-only has methods but
		// nothing a double can stand in for, and emitting the shell would
		// produce a type that satisfies nothing and records nothing.
		if len(methods) == 0 {
			ctx.Diag.Errorf(iface.Pos(),
				"%s: interface %q carries //testkit:%s but has no methods a double can stand in for",
				Name, iface.QName(), DirectiveName)
			continue
		}

		witnesses := witnessesOf(ctx, iface)

		base := sdk.EmitBase(c, iface)

		// Queued in one call rather than two. The pair differs only in its emit
		// kind and output tag, and a second append is where the two would drift.
		if err := sdk.QueueEmit(ctx.Store.Emit(), c, SlotName, iface,
			&Stub{
				BaseEmit:       base,
				RuntimePaths:   GoRuntime(),
				TypeName:       typeName,
				CtorName:       "New" + typeName,
				DelegateToName: typeName + "DelegateTo",
				IfaceName:      iface.Name,
				IfaceRef:       sdk.NewExternal(iface.Package, iface.Name),
				Methods:        methods,
				TypeParams:     golang.TypeParamDecls(iface.TypeParams),
				TypeArgs:       golang.TypeParamNames(iface.TypeParams),
				Witnesses:      witnesses,
			},
			&Tests{
				BaseEmit:     sdk.EmitBaseTagged(base, GoTestOutputTag),
				RuntimePaths: GoRuntime(),
				TypeName:     typeName,
				IfaceName:    iface.Name,
				StubRef:      sdk.NewExternal(iface.Package, typeName),
				CtorRef:      sdk.NewExternal(iface.Package, "New"+typeName),
				IfaceRef:     sdk.NewExternal(iface.Package, iface.Name),
				Methods:      methods,
				TypeArgs:     golang.TypeParamNames(iface.TypeParams),
				TypeParams:   golang.TypeParamDecls(iface.TypeParams),
				Witnesses:    witnesses,
				Generic:      len(iface.TypeParams) > 0 && len(witnesses) == 0,
			},
		); err != nil {
			// Wrapped even though the queue names the plugin and the slot: what
			// it cannot name is which declaration the run was on when it failed,
			// and that is the only part a reader needs to find the source line.
			return fmt.Errorf("%s: queue interface %q: %w", Name, iface.Name, err)
		}
	}
	return nil
}

// doubleConsequence is what this generator loses to an unresolvable
// embed, for the diagnostics [methodset.Resolve] writes.
var doubleConsequence = source.Consequence{
	Partial:    "the double carries whatever the source had already contributed",
	Incomplete: "a double missing a method cannot stand in for the interface it doubles",
}

// witnessesOf resolves the concrete types the companion's entry points
// instantiate the double at, or nil when it cannot.
//
// A source-pinned list wins over derivation and is all-or-nothing: a partially
// pinned list would leave the generator guessing which position a lone entry
// meant, and a wrong guess is a compile error in generated code.
//
// Nothing here checks that a witness satisfies its constraint. It cannot — the
// constraint is a reference into a package the generator never loaded — so a
// wrong witness surfaces when the generated file is compiled. That is a loud
// failure naming the type, which is the best available outcome for a fact the
// generator has no way to know.
func witnessesOf(ctx *sdk.GeneratorContext, iface *sdk.Interface) []sdk.Ref {
	if len(iface.TypeParams) == 0 {
		return nil
	}
	// The directive is read here rather than inside, so the reader is the one
	// that already knows there is one: Generate walks only interfaces carrying
	// it, and a nil guard one call down would be a branch no input can reach.
	if dir := iface.Directive(DirectiveName); dir != nil {
		if pinned, ok := pinnedWitnesses(ctx, iface, dir); ok {
			return pinned
		}
	}
	return golang.Witnesses(iface.TypeParams)
}

// pinnedWitnesses reads the witness key off the interface's stub directive,
// reporting a list whose length does not match the type-parameter list.
//
// The second result distinguishes "the source pinned nothing" from "the source
// pinned something unusable": the first falls through to derivation, the second
// has already been diagnosed and must not be silently replaced by a guess.
//
// Each name is lifted by [golang.RefFor], which renders a predeclared type
// bare and qualifies anything else against the source package — the companion
// lives in an external test package and reaches nothing there unqualified.
func pinnedWitnesses(ctx *sdk.GeneratorContext, iface *sdk.Interface, dir *sdk.Directive) ([]sdk.Ref, bool) {
	// The first directive of this name rather than the first carrying the key.
	// The schema denies negation but does not forbid a second `//testkit:stub`
	// line, so a source putting the witness list on that second line now gets
	// derived witnesses where it once got pinned ones. Both answers are silent,
	// and the shorter rule is the one every source in the corpus writes.
	raw, given := dir.KV[WitnessKey]
	if !given {
		return nil, false
	}
	names := strings.Split(raw, ",")
	if len(names) != len(iface.TypeParams) {
		ctx.Diag.Errorf(iface.Pos(),
			"%s: %s=%q on %s names %d type%s for %d type parameter%s; supply one per parameter",
			Name, WitnessKey, raw, iface.Name,
			len(names), plural(len(names)), len(iface.TypeParams), plural(len(iface.TypeParams)))
		return nil, true
	}
	out := make([]sdk.Ref, 0, len(names))
	for _, n := range names {
		out = append(out, golang.RefFor(strings.TrimSpace(n), iface.Package))
	}
	return out, true
}

// plural returns the suffix that makes a count read correctly.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// methodsOf lifts every method a double carries into the rendered form both
// outputs share. Free function rather than a method: the lifting depends only
// on the source signature and the annotator's stamps, not on plugin options.
//
// Driven off [sdk.MethodSetResult.Entries] rather than Methods, because the
// two are index-aligned and only the entry says which embed a method arrived
// through — the fact the generated field's doc comment carries.
func methodsOf(iface *sdk.Interface, set sdk.MethodSetResult) []Method {
	out := make([]Method, 0, len(set.Entries))
	for _, entry := range set.Entries {
		m := entry.Method
		from, _ := sdk.EmbedName(entry.From)
		out = append(out, Method{
			// One projection replaces four calls: the parameter identifiers,
			// the return fields, the capture locals and the named-return
			// decision are derived together, and the receiver identifier the
			// collision guard reserves is the one the templates bind.
			// Named rather than left to default, because naming it is what
			// makes the guard run: the identifier is made unique against the
			// parameters, so an interface declaring `Put(s Session) error`
			// binds the receiver to something else instead of declaring `s`
			// twice in one signature. The letter is eidos's default, so no
			// output moves except the signatures that did not compile.
			Sig:        golang.SigOf(m, golang.WithReceiverIdent(golang.DefaultReceiverIdent)),
			From:       from,
			CallType:   iface.Name + m.Name + "Call",
			StubType:   iface.Name + m.Name + "Stub",
			ReturnType: iface.Name + m.Name + "Return",
			OnField:    "On" + m.Name,
			Mixins:     shape.Mixins(m.Meta()),
			OrderAfter: orderAfter(m),
			Sequence:   golang.SequenceOf(m),
		})
	}
	return out
}

// orderAfter reads the prerequisite method from the orderafter mixin's
// parameter, or returns empty when the method carries no such constraint.
//
// Three steps, none of which eidos can take for us: confirm the mixin is
// attached, read the stamp, and take the trailing identifier off the qualified
// name the shape resolver rewrote it into. Generated code calls the prerequisite
// on the subject it already holds, so the qualified form is exactly what it
// cannot use.
func orderAfter(m *sdk.Method) string {
	bag := m.Meta()
	if !slices.Contains(shape.Mixins(bag), MixinOrderAfter) {
		return ""
	}
	name, _ := shape.MixinParamKey(MixinOrderAfter, MixinOrderAfterParam).Get(bag)
	return golang.LocalName(name)
}
