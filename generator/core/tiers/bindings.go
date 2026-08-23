// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers

import (
	"slices"
	"strings"

	"go.thesmos.sh/testkit/core/lawid"
)

// Binding is how a generated file instantiates a rule's law: the exported
// struct in `engine/model/law`, and its type arguments after the subject.
//
// A column rather than a derivation because neither half is derivable. The
// type name carries word boundaries and renames the identifier flattened —
// AUTO-CURSOR-NEXT-AFTER-CLOSE is CursorNextAfterCloseSentinel — and the
// argument order is each struct's own: WriteObservable is [T, V, K] where
// ReadAfterWrite is [T, K, V], and a generator that guessed would produce a
// file that fails to compile in whichever corpus package armed it. The
// conformance gate holds every filled row to the shipped struct by reflection.
//
// The column fills as fixtures arm: a rule without one selects and is reported
// unbound by the generated header, never silently dropped.
type Binding struct {
	// Type is the law struct's exported identifier.
	Type string

	// Args are the type arguments after the subject, in the struct's own
	// declaration order.
	Args []BindArg

	// Ptr marks a stateful law — one whose Check keeps memory across calls
	// behind a pointer receiver, so the composite literal must be addressed.
	// The gate holds it to the struct's method set by reflection: a value
	// type with no Check method is a law that must be bound by pointer.
	Ptr bool

	// Timeaware marks a law living in engine/model/timeaware rather than
	// engine/model/law — the clock-shaped family, boxed apart because its
	// checks read time.
	Timeaware bool
}

// BindArg names one type argument, resolved by the generator against the
// interface it is emitting for.
//
// Two bare spellings and four field-qualified ones. The bare pair names the
// shared pools; the qualified forms name a manifest field of the same rule
// and read a type off the method that fills it — which is the only place the
// type exists, since a law like Roundtrip is instantiated at its forward
// role's result and no pool ever draws one.
type BindArg string

// The argument vocabulary. Key and Value are the shared pool types — the same
// two every action draws from, which is what keeps a law's draws colliding
// with the sequences it runs beside. Observation is the composed whole-state
// observation's type — the same derivation the Observe handle renders.
// Partition is the replay partition key: the projection the partition mixin
// names where one is declared, the single anonymous partition otherwise.
const (
	BindKey         BindArg = "key"
	BindValue       BindArg = "value"
	BindObservation BindArg = "observation"
	BindPartition   BindArg = "partition"
)

// The field-qualified prefixes, composed by the constructors below.
const (
	bindResultPrefix = "result:"
	bindInputPrefix  = "input:"
	bindElemPrefix   = "elem:"
	bindScalarPrefix = "scalar:"
)

// ResultOf instantiates at the first non-error result type of the method the
// named manifest field resolves to.
func ResultOf(field string) BindArg { return BindArg(bindResultPrefix + field) }

// InputOf instantiates at the first non-context parameter type of the method
// the named manifest field resolves to.
func InputOf(field string) BindArg { return BindArg(bindInputPrefix + field) }

// ElemOf instantiates at the element type of the stream the named field's
// method drains — a slice's element, or an iterator's yielded value.
func ElemOf(field string) BindArg { return BindArg(bindElemPrefix + field) }

// ScalarOf instantiates at the named field's scalar observation: the method's
// numeric result where it has one, int where the observation is the length of
// a returned slice — the same adaptation the field's closure renders.
func ScalarOf(field string) BindArg { return BindArg(bindScalarPrefix + field) }

// Qualifier splits a field-qualified argument into its form and the manifest
// field it names, false for the bare pool spellings.
func (a BindArg) Qualifier() (form, field string, ok bool) {
	s := string(a)
	for _, prefix := range []string{bindResultPrefix, bindInputPrefix, bindElemPrefix, bindScalarPrefix} {
		if rest, found := strings.CutPrefix(s, prefix); found {
			return strings.TrimSuffix(prefix, ":"), rest, true
		}
	}
	return "", "", false
}

// publisherBoundType is the one struct all three delivery bounds
// instantiate; the mode field is what tells them apart.
const publisherBoundType = "PublisherDeliveryBound"

// publisherArgs is the instantiation the publisher family shares: the
// message at the publish role's input, the subscription at the subscribe
// role's result.
func publisherArgs() []BindArg {
	return []BindArg{InputOf("Publish"), ResultOf("Subscribe")}
}

// BindingFor returns the named law's instantiation spec, and whether the
// column carries one yet.
func BindingFor(law string) (Binding, bool) {
	b, ok := bindings[law]
	return b, ok
}

// Bound returns every law the column covers, sorted, for the gate that holds
// each row to the shipped struct.
func Bound() []string {
	out := make([]string, 0, len(bindings))
	for id := range bindings {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

// bindings is the column, keyed by law identifier.
//
// A law absent here is one no generated closure can compose — a supplied
// comparator, a handle the runner does not offer, a role shape no fixture
// can declare — and the assertion gate's register carries its reason. A law
// present here can still refuse per fixture; the difference is that the
// refusal names the missing piece instead of the missing row.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var bindings = map[string]Binding{
	lawid.Cacheable:             {Type: "Cacheable", Args: []BindArg{BindKey, ResultOf(fieldRead)}},
	lawid.DefaultOnError:        {Type: "DefaultOnError", Args: []BindArg{BindKey, BindValue}},
	lawid.DeleteReturnsNotFound: {Type: "DeleteReturnsNotFound", Args: []BindArg{BindKey, BindValue}},
	lawid.PointInTime:           {Type: "PointInTime", Args: []BindArg{BindKey, BindValue}},
	lawid.ReadAfterWrite:        {Type: "ReadAfterWrite", Args: []BindArg{BindKey, BindValue}},
	lawid.Sticky:                {Type: "Sticky", Args: []BindArg{BindKey, BindValue}, Ptr: true},
	lawid.WriteObservable:       {Type: "WriteObservable", Args: []BindArg{BindValue, BindKey}},

	// The stream family. The hash argument is the value itself — the drained
	// values are comparable, so identity is the strongest fingerprint and the
	// only one nothing has to invent. The two laws over the bare drain
	// instantiate at the drained element rather than the values pool: a
	// read-only stream declares no writer, so no pool ever draws its element.
	lawid.StreamCompletion:   {Type: "StreamCompletion", Args: []BindArg{ElemOf(fieldDrain)}},
	lawid.StreamNoDuplicates: {Type: "StreamNoDuplicates", Args: []BindArg{BindValue, BindValue}},
	lawid.StreamOverMatch:    {Type: "StreamOverMatch", Args: []BindArg{BindValue, BindValue}},
	lawid.StreamPermutation:  {Type: "StreamPermutation", Args: []BindArg{BindValue, BindValue}},
	lawid.StreamReentrant:    {Type: "StreamReentrancy", Args: []BindArg{ElemOf("Collect")}},
	lawid.StreamStableOrder:  {Type: "StreamStableOrder", Args: []BindArg{BindValue}},

	lawid.CausalOrdering:        {Type: "CausalOrdering", Ptr: true, Args: []BindArg{BindKey}},
	lawid.SnapshotIsolationG0:   {Type: "SnapshotIsolationG0", Args: []BindArg{BindKey}},
	lawid.SnapshotIsolationG1:   {Type: "SnapshotIsolationG1", Args: []BindArg{BindKey}},
	lawid.SnapshotIsolationG2:   {Type: "SnapshotIsolationG2", Args: []BindArg{BindKey}},
	lawid.EventualConvergence:   {Type: "EventualConvergence", Args: []BindArg{BindValue, BindObservation}},
	lawid.LeaseReleasedOnCancel: {Type: "LeaseReleasedOnCancel", Args: []BindArg{BindKey}},
	lawid.PoolBalanced:          {Type: "PoolBalancedGetPut"},
	lawid.PoolLeakFree:          {Type: "PoolLeakFree"},
	// K is the partition key, like the rest of the chain family — the
	// anonymous single partition until a partition projection is declared.
	lawid.ReplayCausalOrdering: {
		Type: "ReplayRespectsCausality",
		Args: []BindArg{BindPartition, ElemOf(fieldReplay)},
	},
	lawid.StreamReflectsMutations: {
		Type: "StreamReflectsMutations",
		Args: []BindArg{ElemOf(fieldDrain), ElemOf(fieldDrain)},
	},

	// The self-contained detector laws: the method under its own claim, no
	// second role in the room.
	lawid.AggregatorBounded:        {Type: "AggregatorBounded", Args: []BindArg{ScalarOf(fieldRead)}},
	lawid.CountEqualsReference:     {Type: "CountEqualsReference", Args: []BindArg{ScalarOf(fieldCount)}},
	lawid.LifecycleRespectsContext: {Type: "LifecycleRespectsContext", Args: []BindArg{}},
	lawid.MonotonicNonDecreasing:   {Type: "MonotonicNonDecreasing", Args: []BindArg{ScalarOf(fieldRead)}, Ptr: true},
	lawid.PoisonIdempotentRead:     {Type: "PoisonIdempotentRead", Args: []BindArg{}},
	lawid.PoisonNilOnFresh:         {Type: "PoisonNilOnFresh", Args: []BindArg{}},
	lawid.PredicateConsistent:      {Type: "PredicateConsistency", Args: []BindArg{}},
	lawid.PureDeterministic:        {Type: "PureDeterminism", Args: []BindArg{ResultOf(fieldCall)}},
	lawid.TotalOver:                {Type: "TotalOver", Args: []BindArg{InputOf(fieldCall), ResultOf(fieldCall)}},

	// The write-family laws: a mutation beside the observation that makes it
	// checkable, both spelled by the manifest's own fields.
	lawid.Associative:      {Type: "Associative", Args: []BindArg{InputOf("Apply"), BindObservation}},
	lawid.AtomicWrite:      {Type: "AtomicWrite", Args: []BindArg{InputOf(fieldWrite), BindObservation}},
	lawid.CommutativeWrite: {Type: "CommutativeWrite", Args: []BindArg{InputOf(fieldWrite), BindObservation}},
	lawid.Conservative:     {Type: "Conservative", Args: []BindArg{InputOf(fieldWrite)}},
	lawid.CRDTMerge:        {Type: "CRDTMerge", Args: []BindArg{InputOf(fieldWrite), BindObservation}},
	lawid.IdempotentWrite:  {Type: "IdempotentWrite", Args: []BindArg{InputOf(fieldWrite), BindObservation}},
	lawid.InjectionSafe:    {Type: "InjectionSafe", Args: []BindArg{}},
	lawid.XSSSafe:          {Type: "XSSSafe", Args: []BindArg{}},

	// The contract laws whose roles a generated closure can call directly.
	lawid.AppenderMonotonicOffsets: {
		Type: "AppenderMonotonicOffsets",
		Args: []BindArg{InputOf("Append"), ResultOf("Append")},
		Ptr:  true,
	},
	lawid.CASAtomicOneWinner:       {Type: "CASAtomicOneWinner", Args: []BindArg{InputOf("CAS")}},
	lawid.LeaseDoubleAcquireBlocks: {Type: "LeaseDoubleAcquireBlocks", Args: []BindArg{BindKey}},
	lawid.LeakFree:                 {Type: "LeakFree", Args: []BindArg{}},
	lawid.PersisterRetrievable:     {Type: "PersisterRetrievable", Args: []BindArg{InputOf("Save"), BindKey}},
	lawid.Roundtrip:                {Type: "Roundtrip", Args: []BindArg{ResultOf("Forward")}},
	lawid.LossyRoundtrip:           {Type: "LossyRoundtrip", Args: []BindArg{ResultOf("Forward")}},
	lawid.UpdaterReplaces:          {Type: "UpdaterReplaces", Args: []BindArg{InputOf("Update"), BindKey}},
	lawid.UpserterIdempotent:       {Type: "UpserterIdempotent", Args: []BindArg{InputOf("Upsert"), BindKey}},
	lawid.ValidTransition:          {Type: "ValidTransition", Args: []BindArg{InputOf(fieldWrite), BindObservation}},

	// The chain family, over a slice-replaying log: the replay adapts to the
	// iterator the law drains, and the partition set is the anonymous single
	// partition until a partition projection is declared.
	lawid.AppendOnlyGrows: {
		Type: "AppendOnlyHistoryGrows",
		Args: []BindArg{BindPartition, ElemOf(fieldReplay)},
		Ptr:  true,
	},
	lawid.AppendOnlyNoDrops: {
		Type: "AppendOnlyNoDrops",
		Args: []BindArg{BindPartition, ElemOf(fieldReplay)},
	},

	// The isolated family: each Check corrupts its own throwaway pair, which
	// the runner hands it once per iteration — the conduct census flipped
	// these sound when the [law.Isolated] marker landed.
	lawid.TamperEvident:         {Type: "TamperEvident", Args: []BindArg{InputOf(fieldWrite)}},
	lawid.CursorCloseIdempotent: {Type: "CursorCloseIdempotent", Args: []BindArg{}},
	lawid.CursorNextAfterClose:  {Type: "CursorNextAfterCloseSentinel", Args: []BindArg{ResultOf("Next")}},
	lawid.IdempotentLifecycle:   {Type: "IdempotentLifecycle", Args: []BindArg{BindObservation}},
	lawid.LifecycleAfterClose:   {Type: "LifecycleAfterCloseSentinel", Args: []BindArg{}},
	lawid.PoisonConsistent:      {Type: "PoisonConsistent", Args: []BindArg{}},

	// The clock family: bound conditionally on the ModelClocked option, whose
	// factory builds the subject on the run's own test clock — a law that
	// advances a clock the subject does not read fails every correct
	// implementation.
	lawid.PublisherDelivers:    {Type: "PublisherDelivers", Args: publisherArgs()},
	lawid.PublisherAtLeastOnce: {Type: publisherBoundType, Args: publisherArgs()},
	lawid.PublisherAtMostOnce:  {Type: publisherBoundType, Args: publisherArgs()},
	lawid.PublisherExactlyOnce: {Type: publisherBoundType, Args: publisherArgs()},

	lawid.MonotonicReads:    {Type: "MonotonicReads", Ptr: true, Args: []BindArg{BindKey}},
	lawid.MonotonicWrites:   {Type: "MonotonicWrites", Ptr: true, Args: []BindArg{BindKey}},
	lawid.ReadYourWrites:    {Type: "ReadYourWrites", Ptr: true, Args: []BindArg{BindKey}},
	lawid.WritesFollowReads: {Type: "WritesFollowReads", Ptr: true, Args: []BindArg{BindKey}},

	// The contract-shape family: laws over roles whose signatures carry a
	// handle, a callable or a cursor. Each instantiates off the role's own
	// types, because no shared pool draws a transaction handle or a page
	// token.
	lawid.TwoPhaseMutex: {
		Type: "TwoPhaseCommitOrRollback",
		Args: []BindArg{ResultOf("Begin")},
	},
	lawid.TwoPhaseRollbackAfterCommit: {
		Type: "TwoPhaseNoRollbackAfterCommit",
		Args: []BindArg{ResultOf("Begin")},
	},
	lawid.SagaFullCompensation: {Type: "SagaFullCompensation", Args: []BindArg{BindObservation}},
	lawid.TransactionNoMidTxVisibility: {
		Type: "TransactionNoMidTxVisibility",
		Args: []BindArg{ResultOf("Begin"), BindKey, ResultOf(fieldRead)},
	},
	lawid.SingleflightCoalesces: {
		Type: "SingleflightCoalesces",
		Args: []BindArg{InputOf(fieldCall), ResultOf(fieldCall)},
	},
	lawid.TransactionRollback: {
		Type: "TransactionRollbackOnError",
		Args: []BindArg{BindKey, ResultOf(fieldRead)},
	},
	// K is the page element itself: no key projection derives where the
	// fixture's only reader is the page walk, and identity is the strongest
	// fingerprint the drained element carries — the KeyOf handle renders the
	// matching identity closure.
	lawid.PaginatorNoDuplicates: {
		Type: "PaginatorNoDuplicates",
		Args: []BindArg{ElemOf("Page"), ElemOf("Page"), InputOf("Page")},
	},
	lawid.PaginatorResumable: {
		Type: "PaginatorResumable",
		Args: []BindArg{ElemOf("Page"), InputOf("Page")},
	},
	// The handle at the watch role's answer, and the pools at the trigger's
	// key and value — the member closures read through the handle, so its
	// type is the instantiation's anchor.
	lawid.WatcherReturnsOnChange: {
		Type: "WatcherReturnsOnChange",
		Args: []BindArg{ResultOf("Watch"), BindKey, BindValue},
	},

	lawid.TTLExpiry: {
		Type: "TTLExpiryAfterAdvance", Timeaware: true,
		Args: []BindArg{BindKey, BindValue},
	},
	lawid.DeadlineRespecting: {Type: "DeadlineRespecting", Timeaware: true, Args: []BindArg{}},
	// The key and nothing else: this law asks whether the reading moved,
	// not what it moved to, so the answer's own type never appears.
	lawid.TimeawareMoves:             {Type: "MovesWithTheClock", Timeaware: true, Args: []BindArg{BindKey}},
	lawid.ScheduledFiresAfterAdvance: {Type: "ScheduledFiresAfterAdvance", Timeaware: true, Args: []BindArg{}},
	lawid.Windowed:                   {Type: "Windowed", Args: []BindArg{BindKey}},
	lawid.HashChainIntegrityVerify:   {Type: "HashChainIntegrityViaVerify", Args: []BindArg{}},
	lawid.ReplayDeterministic: {
		Type: "ReplayDeterminism",
		Args: []BindArg{BindPartition, ElemOf(fieldReplay)},
	},
}
