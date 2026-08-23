// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// lawFieldOf fills one manifest entry: a field, nil for one the law defaults,
// or the reason nothing can fill it.
func lawFieldOf(
	b *Bindings, harness *subject.Projection, r tiers.Rule, f tiers.Field, m, keyed *subject.Method,
) (*LawField, string) {
	field := &LawField{
		BaseEmit: b.BaseEmit,
		Name:     f.Name,
		Iface:    b.IfaceRef,
		Key:      b.Keys.Type,
		Value:    b.Values.Type,
	}

	switch f.Kind {
	case tiers.KindDefault:
		// The law's Check defaults it; a generated value would be a second
		// opinion about a number the law already owns.
		return nil, ""
	case tiers.KindMethodName:
		// The method a sibling role field calls, as a literal. Resolved
		// through [roleMethod] on the same From the sibling carries, so
		// the two cannot end up naming different methods.
		role, reason := roleMethod(b, harness, f.From, m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		field.Method = role.Name
		field.KindName = sdk.Kind(LawFieldKindPrefix + "MethodName")
		return field, ""

	case tiers.KindTrace:
		// The runner binds the trace on any law implementing TraceBinder;
		// a generated value would race the binding it already gets.
		return nil, ""
	case tiers.KindSupplied:
		if f.From == optDrain {
			return drainFieldOf(b, harness, f, field, m, keyed)
		}
		if f.From == memberNext || f.From == memberStop {
			// A member param resolves against the handle the watch role
			// answers — the scope the resolver gained — so the closure is
			// derived from the stamp rather than opened as a door.
			return memberFieldOf(b, harness, r, f, field, m, keyed)
		}
		if f.From == "disturb" {
			// The disturbance is derived from the driven writer where one
			// feeds the pool: two adjacent reads with nothing between them
			// check nothing, and the writer is what there is to interleave.
			// Underivable falls back to the optional omission.
			return disturbFieldOf(b, harness, field, m, keyed)
		}
		if f.Optional {
			// The manifest says zero is sound: the law reads the field's
			// absence as the claim's unrefined form, so the binding omits it
			// and the option that would fill it stays a consumer's choice.
			return nil, ""
		}
		return suppliedFieldOf(b, harness, r, f, field, m, keyed)
	case tiers.KindRole:
		got, reason := roleFieldOf(b, harness, r, f, field, m, keyed)
		if reason != "" && f.Optional {
			// The manifest says absence is the claim's unrefined form — a
			// redeliver nothing declares skips the redelivery arm, never
			// the law.
			return nil, ""
		}
		return got, reason
	case tiers.KindConstant:
		return constFieldOf(b, harness, r, f, field, m)
	case tiers.KindGenerator:
		return generatorFieldOf(b, harness, r, f, field, m, keyed)
	case tiers.KindHandle:
		return handleFieldOf(b, harness, r, f, field, m, keyed)
	}
	return nil, f.Name + " has the unknown kind " + string(f.Kind)
}

// observation is the composed whole-state read the before/after laws share.
type observation struct {
	Method   *subject.Method
	Out      sdk.Ref
	Keyed    bool
	TakesCtx bool
}

// observationOf derives the strongest whole-state observation the interface
// offers: the drained collection, the aggregate, or a read of the fixture
// key — in that order, because each earlier one sees strictly more.
func observationOf(
	b *Bindings,
	harness *subject.Projection,
	keyed *subject.Method,
) (*observation, string) {
	var agg, keyedReader *subject.Method
	for i := range harness.Methods {
		m := &harness.Methods[i]
		switch pseudoShape(m) {
		case tiers.ShapeCollector:
			elem, why := collectorElem(b, m)
			if why != "" {
				continue
			}
			return &observation{Method: m, Out: sdk.SliceOf(elem), TakesCtx: m.TakesContext()}, ""
		case shapeAggregator:
			if agg == nil && len(m.CallArgs()) == 0 {
				if _, _, why := resultType(m); why == "" {
					agg = m
				}
			}
		case shapeReader:
			if keyedReader == nil {
				keyedReader = m
			}
		}
	}
	if agg != nil {
		out, _, _ := resultType(agg)
		return &observation{Method: agg, Out: out, TakesCtx: agg.TakesContext()}, ""
	}
	if keyedReader != nil && b.UsesKeys() && b.Keys.Field != "" {
		out, _, why := resultType(keyedReader)
		if why == "" {
			return &observation{
				Method: keyedReader, Out: out, Keyed: true, TakesCtx: keyedReader.TakesContext(),
			}, ""
		}
	}
	if keyed != nil && b.UsesKeys() && b.Keys.Field != "" {
		out, _, why := resultType(keyed)
		if why == "" {
			return &observation{
				Method:   keyed,
				Out:      out,
				Keyed:    true,
				TakesCtx: keyed.TakesContext(),
			}, ""
		}
	}
	return nil, "observes state through no method here — no drain, no aggregate, no keyed read"
}

// resolveArg lifts one binding-row argument into a renderable type.
func resolveArg(
	b *Bindings, harness *subject.Projection, r tiers.Rule, a tiers.BindArg, m, keyed *subject.Method,
) (sdk.Ref, string) {
	switch a {
	case tiers.BindKey:
		if b.Keys.Type == nil {
			// The commonest decline in the corpus, and every instance of it
			// measured is correct rather than a derivation nobody wrote. A
			// law instantiating at a key observes ONE value; an interface
			// drawing no keys offers a count, a drain, or nothing, and each
			// answers for the whole subject instead. Where a drain could
			// stand in, either a claim has already defeated the store model
			// — a read that may lag, a reader that clamps — or the drain is
			// compared after every call already, which states it better.
			return nil, "instantiates at a key type no method here draws"
		}
		return b.Keys.Type, ""
	case tiers.BindValue:
		if b.Values.Type == nil {
			return nil, "instantiates at a value type no method here draws"
		}
		return b.Values.Type, ""
	case tiers.BindObservation:
		obs, reason := observationOf(b, harness, keyed)
		if reason != "" {
			return nil, reason
		}
		return obs.Out, ""
	case tiers.BindPartition:
		// The single anonymous partition, until a partition projection is
		// declared and stamped.
		return sdk.Builtin(builtinString), ""
	}

	form, fieldName, qualified := a.Qualifier()
	if !qualified {
		return nil, "instantiates through " + string(a) + ", which nothing resolves"
	}
	role, reason := ruleFieldRole(b, harness, r, fieldName, m, keyed)
	if reason != "" {
		return nil, reason
	}
	switch form {
	case "result":
		if lawRoleShapes[r.Law][fieldName] == shapeNextOp {
			// NextOp's closure carries the multi-valued return whole, so
			// the law instantiates at the first non-error result — the
			// element a cursor's Next yields beside its ok flag.
			return firstResultType(role)
		}
		if pseudoShape(role) == shapeReaderWithBool {
			// Same reading for the same reason: the flag beside the value
			// is how this read spells absence, not a second thing the law
			// observes. Its closure folds the flag away entirely — see
			// [foldedReadField] — so the law instantiates at the value the
			// fold hands back.
			return firstResultType(role)
		}
		ref, _, why := resultType(role)
		return ref, why
	case "input":
		if len(role.CallArgs()) == 0 {
			return nil, "instantiates at " + role.Name + "'s input, and it takes none"
		}
		return role.CallArgs()[0].Type, ""
	case "elem":
		return drainedElem(b, role)
	case "scalar":
		ref, _, why := scalarType(role)
		return ref, why
	}
	return nil, "instantiates through " + string(a) + ", which nothing resolves"
}

// ruleFieldRole resolves a binding argument's field reference to the method
// that fills it — the same resolution the field itself gets.
func ruleFieldRole(
	b *Bindings, harness *subject.Projection, r tiers.Rule, fieldName string, m, keyed *subject.Method,
) (*subject.Method, string) {
	for _, f := range r.Fields {
		if f.Name != fieldName {
			continue
		}
		if f.Kind != tiers.KindRole {
			return nil, "instantiates through " + fieldName + ", which is not a role field"
		}
		role, reason := roleMethod(b, harness, f.From, m, keyed)
		if reason != "" {
			return nil, fieldName + " " + reason
		}
		return role, ""
	}
	return nil, "instantiates through " + fieldName + ", which the manifest does not name"
}
