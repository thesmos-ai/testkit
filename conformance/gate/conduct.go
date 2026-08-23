// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

import "go.thesmos.sh/testkit/core/lawid"

// Conduct is how a law's Check treats the shared (sut, ref) pair the runner
// hands it — the contract [engine/model/law.Law] states, recorded per law so
// it can be enforced rather than remembered.
//
// The first four conducts keep the pair synchronized and may be bound by the
// generator. The last two do not, and a law carrying one must not gain an
// instantiation row until it is fixed or the runner grows an isolation
// mechanism — which is exactly what the census test holds.
type Conduct string

// The conduct vocabulary.
const (
	// ConductObservational reads only.
	ConductObservational Conduct = "observational"

	// ConductMirrored lands every accepted mutation on both sides.
	ConductMirrored Conduct = "mirrored"

	// ConductSelfCleaning mutates and restores within one Check — an acquire
	// released, a put deleted, a transaction rolled back.
	ConductSelfCleaning Conduct = "self-cleaning"

	// ConductIsolated builds its own subjects through a Factory field and
	// never touches the pair.
	ConductIsolated Conduct = "isolated"

	// ConductNeedsMirror mutates the subject without mirroring — sound alone,
	// unsound interleaved. The fix is mechanical (the mirror helper) and owed
	// before the law's instantiation row lands.
	ConductNeedsMirror Conduct = "needs-mirror"

	// ConductNeedsIsolation corrupts or kills the subject to make its
	// observation — tampering its state, closing it, poisoning it. No mirror
	// repairs that; the law needs a subject of its own, which the runner does
	// not yet offer.
	ConductNeedsIsolation Conduct = "needs-isolation"
)

// Sound reports whether the conduct keeps a shared pair synchronized.
func (c Conduct) Sound() bool {
	switch c {
	case ConductObservational, ConductMirrored, ConductSelfCleaning, ConductIsolated:
		return true
	case ConductNeedsMirror, ConductNeedsIsolation:
		return false
	}
	return false
}

// LawConduct classifies every law in the catalogue.
//
// The classification is by reading each Check — there is nothing mechanical
// that can see a mutation through a closure — which is why it lives here,
// where a test holds it total over the vocabulary and the binding column to
// its verdicts. A law added without a row fails the census by name.
//
//nolint:gochecknoglobals // a census table, read-only, test-facing.
var LawConduct = map[string]Conduct{
	lawid.AggregatorBounded:        ConductObservational,
	lawid.AppendOnlyGrows:          ConductObservational,
	lawid.AppendOnlyNoDrops:        ConductObservational,
	lawid.Cacheable:                ConductObservational,
	lawid.CausalOrdering:           ConductObservational,
	lawid.CountEqualsReference:     ConductObservational,
	lawid.DeadlineRespecting:       ConductObservational,
	lawid.DefaultOnError:           ConductObservational,
	lawid.DeleteReturnsNotFound:    ConductObservational,
	lawid.HashChainIntegrityErr:    ConductObservational,
	lawid.HashChainIntegrityVerify: ConductObservational,
	lawid.LifecycleRespectsContext: ConductObservational,
	lawid.LossyRoundtrip:           ConductObservational,
	lawid.MonotonicNonDecreasing:   ConductObservational,
	lawid.MonotonicReads:           ConductObservational,
	lawid.MonotonicWrites:          ConductObservational,
	lawid.PaginatorNoDuplicates:    ConductObservational,
	lawid.PaginatorResumable:       ConductObservational,
	lawid.PoisonIdempotentRead:     ConductObservational,
	lawid.PoolBalanced:             ConductObservational,
	lawid.PoolLeakFree:             ConductObservational,
	lawid.PredicateConsistent:      ConductObservational,
	lawid.PureDeterministic:        ConductObservational,
	lawid.ReadAfterWrite:           ConductObservational,
	lawid.ReadYourWrites:           ConductObservational,
	lawid.ReplayCausalOrdering:     ConductObservational,
	lawid.ReplayDeterministic:      ConductObservational,
	lawid.Roundtrip:                ConductObservational,
	lawid.SnapshotIsolationG0:      ConductObservational,
	lawid.SnapshotIsolationG1:      ConductObservational,
	lawid.SnapshotIsolationG2:      ConductObservational,
	// Mirrored though it never writes: the sticky subject pins on read, so
	// the observation is a mutation and must land on both sides.
	lawid.Sticky:             ConductMirrored,
	lawid.StreamCompletion:   ConductObservational,
	lawid.StreamNoDuplicates: ConductObservational,
	lawid.StreamOverMatch:    ConductObservational,
	lawid.StreamPermutation:  ConductObservational,
	lawid.StreamReentrant:    ConductObservational,
	lawid.StreamStableOrder:  ConductObservational,
	lawid.TotalOver:          ConductObservational,
	lawid.WritesFollowReads:  ConductObservational,
	lawid.XSSSafe:            ConductObservational,

	lawid.AppenderMonotonicOffsets: ConductMirrored,
	lawid.AtomicWrite:              ConductMirrored,
	lawid.CASAtomicOneWinner:       ConductMirrored,
	lawid.Conservative:             ConductMirrored,
	lawid.IdempotentWrite:          ConductMirrored,
	lawid.InjectionSafe:            ConductMirrored,
	lawid.PersisterRetrievable:     ConductMirrored,
	lawid.PointInTime:              ConductMirrored,
	lawid.PublisherAtLeastOnce:     ConductMirrored,
	lawid.PublisherAtMostOnce:      ConductMirrored,
	lawid.PublisherDelivers:        ConductMirrored,
	lawid.PublisherDelivery:        ConductMirrored,
	lawid.PublisherExactlyOnce:     ConductMirrored,
	lawid.SagaFullCompensation:     ConductMirrored,
	lawid.SingleflightCoalesces:    ConductMirrored,
	lawid.UpdaterReplaces:          ConductMirrored,
	lawid.UpserterIdempotent:       ConductMirrored,
	lawid.ValidTransition:          ConductMirrored,
	lawid.WatcherReturnsOnChange:   ConductMirrored,
	lawid.Windowed:                 ConductMirrored,
	lawid.WriteObservable:          ConductMirrored,

	lawid.LeakFree:                     ConductSelfCleaning,
	lawid.LeaseDoubleAcquireBlocks:     ConductSelfCleaning,
	lawid.LeaseReleasedOnCancel:        ConductSelfCleaning,
	lawid.StreamReflectsMutations:      ConductSelfCleaning,
	lawid.TransactionNoMidTxVisibility: ConductSelfCleaning,
	lawid.TransactionRollback:          ConductSelfCleaning,
	lawid.TwoPhaseMutex:                ConductSelfCleaning,
	lawid.TwoPhaseRollbackAfterCommit:  ConductSelfCleaning,

	lawid.Associative:         ConductIsolated,
	lawid.CRDTMerge:           ConductIsolated,
	lawid.CommutativeWrite:    ConductIsolated,
	lawid.EventualConvergence: ConductIsolated,
	lawid.PoisonNilOnFresh:    ConductIsolated,

	// The corruption laws carry the [law.Isolated] marker: the runner hands
	// each a throwaway pair once per iteration, and the shared pair never
	// meets one. The census test holds marker and verdict to each other.
	lawid.CursorCloseIdempotent: ConductIsolated,
	lawid.CursorNextAfterClose:  ConductIsolated,
	lawid.IdempotentLifecycle:   ConductIsolated,
	lawid.LifecycleAfterClose:   ConductIsolated,
	lawid.PoisonConsistent:      ConductIsolated,
	lawid.TamperEvident:         ConductIsolated,

	// The clock laws: sound exactly where subject and reference age under
	// one [clock.TestClock] — the twins the generated ModelClocked option
	// builds share it, so the pair diverges and converges together. The
	// TTL check even resynchronizes its own unmirrored put by expiring it.
	lawid.TTLExpiry:                  ConductSelfCleaning,
	lawid.ScheduledFiresAfterAdvance: ConductMirrored,

	// Reads only, and the advance it makes is to the shared clock both
	// twins already age under — so it leaves the pair exactly as it found
	// it, in the same state as each other.
	lawid.TimeawareMoves: ConductObservational,
}
