// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"slices"
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// roleFieldOf fills a closure field per its law's transcribed shape.
func roleFieldOf(
	b *Bindings, harness *subject.Projection, r tiers.Rule, f tiers.Field,
	field *LawField, m, keyed *subject.Method,
) (*LawField, string) {
	role, reason := roleMethod(b, harness, f.From, m, keyed)
	if reason != "" {
		return nil, f.Name + " " + reason
	}
	shapes, known := lawRoleShapes[r.Law]
	sh, mapped := shapes[f.Name]
	if !known || !mapped {
		return nil, f.Name + " closes over " + role.Name +
			", and the generator transcribes no closure shape for it"
	}
	field.Method = role.Name
	field.TakesCtx = role.TakesContext()
	field.KindName = sdk.Kind(LawFieldKindPrefix + string(sh))

	switch sh {
	case shapeDrainSeq, shapeScalarLen:
		// Override spellings, never table entries: the slice arms below pick
		// them when the role streams or the observation is a length.
		return nil, f.Name + " names an override shape no table row spells"
	case shapeKeyedRead:
		spec, _ := tiers.BindingFor(r.Law)
		if why := keyedReadMismatch(b, f.Name, role, slices.Contains(spec.Args, tiers.BindValue)); why != "" {
			return nil, why
		}
		// The closure is typed by the role itself — the pools agree where the
		// mismatch check above demands it, and a reader whose value no pool
		// draws (a cache, a persister's load) still compiles.
		field.Key = role.CallArgs()[0].Type
		if pseudoShape(role) == shapeReaderWithBool {
			return foldedReadField(b, f, field, role)
		}
		out, _, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Value = out
		return field, ""

	case shapeValueOp:
		return valueOpField(b, f, field, role)

	case shapeDrainSlice:
		elem, why := drainedElem(b, role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Value = elem
		if !returnsSlice(role) {
			field.KindName = sdk.Kind(LawFieldKindPrefix + string(shapeDrainSeq))
		}
		return field, ""

	case shapeScalar:
		ref, viaLen, why := scalarType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		if len(role.CallArgs()) > 0 {
			return nil, f.Name + " closes over " + role.Name +
				", which takes inputs no nullary observation supplies"
		}
		field.Out = ref
		switch {
		case viaLen:
			field.KindName = sdk.Kind(LawFieldKindPrefix + string(shapeScalarLen))
		case !role.ReturnsError():
			field.KindName = sdk.Kind(LawFieldKindPrefix + "ScalarNoErr")
		}
		if r.Law == lawid.AggregatorBounded && !viaLen && !orderedScalar(role) {
			return nil, f.Name + " observes " + role.Name +
				"'s result, which no bound orders"
		}
		if r.Law == lawid.CountEqualsReference && identityCompared(role) {
			return nil, f.Name + " observes " + role.Name +
				"'s result, a live handle only identity could compare"
		}
		return field, ""

	case shapeBoolCall:
		if len(role.CallArgs()) > 0 || len(role.Returns) != 1 ||
			role.Returns[0].Source == nil || !golang.IsBuiltinNamed(role.Returns[0].Source, "bool") {

			return nil, f.Name + " closes over " + role.Name + ", which is not a bare predicate"
		}
		return field, ""

	case shapeResultCall:
		if len(role.CallArgs()) > 0 || role.ReturnsError() {
			return nil, f.Name + " closes over " + role.Name + ", which is not a bare pure call"
		}
		out, _, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Out = out
		return field, ""

	case shapeInputCall:
		if len(role.CallArgs()) != 1 {
			return nil, f.Name + " closes over " + role.Name +
				", which takes several inputs no single-value closure composes"
		}
		out, _, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.In = role.CallArgs()[0].Type
		field.Out = out
		if !role.ReturnsError() {
			if r.Law == lawid.TotalOver {
				// Totality is the one claim on this shape that reads nothing
				// but the error, so a method with no error to report cannot
				// fail it — the closure supplies the nil and the check finds
				// it. "The most total thing a method can be" is exactly the
				// problem: nothing is left to verify, and a bound check that
				// cannot fail is worth less than an absent one, because the
				// header counts it.
				return nil, f.Name + " closes over " + role.Name +
					", which reports no error, and totality is the claim that it does not — " +
					"the check would pass on every input by construction"
			}
			// The law threads an error the method has no way to report, and
			// the closure's own signature supplies the nil. Adapted rather
			// than refused for the claims that read the *result*: an
			// errorless transformation still has an answer to be wrong.
			field.KindName = sdk.Kind(LawFieldKindPrefix + "InputCallNoErr")
		}
		return field, ""

	case shapeCtxOp:
		if !role.TakesContext() || len(role.CallArgs()) > 0 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name +
				", which is not a context-respecting error operation"
		}
		return field, ""

	case shapeErrOp:
		if len(role.CallArgs()) > 0 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name +
				", which is not a nullary error operation"
		}
		return field, ""

	case shapeKeyedOp:
		if len(role.CallArgs()) != 1 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name + ", which is not a keyed error operation"
		}
		if b.Keys.Q != "" {
			if q, _ := b.keyQOf(role); q != "" && q != b.Keys.Q {
				return nil, f.Name + " closes over " + role.Name +
					", which keys on " + q + " beside a pool of " + b.Keys.Q
			}
		}
		return field, ""

	case shapeKVOp:
		args := role.CallArgs()
		if len(args) != 2 || !errOnly(role) || !stringParam(args[0]) || !stringParam(args[1]) {
			return nil, f.Name + " closes over " + role.Name +
				", which is not a string-keyed string write"
		}
		return field, ""

	case shapeSum:
		if len(role.CallArgs()) > 0 {
			return nil, f.Name + " closes over " + role.Name +
				", which takes inputs no nullary observation supplies"
		}
		_, ret, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		if ret.Source == nil || !integerResult(ret) {
			return nil, f.Name + " observes " + role.Name + "'s result, which no sum totals"
		}
		return field, ""

	case shapeMerge:
		if len(role.CallArgs()) != 1 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name + ", which does not merge one peer"
		}
		return field, ""

	case shapeSave:
		if len(role.CallArgs()) != 1 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name + ", which does not save one value"
		}
		if b.Reference.KeyField == "" {
			return nil, f.Name + " synthesizes the saved identity from the key projection, " +
				"which was not derivable here"
		}
		field.KeyOfName = b.KeyOfName()
		field.In = role.CallArgs()[0].Type
		field.Out = b.Keys.Type
		return field, ""

	case shapeAppendOff:
		if len(role.CallArgs()) != 1 {
			return nil, f.Name + " closes over " + role.Name + ", which does not append one value"
		}
		out, ret, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		if !integerResult(ret) {
			return nil, f.Name + " expects an offset, and " + role.Name + " answers none"
		}
		field.In = role.CallArgs()[0].Type
		field.Out = out
		return field, ""

	case shapeCtxKeyedOp:
		if !role.TakesContext() || len(role.CallArgs()) != 1 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name +
				", which is not a one-key context operation"
		}
		field.Key = role.CallArgs()[0].Type
		return field, ""

	case shapeSubscribe:
		if len(role.CallArgs()) > 0 {
			return nil, f.Name + " closes over " + role.Name + ", which takes inputs no subscription draw supplies"
		}
		out, _, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Out = out
		return field, ""

	case shapeOkOp:
		if len(role.CallArgs()) > 0 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name + ", which is not a nullary error operation"
		}
		return field, ""

	case shapeNextOp:
		if len(role.CallArgs()) > 0 || len(role.Returns) != 3 || !role.ReturnsError() {
			return nil, f.Name + " closes over " + role.Name +
				", which does not answer the (value, more, error) triple"
		}
		field.Out = role.Returns[0].Type
		return field, ""

	case shapeDoOp:
		if len(role.CallArgs()) > 0 {
			return nil, f.Name + " closes over " + role.Name +
				", which takes inputs no nullary corruption supplies"
		}
		return field, ""

	case shapePinnedWrite:
		if len(role.CallArgs()) != 1 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name + ", which does not put one value"
		}
		if b.Values.Pin == "" {
			return nil, f.Name + " pins the drawn key into the value, and this pool pins nothing"
		}
		field.In = role.CallArgs()[0].Type
		field.KeyField = b.Values.Pin
		return field, ""

	case shapeCtxOpFixed:
		if !role.TakesContext() || !errOnly(role) || len(role.CallArgs()) != 1 {
			return nil, f.Name + " closes over " + role.Name +
				", which is not a one-input context operation"
		}
		if len(role.ArgFields) == 0 {
			return nil, f.Name + " anchors on a fixture field the projection does not carry"
		}
		field.KeyField = role.ArgFields[0]
		b.LawsUseFixture = true
		return field, ""

	case shapeScheduleAt:
		if len(role.CallArgs()) != 1 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name + ", which does not take one offset"
		}
		return field, ""

	case shapeCountObs:
		if len(role.CallArgs()) > 0 {
			return nil, f.Name + " closes over " + role.Name +
				", which takes inputs no nullary observation supplies"
		}
		_, ret, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		if !integerResult(ret) {
			return nil, f.Name + " counts " + role.Name + "'s result, which is not a count"
		}
		return field, ""

	case shapeReplay:
		if len(role.CallArgs()) > 0 {
			return nil, f.Name + " closes over " + role.Name +
				", which takes inputs the single-partition replay does not thread"
		}
		elem, why := drainedElem(b, role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		if !returnsSlice(role) {
			return nil, f.Name + " drains " + role.Name +
				", which streams through an iterator this adapter does not compose"
		}
		field.Out = elem
		return field, ""

	case shapeHandleCall:
		if len(role.CallArgs()) > 0 {
			return nil, f.Name + " closes over " + role.Name +
				", which takes inputs no handle draw supplies"
		}
		out, _, why := resultType(role)
		if why != "" {
			return nil, f.Name + " closes over " + role.Name +
				", which answers no handle the terminal pair can thread"
		}
		field.Out = out
		return field, ""

	case shapeHandleOp:
		if len(role.CallArgs()) != 1 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name + ", which does not settle one handle"
		}
		begin, reason := ruleFieldRole(b, harness, r, fBegin, m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		_, ret, why := resultType(begin)
		if why != "" {
			return nil, f.Name + " threads " + begin.Name + "'s handle, and it answers none"
		}
		if hq := shape.QName(role.CallArgs()[0].Source); hq != shape.QName(ret.Source) {
			return nil, f.Name + " settles a " + hq +
				" where " + begin.Name + " answers " + shape.QName(ret.Source)
		}
		field.In = role.CallArgs()[0].Type
		return field, ""

	case shapeSagaRun:
		if len(role.CallArgs()) != 1 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name + ", which does not step one value"
		}
		if !b.UsesValues() {
			return nil, f.Name + " draws steps from the values pool, which no action here declares"
		}
		if q, _ := b.valueQOf(role); q != "" && b.Values.Q != "" && q != b.Values.Q {
			return nil, f.Name + " closes over " + role.Name +
				", which steps " + q + " beside a pool of " + b.Values.Q
		}
		partner, reason := roleMethod(b, harness, "saga.compensate", m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		field.In = role.CallArgs()[0].Type
		field.Pool = poolValues
		field.Partner = partner.Name
		field.PartnerCtx = partner.TakesContext()
		return field, ""

	case shapeComputeCall:
		args := role.CallArgs()
		if len(args) != 2 || args[1].Source == nil || !args[1].Source.IsFunc() {
			return nil, f.Name + " closes over " + role.Name +
				", which takes no compute to deduplicate"
		}
		out, _, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Key = args[0].Type
		field.Out = out
		return field, ""

	case shapeBodyRun:
		args := role.CallArgs()
		if len(args) != 1 || !errOnly(role) || args[0].Source == nil || !args[0].Source.IsFunc() {
			return nil, f.Name + " closes over " + role.Name + ", which accepts no failing body"
		}
		return field, ""

	case shapePageRead:
		if len(role.CallArgs()) != 1 || len(role.Returns) != 4 || !role.ReturnsError() {
			return nil, f.Name + " closes over " + role.Name +
				", which answers no page — no cursor to resume from"
		}
		if cq := shape.QName(role.CallArgs()[0].Source); cq != shape.QName(role.Returns[1].Source) {
			return nil, f.Name + " resumes " + role.Name +
				" at a " + shape.QName(role.Returns[1].Source) + ", which is not its " + cq + " cursor"
		}
		if role.Returns[2].Source == nil || !golang.IsBuiltinNamed(role.Returns[2].Source, "bool") {
			return nil, f.Name + " closes over " + role.Name + ", which never says whether more remains"
		}
		elem, why := drainedElem(b, role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.In = role.CallArgs()[0].Type
		field.Out = elem
		return field, ""

	case shapeKeyedHandle:
		if len(role.CallArgs()) != 1 {
			return nil, f.Name + " closes over " + role.Name +
				", which does not watch one key"
		}
		out, _, why := resultType(role)
		if why != "" {
			return nil, f.Name + " closes over " + role.Name +
				", which answers no handle to read through"
		}
		field.Key = role.CallArgs()[0].Type
		field.Out = out
		return field, ""

	case shapeKeyedWrite:
		if len(role.CallArgs()) != 2 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name +
				", which does not write one value under one key"
		}
		if b.Keys.Q != "" {
			if q, _ := b.keyQOf(role); q != "" && q != b.Keys.Q {
				return nil, f.Name + " closes over " + role.Name +
					", which keys on " + q + " beside a pool of " + b.Keys.Q
			}
		}
		field.Key = role.CallArgs()[0].Type
		field.Value = role.CallArgs()[1].Type
		return field, ""

	case shapeHandleWrite:
		args := role.CallArgs()
		if len(args) != 3 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name +
				", which does not write one value under one key inside an open handle"
		}
		field.In = args[0].Type
		field.Key = args[1].Type
		field.Value = args[2].Type
		return field, ""

	case shapePeerSync:
		if len(role.CallArgs()) != 1 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name +
				", which does not sync with one peer"
		}
		if peer := golang.LocalName(shape.QName(role.CallArgs()[0].Source)); peer != b.IfaceName {
			return nil, f.Name + " closes over " + role.Name +
				", which syncs with a " + peer + " where the replicas are " + b.IfaceName
		}
		return field, ""

	case shapeEachSettle:
		if len(role.CallArgs()) > 0 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name +
				", which is not a nullary settle"
		}
		return field, ""
	}
	return nil, f.Name + " has the unrendered shape " + string(sh)
}

// roleMethod resolves a manifest role to the method whose call fills it:
// the selecting method itself, a shape family, or a partner the selecting
// method's own stamp names.
func roleMethod(
	b *Bindings,
	harness *subject.Projection,
	from string,
	m, keyed *subject.Method,
) (*subject.Method, string) {
	switch from {
	case fromSelf:
		return m, ""
	case "family.reader":
		if keyed == nil {
			return nil, "names the reader family, and the interface has no keyed reader"
		}
		return keyed, ""
	case "family.writer":
		if harness != nil {
			var fallback *subject.Method
			for i := range harness.Methods {
				candidate := &harness.Methods[i]
				if pseudoShape(candidate) != shapeWriter {
					continue
				}
				// The family's writer is the values pool's own feeder: a
				// peer-merging method is writer-shaped too, and a law fed by
				// one writes values no pool ever draws.
				if q, _ := b.valueQOf(candidate); q == b.Values.Q {
					return candidate, ""
				}
				if fallback == nil {
					fallback = candidate
				}
			}
			if b.Values.Q == "" && fallback != nil {
				return fallback, ""
			}
		}
		return nil, "names the writer family, and the interface has no value writer feeding the pool"
	case fromFamilyKeyedWr:
		if harness != nil {
			for i := range harness.Methods {
				candidate := &harness.Methods[i]
				if pseudoShape(candidate) != shapeCompositeWriter {
					continue
				}
				// Same discipline as the plain writer family: the pool's own
				// feeder, so a law writing through this method writes values
				// the reads can draw and compare.
				if q, _ := b.valueQOf(candidate); q == b.Values.Q {
					return candidate, ""
				}
			}
		}
		return nil, "names the keyed-writer family, and the interface has no keyed write feeding the pool"
	case fromFamilyHandleWr:
		// A write threading an open handle: three arguments, the first the
		// handle a begin role answers. Found by shape rather than named by a
		// role, because a contract that declares begin/settle/observe has no
		// word for the staging in between — and a claim about what staging
		// does needs the staging on the interface, not reached past it.
		//
		// By shape alone, with no pool agreement demanded: a law reaching for
		// this family declares its own value domain, because the shared pool
		// on a handle-threading interface draws handles. Requiring the match
		// refuses the one method that fits.
		if harness != nil {
			for i := range harness.Methods {
				candidate := &harness.Methods[i]
				if len(candidate.CallArgs()) == 3 && errOnly(candidate) {
					return candidate, ""
				}
			}
		}
		return nil, "names the handle-writer family, and the interface declares no write threading an open handle"
	case "family.aggregator":
		if harness != nil {
			for i := range harness.Methods {
				candidate := &harness.Methods[i]
				if pseudoShape(candidate) == shapeAggregator && len(candidate.CallArgs()) == 0 {
					return candidate, ""
				}
			}
		}
		return nil, "names the aggregator family, and the interface has no aggregate"
	case fromFamilyCell:
		if harness != nil {
			for i := range harness.Methods {
				candidate := &harness.Methods[i]
				if candidate.Name == m.Name || len(candidate.CallArgs()) > 0 {
					continue
				}
				if _, _, why := resultType(candidate); why == "" {
					return candidate, ""
				}
			}
		}
		return nil, "names the cell family, and the interface has no nullary read"
	}
	if owner, param, ok := strings.Cut(from, "."); ok && !strings.HasPrefix(from, "family.") {
		if m == nil {
			return nil, "names " + from + ", which no selecting method stamps"
		}
		// A mixin's sibling parameter first — `deleteremoves.read` names the
		// method its stamp points at.
		v, stamped := shape.MixinParamKey(owner, param).Get(m.Source.Meta())
		if stamped && v != "" {
			role := methodOf(harness, golang.LocalName(v))
			if role == nil {
				return nil, "names " + from + " = " + v + ", which is not a method of " + b.IfaceName
			}
			return role, ""
		}
		// Then a contract role: the selecting method fills it itself, or its
		// directive's partner key names the sibling that does.
		if held, ownRole := shape.ContractRoleKey(owner).Get(m.Source.Meta()); ownRole && held == param {
			return m, ""
		}
		partner, named := shape.ContractPartnerKey(owner, param).Get(m.Source.Meta())
		if named && partner != "" {
			role := methodOf(harness, golang.LocalName(partner))
			if role == nil {
				return nil, "names " + from + " = " + partner + ", which is not a method of " + b.IfaceName
			}
			return role, ""
		}
		return nil, "names " + from + ", which the selecting method does not stamp"
	}
	return nil, "names " + from + ", which nothing resolves"
}

// foldedReadField fills an observation whose read answers a presence flag
// rather than an error: the closure calls it and reports the run's miss
// identity where the flag is false.
//
// The laws that observe through this field speak `(V, error)` because most
// reads do, and a law declaring both shapes would be two laws with one
// name. The seam is one line and it belongs at the binding, which is the
// only place that knows which identity this run means by absence.
//
// Only where the reference is DERIVED, and the reason is that identity. A
// derived reference mints a miss var for exactly this, and the two sides
// of a comparison then report absence the same way. A twin mints nothing,
// and a fold that invented an error would have the subject and its own
// factory disagreeing about what a miss is called.
func foldedReadField(
	b *Bindings, f tiers.Field, field *LawField, role *subject.Method,
) (*LawField, string) {
	if !b.Reference.Derived() {
		return nil, f.Name + " closes over " + role.Name +
			", which reports a miss as a flag, and this run has no derived " +
			"reference to take the miss identity from"
	}
	field.Value = role.Returns[0].Type
	field.MissSym = b.Reference.MissSym
	field.MissName = b.Reference.MissName
	field.KindName = sdk.Kind(LawFieldKindPrefix + "ReadFolded")
	return field, ""
}

// keyedReadMismatch holds a keyed-read role to the shape its template spells:
// `(ctx, K) (V, error)` at the pools' own types, so a role of another shape —
// or of the right shape over other types — renders a closure that fails to
// compile in whichever package arms it.
func keyedReadMismatch(b *Bindings, fieldName string, role *subject.Method, strictValue bool) string {
	keyQ, _ := b.keyQOf(role)
	valueQ, _ := b.valueQOf(role)
	// A read answering a presence flag is a keyed read; the closure folds
	// the flag into the error channel the law speaks. See [foldedReadField].
	if shape := pseudoShape(role); shape != shapeReader && shape != shapeReaderWithBool {
		return fieldName + " closes over " + role.Name + ", whose shape is " +
			shape + " rather than a keyed reader"
	}
	// The value half is held to the pool only where the law's own row draws
	// it: a windowed count reads int beside string pools, lawfully, because
	// nothing compares its answer to a drawn value.
	if (b.Keys.Q != "" && keyQ != b.Keys.Q) ||
		(strictValue && b.Values.Q != "" && valueQ != b.Values.Q) {

		return fieldName + " closes over " + role.Name + ", which reads (" + keyQ +
			" → " + valueQ + ") beside pools of (" + b.Keys.Q + ", " + b.Values.Q + ")"
	}
	return ""
}
