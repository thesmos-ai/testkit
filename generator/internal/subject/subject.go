// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package subject is the interface as every generator sees it: the naming
// each one emits declarations from, the method set with the
// classifications the annotator stamped, and the derived inputs the
// checks draw.
//
// It exists because the projection had no home of its own. It lived in
// [generator/suite], which is also its largest consumer, so the model
// tier read the harness generator's queued value — and when that value
// was reshaped, the model tier broke in forty-one places and was
// unregistered for a month.
//
// The rule that follows, and the reason this package must stay a leaf:
// one derivation, read by many plugins, owned by none of them.
// RFC-0002 was right that satellites must not re-derive what the harness
// already computed — two derivations of one thing disagree silently, and
// a benchmark seeded differently from the assertion it mirrors is the
// failure it named. What it got wrong was where the one derivation
// should live. Nothing about "one projection, many readers" requires the
// projection to sit inside the biggest reader.
//
// So: no plugin imports another plugin. What a plugin needs about the
// SOURCE comes from here; what it needs about another plugin's OUTPUT is
// that plugin's naming to publish, and its own to keep.
package subject

import (
	"slices"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"
)

// Subject is what every emit value in this file needs to name the interface it
// is about.
//
// Embedded in both the harness and each check rather than reached through a
// back-pointer from one to the other: the emit graph is walked and versioned,
// and a cycle in it is a hazard for the sake of three strings.
type Subject struct {
	// IfaceName is the source interface's identifier, which names every
	// generated declaration.
	IfaceName string

	// IfaceRef qualifies the interface. The harness is routed into its own
	// package in the ordinary case, where it is not reachable unqualified.
	//
	// A reference rather than an expression: this appears in type position —
	// a parameter, a struct field — and the two render through different
	// builtins. [sdk.NewExternal] builds the expression form, which is what a
	// call site needs and what a type position rejects. [sdk.External] is the
	// reference form and would serve; [golang.RefFor] is taken instead because
	// it answers for a predeclared name too, so one call covers every type an
	// interface could be named by.
	IfaceRef sdk.Ref

	// IntegrationEnv is the variable a run sets to include integration-only
	// checks. See [suite.GoIntegrationEnv].
	IntegrationEnv string

	// Runtime is testkit's module root, where the assertion helpers the
	// generated checks call live. The backend's `external` builtin turns a path
	// and a symbol into a qualified reference and registers the import, so a
	// path is all a template needs.
	Runtime string

	// ClockRef is [clock.Clock] in type position — a config field, a parameter.
	//
	// Built here rather than composed in a template, because `external` yields
	// the expression form and a type position rejects it. The two render
	// through different builtins, and the mismatch is a render error rather
	// than a compile one, so it surfaces as a file that came out short.
	ClockRef sdk.Ref

	// TypeParams is the source interface's type-parameter list in declaration
	// form, which `renderTypeParams` spells as `[K comparable, V any]`. Empty
	// for a non-generic interface, where the helper renders nothing.
	TypeParams []*sdk.EmitTypeParam

	// TypeArgs is the same list in use position — `[K, V]`, or empty.
	//
	// Every generated identifier naming a type that carries parameters has to
	// carry it too, since a generic type cannot be referenced bare. That
	// includes the subject: `Store` alone is not a type, `Store[K, V]` is.
	TypeArgs string
}

// Projection is the whole of what a generator needs to know about the
// interface under test: what it is called, the methods it declares, and
// the inputs a check draws.
//
// Carried as one value because every satellite needs all three and none
// of them needs a fourth. Before it existed, each satellite took the
// harness generator's own emit value as its parameter — which reads as a
// dependency on the harness and is not one, since what those signatures
// touched was the projection the harness happens to embed. The
// distinction shows the moment the harness's shape changes: a satellite
// named on the emit value breaks, one named on this does not, and that
// is exactly what unregistered the model tier for a month.
//
// The harness generator embeds it rather than restating the fields, so
// there is one derivation and every reader sees the same one.
type Projection struct {
	Subject

	// Fixture is the derived input set every check draws its values from.
	Fixture Fixture

	// Methods is the interface's method set in declaration order, each
	// carrying the classifications the annotator stamped on it.
	Methods []Method
}

// Method is one method of the subject interface, with the naming this generator
// adds to the shared signature projection.
type Method struct {
	// Sig is the source signature in rendered form, embedded so `.Name`,
	// `.Params`, `.Returns` and `.ReturnsError` promote onto the method.
	//
	// From [golang.Sig] rather than derived here, because every Go generator
	// projects the same source the same way and four independent
	// implementations had already disagreed about it.
	*golang.Sig

	// CheckType is the identifier of this method's extension point —
	// `<Iface><Method>Check`. Every generated check for the method is a value
	// of it, and so is a consumer's, which is what lets them compose.
	CheckType string

	// ArgFields names the fixture field each of the method's non-context
	// parameters is supplied from, in order.
	//
	// The fixture's names rather than the parameters' own, because two methods
	// naming one parameter at different types get a field each. Carried on the
	// method so the extension point's call site and the generated checks read
	// one answer: they did not, and a consumer's check was handed the other
	// method's value.
	ArgFields []string

	// IntegrationOnly reports that this method reaches something outside the
	// process, so its checks run only where that something exists.
	//
	// Carried on the projection rather than asked of the mixin list in a
	// template, because the template's job is to spell the guard and not to
	// know which classification implies one.
	IntegrationOnly bool

	// Mixins names the classifications the annotator attached, and Contracts
	// the same for contract roles.
	//
	// Carried on the projection rather than read from the source node at each
	// use. A check is selected once and rendered later, and the node is not in
	// scope by then — but more to the point, two derivations of the same stamp
	// are two chances to disagree about what the run classified.
	Mixins, Contracts []string

	// mixinParams holds each attached mixin's KV arguments, keyed
	// `<mixin>.<param>`.
	//
	// Unexported with an accessor, because a template reaching a map by a
	// composed key would spell the composition itself — and the one thing
	// worth hiding here is that a sibling param arrives qualified and has to be
	// cut back down to the local name a generated call can use.
	MixinParams map[string]string

	// contractRoles holds the role this method fills in each contract it belongs
	// to, and contractPartners the role-keyed partners beside it.
	//
	// Two maps rather than one, because the axis keys its stamps two ways and
	// flattening them would need a discriminator this would have to invent.
	// Unexported with accessors for the reason mixinParams is: the composed key
	// is spelling, and a template should ask a question rather than build one.
	//
	// The third stamp a contract can carry — an opaque param — is read by
	// nothing here: every check calls what it names, and a param is by
	// definition a value with no callable in it.
	ContractRoles, ContractPartners map[string]string

	// contractParams holds the KV arguments a contract declares — the
	// conflict sentinel an if-absent write reports, and whatever follows
	// it. Apart from the two above because a param is neither a role nor
	// a callable: nothing resolves it, so it arrives as written.
	ContractParams map[string]string
}

// HasMixin reports whether the annotator attached the named classification.
func (m Method) HasMixin(name string) bool { return slices.Contains(m.Mixins, name) }

// MixinParam returns a mixin's KV argument, and whether one was written.
//
// The value verbatim. A param the mixin declares as a sibling arrives as a
// qualified name, which is right for identity and wrong for a call site — see
// [Method.ContractPartner] for the axis that resolves one.
func (m Method) MixinParam(name, param string) (string, bool) {
	v, ok := m.MixinParams[name+"."+param]
	return v, ok
}

// TakesContext reports whether the method's first parameter is a context.
//
// The gate on three of the five signature-derived checks: cancellation, an
// expired deadline and a nil context are all claims about a parameter a method
// may not take, and emitting them for one that does not would not compile.
func (m Method) TakesContext() bool {
	return len(m.Params) > 0 && golang.IsContext(m.Params[0].Source)
}

// Shape returns the detector the annotator stamped on this method, empty when
// it stamped none.
func (m Method) Shape() string {
	if m.Source == nil {
		return ""
	}
	return shape.Get(m.Source.Meta())
}

// VariadicParam returns the method's variadic parameter, or nil.
//
// Go allows at most one and only in final position, so one answer covers the
// signature. Present so the generated file can state a narrowing a reader would
// otherwise have to infer: the fixture derives one value per parameter, so a
// generated check calls a variadic method with exactly one element.
func (m Method) VariadicParam() *golang.Param {
	for i := range m.Params {
		if m.Params[i].Variadic {
			return &m.Params[i]
		}
	}
	return nil
}

// CallArgs returns the parameters a generated call passes after the context,
// which is every parameter for a method that takes none.
func (m Method) CallArgs() []golang.Param {
	if m.TakesContext() {
		return m.Params[1:]
	}
	return m.Params
}

// HasInput reports whether the method takes anything after its context.
//
// The only lever a harness has over a subject. A parameterless method can still
// fail — a closed store, a dropped connection — but not because of anything the
// suite chose, so a check whose meaning is "this input misses" cannot reach the
// failure it is about and would demand one from a correct implementation.
func (m Method) HasInput() bool { return len(m.CallArgs()) > 0 }

// ValueReturns returns the result slots that are not the error, which is what
// a zero-value check compares.
func (m Method) ValueReturns() []golang.Return {
	out := make([]golang.Return, 0, len(m.Returns))
	for _, r := range m.Returns {
		if !r.Error {
			out = append(out, r)
		}
	}
	return out
}

// Fixture is the derived input set for one interface.
type Fixture struct {
	// TypeName is the generated struct's identifier — `<Iface>Fixture`.
	TypeName string

	// CtorName is the identifier of the function returning the derived values,
	// which a consumer reads to see what they would be overriding.
	CtorName string

	Fields []FixtureField

	// groups records which parameter each field was derived from, so a check
	// can name the field its own argument landed in — which is not the
	// parameter's name wherever two types contest one.
	Groups []ParamGroup
}

// FieldFor names the fixture field a method's parameter is supplied from.
//
// Falls back to the parameter's own name, which is what the fixture calls a
// field no other method contests. Every caller asks about a parameter of a
// method the fixture was built from, so the fallback answers the same thing the
// loop would — and a `""` would compose `cfg.Fixture.` into generated source
// rather than failing where a reader could see it.
func (f Fixture) FieldFor(p golang.Param) string {
	want := DrawField(p)
	for _, g := range f.Groups {
		if DrawField(g.Param) == want && g.Param.Source.Equal(p.Source) {
			return g.Name
		}
	}
	return p.Field
}

// Field returns the field of that name, and whether one was derived.
func (f Fixture) Field(name string) (FixtureField, bool) {
	for _, x := range f.Fields {
		if x.Name == name {
			return x, true
		}
	}
	return FixtureField{}, false
}

// FixtureField is one derived input, with the second value that makes a check
// able to fail.
type FixtureField struct {
	// Name is the exported field the generated struct declares — the Pascal
	// form of the parameter's identifier.
	Name string

	// Type is the parameter's type, rendered through the backend so the file
	// registers whatever import it needs.
	Type sdk.Ref

	// Sample and Other are the two derived values, for a parameter whose type
	// yields one whole. Empty for a struct, whose value is composed from Parts.
	Sample, Other golang.Sample

	// Parts is the per-field pair for a struct parameter, in declaration order.
	//
	// A struct's value is composed rather than carried as text, because a field
	// whose own type is a struct needs its type spelled beside its braces — and
	// only the backend knows how to spell it for this file, and to register the
	// import it needs. Text alone renders `{F: "x"}`, which is not a value.
	Parts []FixturePart

	// Variadic reports that the parameter this field was derived from was
	// declared `...T`, so the field holds one element rather than the list the
	// method takes.
	//
	// Carried only to be said out loud in the generated file. Nothing about the
	// derivation changes — [golang.Param] keeps Type as the element type, which
	// is the type of the one value a check is handed.
	Variadic bool

	// Pool is the config field this value draws from, empty where the
	// declaration carries no role.
	//
	// Matched by NAME rather than by a carried reference, because the two
	// projections read the same stamp from the same declaration: a role
	// opens `<Name>Pool` and a fixture field takes `<Name>`, so a pool
	// and the value it feeds cannot disagree about which declaration they
	// came from without disagreeing about its name.
	//
	// Set means the emitted fixture reads `cfg.<Pool>[0]` and `[1]` in
	// place of the derived literals, which is what makes a consumer's
	// override reach every check that draws.
	Pool string

	// Companion calls the type's `<Type>Defaults()`, and wins over Sample where
	// the source declares one.
	//
	// Only the sample half. Other is "a value that should not be found", which
	// is what a miss check needs and is a different claim from "a value this
	// type accepts" — one function cannot answer both, and asking for a second
	// convention to supply the alternate would cost more than the miss check
	// gains.
	Companion *sdk.Expr
}

// Composed reports whether this field's value is built from Parts rather than
// carried whole.
func (f FixtureField) Composed() bool { return len(f.Parts) > 0 }

// Choose flattens this field to one of its two values.
func (f FixtureField) Choose(alternate bool) FixtureValue {
	out := FixtureValue{Type: f.Type, Value: f.Sample}
	if alternate {
		out.Value = f.Other
		out.Alternate = 1
	}
	for _, p := range f.Parts {
		v := p.Sample
		if alternate {
			v = p.Other
		}
		out.Parts = append(out.Parts, FixtureValuePart{Name: p.Name, Value: v, Pool: p.Pool})
	}
	return out
}

// OtherName is the identifier of the companion field.
func (f FixtureField) OtherName() string { return f.Name + OtherSuffix }

// OK reports whether both of this field's values could be produced: the
// sample — a companion or a derived literal — and, separately, the alternate.
//
// Separately, because the companion answers only the sample half: "a value
// that should not be found" is a different claim from "a value this type
// accepts", and a companion accepted as proof of both let the alternate
// render as a silent zero — which real data collides with, and which turns a
// miss check into a comparison against nothing in particular.
//
// A parameter whose type admits no literal and declares no companion — a
// channel, a func, a type from a package the run never read — yields neither,
// and the one check whose meaning is the value is dropped rather than emitted
// against something nobody could write.
func (f FixtureField) OK() bool {
	if f.Composed() {
		return true
	}
	return (f.Companion != nil || f.Sample.OK()) && f.Other.OK()
}

// Reason phrases why nothing could be derived for this field.
//
// Only [golang.RefusedNoLiteral] is a fact about the type. The rest describe
// this run's own input — a package the patterns did not reach, a walk that hit
// its budget — and reporting one of those as settled sends an author to change
// source that is already correct.
func (f FixtureField) Reason() string {
	if f.Sample.Refusal.Incomplete() {
		return "which this run did not resolve, so no value was derived for it"
	}
	return "which no literal can be written for"
}

// QNameDuration is time.Duration in the annotator's stamp spelling — the
// vocabulary a pool's Q and a fixture part's Q are both written in.
//
// One home, because two tiers ask about it from opposite ends: the suite
// decides what value a duration field takes, and the model tier reads a
// law's quantity back off the one it wrote.
const QNameDuration = "time.Duration"

// FixturePart is one field of a composed struct value.
type FixturePart struct {
	// Name is the field's identifier in the composite literal.
	Name string

	// Q is the field's type in the annotator's stamp spelling, so a reader
	// can ask what a part IS without resolving it again.
	//
	// Needed because a sample does not always carry its type: one spelled
	// as an expression — `2 * time.Minute` — leaves Ref empty by contract,
	// and the type is exactly what a law binding a quantity has to match on.
	Q string

	// Sample and Other are the two values for it.
	Sample, Other golang.Sample

	// Pool is the config field this part draws from, empty where the
	// declaration carries no role. See [FixtureField.Pool].
	Pool string
}

// FixtureValue is one of a field's two values, flattened for rendering.
//
// The template needs "this field's sample" and "this field's alternate" spelled
// identically, and text/template cannot pass which one it wants down to a
// sub-template. Choosing here keeps one spelling instead of two loops that
// could drift.
type FixtureValue struct {
	Type  sdk.Ref
	Value golang.Sample
	Parts []FixtureValuePart

	// Alternate is the pool index a roled part draws — 0 for the
	// canonical member, 1 for the one that funds a miss. Carried as the
	// index rather than as a bool because that is what the emitted
	// subscript spells, and converting a bool at the template would put
	// the pool's member policy in two places.
	Alternate int
}

// FixtureValuePart is one field of a composed [FixtureValue].
type FixtureValuePart struct {
	Name  string
	Value golang.Sample

	// Pool is the config field this part draws from, empty for an
	// unroled field of a composed value.
	Pool string
}

// OtherSuffix names the companion field holding a second, different value for
// the same parameter.
//
// Two values rather than one, for every parameter: a check comparing a result
// against a single input passes whenever the subject happened to be seeded with
// it, and a miss check whose key happens to hit asserts nothing and reports
// success. The pair is what makes both able to fail.
const OtherSuffix = "Other"

// DrawWord is the word a parameter is drawn under: its declared type's
// own name where it has one, and the parameter's name otherwise.
//
// Here rather than beside the harness's other naming, because the field
// a parameter lands in is a fact about the SOURCE and every tier that
// draws has to agree on it. The model tier's pools index the harness's
// fixture fields by this name; if the two computed it differently the
// sequences would draw values the checks never wrote.
func DrawWord(p golang.Param) string {
	if p.Source != nil && p.Source.Name != "" && !golang.IsPredeclared(p.Source.Name) {
		return p.Source.Name
	}
	return p.Name
}

// DrawField is [DrawWord] as the fixture's exported field.
func DrawField(p golang.Param) string { return golang.ExportedName(DrawWord(p)) }

// ParamGroup is one fixture field: a parameter name at one type, and the first
// method that introduced it.
type ParamGroup struct {
	// name is the field's identifier, which is the parameter's own where no
	// other method takes that name at a different type.
	Name string

	// method is the first method in method-set order to take this pair, which
	// is what disambiguates a name two types share.
	Method string

	Param golang.Param
}

// GroupParams collects the interface's parameters into one field per name and
// type, in method-set order.
//
// # Why the pair rather than the name
//
// A `key string` on the reader and one on the deleter are the same value as far
// as a conformance run is concerned, and giving them separate fields would let a
// consumer override one and silently not the other.
//
// But a name is not a type. `Put(ctx, s Session)` beside `Get(ctx, s string)` is
// ordinary Go — nothing stops two methods naming their parameters alike — and a
// fixture keyed on the name alone holds one of them and hands it to the method
// that takes the other, which does not compile. An earlier version diagnosed
// that and told the author to rename a parameter, which is bad advice about
// correct source.
//
// # How a shared name is disambiguated
//
// By the method that introduced each type, not by the type itself: a composite
// has no name to spell, and `SSlice` would be a spelling this package invented.
// `PutS` and `GetS` name something the reader can find in the source. Only a
// contested name is qualified; a name carrying one type keeps it.
//
// The qualified spelling can in principle meet an uncontested parameter
// literally named `PutS`. Nothing here detects that, and nothing needs to: two
// fields of one name is a struct the toolchain refuses, so the cost is a
// compile error over generated source rather than a check quietly handed the
// wrong value.
func GroupParams(methods []Method) []ParamGroup {
	var groups []ParamGroup
	byField := map[string]int{}
	for _, m := range methods {
		for _, p := range m.CallArgs() {
			if FindGroup(groups, p) {
				continue
			}
			groups = append(groups, ParamGroup{
				Name: DrawField(p), Method: m.Name, Param: p,
			})
			byField[DrawField(p)]++
		}
	}
	// Qualify every group whose parameter name another type also claims, so
	// neither spelling is privileged by the order the walk happened to take.
	for i := range groups {
		if byField[DrawField(groups[i].Param)] > 1 {
			groups[i].Name = groups[i].Method + DrawField(groups[i].Param)
		}
	}
	return groups
}

// FindGroup reports whether a group already holds this parameter's name and
// type.
func FindGroup(groups []ParamGroup, p golang.Param) bool {
	want := DrawField(p)
	for _, g := range groups {
		if DrawField(g.Param) == want && g.Param.Source.Equal(p.Source) {
			return true
		}
	}
	return false
}

// HasContractRole reports whether the annotator stamped this method as filling
// the named role of the named contract.
//
// Both halves, because a contract is a protocol rather than a property: an
// `outbox` check written against the subscriber would call the wrong method,
// and the role is the only thing that distinguishes the two members.
func (m Method) HasContractRole(contract, role string) bool {
	return m.ContractRoles[contract] == role
}

// ContractPartner returns the local identifier a contract's role-keyed partner
// names, empty where the directive named none.
//
// Local, for the reason [Method.MixinParam] gives: the resolver rewrites a
// partner into a qualified name so it is unambiguous across packages, and a
// generated call is on a subject the check already holds.
func (m Method) ContractPartner(contract, role string) string {
	return golang.LocalName(m.ContractPartners[contract+"."+role])
}

// ContractParam returns a contract's KV argument, and whether one was written.
//
// Verbatim, unlike [Method.ContractPartner]: the resolver does not rewrite a
// param, because a param names a value rather than a callable and there is
// nothing to resolve it against. One that happens to name a package-level
// sentinel is qualified by whoever emits it — the same treatment
// [Method.MixinParam] gets.
func (m Method) ContractParam(contract, param string) (string, bool) {
	v, ok := m.ContractParams[contract+"."+param]
	return v, ok
}

// Classifications is the method's whole stamp set in tiers' one
// namespace — detector shape, mixins, contract memberships. The one
// home of the composition: the model generator selects from exactly
// this, so the two tiers cannot disagree about what the run
// classified.
func (m Method) Classifications() []string {
	var out []string
	if s := m.Shape(); s != "" {
		out = append(out, s)
	}
	out = append(out, m.Mixins...)
	return append(out, m.Contracts...)
}
