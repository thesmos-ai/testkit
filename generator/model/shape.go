// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"slices"
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// firstResultType is a method's first non-error result — the instantiation
// point for a law whose closure shape returns the method's results whole,
// where [resultType]'s single-valued strictness would refuse the method.
func firstResultType(m *subject.Method) (sdk.Ref, string) {
	for i := range m.Returns {
		ret := &m.Returns[i]
		if ret.Source != nil && golang.IsError(ret.Source) {
			continue
		}
		return ret.Type, ""
	}
	return nil, "observes through " + m.Name + ", which answers nothing to observe"
}

// resultType is a method's single non-error result.
func resultType(m *subject.Method) (sdk.Ref, *golang.Return, string) {
	results := make([]*golang.Return, 0, len(m.Returns))
	for i := range m.Returns {
		ret := &m.Returns[i]
		if ret.Source != nil && golang.IsError(ret.Source) {
			continue
		}
		results = append(results, ret)
	}
	if len(results) == 0 {
		return nil, nil, "observes through " + m.Name + ", which answers nothing to observe"
	}
	if len(results) > 1 {
		return nil, nil, "observes through " + m.Name +
			", which answers several results no single-valued closure returns"
	}
	return results[0].Type, results[0], ""
}

// scalarType is a method's scalar observation: its single non-error result,
// or the length of the slice it returns.
func scalarType(m *subject.Method) (ref sdk.Ref, viaLen bool, reason string) {
	if returnsSlice(m) {
		return sdk.Builtin(builtinInt), true, ""
	}
	r, _, why := resultType(m)
	return r, false, why
}

// drainedElem is the element type of the stream a method drains — a slice's
// element, or the stamped yield of an iterator.
func drainedElem(b *Bindings, m *subject.Method) (sdk.Ref, string) {
	if returnsSlice(m) {
		return collectorElem(b, m)
	}
	q, stamped := b.valueQOf(m)
	if !stamped || q == "" {
		return nil, "drains " + m.Name + ", which streams elements no stamp names"
	}
	ref, err := golang.RefForQualified(q, b.IfaceName)
	if err != nil {
		return nil, "drains " + q + ", which no closure can spell: " + err.Error()
	}
	return ref, ""
}

// errOnly reports whether the method returns exactly one error and nothing
// else.
func errOnly(m *subject.Method) bool {
	return len(m.Returns) == 1 && m.Returns[0].Source != nil &&
		golang.IsError(m.Returns[0].Source)
}

// stringParam reports whether the parameter is a bare string.
func stringParam(p golang.Param) bool {
	return p.Source != nil && golang.IsBuiltinNamed(p.Source, builtinString)
}

// integerResult reports whether the return slot is a builtin integer — the
// shape an offset or a conserved sum totals.
func integerResult(ret *golang.Return) bool {
	if ret.Source == nil {
		return false
	}
	for _, name := range builtinInts {
		if golang.IsBuiltinNamed(ret.Source, name) {
			return true
		}
	}
	return false
}

// identityCompared reports whether the method's first result is a live
// handle — a channel, a function, a pointer — that `!=` compares by identity,
// which two independently built sides never share.
func identityCompared(m *subject.Method) bool {
	if len(m.Returns) == 0 || m.Returns[0].Source == nil {
		return false
	}
	src := m.Returns[0].Source
	return golang.IsChannel(src) || src.IsFunc() || shape.GoPointerElem(src) != nil
}

// orderedScalar reports whether the method's single result is a type `<`
// orders — the builtin integers, floats and string.
func orderedScalar(m *subject.Method) bool {
	_, ret, why := resultType(m)
	if why != "" || ret.Source == nil {
		return false
	}
	for _, name := range builtinOrdered {
		if golang.IsBuiltinNamed(ret.Source, name) {
			return true
		}
	}
	return false
}

// numericScalar reports that the method answers a number — a quantity,
// rather than a payload or a handle that happens to be comparable.
func numericScalar(m *subject.Method) bool {
	_, ret, why := resultType(m)
	if why != "" || ret.Source == nil {
		return false
	}
	for _, name := range builtinNumeric {
		if golang.IsBuiltinNamed(ret.Source, name) {
			return true
		}
	}
	return false
}

// transitionPairs parses a workflow's `from>to[,from>to…]` stamp.
func transitionPairs(value string) ([][2]string, string) {
	var out [][2]string
	for part := range strings.SplitSeq(value, ",") {
		from, to, ok := strings.Cut(strings.TrimSpace(part), ">")
		if !ok || from == "" || to == "" {
			return nil, "reads " + value + ", which is not a from>to transition list"
		}
		out = append(out, [2]string{from, to})
	}
	return out, ""
}

// seqArity reports how many values a method's streamed result yields — 1 for
// an `iter.Seq`, 2 for an `iter.Seq2` — and zero where the method streams
// nothing.
//
// Named types from the standard library rather than a shape stamp, because
// this is not a classification: it is the fact that the result's zero value
// is a nil function. A defect that answers the zero for a stream hands the
// law a nil iterator, and ranging over one panics before the law is asked.
func seqArity(m *subject.Method) int {
	if len(m.Returns) != 1 || m.Returns[0].Source == nil {
		return 0
	}
	t := m.Returns[0].Source
	if t.Package != "iter" {
		return 0
	}
	switch t.Name {
	case "Seq":
		return 1
	case "Seq2":
		return 2
	}
	return 0
}

// returnsSlice reports whether the method's first result is a slice.
func returnsSlice(m *subject.Method) bool {
	return len(m.Returns) > 0 && m.Returns[0].Source != nil &&
		shape.GoSliceElem(m.Returns[0].Source) != nil
}

// stampValue reads one classification parameter, by the raw key the manifest
// spells — off the selecting method first, and for a contract parameter off
// every carrier of the same contract, because the stamp lives on the
// directive host and any role method may be the one selecting the rule.
func stampValue(harness *subject.Projection, m *subject.Method, key string) (string, bool) {
	if v, ok := sdk.EnsureKey(key, sdk.StringParser).Get(m.Source.Meta()); ok && v != "" {
		return v, true
	}
	contract, isContract := contractOfParamKey(key)
	if !isContract || harness == nil {
		return "", false
	}
	for i := range harness.Methods {
		carrier := &harness.Methods[i]
		if !slices.Contains(carrier.Contracts, contract) {
			continue
		}
		if v, ok := sdk.EnsureKey(key, sdk.StringParser).Get(carrier.Source.Meta()); ok && v != "" {
			return v, true
		}
	}
	return "", false
}

// contractOfParamKey extracts the contract a param stamp key belongs to,
// false for a mixin's.
func contractOfParamKey(key string) (string, bool) {
	rest, found := strings.CutPrefix(key, "shape.contract.")
	if !found {
		return "", false
	}
	name, _, ok := strings.Cut(rest, ".param.")
	return name, ok && name != ""
}

// splitQualified splits a resolver-qualified name into its package path and
// trailing identifier.
func splitQualified(v string) (pkg, name string, ok bool) {
	i := strings.LastIndexByte(v, '.')
	if i <= 0 || i == len(v)-1 {
		return "", "", false
	}
	return v[:i], v[i+1:], true
}

// contractParamNames returns the named contract's declared parameters.
func contractParamNames(name string) []string {
	for _, c := range contracts.All() {
		if c.Name == name {
			return paramKeys(c.Params)
		}
	}
	return nil
}

// paramKeys flattens a param schema to its keys — the spelling the stamp
// lookups compose with; the kinds are the resolver's business.
func paramKeys(params []shape.Param) []string {
	out := make([]string, len(params))
	for i, p := range params {
		out[i] = p.Key
	}
	return out
}

// answeringWriterOf finds a write that answers the stored state — one value
// in, the same type out beside the error — or nil. Structural rather than
// stamped, so a hand-built projection in a test answers the same way the
// annotated corpus does.
func answeringWriterOf(harness *subject.Projection) *subject.Method {
	if harness == nil {
		return nil
	}
	for i := range harness.Methods {
		m := &harness.Methods[i]
		args := m.CallArgs()
		if len(args) != 1 || len(m.Returns) != 2 || !m.ReturnsError() {
			continue
		}
		in, inOK := args[0].Source, args[0].Source != nil
		out, outOK := m.Returns[0].Source, m.Returns[0].Source != nil
		if inOK && outOK && shape.QName(in) != "" && shape.QName(in) == shape.QName(out) {
			return m
		}
	}
	return nil
}

// methodOf finds one projection method by name; the adapter was built from
// the same list, so a miss is unreachable.
func methodOf(harness *subject.Projection, name string) *subject.Method {
	for i := range harness.Methods {
		if harness.Methods[i].Name == name {
			return &harness.Methods[i]
		}
	}
	return nil
}

// missSentinelOf reports the declaration's own miss sentinel: the first
// sentinel= or notfound= a mixin stamps anywhere in the method set,
// qualified by the resolver. Routed into the derived oracle's constructor,
// it is what lets a sentinel-checking law's guard match the identity the
// fixture declared — against a minted private error the guard never passes,
// and the law it feeds is dead without anyone saying so.
func missSentinelOf(harness *subject.Projection) *sdk.Expr {
	for i := range harness.Methods {
		m := &harness.Methods[i]
		v, declared := subject.MissSentinel(*m)
		if !declared {
			continue
		}
		if pkg, name, qualified := splitQualified(v); qualified {
			return sdk.NewExternal(pkg, name)
		}
	}
	return nil
}

// historyDrained reports whether any classification marks the drained
// slice as an event log — the refinement that outranks every keyed
// election, because a map oracle collapses the repeats a log faithfully
// holds.
func historyDrained(harness *subject.Projection) bool {
	for i := range harness.Methods {
		m := &harness.Methods[i]
		drains := slices.ContainsFunc(m.Mixins, tiers.DrainsHistory) ||
			slices.ContainsFunc(m.Contracts, tiers.DrainsHistory)
		if drains {
			return true
		}
	}
	return false
}

// spelling renders a value type for a refusal reason, naming the absence
// where a method carries no stamp at all — "takes  where" reads as a
// rendering bug and sends the reader to the generator instead of to the
// method whose classification is missing.
func spelling(q string) string {
	if q == "" {
		return "no stamped value type"
	}
	return q
}

// partnerMethods maps each method excluded by a mixin's sibling reference to
// the `<mixin>.<param>: <reason>` that claims it.
//
// Only the references whose role overrides their shape exclude: a validator
// is writer-shaped, and a sequence that drives it as a writer corrupts the
// reference with stores the subject never made. Most references name an
// ordinary method — a put that is a writer — and those stay in the sequences;
// [tiers.PartnerDriven] is the classification, held total by the census. The
// registry is consulted for which parameters are sibling references rather
// than values — that is the annotator's vocabulary, not a spelling this
// plugin owns.
func partnerMethods(iface *sdk.Interface) map[string]string {
	out := map[string]string{}
	for _, m := range iface.Methods {
		for _, name := range shape.Mixins(m.Meta()) {
			for _, p := range siblingParams(name) {
				v, ok := shape.MixinParamKey(name, p).Get(m.Meta())
				if !ok || v == "" {
					continue
				}
				if driven, reason := tiers.PartnerDriven(name, p); !driven {
					out[golang.LocalName(v)] = "the " + name + "." + p + " partner — " + reason
				}
			}
		}
	}
	return out
}

// siblingParams returns the named mixin's sibling-reference parameters —
// the callable-kinded keys, whose values name methods of this interface.
// Member-kinded keys stay out: they name methods of a role's answered
// handle, and a sibling scan claiming them would mark an interface method
// that merely shares the name.
func siblingParams(name string) []string {
	for _, m := range mixins.All() {
		if m.Name == name {
			var out []string
			for _, p := range m.Params {
				if p.Kind == shape.KindCallable {
					out = append(out, p.Key)
				}
			}
			return out
		}
	}
	return nil
}

// directiveValue reads one key off the interface's own directive.
func directiveValue(iface *sdk.Interface, key string) (string, bool) {
	for _, dir := range iface.Directives() {
		if string(dir.Name) != string(DirectiveName) {
			continue
		}
		if v, ok := dir.KV[key]; ok && v != "" {
			return v, true
		}
	}
	return "", false
}
