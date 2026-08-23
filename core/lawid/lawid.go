// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package lawid

// The identifiers, grouped by the file that implements them — [engine/model/law]
// throughout except for the clock-shaped block, which names its own package —
// so a reader diffing the two finds them in the same order.
//
// Every value carries the `AUTO-` prefix, which is what marks a law as derived
// from a declaration rather than written by a consumer. [TestEveryIDIsWellFormed]
// holds that, and holds every value distinct — a copy-pasted duplicate would
// give two laws one identity, and the second would silently replace the first
// in any map keyed by ID.
const (
	// aggregator.go — laws over a method that reports a total.
	AggregatorBounded = "AUTO-AGGREGATOR-BOUNDED"
	Associative       = "AUTO-ASSOCIATIVE"
	Conservative      = "AUTO-CONSERVATIVE"
	Windowed          = "AUTO-WINDOWED"

	// causal.go — happens-before over a multi-client trace.
	CausalOrdering = "AUTO-CAUSAL-ORDERING"

	// chain.go — laws over an append-and-replay pair.
	AppendOnlyGrows          = "AUTO-APPEND-ONLY-GROWS"
	AppendOnlyNoDrops        = "AUTO-APPEND-ONLY-NO-DROPS"
	HashChainIntegrityErr    = "AUTO-HASH-CHAIN-INTEGRITY-ERR"
	HashChainIntegrityVerify = "AUTO-HASH-CHAIN-INTEGRITY-VERIFY"
	ReplayCausalOrdering     = "AUTO-REPLAY-CAUSAL-ORDERING"
	ReplayDeterministic      = "AUTO-REPLAY-DETERMINISTIC"

	// composite.go — laws needing two or more members of one protocol.
	CursorCloseIdempotent       = "AUTO-CURSOR-CLOSE-IDEMPOTENT"
	CursorNextAfterClose        = "AUTO-CURSOR-NEXT-AFTER-CLOSE"
	PoolBalanced                = "AUTO-POOL-BALANCED"
	PoolLeakFree                = "AUTO-POOL-LEAK-FREE"
	SagaFullCompensation        = "AUTO-SAGA-FULL-COMPENSATION"
	TwoPhaseMutex               = "AUTO-TWO-PHASE-MUTEX"
	TwoPhaseRollbackAfterCommit = "AUTO-TWO-PHASE-ROLLBACK-AFTER-COMMIT"

	// contract.go — laws a named contract's roles select.
	AppenderMonotonicOffsets     = "AUTO-APPENDER-MONOTONIC-OFFSETS"
	CASAtomicOneWinner           = "AUTO-CAS-ATOMIC-ONE-WINNER"
	LeaseDoubleAcquireBlocks     = "AUTO-LEASE-DOUBLE-ACQUIRE-BLOCKS"
	LeaseReleasedOnCancel        = "AUTO-LEASE-RELEASED-ON-CANCEL"
	PaginatorNoDuplicates        = "AUTO-PAGINATOR-NO-DUPLICATES"
	PaginatorResumable           = "AUTO-PAGINATOR-RESUMABLE"
	PersisterRetrievable         = "AUTO-PERSISTER-RETRIEVABLE"
	SingleflightCoalesces        = "AUTO-SINGLEFLIGHT-COALESCES"
	TransactionNoMidTxVisibility = "AUTO-TRANSACTION-NO-MID-TX-VISIBILITY"
	TransactionRollback          = "AUTO-TRANSACTION-ROLLBACK"
	UpdaterReplaces              = "AUTO-UPDATER-REPLACES"
	UpserterIdempotent           = "AUTO-UPSERTER-IDEMPOTENT"
	WatcherReturnsOnChange       = "AUTO-WATCHER-RETURNS-ON-CHANGE"

	// The publisher family. One law type reports under four identifiers,
	// selected by the delivery mode the contract declares — so a rule
	// choosing between them reads `mode`, not the contract name alone.
	PublisherDelivers    = "AUTO-PUBLISHER-DELIVERS"
	PublisherDelivery    = "AUTO-PUBLISHER-DELIVERY"
	PublisherAtLeastOnce = "AUTO-PUBLISHER-AT-LEAST-ONCE"
	PublisherAtMostOnce  = "AUTO-PUBLISHER-AT-MOST-ONCE"
	PublisherExactlyOnce = "AUTO-PUBLISHER-EXACTLY-ONCE"

	// eventual.go — convergence across replicas.
	EventualConvergence = "AUTO-EVENTUAL-CONVERGENCE"

	// law.go — the base laws every keyed store owes.
	CountEqualsReference  = "AUTO-COUNT-EQUALS-REFERENCE"
	CRDTMerge             = "AUTO-CRDT-MERGE"
	DeleteReturnsNotFound = "AUTO-DELETE-RETURNS-NOT-FOUND"
	ReadAfterWrite        = "AUTO-READ-AFTER-WRITE"

	// lifecycle.go — laws over setup, teardown and the poisoned state after.
	IdempotentLifecycle      = "AUTO-IDEMPOTENT-LIFECYCLE"
	LeakFree                 = "AUTO-LEAK-FREE"
	LifecycleAfterClose      = "AUTO-LIFECYCLE-AFTER-CLOSE"
	LifecycleRespectsContext = "AUTO-LIFECYCLE-RESPECTS-CONTEXT"
	PoisonConsistent         = "AUTO-POISON-CONSISTENT"
	PoisonIdempotentRead     = "AUTO-POISON-IDEMPOTENT-READ"
	PoisonNilOnFresh         = "AUTO-POISON-NIL-ON-FRESH"

	// perclient.go — session guarantees, read from the per-iteration trace.
	MonotonicReads    = "AUTO-MONOTONIC-READS"
	MonotonicWrites   = "AUTO-MONOTONIC-WRITES"
	ReadYourWrites    = "AUTO-READ-YOUR-WRITES"
	WritesFollowReads = "AUTO-WRITES-FOLLOW-READS"

	// predicate.go, pure.go — self-consistency, needing no reference.
	PredicateConsistent = "AUTO-PREDICATE-CONSISTENT"
	PureDeterministic   = "AUTO-PURE-DETERMINISTIC"

	// reader.go — laws over a read that takes a key.
	Cacheable              = "AUTO-CACHEABLE"
	DefaultOnError         = "AUTO-DEFAULT-ON-ERROR"
	MonotonicNonDecreasing = "AUTO-MONOTONIC-NON-DECREASING"
	PointInTime            = "AUTO-POINT-IN-TIME"
	Sticky                 = "AUTO-STICKY"

	// snapshot.go — the isolation anomalies, over a transaction history.
	SnapshotIsolationG0 = "AUTO-SNAPSHOT-ISOLATION-G0"
	SnapshotIsolationG1 = "AUTO-SNAPSHOT-ISOLATION-G1"
	SnapshotIsolationG2 = "AUTO-SNAPSHOT-ISOLATION-G2"

	// stateless.go — laws about a function rather than a state machine.
	LossyRoundtrip = "AUTO-LOSSY-ROUNDTRIP"
	Roundtrip      = "AUTO-ROUNDTRIP"
	TotalOver      = "AUTO-TOTAL-OVER"

	// The clock-shaped laws, which live in [engine/model/timeaware] rather
	// than in `law` because each needs a controlled clock to advance before
	// it can observe anything. They implement the same interface and report
	// the same way, so they are identified here beside the rest.
	DeadlineRespecting         = "AUTO-DEADLINE-RESPECTING"
	ScheduledFiresAfterAdvance = "AUTO-SCHEDULED-FIRES-AFTER-ADVANCE"
	TimeawareMoves             = "AUTO-TIMEAWARE-MOVES"
	TTLExpiry                  = "AUTO-TTL-EXPIRY"

	// stream.go — laws over a method that yields many values.
	StreamCompletion        = "AUTO-STREAM-COMPLETION"
	StreamNoDuplicates      = "AUTO-STREAM-NO-DUPLICATES"
	StreamOverMatch         = "AUTO-STREAM-OVER-MATCH"
	StreamPermutation       = "AUTO-STREAM-PERMUTATION"
	StreamReentrant         = "AUTO-STREAM-REENTRANT"
	StreamReflectsMutations = "AUTO-STREAM-REFLECTS-MUTATIONS"
	StreamStableOrder       = "AUTO-STREAM-STABLE-ORDER"

	// writer.go — laws over a method that takes a value and stores it.
	AtomicWrite      = "AUTO-ATOMIC-WRITE"
	CommutativeWrite = "AUTO-COMMUTATIVE-WRITE"
	IdempotentWrite  = "AUTO-IDEMPOTENT-WRITE"
	InjectionSafe    = "AUTO-INJECTION-SAFE"
	TamperEvident    = "AUTO-TAMPER-EVIDENT"
	ValidTransition  = "AUTO-VALID-TRANSITION"
	WriteObservable  = "AUTO-WRITE-OBSERVABLE"
	XSSSafe          = "AUTO-XSS-SAFE"
)

// All returns every identifier this package declares, sorted.
//
// The census the gates count against. A law added to [engine/model/law] with no
// constant here is invisible to the model generator's selection rules, and the
// only way to notice is to hold this list up to the law package — which is what
// the conformance gate does with it.
//
// Completeness is proven there rather than here, for two reasons. This module
// may not import the `go` toolchain packages, so nothing local can read the
// declaration; and reflecting over the law types compares this list against
// what the catalogue actually reports, which is a stronger claim than
// comparing it against the constants beside it.
func All() []string {
	return []string{
		AggregatorBounded,
		AppendOnlyGrows,
		AppendOnlyNoDrops,
		AppenderMonotonicOffsets,
		Associative,
		AtomicWrite,
		Cacheable,
		CASAtomicOneWinner,
		CausalOrdering,
		CommutativeWrite,
		Conservative,
		CountEqualsReference,
		CRDTMerge,
		CursorCloseIdempotent,
		CursorNextAfterClose,
		DeadlineRespecting,
		DefaultOnError,
		DeleteReturnsNotFound,
		EventualConvergence,
		HashChainIntegrityErr,
		HashChainIntegrityVerify,
		IdempotentLifecycle,
		IdempotentWrite,
		InjectionSafe,
		LeakFree,
		LeaseDoubleAcquireBlocks,
		LeaseReleasedOnCancel,
		LifecycleAfterClose,
		LifecycleRespectsContext,
		LossyRoundtrip,
		MonotonicNonDecreasing,
		MonotonicReads,
		MonotonicWrites,
		PaginatorNoDuplicates,
		PaginatorResumable,
		PersisterRetrievable,
		PointInTime,
		PoisonConsistent,
		PoisonIdempotentRead,
		PoisonNilOnFresh,
		PoolBalanced,
		PoolLeakFree,
		PredicateConsistent,
		PublisherAtLeastOnce,
		PublisherAtMostOnce,
		PublisherDelivers,
		PublisherDelivery,
		PublisherExactlyOnce,
		PureDeterministic,
		ReadAfterWrite,
		ReadYourWrites,
		ReplayCausalOrdering,
		ReplayDeterministic,
		Roundtrip,
		SagaFullCompensation,
		ScheduledFiresAfterAdvance,
		SingleflightCoalesces,
		SnapshotIsolationG0,
		SnapshotIsolationG1,
		SnapshotIsolationG2,
		Sticky,
		StreamCompletion,
		StreamNoDuplicates,
		StreamOverMatch,
		StreamPermutation,
		StreamReentrant,
		StreamReflectsMutations,
		StreamStableOrder,
		TamperEvident,
		TimeawareMoves,
		TotalOver,
		TransactionNoMidTxVisibility,
		TransactionRollback,
		TTLExpiry,
		TwoPhaseMutex,
		TwoPhaseRollbackAfterCommit,
		UpdaterReplaces,
		UpserterIdempotent,
		ValidTransition,
		WatcherReturnsOnChange,
		Windowed,
		WriteObservable,
		WritesFollowReads,
		XSSSafe,
	}
}
