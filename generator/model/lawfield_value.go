// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"strconv"
	"strings"
	"time"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// valueOpField fills a single-value mutation closure: the role's one value
// input, or a composite write anchored on the fixture key.
func valueOpField(
	b *Bindings,
	f tiers.Field,
	field *LawField,
	role *subject.Method,
) (*LawField, string) {
	args := role.CallArgs()
	switch {
	case len(args) == 1:
		field.In = args[0].Type
		if !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name + ", which answers more than an error"
		}
		return field, ""
	case len(args) == 2 && b.UsesKeys() && b.Keys.Field != "":
		// A composite write anchored on the fixture key: the law repeats one
		// (key, value) pair, which is its claim restricted to a key every
		// other draw revisits.
		q, _ := b.keyQOf(role)
		if q != "" && b.Keys.Q != "" && q != b.Keys.Q {
			return nil, f.Name + " closes over " + role.Name +
				", which keys on " + q + " beside a pool of " + b.Keys.Q
		}
		if !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name + ", which answers more than an error"
		}
		field.In = args[1].Type
		field.KeyField = b.Keys.Field
		field.KindName = sdk.Kind(LawFieldKindPrefix + "WriteFixedKey")
		b.LawsUseFixture = true
		return field, ""
	default:
		return nil, f.Name + " closes over " + role.Name +
			", which takes several inputs no single-value closure composes"
	}
}

// constFieldOf fills a stamped constant: a qualified sentinel, a numeric
// literal, or the workflow's transition list.
func constFieldOf(
	b *Bindings, harness *subject.Projection,
	r tiers.Rule, f tiers.Field, field *LawField, m *subject.Method,
) (*LawField, string) {
	value, ok := stampValue(harness, m, f.From)
	if !ok {
		if f.Optional {
			return nil, ""
		}
		return nil, f.Name + " reads the " + f.From + " stamp, which this declaration does not carry"
	}

	if r.Law == lawid.ValidTransition && f.Name == "Allowed" {
		pairs, why := transitionPairs(value)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Pairs = pairs
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Transitions")
		return field, ""
	}

	if r.Law == lawid.PublisherAtLeastOnce || r.Law == lawid.PublisherAtMostOnce ||
		r.Law == lawid.PublisherExactlyOnce {
		// The mode spelling is the engine's own enum, not a symbol the
		// source declares — the directive says which claim, the law package
		// says what it is called.
		mode, spelled := deliveryModes[value]
		if !spelled {
			return nil, f.Name + "'s stamp names " + value + ", which is not a delivery mode"
		}
		field.Const = sdk.NewExternal(LawPkg, mode)
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Sentinel")
		return field, ""
	}

	if pkg, name, qualified := splitQualified(value); qualified {
		field.Const = sdk.NewExternal(pkg, name)
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Sentinel")
		return field, ""
	}
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		// The declared bound reaches three places — the harness
		// constructors take it, this law enforces it, and a consumer's
		// own policy check needs the same number — so it is emitted once
		// as a constant and named here. See [projection.LimitConst],
		// whose docblock says exactly this and which the law site was
		// not honouring: the file declared mixedCapacity = 5 and then
		// wrote Max: 5 beside it.
		field.Lit = value
		if f.From == tiers.ParamBoundedLimit {
			field.Lit = projection.LimitConst(projection.Token(b.IfaceName))
		}
		field.KindName = sdk.Kind(LawFieldKindPrefix + "ConstLit")
		return field, ""
	}
	if count, unit, ok := durationParts(value); ok {
		// The count and the unit the stamp spelled, not the nanoseconds
		// they multiply out to. Both compile to the same value and only
		// one of them can be read against the declaration: `timeout=100ms`
		// and `Timeout: 100000000` agree, and nothing shows that they do.
		field.Lit = count
		field.Const = sdk.NewExternal("time", unit)
		field.KindName = sdk.Kind(LawFieldKindPrefix + "ConstDur")
		return field, ""
	}
	if d, err := time.ParseDuration(value); err == nil {
		// A compound like "1h30m" has no single symbol to multiply, so it
		// keeps the nanosecond spelling: exact, if not readable against
		// the stamp.
		field.Lit = strconv.FormatInt(d.Nanoseconds(), 10)
		field.KindName = sdk.Kind(LawFieldKindPrefix + "ConstLit")
		return field, ""
	}
	return nil, f.Name + "'s stamp names " + value +
		", which is neither a qualified symbol nor a number"
}

// durationUnits are the stamp suffixes this spells back as a symbol,
// longest first so "ms" is not read as "s".
//
//nolint:gochecknoglobals // a vocabulary table, read-only after init.
var durationUnits = []struct{ suffix, name string }{
	{"ns", "Nanosecond"},
	{"us", "Microsecond"},
	{"ms", "Millisecond"},
	{"s", "Second"},
	{"m", "Minute"},
	{"h", "Hour"},
}

// durationParts splits a stamped duration into the count and the time
// package's name for its unit.
//
// Only the single-unit form, which is what a stamp writes: "100ms",
// "1h". A compound like "1h30m" has no one symbol to multiply, and
// falls back to the nanosecond literal rather than being spelled wrong.
func durationParts(value string) (count, unit string, ok bool) {
	if _, err := time.ParseDuration(value); err != nil {
		return "", "", false
	}
	for _, u := range durationUnits {
		digits, found := strings.CutSuffix(value, u.suffix)
		if !found || digits == "" {
			continue
		}
		if _, err := strconv.ParseInt(digits, 10, 64); err != nil {
			return "", "", false
		}
		return digits, u.name, true
	}
	return "", "", false
}

// generatorFieldOf fills a pool field: the run's shared pools, or a
// law-declared one for a domain the sequences never draw.
func generatorFieldOf(
	b *Bindings, harness *subject.Projection, r tiers.Rule, f tiers.Field,
	field *LawField, m, keyed *subject.Method,
) (*LawField, string) {
	switch f.From {
	case poolKeys:
		if !b.UsesKeys() {
			return nil, f.Name + " draws from the " + f.From + " pool, which no action here declares"
		}
		field.Pool = f.From
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Keys")
		return field, ""
	case poolValues:
		if !b.UsesValues() {
			return nil, f.Name + " draws from the " + f.From + " pool, which no action here declares"
		}
		field.Pool = f.From
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Values")
		return field, ""
	case poolInputs:
		elem, q, why := lawInputElem(b, harness, r, m, keyed)
		if why != "" {
			return nil, f.Name + " " + why
		}
		if reason := b.addLawPool(LawPool{Name: poolInputs, Q: q, Elem: elem}); reason != "" {
			return nil, f.Name + " " + reason
		}
		field.Pool = poolInputs
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Pool")
		return field, ""
	case poolPayloads:
		if reason := b.addLawPool(LawPool{
			Name: poolPayloads, Q: builtinString, Elem: sdk.Builtin(builtinString), Adversarial: true,
		}); reason != "" {
			return nil, f.Name + " " + reason
		}
		field.Pool = poolPayloads
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Pool")
		return field, ""
	case "messages":
		// The publish role is the writer feeding the values pool, so the
		// messages a law publishes are the values the sequences publish —
		// one pool, colliding by construction.
		if !b.UsesValues() {
			return nil, f.Name + " draws from the values pool, which no action here declares"
		}
		field.Pool = poolValues
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Values")
		return field, ""

	case poolReadback:
		// The observed reader's answer is the domain: the law's writes travel
		// through a door, so no role input names what the store holds and
		// only the read-back says.
		role, reason := ruleFieldRole(b, harness, r, fRead, m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		out, ret, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		if reason := b.addLawPool(LawPool{
			Name: poolReadback, Q: shape.QName(ret.Source), Elem: out,
		}); reason != "" {
			return nil, f.Name + " " + reason
		}
		field.Pool = poolReadback
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Pool")
		return field, ""

	case poolOffsets:
		// Bounded durations rather than arbitrary ones: an offset past the
		// advance horizon never fires inside the law's own window, and a
		// negative one is a schedule for the past nothing promises.
		elem, err := golang.RefForQualified("time.Duration", b.IfaceName)
		if err != nil {
			return nil, f.Name + " spells time.Duration, which no ref composes: " + err.Error()
		}
		if reason := b.addLawPool(LawPool{
			Name: poolOffsets, Q: "time.Duration", Elem: elem, Offsets: true,
		}); reason != "" {
			return nil, f.Name + " " + reason
		}
		field.Pool = poolOffsets
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Pool")
		return field, ""
	}
	return nil, f.Name + " draws from the " + f.From + " pool, which this build does not compose"
}

// lawInputElem is the element type of a law's wide input pool: the first
// role field's own input, which is the domain the stateless claim ranges
// over.
func lawInputElem(
	b *Bindings, harness *subject.Projection, r tiers.Rule, m, keyed *subject.Method,
) (sdk.Ref, string, string) {
	for _, f := range r.Fields {
		if f.Kind != tiers.KindRole {
			continue
		}
		role, reason := roleMethod(b, harness, f.From, m, keyed)
		if reason != "" || len(role.CallArgs()) == 0 {
			continue
		}
		arg := role.CallArgs()[0]
		return arg.Type, shape.QName(arg.Source), ""
	}
	return nil, "", "draws a domain no role here states"
}

// deliveryModes maps the directive's mode spellings to the engine enum.
//
//nolint:gochecknoglobals // a vocabulary table, read-only after init.
var deliveryModes = map[string]string{
	"at-least-once": "DeliveryAtLeastOnce",
	"at-most-once":  "DeliveryAtMostOnce",
	"exactly-once":  "DeliveryExactlyOnce",
}

// addLawPool registers a law-declared pool, refusing a second element type
// under one name.
func (b *Bindings) addLawPool(p LawPool) string {
	for _, held := range b.LawPools {
		if held.Name == p.Name {
			if held.Q == p.Q {
				return ""
			}
			return "draws " + p.Q + " from the " + p.Name + " pool, which already draws " + held.Q
		}
	}
	b.LawPools = append(b.LawPools, p)
	return ""
}
