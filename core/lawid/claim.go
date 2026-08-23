// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package lawid

import (
	"fmt"
	"slices"
	"strings"
)

// Claim is one law's human claim: the sentence a lock row, a report,
// and a skipped subtest speak for it. It lives beside the identifier
// for the reason the identifier lives here at all — the generator
// writes the sentence into manifests and the engine reports outcomes
// under it, and a wording spelled in two modules drifts where no
// compiler can see.
//
// Parametric where the wording names something only a declaration
// knows. The placeholders are the closed vocabulary below; a claim
// interpolating anything else fails this package's own census, and
// [Claim.Fill] refuses a sentence left half-filled rather than
// publishing a bracket into a manifest.
type Claim string

// The placeholder vocabulary a claim may interpolate. Each names a
// fact the selecting declaration carries; the consumer filling a
// claim resolves them from its own stamps, and over-supplying is
// free — an absent placeholder ignores its pair.
const (
	// PlaceClose is the close method the selecting declaration names:
	// the after-close teardown, or a produced handle's release.
	PlaceClose = "{close}"
	// PlaceNext is the produced handle's reader.
	PlaceNext = "{next}"
	// PlaceProduced is the contract's own word for the produced
	// handle.
	PlaceProduced = "{produced}"
	// PlaceSubject is the subject interface's token.
	PlaceSubject = "{subject}"
)

// Placeholders enumerates the vocabulary, for the census that holds
// every claim's tokens to it.
func Placeholders() []string {
	return []string{PlaceClose, PlaceNext, PlaceProduced, PlaceSubject}
}

// Fill interpolates placeholder/value pairs and refuses a claim left
// unfilled: a leftover bracket in a manifest row would read as prose
// and diff forever after.
func (c Claim) Fill(pairs ...string) (string, error) {
	if len(pairs)%2 != 0 {
		return "", fmt.Errorf("lawid: Fill takes placeholder/value pairs, got %d values", len(pairs))
	}
	out := string(c)
	for i := 0; i < len(pairs); i += 2 {
		out = strings.ReplaceAll(out, pairs[i], pairs[i+1])
	}
	for _, p := range Placeholders() {
		if strings.Contains(out, p) {
			return "", fmt.Errorf("lawid: claim %q left %s unfilled", string(c), p)
		}
	}
	return out, nil
}

// ClaimOf returns the law's claim, false for an identifier this
// package does not word yet — the consumer's signal to refuse the
// row by name rather than invent a sentence. Wordings accrete toward
// the full registry under the conformance corpus, which surfaces
// every unworded law the day a fixture stamps its classification.
func ClaimOf(id string) (Claim, bool) {
	w, ok := worded()[id]
	return w.claim, ok
}

// AccessorOf returns the law's spelling in a generated check index —
// the method a consumer calls to name the check, as in
// `ix.Model.CloseIdempotent()`.
//
// Here rather than in the emitter because it is a fact about the law,
// and the law's facts have one home. The identifier cannot be derived
// from the constant's name either: the contract word that qualifies a
// law globally (Cursor, Lease) is redundant inside an index already
// scoped to one interface, and which part is redundant is a judgment
// per law rather than a prefix rule.
func AccessorOf(id string) (string, bool) {
	w, ok := worded()[id]
	return w.accessor, ok
}

// IsLaw reports whether the identifier is one this package registers.
//
// The question a caller holding a bare identity segment has to ask
// before deciding what an unworded one means: a registered law with no
// wording is a gap to refuse by name, while a segment that was never a
// law is somebody else's vocabulary and none of this package's
// business. Prefix-sniffing for AUTO- cannot tell them apart — the
// runtime spells segments that way too.
func IsLaw(id string) bool { return slices.Contains(All(), id) }

// ConstOf returns the Go identifier this package declares the law
// under, for emitted code that must name the law rather than repeat
// its text — `lawid.CursorCloseIdempotent`, not the AUTO- string.
//
// Carried as data because Go cannot ask a constant for its own name,
// and a generated file spelling the literal would be the one place
// the identifier is not the single home.
func ConstOf(id string) (string, bool) {
	w, ok := worded()[id]
	return w.constant, ok
}

// law is what this package knows about one identifier.
//
// One row per law rather than a map per fact: a claim and an accessor
// added in separate places is a law worded in one and unspellable in
// the other, which no census catches until a fixture stamps it.
type law struct {
	claim Claim

	// accessor is the law's spelling in a generated index; constant
	// is the identifier this package declares it under.
	accessor, constant string
}

// worded is the laws the proof-of-concept corpus pinned. The claim
// spellings are its manifests', verbatim; the accessors are its
// generated indexes'.
func worded() map[string]law {
	return map[string]law{
		TTLExpiry: {
			"an entry stops being readable once its lifetime has run out",
			"Expiry", "TTLExpiry",
		},

		// The rest of the registry, each worded from the law's own body
		// rather than from its name. The accessor is the identifier this
		// package declares the law under: an index is scoped to one
		// interface, so a shorter word would often read better, but
		// shortening invites two laws meeting under one name — a
		// paginator and a stream both wanting NoDuplicates — and a
		// collision there is a generated file that does not compile.
		Associative: {
			"applying the same values in either order leaves the same observation",
			"Associative", "Associative",
		},
		CommutativeWrite: {
			"two writes in either order leave the same observation",
			"CommutativeWrite", "CommutativeWrite",
		},
		Conservative: {
			"a mutation leaves the sum of the conserved field unchanged",
			"Conservative", "Conservative",
		},
		Windowed: {
			"the window reflects an event until the clock passes the window, and not after",
			"Windowed", "Windowed",
		},
		CausalOrdering: {
			"the run's operations are consistent with one causal order",
			"CausalOrdering", "CausalOrdering",
		},
		HashChainIntegrityVerify: {
			"the chain verifies its own integrity after every operation",
			"HashChainIntegrityVerify", "HashChainIntegrityVerify",
		},
		ReplayCausalOrdering: {
			"every replayed entry appears after the entries it depends on",
			"ReplayCausalOrdering", "ReplayCausalOrdering",
		},
		PoolBalanced: {
			"the pool's outstanding count never goes negative and returns to zero at rest",
			"PoolBalanced", "PoolBalanced",
		},
		PoolLeakFree: {
			"with no cycle outstanding, the pool reports itself balanced",
			"PoolLeakFree", "PoolLeakFree",
		},
		SagaFullCompensation: {
			"a saga whose step fails leaves the state it started from",
			"SagaFullCompensation", "SagaFullCompensation",
		},
		TwoPhaseMutex: {
			"once a transaction commits or rolls back, the other reports it closed",
			"TwoPhaseMutex", "TwoPhaseMutex",
		},
		TwoPhaseRollbackAfterCommit: {
			"rolling back a committed transaction reports it closed",
			"TwoPhaseRollbackAfterCommit", "TwoPhaseRollbackAfterCommit",
		},
		CASAtomicOneWinner: {
			"two concurrent writes from one version leave exactly one winner",
			"CASAtomicOneWinner", "CASAtomicOneWinner",
		},
		PaginatorNoDuplicates: {
			"a full walk emits every element at most once",
			"PaginatorNoDuplicates", "PaginatorNoDuplicates",
		},
		PaginatorResumable: {
			"resuming from a cursor yields exactly the suffix the full walk would have",
			"PaginatorResumable", "PaginatorResumable",
		},
		PersisterRetrievable: {
			"what a save answers is what a read finds",
			"PersisterRetrievable", "PersisterRetrievable",
		},
		SingleflightCoalesces: {
			"concurrent calls for one key compute at most once",
			"SingleflightCoalesces", "SingleflightCoalesces",
		},
		TransactionNoMidTxVisibility: {
			"a write inside an open transaction is invisible until it commits",
			"TransactionNoMidTxVisibility", "TransactionNoMidTxVisibility",
		},
		TransactionRollback: {
			"a transaction whose body errs leaves none of its writes visible",
			"TransactionRollback", "TransactionRollback",
		},
		UpdaterReplaces: {
			"the second update of a key is the one that reads back",
			"UpdaterReplaces", "UpdaterReplaces",
		},
		UpserterIdempotent: {
			"upserting the same value twice leaves what the first upsert left",
			"UpserterIdempotent", "UpserterIdempotent",
		},
		WatcherReturnsOnChange: {
			"a watch established before a change observes it",
			"WatcherReturnsOnChange", "WatcherReturnsOnChange",
		},
		PublisherDelivery: {
			"every subscriber's delivery count is within the declared bound",
			"PublisherDelivery", "PublisherDelivery",
		},
		PublisherAtMostOnce: {
			"no subscriber receives a published message more than once",
			"PublisherAtMostOnce", "PublisherAtMostOnce",
		},
		PublisherExactlyOnce: {
			"every subscriber receives a published message exactly once",
			"PublisherExactlyOnce", "PublisherExactlyOnce",
		},
		EventualConvergence: {
			"replicas given disjoint writes converge once they exchange state",
			"EventualConvergence", "EventualConvergence",
		},
		CRDTMerge: {
			"two replicas that merge each other's state end up observably the same",
			"CRDTMerge", "CRDTMerge",
		},
		DeleteReturnsNotFound: {
			"where the reference reports the miss sentinel, so does the subject",
			"DeleteReturnsNotFound", "DeleteReturnsNotFound",
		},
		IdempotentLifecycle: {
			"calling the lifecycle method twice leaves what calling it once left",
			"IdempotentLifecycle", "IdempotentLifecycle",
		},
		LeakFree: {
			"repeated open-and-close cycles leak no goroutines",
			"LeakFree", "LeakFree",
		},
		PoisonIdempotentRead: {
			"two consecutive reads of the poison answer the same thing",
			"PoisonIdempotentRead", "PoisonIdempotentRead",
		},
		PoisonNilOnFresh: {
			"a freshly built subject reports no poison",
			"PoisonNilOnFresh", "PoisonNilOnFresh",
		},
		MonotonicReads: {
			"a client's successive reads of a key never go backwards",
			"MonotonicReads", "MonotonicReads",
		},
		MonotonicWrites: {
			"a client's successive writes to a key are stamped in issue order",
			"MonotonicWrites", "MonotonicWrites",
		},
		ReadYourWrites: {
			"a client reading after its own write sees that write or a later one",
			"ReadYourWrites", "ReadYourWrites",
		},
		WritesFollowReads: {
			"a client's write is stamped no older than what it has read",
			"WritesFollowReads", "WritesFollowReads",
		},
		PredicateConsistent: {
			"the predicate answers the same on repeated calls",
			"PredicateConsistent", "PredicateConsistent",
		},
		PureDeterministic: {
			"the call answers the same on repeated calls",
			"PureDeterministic", "PureDeterministic",
		},
		DefaultOnError: {
			"a read that errs answers the declared default",
			"DefaultOnError", "DefaultOnError",
		},
		MonotonicNonDecreasing: {
			"the count never decreases across calls",
			"MonotonicNonDecreasing", "MonotonicNonDecreasing",
		},
		PointInTime: {
			"a read at a time answers what was committed at or before it",
			"PointInTime", "PointInTime",
		},
		Sticky: {
			"once a key resolves, it keeps resolving to the same value",
			"Sticky", "Sticky",
		},
		SnapshotIsolationG0: {
			"the recorded transaction history has no write cycles",
			"SnapshotIsolationG0", "SnapshotIsolationG0",
		},
		SnapshotIsolationG1: {
			"the recorded transaction history has no aborted, intermediate or cyclic reads",
			"SnapshotIsolationG1", "SnapshotIsolationG1",
		},
		SnapshotIsolationG2: {
			"the recorded transaction history has no write skew",
			"SnapshotIsolationG2", "SnapshotIsolationG2",
		},
		LossyRoundtrip: {
			"encoding what a decode produced gives the same encoding back",
			"LossyRoundtrip", "LossyRoundtrip",
		},
		Roundtrip: {
			"decoding what was encoded gives back what went in",
			"Roundtrip", "Roundtrip",
		},
		TotalOver: {
			"every input in the declared domain answers something other than the zero value",
			"TotalOver", "TotalOver",
		},
		StreamCompletion: {
			"the stream terminates within the declared limit",
			"StreamCompletion", "StreamCompletion",
		},
		StreamNoDuplicates: {
			"each drain emits every element at most once",
			"StreamNoDuplicates", "StreamNoDuplicates",
		},
		StreamOverMatch: {
			"the drain holds everything that was written, and may hold more",
			"StreamOverMatch", "StreamOverMatch",
		},
		StreamPermutation: {
			"the drain is a permutation of what was written",
			"StreamPermutation", "StreamPermutation",
		},
		StreamReentrant: {
			"draining twice yields the same items",
			"StreamReentrant", "StreamReentrant",
		},
		StreamReflectsMutations: {
			"a written item appears in the next drain, and a deleted one stops appearing",
			"StreamReflectsMutations", "StreamReflectsMutations",
		},
		StreamStableOrder: {
			"the drain's order follows the declared one",
			"StreamStableOrder", "StreamStableOrder",
		},
		AtomicWrite: {
			"a write that errs leaves the observable state unchanged",
			"AtomicWrite", "AtomicWrite",
		},
		InjectionSafe: {
			"a payload of metacharacters round-trips as literal data",
			"InjectionSafe", "InjectionSafe",
		},
		TamperEvident: {
			"tampering with stored data is detectable afterwards",
			"TamperEvident", "TamperEvident",
		},
		ValidTransition: {
			"a write only moves the field along a declared transition",
			"ValidTransition", "ValidTransition",
		},
		XSSSafe: {
			"no script-capable markup survives rendering",
			"XSSSafe", "XSSSafe",
		},

		// The writer pair. Both are about what a write leaves behind,
		// and both say so through the reader that observes it — neither
		// claims anything about a write nothing reads back.
		WriteObservable: {
			"a written value is readable under the key it was written with",
			"WriteObservable", "WriteObservable",
		},
		IdempotentWrite: {
			"writing the same value twice leaves what writing it once left",
			"IdempotentWrite", "IdempotentWrite",
		},

		// Worded for the cancelled context specifically, not for
		// contexts generally: the law calls with one already cancelled
		// and demands that error back, and says nothing about a
		// deadline or about cancellation part-way through.
		LifecycleRespectsContext: {
			"a lifecycle call handed an already-cancelled context reports the cancellation",
			"RespectsContext", "LifecycleRespectsContext",
		},

		// Within one iteration, which is the whole claim: a cache that
		// forgets between iterations is not what this catches.
		Cacheable: {
			"repeated reads of one key answer the same value",
			"Cacheable", "Cacheable",
		},
		AggregatorBounded: {
			"the count stays inside the declared bound",
			"Bounded", "AggregatorBounded",
		},

		// The chain trio. Grows and NoDrops sound alike and are not:
		// one is about the order of what is there, the other about
		// whether it is there at all.
		AppendOnlyGrows: {
			"every replay extends the last one rather than rewriting it",
			"Grows", "AppendOnlyGrows",
		},
		AppendOnlyNoDrops: {
			"every entry the chain acknowledged is in its replay",
			"NoDrops", "AppendOnlyNoDrops",
		},
		ReplayDeterministic: {
			"two replays of one chain answer the same entries",
			"ReplayDeterministic", "ReplayDeterministic",
		},

		// The two comparison laws. Both word what they actually do —
		// compare the subject against the reference — rather than the
		// property a reader might assume from the name: ReadAfterWrite
		// never writes, and neither says anything about a subject taken
		// on its own.
		ReadAfterWrite: {
			"every key reads the same on the subject as on the reference",
			"ReadsAgree", "ReadAfterWrite",
		},
		CountEqualsReference: {
			"the subject counts what the reference counts",
			"Counts", "CountEqualsReference",
		},
		// The publisher pair, worded as the proof-of-concept corpus
		// spelled them.
		PublisherDelivers: {
			"a message published after subscribers registered reaches every one of them",
			"Delivers", "PublisherDelivers",
		},
		PublisherAtLeastOnce: {
			"every subscriber's delivery count for a published message is one or more",
			"AtLeastOnce", "PublisherAtLeastOnce",
		},
		DeadlineRespecting: {
			"an operation given a deadline returns once that deadline fires",
			"Deadline", "DeadlineRespecting",
		},
		TimeawareMoves: {
			"what {subject} reports for a key changes when the clock does",
			"MovesWithTheClock", "TimeawareMoves",
		},
		ScheduledFiresAfterAdvance: {
			// At least, not exactly: the law shares its subject with an
			// action stream scheduling work of its own, and whatever of
			// that is pending fires inside the same advance. The wording
			// says what the law checks, which is the whole reason it is
			// written here rather than inferred from the identifier.
			"work scheduled for a time already passed has fired",
			"Scheduled", "ScheduledFiresAfterAdvance",
		},
		LifecycleAfterClose: {
			"once {close} has run, every method reports the closed sentinel",
			"AfterClose", "LifecycleAfterClose",
		},
		PoisonConsistent: {
			"once the {subject} reports it is closed, it keeps reporting it",
			"PoisonConsistent", "PoisonConsistent",
		},
		CursorCloseIdempotent: {
			"a second {close} on a {produced} changes nothing",
			"CloseIdempotent", "CursorCloseIdempotent",
		},
		CursorNextAfterClose: {
			"once a {produced} is closed, {next} reports the closed sentinel",
			"NextAfterClose", "CursorNextAfterClose",
		},
		AppenderMonotonicOffsets: {
			"offsets of successive appends strictly increase",
			"MonotonicOffsets", "AppenderMonotonicOffsets",
		},
		LeaseDoubleAcquireBlocks: {
			"a second acquire of a held key reports the held sentinel",
			"DoubleAcquire", "LeaseDoubleAcquireBlocks",
		},
		LeaseReleasedOnCancel: {
			"a held lease frees once its acquiring context is cancelled",
			"ReleasedOnCancel", "LeaseReleasedOnCancel",
		},
	}
}
