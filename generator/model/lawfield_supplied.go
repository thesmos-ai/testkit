// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// suppliedFieldOf builds a consumer-supplied door: the closure type spelled
// at this fixture's instantiation, the config field the guarded
// registration reads, and the option that fills it. A type the fixture
// cannot spell keeps the refusal, naming what is missing.
func suppliedFieldOf(
	b *Bindings, harness *subject.Projection, r tiers.Rule, f tiers.Field,
	field *LawField, m, keyed *subject.Method,
) (*LawField, string) {
	shapes, known := suppliedShapes[r.Law]
	sh, mapped := shapes[f.Name]
	if !known || !mapped {
		return nil, f.Name + " waits on the " + f.From + " option, which no generated value can stand in for"
	}

	opt := &SuppliedOption{
		Field:  f.Name,
		Config: strings.ToLower(f.Name[:1]) + f.Name[1:],
		Shape:  sh,
		Iface:  b.IfaceRef,
	}
	switch sh {
	case supClientOpPred, supTxnHistory, supKeyPred:
		if b.Keys.Type == nil {
			return nil, f.Name + " is typed at a key no method here draws"
		}
		opt.Key = b.Keys.Type
	case supElemPred, supElemList:
		elem, why := drainedElem(b, m)
		if why != "" {
			return nil, f.Name + " " + why
		}
		opt.Elem = elem
	case supMerge:
		if harness == nil {
			return nil, f.Name + " merges the observed state, and this interface observes state through no method here"
		}
		obs, why := observationOf(b, harness, keyed)
		if why != "" {
			return nil, f.Name + " merges the observed state, and this interface " + why
		}
		opt.Out = obs.Out
	case supEntryID, supDependsOn:
		role, reason := roleMethod(b, harness, "chain.replay", m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		elem, why := drainedElem(b, role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		opt.Elem = elem
	case supSubjPred, supStats:
		// The subject alone; nothing more to resolve.
	}

	if why := b.addSuppliedOption(opt); why != "" {
		return nil, f.Name + " " + why
	}
	field.KindName = sdk.Kind(LawFieldKindPrefix + "SuppliedField")
	field.Pool = opt.Config
	// The refs the door's own type is spelled from. Copied onto the field
	// rather than looked up through the option, because the field renders
	// standalone: the backend dispatches it by kind with no path back to
	// the binding that holds it.
	field.Shape, field.Iface, field.Key, field.Elem, field.Out =
		opt.Shape, opt.Iface, opt.Key, opt.Elem, opt.Out
	return field, ""
}

// suppliedShapes transcribes each supplied law field's closure type from
// the engine structs — the third transcription beside the binding rows and
// the role shapes. An entry is what lets the generator spell the typed
// option a consumer arms the law through.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var suppliedShapes = map[string]map[string]string{
	lawid.CausalOrdering:        {"HappensBefore": supClientOpPred},
	lawid.StreamStableOrder:     {"Less": supElemPred},
	lawid.StreamOverMatch:       {"Required": supElemList},
	lawid.StreamPermutation:     {"Expected": supElemList},
	lawid.SnapshotIsolationG0:   {fHistory: supTxnHistory},
	lawid.SnapshotIsolationG1:   {fHistory: supTxnHistory},
	lawid.SnapshotIsolationG2:   {fHistory: supTxnHistory},
	lawid.EventualConvergence:   {fMerge: supMerge},
	lawid.LeaseReleasedOnCancel: {"Free": supKeyPred},
	lawid.PoolBalanced:          {"Stats": supStats},
	lawid.PoolLeakFree:          {"Balanced": supSubjPred},
	lawid.ReplayCausalOrdering:  {fEntryID: supEntryID, "DependsOn": supDependsOn},
}

// drainFieldOf derives the subscription sweep where the subscribe role
// answers a channel, or refuses with the option that would serve instead.
//
// The derived form is the synchronous floor: everything a subscriber is
// owed is in its channel by the time Publish returns, and the sweep reads
// what is there without blocking. An asynchronous publisher supplies its
// own drain through the generated option, which outranks this derivation —
// the property prefers the config's closure and falls back to the sweep.
// disturbFieldOf derives the point-in-time disturbance: a drawn value
// written under an adjacent key, between the law's two reads. Adjacent
// deliberately — a same-key overwrite is invisible-by-right only under
// resolution pinning, which is the sticky claim; what this shape can
// promise is that concurrent activity elsewhere does not perturb the key
// being read. The write rides the values-feeding writer and the key
// projection the reference keys on; where either is absent the field stays
// omitted — the law then checks read purity alone, the claim's floor.
func disturbFieldOf(
	b *Bindings, harness *subject.Projection, field *LawField, m, keyed *subject.Method,
) (*LawField, string) {
	writer, reason := roleMethod(b, harness, "family.writer", m, keyed)
	if reason != "" || !b.UsesValues() || !b.UsesKeys() || b.Reference.KeyField == "" {
		return nil, ""
	}
	field.Method = writer.Name
	field.TakesCtx = writer.TakesContext()
	field.Pool = poolValues
	field.KeyField = b.Reference.KeyField
	field.KindName = sdk.Kind(LawFieldKindPrefix + "DisturbWrite")
	return field, ""
}

// memberFieldOf fills a closure over a member of the handle the watch role
// answers: the resolver stamped the member's qualified name, the closure
// spells the call, and the compile gate in the armed package holds the
// member to the shape the law's field declares.
func memberFieldOf(
	b *Bindings, harness *subject.Projection, r tiers.Rule, f tiers.Field,
	field *LawField, m, keyed *subject.Method,
) (*LawField, string) {
	watch, reason := ruleFieldRole(b, harness, r, fWatch, m, keyed)
	if reason != "" {
		return nil, f.Name + " " + reason
	}
	out, _, why := resultType(watch)
	if why != "" {
		return nil, f.Name + " reads through " + watch.Name + "'s handle, and it answers none"
	}
	member, stamped := stampValue(harness, m, paramWatcherKey+f.From)
	if !stamped {
		return nil, f.Name + " reads the " + f.From + "= member, which the watcher directive does not name"
	}
	if f.From == memberNext && b.Values.Type == nil {
		return nil, f.Name + " yields a value no method here draws"
	}
	field.Out = out
	field.KeyField = golang.LocalName(member)
	switch f.From {
	case memberNext:
		field.KindName = sdk.Kind(LawFieldKindPrefix + "MemberNext")
	case memberStop:
		field.KindName = sdk.Kind(LawFieldKindPrefix + "MemberStop")
	}
	return field, ""
}

func drainFieldOf(
	b *Bindings, harness *subject.Projection, f tiers.Field,
	field *LawField, m, keyed *subject.Method,
) (*LawField, string) {
	role, reason := roleMethod(b, harness, "publisher.subscribe", m, keyed)
	if reason != "" {
		return nil, f.Name + " " + reason
	}
	out, ret, why := resultType(role)
	if why != "" {
		return nil, f.Name + " " + why
	}
	isChan := false
	if ret.Source != nil {
		isChan, _ = golang.MetaIsChannel.Get(ret.Source.Meta())
	}
	if !isChan {
		return nil, f.Name + " waits on the " + f.From + " option — the subscription " +
			"answers no channel this sweep can read"
	}
	if b.Publisher == nil {
		msgQ, stamped := golang.MetaChanElem.Get(ret.Source.Meta())
		if !stamped || msgQ == "" {
			return nil, f.Name + " drains a channel whose element no stamp names"
		}
		msg, err := golang.RefForQualified(b.substQ(msgQ), b.IfaceName)
		if err != nil {
			return nil, f.Name + " drains " + msgQ + ", which no closure can spell: " + err.Error()
		}
		b.Publisher = &PublisherSpec{
			DrainName: strings.ToLower(b.IfaceName[:1]) + b.IfaceName[1:] + "DrainSubscription",
			Sub:       out,
			Msg:       msg,
		}
	}
	field.KindName = sdk.Kind(LawFieldKindPrefix + "DrainSub")
	field.KeyOfName = "drainSub"
	return field, ""
}
