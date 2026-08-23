// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// fixedKeyProjection is the key projection for a law whose write takes the
// key beside the value: it answers which of the key pool's keys a given
// value is written under.
//
// No key rides inside such a value — the writer takes it as an argument —
// so a value-to-key projection cannot be READ off the type. It can still
// be chosen, and choosing one is what makes the law's claim answerable.
// The projection that answers the fixture's one key for every value does
// not: with the whole history on a single slot, a subject that ignores
// its key argument entirely reads back correctly every time, and
// [legs.SpreadKey] plus the engine's own test for it is where that was
// settled. The pool's keys, picked by the value, put two values on two
// keys instead.
//
// The same restriction [valueOpField] puts on the write, and the two have
// to agree. That arm anchors a composite write on the fixture key for
// exactly this reason; if it stops applying, this has to stop with it, or
// the law would read a key nothing was written to and call a correct
// subject wrong.
func fixedKeyProjection(
	b *Bindings, harness *subject.Projection,
	r tiers.Rule, f tiers.Field, field *LawField, m, keyed *subject.Method,
) (*LawField, string) {
	underivable := f.Name + " needs the key projection, which was not derivable here"
	if b.Keys.Field == "" || b.Keys.Type == nil {
		return nil, underivable
	}
	if why := spreadDecline(b); why != "" {
		return nil, f.Name + " " + why
	}
	role, reason := ruleFieldRole(b, harness, r, fWrite, m, keyed)
	if reason != "" {
		return nil, underivable
	}
	if pseudoShape(role) != shapeCompositeWriter || len(role.CallArgs()) != 2 {
		return nil, underivable
	}
	field.Key = b.Keys.Type
	field.KeyField = b.Keys.Field
	field.KeyOtherField = b.Keys.OtherField
	field.KeyPoolField = b.Keys.PoolField
	field.KindName = sdk.Kind(LawFieldKindPrefix + "KeyOfSpread")
	b.LawsUseFixture = true
	return field, ""
}

// spreadDecline is why this run cannot pick a law's write key from the
// pool by the value, empty where it can.
//
// Asked by both halves of the binding, which is the point. The write and
// the projection have to name the same key — a write anchored on one key
// under a projection that answers another reads a key nothing was
// written to, and calls a correct subject wrong — so neither arm decides
// on its own.
func spreadDecline(b *Bindings) string {
	if b.Keys.OtherField == "" && b.Keys.PoolField == "" {
		// One key is the collapsed case: every value goes to it, so the
		// law never places two values apart, and a subject that ignores
		// its key argument entirely reads back correctly every time. See
		// [legs.SpreadKey], and the engine's own test for the shape.
		return "would answer the fixture's one key for every value, and this " +
			"claim is that a value is readable under the key it went to — with " +
			"one key there is no other key it could have gone to"
	}
	if m := methodNamed(b, b.EvictingRead); b.EvictingRead != "" && m != nil {
		// Spreading the writes over the pool is what makes the claim
		// answerable, and on a store that evicts it is also what makes the
		// law wrong: the second write can drop the first, and a read that
		// then misses is the store keeping its own promise rather than a
		// value going to the wrong key.
		return "spreads the writes across the key pool so the key half of this " +
			"claim is reachable, and " + m.Name + " may evict what an earlier write " +
			"left — the read that then misses is the store's own rule, not a divergence"
	}
	if !b.Reference.Derived() {
		// The twin floor is where a declared claim defeated store
		// modelling, and "what went in comes back out" IS the store model.
		// The corpus taught this the first time this arm ran: it bound the
		// law on the accumulates fixture, whose whole claim is that a
		// repeated write COMPOUNDS — so the read answers a running total,
		// the law compared it against the last value written, and a correct
		// subject failed at step one.
		return "would place the run's values across the key pool, and the " +
			"reference here is the subject's own factory — so a claim on this " +
			"interface has already defeated the store model this law is"
	}
	return ""
}

// handleFieldOf fills a handle the generated file constructs and shares.
func handleFieldOf(
	b *Bindings, harness *subject.Projection, r tiers.Rule, f tiers.Field,
	field *LawField, m, keyed *subject.Method,
) (*LawField, string) {
	switch f.From {
	case handleKeyProjection:
		if b.Reference.KeyField != "" {
			field.KeyOfName = b.KeyOfName()
			field.KindName = sdk.Kind(LawFieldKindPrefix + "KeyOf")
			return field, ""
		}
		if r.Law == lawid.PaginatorNoDuplicates {
			// Identity over the page element: no projection derives where the
			// only reader is the walk itself, and the element is comparable —
			// the binding row instantiates K at the element for the same
			// reason.
			role, reason := ruleFieldRole(b, harness, r, fPage, m, keyed)
			if reason != "" {
				return nil, f.Name + " " + reason
			}
			elem, why := drainedElem(b, role)
			if why != "" {
				return nil, f.Name + " " + why
			}
			field.Value = elem
			field.KindName = sdk.Kind(LawFieldKindPrefix + "KeyOfIdentity")
			return field, ""
		}
		return fixedKeyProjection(b, harness, r, f, field, m, keyed)

	case "identity-hash":
		// Identity over the drained element: the hash argument is the value
		// itself, so the closure needs only the element's type.
		if elem, why := hashElem(b, harness, r, m, keyed); why == "" {
			field.Value = elem
		}
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Hash")
		return field, ""

	case handleReferenceMiss:
		// What the DERIVED reference answers for a key it does not hold.
		// A twin has no such identity — it answers whatever the subject
		// does — so the law falls back to its sentinel there, which is
		// then the same error on both sides.
		if b.Reference.MissSym == nil && b.Reference.MissName == "" {
			return nil, ""
		}
		field.MissSym, field.MissName = b.Reference.MissSym, b.Reference.MissName
		field.KindName = sdk.Kind(LawFieldKindPrefix + "ReferenceMiss")
		return field, ""

	case "subject-factory":
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Factory")
		return field, ""

	case handleClassifier:
		spec, why := sessionSpecOf(b, harness, r, m, keyed)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Classify")
		field.KeyOfName = spec.ClassifyName
		return field, ""

	case "natural-order":
		role, reason := roleMethod(b, harness, fromSelf, m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		if !orderedScalar(role) {
			return nil, f.Name + " orders " + role.Name + "'s result, which the language does not"
		}
		out, _, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Out = out
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Less")
		return field, ""

	case "observation":
		obs, reason := observationOf(b, harness, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		field.Method = obs.Method.Name
		field.TakesCtx = obs.TakesCtx
		field.Out = obs.Out
		if obs.Keyed {
			field.KeyField = b.Keys.Field
			field.KindName = sdk.Kind(LawFieldKindPrefix + "ObserveKeyed")
			b.LawsUseFixture = true
		} else {
			field.KindName = sdk.Kind(LawFieldKindPrefix + "ObserveCall")
		}
		return field, ""

	case "partitions":
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Partitions")
		return field, ""

	case "clock":
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Advance")
		return field, ""

	case handleCoalesce:
		// The law's own instrumentation: the compute it hands every caller,
		// counting how often the subject actually ran it.
		call, reason := ruleFieldRole(b, harness, r, fCall, m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		out, _, why := resultType(call)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Out = out
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Compute")
		return field, ""

	case "coalesce-counter":
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Counter")
		return field, ""

	case handleVersionStamp:
		// The version-coherent draw: read the cell, copy its version member
		// into the drawn attempt. Both types carry the member — the stamp
		// key names one field of one payload — and a fixture where they
		// drift fails to compile in the package that armed it.
		cell, reason := ruleFieldRole(b, harness, r, fRead, m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		attempt, reason := ruleFieldRole(b, harness, r, fCAS, m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		if len(attempt.CallArgs()) == 0 {
			return nil, f.Name + " stamps " + attempt.Name + "'s attempt, and it takes none"
		}
		member, stamped := stampValue(harness, m, paramCASVersion)
		if !stamped {
			return nil, f.Name + " reads the version member, and the cas directive names none"
		}
		field.Method = cell.Name
		field.TakesCtx = cell.TakesContext()
		field.In = attempt.CallArgs()[0].Type
		field.KeyField = golang.LocalName(member)
		field.KindName = sdk.Kind(LawFieldKindPrefix + "VersionStamp")
		return field, ""

	case handleWriteLog:
		// The write log a drain claim is judged against: what a collection
		// should hold is what went into it, and the action stream is the
		// only party that knows. The same recording the chain's history
		// takes, resolved off the write rather than off an append role — a
		// mixin declares no contract and names no partner, so the write is
		// found by what it carries.
		drainRole, reason := ruleFieldRole(b, harness, r, fDrain, m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		elem, why := drainedElem(b, drainRole)
		if why != "" {
			return nil, f.Name + " " + why
		}
		writer, why := writeOfDrained(b, harness, drainRole)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Method = writer
		field.Value = elem
		field.KindName = sdk.Kind(LawFieldKindPrefix + "HistoryRef")
		return field, ""

	case handleHistoryLog:
		// The append-recording history: a property-level log of every
		// successful append the sequences drove, cleared by the runner each
		// iteration. The field rides the append role so the inert check
		// catches a derived reference answering it inertly.
		appendRole, reason := roleMethod(b, harness, "chain.append", m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		replayRole, reason := ruleFieldRole(b, harness, r, fReplay, m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		elem, why := drainedElem(b, replayRole)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Method = appendRole.Name
		field.Value = elem
		field.KindName = sdk.Kind(LawFieldKindPrefix + "HistoryRef")
		return field, ""
	}
	return nil, f.Name + " needs the " + f.From + " handle, which this build does not construct"
}

// writeOfDrained names the driven method that puts the drained element
// in, and why none does.
//
// Matched on what the method carries rather than on its name or its
// position: an interface may declare several writes, and the one whose
// value is what the drain yields is the one whose record the claim is
// about. Two of them writing the same element is refused rather than
// resolved — a log filled by one of two writes states a claim about half
// the history and reads as a claim about all of it.
//
// Driven, too: a method with no action never runs, so a log riding it
// would be empty on every draw and the claim would hold over nothing.
func writeOfDrained(b *Bindings, harness *subject.Projection, drain *subject.Method) (string, string) {
	want, why := drainedQName(b, drain)
	if why != "" {
		return "", why
	}
	var found string
	for i := range harness.Methods {
		write := &harness.Methods[i]
		if pseudoShape(write) != shapeWriter || b.actionFor(write.Name) == nil {
			continue
		}
		if q, stamped := b.valueQOf(write); !stamped || q != want {
			continue
		}
		if found != "" {
			return "", "reads what was written, and " + found + " and " + write.Name +
				" both write " + want + " — which of them the claim is about is not derivable"
		}
		found = write.Name
	}
	if found == "" {
		return "", "reads what was written, and no driven method here writes " + want
	}
	return found, ""
}

// drainedQName is [drainedElem]'s answer as the stamp spells it, for
// comparing against what a write carries — the reference the other
// returns is a rendering, and two spellings of one type render alike
// without being equal values.
func drainedQName(b *Bindings, m *subject.Method) (string, string) {
	if returnsSlice(m) {
		return shape.QName(shape.GoSliceElem(m.Returns[0].Source)), ""
	}
	q, stamped := b.valueQOf(m)
	if !stamped || q == "" {
		return "", "drains " + m.Name + ", which streams elements no stamp names"
	}
	return q, ""
}

// hashElem resolves the identity hash's element: the drained element of the
// same rule's Drain field where one exists, the values pool otherwise.
func hashElem(
	b *Bindings, harness *subject.Projection, r tiers.Rule, m, keyed *subject.Method,
) (sdk.Ref, string) {
	for _, f := range r.Fields {
		if f.Kind != tiers.KindRole || (f.Name != fDrain && f.Name != "Collect") {
			continue
		}
		role, reason := roleMethod(b, harness, f.From, m, keyed)
		if reason != "" {
			return nil, reason
		}
		return drainedElem(b, role)
	}
	if b.Values.Type != nil {
		return b.Values.Type, ""
	}
	return nil, "hashes a value type no method here draws"
}
