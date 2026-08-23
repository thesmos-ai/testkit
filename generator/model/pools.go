// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// poolsOf fills the shared pools from the fixture fields the harness already
// derived, then decides how far past them the value draws reach.
//
// The composite writer supplies the value pool where one exists: a plain
// writer beside it is usually a delete or a touch, whose one argument is a
// key, and a values pool drawn from that would feed keys to every value slot.
//
// The keys stay the fixture pair: collision density is what makes a read
// revisit a write and an overwrite land on held state, and a wide key pool
// would pass every comparison over a history that never collides. The values
// widen wherever the claims license it, and — where the value carries its own
// key — pin that field to the keys pool, so the same key is rewritten under
// different bodies: the overwrite two fixed fixture values never draw.
func poolsOf(
	ctx *sdk.GeneratorContext,
	b *Bindings,
	harness *subject.Projection,
	keyed, valued, composite *subject.Method,
	genFunc string,
) {
	switch {
	case keyed != nil:
		arg := keyed.CallArgs()[0]
		keyQ, _ := b.keyQOf(keyed)
		b.Keys = Pool{
			Field:      keyed.ArgFields[0],
			OtherField: keyed.ArgFields[0] + subject.OtherSuffix,
			Type:       arg.Type,
			Q:          keyQ,
		}
	case composite != nil:
		// A keyed writer with no reader still draws keys; its own first
		// argument's fixture pair is the pool.
		arg := composite.CallArgs()[0]
		keyQ, _ := b.keyQOf(composite)
		b.Keys = Pool{
			Field:      composite.ArgFields[0],
			OtherField: composite.ArgFields[0] + subject.OtherSuffix,
			Type:       arg.Type,
			Q:          keyQ,
		}
	}
	switch {
	case composite != nil:
		arg := composite.CallArgs()[1]
		valueQ, _ := b.valueQOf(composite)
		b.Values = Pool{
			Field:      composite.ArgFields[1],
			OtherField: composite.ArgFields[1] + subject.OtherSuffix,
			Type:       arg.Type,
			Q:          valueQ,
		}
	case valued != nil:
		arg := valued.CallArgs()[0]
		valueQ, _ := b.valueQOf(valued)
		b.Values = Pool{
			Field:      valued.ArgFields[0],
			OtherField: valued.ArgFields[0] + subject.OtherSuffix,
			Type:       arg.Type,
			Q:          valueQ,
		}
	default:
		return
	}

	if restricted := widenValues(ctx, b, harness, genFunc); restricted {
		// A restricting claim holds the pool to values the harness has
		// proven accepted — recombining a proven body with another key is
		// already a value nothing proved.
		return
	}
	pinValues(ctx, b, keyed, valued, composite)
}

// poolFields records which shared pools draw from a config field, so the
// tier samples the whole pool rather than the fixture's pair.
//
// The pair is two values a fixed sequence needs; the pool is what a drawn
// one does, and the difference is the hostile member the transforms add to
// a derived pool. It was derived for every roled declaration and read by
// nothing at all until this.
func poolFields(ctx *sdk.GeneratorContext, b *Bindings, pools []projection.PoolPlan) {
	for _, p := range pools {
		// Matched on the ROLE the declaration stamped, not on the element
		// type: the role is the author saying what a value is FOR, and two
		// roles at one type — a key and a payload both spelled string — are
		// two pools whichever way their members render.
		switch {
		case p.Role == projection.RolePayload && b.Values.Type != nil:
			b.Values.PoolField = p.Field
		case p.Role == projection.RoleKey && b.Keys.Type != nil:
			b.Keys.PoolField = p.Field
		}
	}
	// The hostile arm asks about the TYPE, not about the config. A
	// declaration that stamped no role still draws strings, and the
	// adversarial half of the string space is exactly as relevant to it —
	// it simply has no pool for a consumer to narrow, so there is nothing
	// to gate the widening on. Reading the two questions as one is what
	// kept the corpus's most valuable code path down to a single witness.
	b.Values.Hostile = b.Values.Type != nil && stringUnder(ctx, b.Values.Q)
	b.Keys.Hostile = b.Keys.Type != nil && stringUnder(ctx, b.Keys.Q)
}

// stringUnder reports whether the named type is a string under its own
// name — the one shape a hostile string converts into without a
// constructor this generator would have to invent.
func stringUnder(ctx *sdk.GeneratorContext, typeQ string) bool {
	// A bare string is the identity case, and refusing it was an
	// accident of looking only at named types: the conversion this
	// guards against inventing is the one from a hostile string INTO
	// the drawn type, and for string that conversion is nothing at all.
	if typeQ == builtinString {
		return true
	}
	for cand := range ctx.Reader.Aliases().All() {
		if cand.Package+"."+cand.Name != typeQ {
			continue
		}
		// The target rather than the kind: the kind answers "basic" for
		// every predeclared scalar, and a hostile string converts into a
		// string and nothing else.
		return cand.Target != nil && shape.QName(cand.Target) == builtinString
	}
	return false
}

// Pool is one shared value source: a fixture field and its companion, and how
// far past them the draws reach.
type Pool struct {
	// Field and OtherField name the fixture fields the pool samples — dotted
	// where the key rides inside a fixture value rather than beside it.
	Field, OtherField string

	// Type is the drawn type, for the slice literal's element clause. Q is
	// the same type in the annotator's stamp spelling, which is what a law
	// role's stamps are compared against.
	Type sdk.Ref
	Q    string

	// GenFunc names the consumer's generator constructor where the gen=
	// directive key supplied one; the wide arm draws through it instead of
	// reflection.
	GenFunc string

	// Wide reports that the pool blends the fixture pair with arbitrary
	// [model.Make] draws; WhyNarrow is the header's reason where it cannot.
	Wide      bool
	WhyNarrow string

	// Hostile marks a pool the adversarial half of the string space can be
	// blended into: the element is a string under its own name, so a
	// hostile string converts to it without a constructor anyone has to
	// guess.
	Hostile bool

	// PoolField is the config field this pool draws from where the role
	// declared one, empty otherwise.
	//
	// Drawing the FIELD rather than the fixture's pair is what puts the
	// hostile member in reach: the transforms add one to a pool derived
	// from the type, and a pool the run passed carries exactly what it
	// passed. The provenance rule needs no flag — pass a pool and the
	// hostile member is not in it.
	PoolField string

	// Pin is the value field overwritten with a keys-pool draw, so every
	// drawn value lands on a key the reads revisit — empty where the key is
	// an argument beside the value, or where a restricting claim holds the
	// pool to values the harness has proven accepted.
	Pin string
}

// Identity reports that the hostile transform is a conversion from string
// to string, and so no conversion at all.
//
// The blend lifts a drawn hostile string into the pool's element type, and
// where that type IS string the lift is `string(s)` — legal, a no-op, and
// a line every reader of the generated file stops on to check they have
// not missed something. Thirty-six of them across the corpus said nothing.
func (p Pool) Identity() bool { return p.Q == builtinString }

// Read and OtherRead spell the two draws as calls on the harness's
// fixture: `fx.Key()`, or `fx.Value().Key` where the key rides inside a
// fixture value rather than beside it.
//
// The harness exposes its inputs as accessors and holds the values
// unexported, which is what lets a supplied value and a derived one read
// the same. A dotted field is a path THROUGH one of those accessors, so
// only its head is a call — spelling the whole path as one is how
// `fx.Value.Key()` reached a generated file.
func (p Pool) Read() string { return fixtureRead(p.Field) }

// OtherRead is [Pool.Read] for the second, different value of the pair.
func (p Pool) OtherRead() string { return fixtureRead(p.OtherField) }

func fixtureRead(field string) string {
	if field == "" {
		return ""
	}
	head, rest, dotted := strings.Cut(field, ".")
	if !dotted {
		return "fx." + head + "()"
	}
	return "fx." + head + "()." + rest
}

// pinValues pins the value's key field to the keys pool where the value
// carries its own key, deriving the pool itself where no reader supplies one.
//
// A supplied or twin reference skipped the key-field derivation, so the pin
// retries it best-effort: found, the wide pool lands on pooled keys the way a
// derived one does. Not found, a supplied reference narrows — it models a
// store, and wide values keyed afresh would never collide with a key the
// reads revisit — where the twin stays wide: its comparisons are twin against
// twin, and two twins agree about a miss as readily as a hit.
func pinValues(ctx *sdk.GeneratorContext, b *Bindings, keyed, valued, composite *subject.Method) {
	pin := b.Reference.KeyField
	if pin == "" && !b.Reference.Derived() && keyed != nil && valued != nil && composite == nil {
		keyQ, _ := b.keyQOf(keyed)
		readV, _ := b.valueQOf(keyed)
		wroteV, _ := b.valueQOf(valued)
		if readV != "" && readV == wroteV {
			pin, _ = keyFieldOf(ctx, readV, keyQ)
		}
		if pin == "" && b.Reference.Supplied() {
			b.Values.Wide = false
			b.Values.WhyNarrow = "no key field is derivable to pin a wide draw onto the pooled keys"
			return
		}
	}
	b.Values.Pin = pin
	if pin != "" && keyed == nil {
		// The drain path has no reader to supply keys; the fixture values'
		// own key fields are the colliding set.
		b.Keys.Field = b.Values.Field + "." + pin
		b.Keys.OtherField = b.Values.OtherField + "." + pin
	}
}

// widenValues decides how far past the fixture pair the values pool reaches,
// reporting whether a restricting claim made the decision.
//
// Wide is the default the contract earns: a writer with no restricting claim
// accepts any value of its type, so the pool blends the fixture pair with
// arbitrary draws and a subject refusing one has broken its own claim. A
// [tiers.ValueRestriction] inverts the license, and a type whose graph this
// build cannot see to the bottom would arm a [model.Make] panic instead of a
// deeper run; both keep the pair, and the header says which.
func widenValues(ctx *sdk.GeneratorContext, b *Bindings, harness *subject.Projection, genFunc string) bool {
	// A supplied generator outranks every verdict below: the consumer
	// authored the domain, which is more than a reflection walk or a claim
	// scan can ever know.
	if genFunc != "" {
		b.Values.GenFunc = genFunc
		b.Values.Wide = true
		return false
	}
	drawing := map[string]bool{}
	for _, a := range b.Actions {
		if a.Pool == poolValues {
			drawing[a.Method] = true
		}
	}
	var valueQ string
	for i := range harness.Methods {
		m := &harness.Methods[i]
		if !drawing[m.Name] {
			continue
		}
		for _, mixin := range m.Mixins {
			if reason, restricted := tiers.ValueRestriction(mixin); restricted {
				b.Values.WhyNarrow = "the " + mixin + " claim on " + m.Name + " " + reason
				return true
			}
		}
		if q, ok := b.valueQOf(m); ok && q != "" {
			valueQ = q
		}
	}
	if valueQ == "" {
		// A drawing method with no value stamp — a mutator's key, say —
		// still has the pool's own spelling to be checked against.
		valueQ = b.Values.Q
	}
	if why := unmakeable(ctx, valueQ, map[string]bool{}); why != "" {
		b.Values.WhyNarrow = why
		return false
	}
	b.Values.Wide = true
	return false
}

// unmakeable reports why [model.Make] cannot be trusted to draw the named
// type, empty where it can.
//
// rapid resolves the type graph at run time and panics on a kind it cannot
// draw, so the walk here is the mirror of that resolution over what this
// build can see: a scalar draws, an exported struct field recurses, an
// unexported one is skipped the way Make skips it, and anything else — a
// declaration out of reach, an interface, a spelling the frontend does not
// resolve to a struct — keeps the pool narrow instead of arming a panic.
func unmakeable(ctx *sdk.GeneratorContext, typeQ string, seen map[string]bool) string {
	if scalarKinds[typeQ] || seen[typeQ] {
		return ""
	}
	seen[typeQ] = true
	for cand := range ctx.Reader.Structs().All() {
		if cand.Package+"."+cand.Name != typeQ {
			continue
		}
		for _, f := range cand.Fields {
			if r, _ := utf8.DecodeRuneInString(f.Name); !unicode.IsUpper(r) {
				// Unexported: Make leaves it zero, which draws fine.
				continue
			}
			if why := unmakeable(ctx, shape.QName(f.Type), seen); why != "" {
				return why
			}
		}
		return ""
	}
	return typeQ + " reaches a type this build cannot prove a wide draw serves"
}

// scalarKinds is the predeclared vocabulary a wide draw serves unconditionally.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var scalarKinds = map[string]bool{
	"bool": true, builtinString: true, "byte": true, "rune": true,
	builtinInt: true, "int8": true, "int16": true, "int32": true, builtin64: true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"uintptr": true, "float32": true, "float64": true,
}

// feederOf picks the writer whose values fill the pool: the one agreeing
// with the reader — or, with no reader, with the collector's element — else
// the first in declaration order, so the choice never depends on where a
// method sits in the source.
func feederOf(b *Bindings, keyed, collector *subject.Method, writers []*subject.Method) *subject.Method {
	if len(writers) == 0 {
		return nil
	}
	want := ""
	switch {
	case keyed != nil:
		want, _ = b.valueQOf(keyed)
	case collector != nil:
		want = shape.QName(shape.GoSliceElem(collector.Returns[0].Source))
	}
	if want != "" {
		for _, w := range writers {
			if q, _ := b.valueQOf(w); q == want {
				return w
			}
		}
	}
	return writers[0]
}

// valueSourceOf names the one method whose value type the values pool takes,
// in the order [poolsOf] resolves it: a composite writer outranks a plain one
// because its second argument is the body a store holds, and where neither
// exists the first mutator's argument is all the pool has to go on.
//
// Two callers, one answer, deliberately: [poolsOf] types the pool from this
// method and the mismatch guard in [bindingsOf] refuses every other drawer
// that disagrees with it. Deriving the guard's baseline separately is what
// let a composite type the pool while the guard measured against a writer —
// admitting a drawer that mismatched and refusing one that matched, both
// silently, and both surfacing only as generated code that will not compile.
func valueSourceOf(valued, composite, valueFallback *subject.Method) *subject.Method {
	switch {
	case composite != nil:
		return composite
	case valued != nil:
		return valued
	default:
		return valueFallback
	}
}
