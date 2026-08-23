// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

import (
	"reflect"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/engine/model/law"
	"go.thesmos.sh/testkit/engine/model/timeaware"
)

// LawTypes maps every declared identifier to the struct that reports it.
//
// This module is the only one that can hold this: the identifiers live in the
// root module, the laws in `engine`, and the selection rules in `generator` —
// and none of those three depends on the other two. Reading all three at once
// is what the conformance module is for.
//
// The type arguments are stand-ins. Nothing here calls a law or compares a
// value; the gate reads the struct's field set and its identifier, and both are
// the same whatever the type parameters are instantiated at. The choice is
// simply the simplest that satisfies each constraint.
//
// Hand-written, and held to the catalogue in both directions by
// [TestEveryLawIsAccountedFor]: a law added upstream with no entry here fails,
// and an entry for a law that no longer exists fails to compile.
//
//nolint:gochecknoglobals // a lookup table, read-only after init
var LawTypes = map[string]reflect.Type{
	lawid.AggregatorBounded:            reflect.TypeFor[law.AggregatorBounded[any, int]](),
	lawid.AppendOnlyGrows:              reflect.TypeFor[law.AppendOnlyHistoryGrows[any, string, any]](),
	lawid.AppendOnlyNoDrops:            reflect.TypeFor[law.AppendOnlyNoDrops[any, string, any]](),
	lawid.AppenderMonotonicOffsets:     reflect.TypeFor[law.AppenderMonotonicOffsets[any, any, int]](),
	lawid.Associative:                  reflect.TypeFor[law.Associative[any, any, any]](),
	lawid.AtomicWrite:                  reflect.TypeFor[law.AtomicWrite[any, any, any]](),
	lawid.CASAtomicOneWinner:           reflect.TypeFor[law.CASAtomicOneWinner[any, any]](),
	lawid.CRDTMerge:                    reflect.TypeFor[law.CRDTMerge[any, any, any]](),
	lawid.Cacheable:                    reflect.TypeFor[law.Cacheable[any, string, any]](),
	lawid.CausalOrdering:               reflect.TypeFor[law.CausalOrdering[any, string]](),
	lawid.CommutativeWrite:             reflect.TypeFor[law.CommutativeWrite[any, any, any]](),
	lawid.Conservative:                 reflect.TypeFor[law.Conservative[any, any]](),
	lawid.CountEqualsReference:         reflect.TypeFor[law.CountEqualsReference[any, string]](),
	lawid.CursorCloseIdempotent:        reflect.TypeFor[law.CursorCloseIdempotent[any]](),
	lawid.CursorNextAfterClose:         reflect.TypeFor[law.CursorNextAfterCloseSentinel[any, any]](),
	lawid.DeadlineRespecting:           reflect.TypeFor[timeaware.DeadlineRespecting[any]](),
	lawid.DefaultOnError:               reflect.TypeFor[law.DefaultOnError[any, string, any]](),
	lawid.DeleteReturnsNotFound:        reflect.TypeFor[law.DeleteReturnsNotFound[any, string, any]](),
	lawid.EventualConvergence:          reflect.TypeFor[law.EventualConvergence[any, any, any]](),
	lawid.HashChainIntegrityErr:        reflect.TypeFor[law.HashChainIntegrityViaErr[any]](),
	lawid.HashChainIntegrityVerify:     reflect.TypeFor[law.HashChainIntegrityViaVerify[any]](),
	lawid.IdempotentLifecycle:          reflect.TypeFor[law.IdempotentLifecycle[any, any]](),
	lawid.IdempotentWrite:              reflect.TypeFor[law.IdempotentWrite[any, any, any]](),
	lawid.InjectionSafe:                reflect.TypeFor[law.InjectionSafe[any]](),
	lawid.LeakFree:                     reflect.TypeFor[law.LeakFree[any]](),
	lawid.LeaseDoubleAcquireBlocks:     reflect.TypeFor[law.LeaseDoubleAcquireBlocks[any, string]](),
	lawid.LeaseReleasedOnCancel:        reflect.TypeFor[law.LeaseReleasedOnCancel[any, string]](),
	lawid.LifecycleAfterClose:          reflect.TypeFor[law.LifecycleAfterCloseSentinel[any]](),
	lawid.LifecycleRespectsContext:     reflect.TypeFor[law.LifecycleRespectsContext[any]](),
	lawid.LossyRoundtrip:               reflect.TypeFor[law.LossyRoundtrip[any, any]](),
	lawid.MonotonicNonDecreasing:       reflect.TypeFor[law.MonotonicNonDecreasing[any, any]](),
	lawid.MonotonicReads:               reflect.TypeFor[law.MonotonicReads[any, string]](),
	lawid.MonotonicWrites:              reflect.TypeFor[law.MonotonicWrites[any, string]](),
	lawid.PaginatorNoDuplicates:        reflect.TypeFor[law.PaginatorNoDuplicates[any, any, string, any]](),
	lawid.PaginatorResumable:           reflect.TypeFor[law.PaginatorResumable[any, any, any]](),
	lawid.PersisterRetrievable:         reflect.TypeFor[law.PersisterRetrievable[any, any, string]](),
	lawid.PointInTime:                  reflect.TypeFor[law.PointInTime[any, string, any]](),
	lawid.PoisonConsistent:             reflect.TypeFor[law.PoisonConsistent[any]](),
	lawid.PoisonIdempotentRead:         reflect.TypeFor[law.PoisonIdempotentRead[any]](),
	lawid.PoisonNilOnFresh:             reflect.TypeFor[law.PoisonNilOnFresh[any]](),
	lawid.PoolBalanced:                 reflect.TypeFor[law.PoolBalancedGetPut[any]](),
	lawid.PoolLeakFree:                 reflect.TypeFor[law.PoolLeakFree[any]](),
	lawid.PredicateConsistent:          reflect.TypeFor[law.PredicateConsistency[any]](),
	lawid.PublisherAtLeastOnce:         reflect.TypeFor[law.PublisherDeliveryBound[any, string, any]](),
	lawid.PublisherAtMostOnce:          reflect.TypeFor[law.PublisherDeliveryBound[any, string, any]](),
	lawid.PublisherDelivers:            reflect.TypeFor[law.PublisherDelivers[any, string, any]](),
	lawid.PublisherDelivery:            reflect.TypeFor[law.PublisherDeliveryBound[any, string, any]](),
	lawid.PublisherExactlyOnce:         reflect.TypeFor[law.PublisherDeliveryBound[any, string, any]](),
	lawid.PureDeterministic:            reflect.TypeFor[law.PureDeterminism[any, any]](),
	lawid.ReadAfterWrite:               reflect.TypeFor[law.ReadAfterWrite[any, string, any]](),
	lawid.ReadYourWrites:               reflect.TypeFor[law.ReadYourWrites[any, string]](),
	lawid.ReplayCausalOrdering:         reflect.TypeFor[law.ReplayRespectsCausality[any, string, any]](),
	lawid.ReplayDeterministic:          reflect.TypeFor[law.ReplayDeterminism[any, string, any]](),
	lawid.Roundtrip:                    reflect.TypeFor[law.Roundtrip[any, any]](),
	lawid.SagaFullCompensation:         reflect.TypeFor[law.SagaFullCompensation[any, string]](),
	lawid.ScheduledFiresAfterAdvance:   reflect.TypeFor[timeaware.ScheduledFiresAfterAdvance[any]](),
	lawid.SingleflightCoalesces:        reflect.TypeFor[law.SingleflightCoalesces[any, string, any]](),
	lawid.SnapshotIsolationG0:          reflect.TypeFor[law.SnapshotIsolationG0[any, string]](),
	lawid.SnapshotIsolationG1:          reflect.TypeFor[law.SnapshotIsolationG1[any, string]](),
	lawid.SnapshotIsolationG2:          reflect.TypeFor[law.SnapshotIsolationG2[any, string]](),
	lawid.Sticky:                       reflect.TypeFor[law.Sticky[any, string, any]](),
	lawid.TimeawareMoves:               reflect.TypeFor[timeaware.MovesWithTheClock[any, string]](),
	lawid.StreamCompletion:             reflect.TypeFor[law.StreamCompletion[any, any]](),
	lawid.StreamNoDuplicates:           reflect.TypeFor[law.StreamNoDuplicates[any, any, string]](),
	lawid.StreamOverMatch:              reflect.TypeFor[law.StreamOverMatch[any, any, string]](),
	lawid.StreamPermutation:            reflect.TypeFor[law.StreamPermutation[any, any, string]](),
	lawid.StreamReentrant:              reflect.TypeFor[law.StreamReentrancy[any, any]](),
	lawid.StreamReflectsMutations:      reflect.TypeFor[law.StreamReflectsMutations[any, any, string]](),
	lawid.StreamStableOrder:            reflect.TypeFor[law.StreamStableOrder[any, any]](),
	lawid.TTLExpiry:                    reflect.TypeFor[timeaware.TTLExpiryAfterAdvance[any, string, any]](),
	lawid.TamperEvident:                reflect.TypeFor[law.TamperEvident[any, any]](),
	lawid.TotalOver:                    reflect.TypeFor[law.TotalOver[any, any, string]](),
	lawid.TransactionNoMidTxVisibility: reflect.TypeFor[law.TransactionNoMidTxVisibility[any, any, string, any]](),
	lawid.TransactionRollback:          reflect.TypeFor[law.TransactionRollbackOnError[any, string, any]](),
	lawid.TwoPhaseMutex:                reflect.TypeFor[law.TwoPhaseCommitOrRollback[any, any]](),
	lawid.TwoPhaseRollbackAfterCommit:  reflect.TypeFor[law.TwoPhaseNoRollbackAfterCommit[any, any]](),
	lawid.UpdaterReplaces:              reflect.TypeFor[law.UpdaterReplaces[any, any, string]](),
	lawid.UpserterIdempotent:           reflect.TypeFor[law.UpserterIdempotent[any, any, string]](),
	lawid.ValidTransition:              reflect.TypeFor[law.ValidTransition[any, any, string]](),
	lawid.WatcherReturnsOnChange:       reflect.TypeFor[law.WatcherReturnsOnChange[any, any, string, any]](),
	lawid.Windowed:                     reflect.TypeFor[law.Windowed[any, string]](),
	lawid.WriteObservable:              reflect.TypeFor[law.WriteObservable[any, any, string]](),
	lawid.WritesFollowReads:            reflect.TypeFor[law.WritesFollowReads[any, string]](),
	lawid.XSSSafe:                      reflect.TypeFor[law.XSSSafe[any]](),
}

// modeSwitched are the identifiers one law reports depending on a field's
// value rather than from its type alone.
//
// [law.PublisherDeliveryBound] switches on the delivery mode it was given, so
// a zero value reports the unrefined identifier and the other three are
// unreachable without setting the field. The gate sets it, because an
// identifier nothing can produce is exactly what it exists to notice.
//
//nolint:gochecknoglobals // a lookup table, read-only after init
var modeSwitched = map[string]law.DeliveryMode{
	// At-least-once is the zero value, so the unrefined identity is not what
	// an unset field reports — it is the default arm, reachable only for a
	// mode outside the declared three. Probing it with one is the only way to
	// observe the identity this census excuses as unreachable.
	lawid.PublisherDelivery:    law.DeliveryMode(-1),
	lawid.PublisherAtLeastOnce: law.DeliveryAtLeastOnce,
	lawid.PublisherAtMostOnce:  law.DeliveryAtMostOnce,
	lawid.PublisherExactlyOnce: law.DeliveryExactlyOnce,
}

// ReportedID returns the identifier an instance of that law type reports.
//
// Constructed through reflection rather than by calling a hand-written
// constructor per law: eighty-three closures would be a second transcription
// of the catalogue, and a gate that can drift from what it checks is not a
// gate. A pointer receiver is handled by addressing the value, which is what
// the stateful laws declare.
func ReportedID(id string, t reflect.Type) (string, bool) {
	v := reflect.New(t)
	if mode, switched := modeSwitched[id]; switched {
		v.Elem().FieldByName("Mode").Set(reflect.ValueOf(mode))
	}

	// Addressed rather than dereferenced: a pointer's method set contains
	// both the value-receiver and pointer-receiver forms, so one lookup
	// reaches the stateful laws that declare theirs on the pointer as well as
	// the plain ones that do not.
	m := v.MethodByName("ID")
	if !m.IsValid() {
		return "", false
	}
	return m.Call(nil)[0].String(), true
}

// timeawareAnchor keeps the import honest: the clock-shaped laws live in their
// own package, and dropping their entries from the table would otherwise leave
// an unused import rather than a failing test.
var _ = timeaware.Barrier{}

// UnreachableLaws records every shipped law no selection rule names, and what
// would have to exist first.
//
// The census docs/adr/0017 asks for, one level below classifications. A law
// nothing selects ships, is tested in `engine`, and can never run from a
// declaration — which is a defect when a rule was simply not written and a
// boundary when nothing can express the selection. Recording the second is
// what makes the first visible.
//
//nolint:gochecknoglobals // a lookup table, read-only after init
var UnreachableLaws = map[string]string{
	lawid.PublisherDelivery: "the identity the delivery law falls back to when " +
		"the declared mode is one it does not recognise; every rule selects a " +
		"known mode, so nothing reaches it",
	lawid.HashChainIntegrityErr: "selected by a stored-error accessor sitting on " +
		"a chain interface, which is a fact about the interface rather than " +
		"about one method — the selector reads a single method's stamps",
}
