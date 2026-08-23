// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers

import "go.thesmos.sh/testkit/core/lawid"

// The classifications a rule needs, spelled once.
//
// Named rather than written inline because a classification is a string eidos
// owns, and a typo in one of eighty rule literals selects nothing, silently,
// for exactly one law. The conformance gate holds every one of these to the
// live registry, so a rename upstream fails a test rather than quietly
// unbinding a law.
const (
	// Detectors.
	shapeAggregator      = "aggregator"
	shapeBatchReader     = "batchreader"
	shapeCloser          = "closer"
	shapeCompositeWriter = "compositewriter"
	shapeLifecycle       = "lifecycle"
	shapeLookup          = "lookup"
	shapeMultiAggregator = "multiaggregator"
	shapeMultiArgWriter  = "multiargwriter"
	shapeMultiReader     = "multireader"
	shapeMutator         = "mutator"
	shapePointerReader   = "pointerreader"
	shapePoisonAccessor  = "poisonaccessor"
	shapePredicate       = "predicate"
	shapePure            = "pure"
	shapeReader          = "reader"
	shapeReaderNoError   = "readernoerror"
	shapeReaderWithBool  = "readerwithbool"
	shapeStreamConsumer  = "streamconsumer"
	shapeStreamReader    = "streamreader"
	shapeVoidLifecycle   = "voidlifecycle"
	shapeWriter          = "writer"
	shapeAnsweringWriter = "answeringwriter"

	// Mixins.
	mixinAccumulates       = "accumulates"
	mixinAssociative       = "associative"
	mixinAtomic            = "atomic"
	mixinBounded           = "bounded"
	mixinCacheable         = "cacheable"
	mixinCausal            = "causal"
	mixinCommutative       = "commutative"
	mixinConservative      = "conservative"
	mixinCRDTMerge         = "crdtmerge"
	mixinDefaultOnError    = "defaultonerror"
	mixinDeleteRemoves     = "deleteremoves"
	mixinEventually        = "eventually"
	mixinHooks             = "hooks"
	mixinIdempotent        = "idempotent"
	mixinIndexed           = "indexed"
	mixinInjectionSafe     = "injectionsafe"
	mixinLeakFree          = "leakfree"
	mixinLifecycleAfter    = "lifecycleafterclose"
	mixinMonotonic         = "monotonic"
	mixinMonotonicReads    = "monotonicreads"
	mixinMonotonicWrites   = "monotonicwrites"
	mixinOverMatch         = "overmatch"
	mixinNoDuplicates      = "noduplicates"
	mixinOrderAfter        = "orderafter"
	mixinPartition         = "partition"
	mixinPermutation       = "permutation"
	mixinPointInTime       = "pointintime"
	mixinPoisonable        = "poisonable"
	mixinReadAfterWrite    = "readafterwrite"
	mixinReadYourWrites    = "readyourwrites"
	mixinSample            = "sample"
	mixinScheduled         = "scheduled"
	mixinSerializable      = "serializable"
	mixinSideEffect        = "sideeffect"
	mixinSnapshotIsolation = "snapshotisolation"
	mixinStableOrder       = "stableorder"
	mixinSticky            = "sticky"
	mixinStreamReflects    = "streamreflectsmutations"
	mixinTamperEvident     = "tamperevident"
	mixinValidates         = "validates"
	mixinWrappedVia        = "wrappedvia"
	mixinTimeaware         = "timeaware"
	mixinTimeout           = "timeout"
	mixinTotal             = "total"
	mixinTTL               = "ttl"
	mixinWindowed          = "windowed"
	mixinWritesFollowReads = "writesfollowreads"
	mixinXSSSafe           = "xsssafe"

	// Contracts.
	contractAppender     = "appender"
	contractBatchWriter  = "batch-writer"
	contractCache        = "cache"
	contractCAS          = "cas"
	contractChain        = "chain"
	contractCodec        = "codec"
	contractCursor       = "cursor"
	contractLease        = "lease"
	contractPagination   = "pagination"
	contractPersister    = "persister"
	contractPool         = "pool"
	contractPublisher    = "publisher"
	contractSaga         = "saga"
	contractSingleflight = "singleflight"
	contractTransaction  = "transaction"
	contractTx           = "tx"
	contractUpdater      = "updater"
	contractUpserter     = "upserter"
	contractWatcher      = "watcher"
	contractWorkflow     = "workflow"
)

// The shared generator pools, so a law and the actions that feed it draw from
// one key space. Two laws naming different pools for the same keys is the
// defect this vocabulary exists to make unwritable: readers stop revisiting
// what writers wrote, and every comparison passes over a history with no
// conflicts in it.
const (
	genKeys     = "keys"
	genValues   = "values"
	genInputs   = "inputs"
	genMessages = "messages"
	genOffsets  = "offsets"
	genPayloads = "payloads"

	// genReadback is a law-declared pool at the observed reader's answer —
	// for a law whose writes travel through a door, so no role input names
	// the domain and only the read-back says what the store holds.
	genReadback = "readback"
)

// The parameter stamps a rule reads, in the annotator's own spelling.
//
// Constants because a stamp key is one thing that appears in several rules,
// unlike a law's field name, which is several things that happen to share a
// spelling. A typo here selects nothing and reports nothing; the gate holds
// each to the live registry for exactly that reason.
const (
	paramBatchMode           = "shape.contract.batch-writer.param.mode"
	ParamBoundedLimit        = "shape.mixin.bounded.limit"
	paramBoundedMin          = "shape.mixin.bounded.min"
	paramCASMismatch         = "shape.contract.cas.param.mismatch"
	paramCursorSentinel      = "shape.contract.cursor.param.sentinel"
	paramDeleteSentinel      = "shape.mixin.deleteremoves.sentinel"
	paramLeaseHeld           = "shape.contract.lease.param.held"
	paramLeaseTimeout        = "shape.contract.lease.param.timeout"
	paramLifecycleClose      = "shape.mixin.lifecycleafterclose.close"
	paramLifecycleSentinel   = "shape.mixin.lifecycleafterclose.sentinel"
	paramTimeoutDuration     = "shape.mixin.timeout.duration"
	paramTransactionNotFound = "shape.contract.transaction.param.notfound"
	paramTTLDuration         = "shape.mixin.ttl.duration"
	paramTTLNotFound         = "shape.mixin.ttl.notfound"
	paramTxClosed            = "shape.contract.tx.param.closed"
	paramWindowedWindow      = "shape.mixin.windowed.window"
	paramCodecFidelity       = "shape.contract.codec.param.fidelity"
	ParamPublisherMode       = "shape.contract.publisher.param.mode"
	paramWorkflowTransitions = "shape.contract.workflow.param.transitions"
)

// The options a consumer fills where nothing derives the field.
//
// Named where more than one law waits on the same one, so the header a
// consumer reads and the option they call agree. Most are waiting on eidos
// rather than on them — see the rule that names each.
const (
	optClosed   = "closed"
	optNotFound = "notfound"
	optDrain    = "drain"
)

// The law field names more than one rule fills.
//
// Each names one concept wherever it appears — Values is always the value
// pool, Read is always the observation the claim compares against — so the
// constant asserts a real unity rather than a coincidence of spelling. A field
// unique to one law stays a literal beside it, where a reader can see it
// without a lookup.
const (
	fieldAdvance    = "Advance"
	fieldCall       = "Call"
	fieldClose      = "Close"
	fieldCount      = "Count"
	fieldDrain      = "Drain"
	fieldFactory    = "Factory"
	fieldHash       = "Hash"
	fieldKeyOf      = "KeyOf"
	fieldKeys       = "Keys"
	fieldPartitions = "Partitions"
	fieldPut        = "Put"
	fieldRead       = "Read"
	fieldReplay     = "Replay"
	fieldHistory    = "History"
	fieldSentinel   = "Sentinel"
	fieldValues     = "Values"
	fieldWrite      = "Write"
)

// The role vocabulary [Field.From] uses.
const (
	roleSelf = "self"

	familyReader     = "family.reader"
	familyWriter     = "family.writer"
	familyAggregator = "family.aggregator"

	// familyKeyedWriter is a write at both pools — `(ctx, K, V) error` — for
	// a law whose claim names the key it wrote as well as the value. The
	// plain writer family cannot stand in: a law that reads back what it
	// wrote has to have chosen the key, and a one-argument writer decides
	// that for itself.
	familyKeyedWriter = "family.keyedwriter"

	// familyHandleWriter is a write threading an open handle — `(ctx, H, K,
	// V) error` — for a claim about what a scope stages before it settles.
	// A contract declaring begin, settle and observe has no word for the
	// staging in between, and a claim about what staging does needs the
	// staging on the interface rather than reached past it.
	familyHandleWriter = "family.handlewriter"

	// familyCell is a nullary read of a single-slot subject — `(ctx) (V,
	// error)` — the observation a cell-shaped law compares against, which
	// the keyed reader family cannot stand in for.
	familyCell = "family.cell"
)

// The handles the generated file constructs and shares.
const (
	handleKeyOf = "key-projection"
	// handleReferenceMiss is the identity the derived reference reports
	// for a key it does not hold — the minted var, or the declaration's
	// own sentinel where one is stamped and the reference was built with
	// it.
	handleReferenceMiss = "reference-miss"
	handleFactory       = "subject-factory"
	handleHistory       = "history"
	handleWriteLog      = "write-log"
	handleClock         = "clock"
	handleIdentity      = "identity-hash"
	handleOrder         = "natural-order"
	handleClassify      = "trace-classifier"

	// handleVersionStamp is the version-coherent draw: an attempt stamped
	// at the cell's current version, which is what makes "exactly one
	// winner" a theorem — two attempts at a stale version are two
	// mismatches and no winner.
	handleVersionStamp = "version-stamp"

	// handleObserve is the composed whole-state observation: the batch read
	// where the interface streams its state, the aggregate where it counts
	// it, the fixture-keyed read where it stores it. One derivation, because
	// every law comparing state before and after a mutation must observe the
	// same way or two laws would disagree about what "the state" is.
	handleObserve = "observation"
)

// observed is the read-back every write-family law compares against.
//
// One spelling, because the claim is always the same one — "the write is
// visible" — and a law that named its own reader would let two rules disagree
// about which method observes the same subject.
func observed(name string) Field {
	return Field{Name: name, Kind: KindRole, From: familyReader}
}

// observation is the whole-state observation a before/after comparison law
// reads — a handle the generated file composes from whatever the interface
// offers, where [observed]'s keyed read demands a key the law has no way to
// choose. Every law spells the field Observe, which is the unity the shared
// derivation asserts.
func observation() Field {
	return Field{Name: "Observe", Kind: KindHandle, From: handleObserve}
}

// writeObservable is the manifest three writer shapes share.
//
// They differ in arity and in nothing else: a composite writer takes the key
// beside the value and a multi-arg writer takes several, but the property is
// one property, so the manifest is one manifest.
func writeObservable(shape string) Rule {
	return Rule{
		Law:   lawid.WriteObservable,
		Needs: []string{shape},
		Fields: []Field{
			{Name: fieldWrite, Kind: KindRole, From: roleSelf},
			observed(fieldRead),
			{Name: fieldValues, Kind: KindGenerator, From: genValues},
			{Name: fieldKeyOf, Kind: KindHandle, From: handleKeyOf},
		},
	}
}

// countEquals is the manifest both aggregator shapes share.
func countEquals(shape string) Rule {
	return Rule{
		Law:    lawid.CountEqualsReference,
		Needs:  []string{shape},
		Fields: []Field{{Name: fieldCount, Kind: KindRole, From: roleSelf}},
	}
}

// perClient is the manifest the four session-guarantee laws share.
//
// Each reads the per-iteration trace rather than calling anything, so the
// binding fills nothing: the runner binds the trace through law.TraceBinder,
// and the classifier is derived from the shape that says which argument is the
// key and which calls are reads.
func perClient(law, mixin string) Rule {
	return Rule{
		Law:   law,
		Needs: []string{mixin},
		Fields: []Field{
			{Name: "Classify", Kind: KindHandle, From: handleClassify},
			{Name: "Trace", Kind: KindTrace},
		},
	}
}

// convergent is the manifest the three replica-algebra laws share.
func convergent(law, mixin string, extra ...Field) Rule {
	fields := make([]Field, 0, 3+len(extra))
	fields = append(fields,
		Field{Name: fieldFactory, Kind: KindHandle, From: handleFactory},
		Field{Name: fieldValues, Kind: KindGenerator, From: genValues},
		observation(),
	)
	return Rule{Law: law, Needs: []string{mixin}, Fields: append(fields, extra...)}
}

// snapshotIsolation is the manifest the anomaly laws share.
func snapshotIsolation(law string) Rule { return anomaly(law, mixinSnapshotIsolation) }

// anomaly is the manifest every Adya-phenomenon law shares, under whichever
// claim forbids it.
func anomaly(law, claim string) Rule {
	return Rule{
		Law:   law,
		Needs: []string{claim},
		Fields: []Field{
			// The transaction history the subject exposes. Not a stamp's to
			// name: a subject that cannot report its own history cannot be
			// asked about isolation at all, so this law does not bind without
			// one rather than binding against a fiction.
			{Name: fieldHistory, Kind: KindSupplied, From: "history"},
		},
	}
}

// rules is the catalogue.
//
// Grouped by the axis of the classification that leads each rule, and within a
// group by that classification, so a reader looking up "what does `cursor` owe"
// finds its rules adjacent. [Rules] hands out a copy; nothing mutates it.
//
// repeated literals are law field names, and a shared constant for two structs
// that happen to spell a field the same way would assert a unity that is not
// there — Read on a paginator and Read on a lease are different fields.
//
//nolint:gochecknoglobals,goconst // a lookup table, read-only after init; the
var rules = []Rule{
	// ============================================================ detectors

	writeObservable(shapeWriter),
	writeObservable(shapeCompositeWriter),
	writeObservable(shapeMultiArgWriter),

	countEquals(shapeAggregator),
	countEquals(shapeMultiAggregator),

	// Two laws from one shape: a drain must terminate, and draining twice must
	// agree. Different claims about one method, which is why the selector
	// cannot be a table entry naming a single law.
	{
		Law:   lawid.StreamCompletion,
		Needs: []string{shapeStreamReader},
		Fields: []Field{
			{Name: fieldDrain, Kind: KindRole, From: roleSelf},
			{Name: "Limit", Kind: KindDefault},
		},
	},
	{
		Law:    lawid.StreamReentrant,
		Needs:  []string{shapeStreamReader},
		Fields: []Field{{Name: "Collect", Kind: KindRole, From: roleSelf}},
	},

	{
		Law:   lawid.LifecycleRespectsContext,
		Needs: []string{shapeLifecycle},
		Fields: []Field{
			{Name: "Op", Kind: KindRole, From: roleSelf},
			// An interface declaring begin, commit and rollback registers
			// this law three times under one row, so a failure that did
			// not carry the method named the claim and left the reader to
			// find out where.
			{Name: "Name", Kind: KindMethodName, From: roleSelf},
		},
	},

	// Neither needs a reference: both compare the subject against itself,
	// which is what lets them bind on an interface no oracle models.
	{
		Law:   lawid.PredicateConsistent,
		Needs: []string{shapePredicate},
		Fields: []Field{
			{Name: fieldCall, Kind: KindRole, From: roleSelf},
			{Name: "N", Kind: KindDefault},
		},
	},
	{
		Law:   lawid.PureDeterministic,
		Needs: []string{shapePure},
		Fields: []Field{
			{Name: fieldCall, Kind: KindRole, From: roleSelf},
			{Name: "N", Kind: KindDefault},
		},
	},

	// Both need the mixin beside the shape, and the mixin is what makes them
	// true. `poisonaccessor` is a signature — a nullary bare-error callable —
	// and `Err() error`, `Close() error` and `Ping() error` are all of them.
	// Selecting a latch's laws from a signature claimed every method of that
	// shape, and the read-purity law then failed every correct close-once
	// teardown: the first call answers nil, the second an error, which is what
	// `(a==nil)!=(b==nil)` reports as a defect.
	//
	// `poisonable induce=` is the declaration that a latch exists, naming the
	// operation that trips it. A latch nothing can trip is not a latch, so the
	// claim is exactly the evidence these laws were missing.
	{
		Law:   lawid.PoisonNilOnFresh,
		Needs: []string{shapePoisonAccessor, mixinPoisonable},
		Fields: []Field{
			{Name: fieldFactory, Kind: KindHandle, From: handleFactory},
			{Name: "Probe", Kind: KindRole, From: roleSelf},
		},
	},
	{
		Law:    lawid.PoisonIdempotentRead,
		Needs:  []string{shapePoisonAccessor, mixinPoisonable},
		Fields: []Field{{Name: "Probe", Kind: KindRole, From: roleSelf}},
	},

	// =============================================================== mixins

	convergent(lawid.Associative, mixinAssociative,
		Field{Name: "Apply", Kind: KindRole, From: roleSelf}),
	convergent(lawid.CommutativeWrite, mixinCommutative,
		Field{Name: fieldWrite, Kind: KindRole, From: roleSelf}),
	convergent(lawid.CRDTMerge, mixinCRDTMerge,
		Field{Name: fieldWrite, Kind: KindRole, From: familyWriter},
		Field{Name: "Merge", Kind: KindRole, From: roleSelf}),

	{
		Law:   lawid.AtomicWrite,
		Needs: []string{mixinAtomic},
		Fields: []Field{
			{Name: fieldWrite, Kind: KindRole, From: roleSelf},
			{Name: fieldValues, Kind: KindGenerator, From: genValues},
			observation(),
		},
	},

	{
		Law:   lawid.AggregatorBounded,
		Needs: []string{mixinBounded},
		Fields: []Field{
			{Name: fieldRead, Kind: KindRole, From: roleSelf},
			// `bounded` names one bound, so the floor has no stamp behind it.
			// Optional because zero is the floor of the counting shapes this
			// attaches to; a signed quantity needs the option until eidos
			// carries a second parameter.
			{Name: "Min", Kind: KindConstant, From: paramBoundedMin, Optional: true},
			{Name: "Max", Kind: KindConstant, From: ParamBoundedLimit},
		},
	},

	{
		Law:   lawid.Cacheable,
		Needs: []string{mixinCacheable},
		Fields: []Field{
			{Name: fieldRead, Kind: KindRole, From: roleSelf},
			{Name: fieldKeys, Kind: KindGenerator, From: genKeys},
		},
	},

	// The mixin sits on the reader and names the writer it reflects, so the
	// read is `self`. The `write` partner feeds the action set rather than the
	// law: the claim is about what a read returns, and the write that made it
	// true is a step, not a field.
	{
		Law:   lawid.ReadAfterWrite,
		Needs: []string{mixinReadAfterWrite},
		Fields: []Field{
			{Name: fieldRead, Kind: KindRole, From: roleSelf},
			{Name: fieldKeys, Kind: KindGenerator, From: genKeys},
		},
	},

	{
		// Not the bare per-client shape its three siblings have: causality is
		// a relation over operations, and no stamp can state which of two
		// writes precedes the other. The consumer's domain decides.
		Law:   lawid.CausalOrdering,
		Needs: []string{mixinCausal},
		Fields: []Field{
			{Name: "Classify", Kind: KindHandle, From: handleClassify},
			{Name: "HappensBefore", Kind: KindSupplied, From: "happens-before"},
			{Name: "Trace", Kind: KindTrace},
		},
	},
	perClient(lawid.MonotonicReads, mixinMonotonicReads),
	perClient(lawid.MonotonicWrites, mixinMonotonicWrites),
	perClient(lawid.WritesFollowReads, mixinWritesFollowReads),

	{
		Law:   lawid.Conservative,
		Needs: []string{mixinConservative},
		Fields: []Field{
			{Name: "Sum", Kind: KindRole, From: familyAggregator},
			{Name: fieldWrite, Kind: KindRole, From: roleSelf},
			{Name: fieldValues, Kind: KindGenerator, From: genValues},
		},
	},

	{
		Law:   lawid.DefaultOnError,
		Needs: []string{mixinDefaultOnError},
		Fields: []Field{
			{Name: fieldRead, Kind: KindRole, From: roleSelf},
			{Name: fieldKeys, Kind: KindGenerator, From: genKeys},
			// The zero value is the default a signature states; a different
			// one is a claim no stamp carries.
			{Name: "Default", Kind: KindSupplied, From: "default", Optional: true},
			{Name: "Eq", Kind: KindDefault},
		},
	},

	{
		Law:   lawid.DeleteReturnsNotFound,
		Needs: []string{mixinDeleteRemoves},
		Fields: []Field{
			{Name: fieldRead, Kind: KindRole, From: "deleteremoves.read"},
			{Name: fieldKeys, Kind: KindGenerator, From: genKeys},
			// No classification names the not-found sentinel, and a
			// nil one fails every correct subject — so this does not bind.
			{Name: fieldSentinel, Kind: KindConstant, From: paramDeleteSentinel},
			// The reference's own miss, which is what the law's guard
			// compares against. Left to the sentinel above, the guard asked
			// a plain map for a tombstone it never heard of and the law
			// held vacuously for every subject.
			{Name: "RefMiss", Kind: KindHandle, From: handleReferenceMiss},
		},
	},

	{
		Law:   lawid.EventualConvergence,
		Needs: []string{mixinEventually},
		Fields: []Field{
			{Name: fieldFactory, Kind: KindHandle, From: handleFactory},
			{Name: "Replicas", Kind: KindDefault},
			{Name: fieldWrite, Kind: KindRole, From: familyWriter},
			{Name: fieldValues, Kind: KindGenerator, From: genValues},
			// `eventually` names neither the quiet window nor the
			// anti-entropy round. Settle is skippable; Sync is the law.
			{Name: "Settle", Kind: KindRole, From: "eventually.settle", Optional: true},
			{Name: "Sync", Kind: KindRole, From: "eventually.sync"},
			// The whole-state observation, not the keyed reader: the binding
			// row instantiates S at the observation, and one derivation must
			// answer for both or two laws disagree about what "the state" is.
			{Name: "Snapshot", Kind: KindHandle, From: handleObserve},
			// The join of the replica lattice — the consumer's algebra, and
			// the one field on this law nothing could derive.
			{Name: "Merge", Kind: KindSupplied, From: "merge"},
			{Name: "Equal", Kind: KindDefault},
		},
	},

	// `idempotent` beside a writer and beside a lifecycle are different laws.
	// Neither rule needs a precedence tiebreak: a method carries one detector
	// shape, so at most one of these can match.
	{
		Law:   lawid.IdempotentWrite,
		Needs: []string{mixinIdempotent, shapeWriter},
		Fields: []Field{
			{Name: fieldWrite, Kind: KindRole, From: roleSelf},
			{Name: fieldValues, Kind: KindGenerator, From: genValues},
			observation(),
		},
	},
	{
		// The same claim on a keyed put. Without this row a composite writer
		// carrying `idempotent` selected nothing at all — not even an unbound
		// header line — which is the silence docs/adr/0017 forbids.
		Law:   lawid.IdempotentWrite,
		Needs: []string{mixinIdempotent, shapeCompositeWriter},
		Fields: []Field{
			{Name: fieldWrite, Kind: KindRole, From: roleSelf},
			{Name: fieldValues, Kind: KindGenerator, From: genValues},
			observation(),
		},
	},
	{
		Law:   lawid.IdempotentLifecycle,
		Needs: []string{mixinIdempotent, shapeLifecycle},
		Fields: []Field{
			{Name: fieldCall, Kind: KindRole, From: roleSelf},
			observation(),
		},
	},

	{
		Law:   lawid.InjectionSafe,
		Needs: []string{mixinInjectionSafe},
		Fields: []Field{
			{Name: "Store", Kind: KindRole, From: roleSelf},
			{Name: "Load", Kind: KindRole, From: familyReader},
			{Name: "Payloads", Kind: KindGenerator, From: genPayloads},
			{Name: "CanaryKey", Kind: KindDefault},
			{Name: "CanaryValue", Kind: KindDefault},
		},
	},

	{
		Law:   lawid.LeakFree,
		Needs: []string{mixinLeakFree},
		Fields: []Field{
			// `leakfree` names neither half of the cycle whose
			// balance is the whole claim.
			{Name: "Open", Kind: KindRole, From: "leakfree.open"},
			{Name: fieldClose, Kind: KindRole, From: "leakfree.close"},
			{Name: "Cycles", Kind: KindDefault},
			{Name: "Tolerance", Kind: KindDefault},
			{Name: "Outstanding", Kind: KindRole, From: "family.aggregator", Optional: true},
		},
	},

	{
		Law:   lawid.LifecycleAfterClose,
		Needs: []string{mixinLifecycleAfter},
		Fields: []Field{
			{Name: fieldClose, Kind: KindRole, From: "lifecycleafterclose.close"},
			{Name: "Op", Kind: KindRole, From: roleSelf},
			// Ops is the law's multi-probe arm, deliberately left at its
			// default here: this generator binds one law per stamped host
			// through Op, so every stamped method is probed by its own
			// binding and the probe map has nothing to add. An emitter
			// that folds the stamped set into ONE binding fills Ops
			// instead — one probe per stamped method — because a claim
			// about several methods backed by one probe is the
			// silent-green class the law widened to kill.
			{Name: "Ops", Kind: KindDefault},
			{Name: fieldSentinel, Kind: KindConstant, From: paramLifecycleSentinel},
		},
	},

	{
		Law:   lawid.MonotonicNonDecreasing,
		Needs: []string{mixinMonotonic},
		Fields: []Field{
			{Name: fieldRead, Kind: KindRole, From: roleSelf},
			{Name: "Less", Kind: KindHandle, From: handleOrder},
		},
	},

	{
		Law:   lawid.StreamOverMatch,
		Needs: []string{mixinOverMatch},
		Fields: []Field{
			{Name: fieldDrain, Kind: KindRole, From: roleSelf},
			{Name: fieldHistory, Kind: KindHandle, From: handleWriteLog},
			{Name: fieldHash, Kind: KindHandle, From: handleIdentity},
		},
	},
	{
		Law:   lawid.StreamPermutation,
		Needs: []string{mixinPermutation},
		Fields: []Field{
			{Name: fieldDrain, Kind: KindRole, From: roleSelf},
			{Name: fieldHistory, Kind: KindHandle, From: handleWriteLog},
			{Name: fieldHash, Kind: KindHandle, From: handleIdentity},
		},
	},
	{
		Law:   lawid.StreamStableOrder,
		Needs: []string{mixinStableOrder},
		Fields: []Field{
			{Name: fieldDrain, Kind: KindRole, From: roleSelf},
			// "Stable" names a semantic order — insertion, key-ascending —
			// and no signature reveals which. One of the two fields in the
			// whole catalogue nothing can derive.
			{Name: "Less", Kind: KindSupplied, From: "order"},
		},
	},

	// Two claims, and the split is the whole point. Snapshot isolation forbids
	// dirty writes and dirty, intermediate and circular reads, and it
	// *permits* G2 — write skew is its canonical allowed anomaly, the price
	// the model is defined to pay. Selecting the anti-dependency-cycle check
	// from it reddened every correct SI store, and the claim could not be
	// declared without declaring the check that contradicts it.
	//
	// `serializable` is the claim that forbids G2, and it is a sibling rather
	// than a level: a store is not snapshot-isolated at level serializable.
	// A store claiming both earns all three, which is what a serializable
	// store owes.
	snapshotIsolation(lawid.SnapshotIsolationG0),
	snapshotIsolation(lawid.SnapshotIsolationG1),
	anomaly(lawid.SnapshotIsolationG2, mixinSerializable),

	{
		Law:   lawid.Sticky,
		Needs: []string{mixinSticky},
		Fields: []Field{
			{Name: fieldRead, Kind: KindRole, From: roleSelf},
			{Name: fieldKeys, Kind: KindGenerator, From: genKeys},
			{Name: "Eq", Kind: KindDefault},
		},
	},

	{
		Law:   lawid.StreamReflectsMutations,
		Needs: []string{mixinStreamReflects},
		Fields: []Field{
			{Name: fieldPut, Kind: KindRole, From: "streamreflectsmutations.mutate"},
			// The mixin names the write half only.
			{Name: "Delete", Kind: KindRole, From: "streamreflectsmutations.delete"},
			{Name: fieldDrain, Kind: KindRole, From: roleSelf},
			{Name: fieldValues, Kind: KindGenerator, From: genValues},
			{Name: fieldHash, Kind: KindHandle, From: handleIdentity},
		},
	},

	{
		Law:   lawid.TamperEvident,
		Needs: []string{mixinTamperEvident},
		Fields: []Field{
			{Name: fieldWrite, Kind: KindRole, From: familyWriter},
			// Neither the tamper nor the verify is named by the mixin.
			{Name: "Tamper", Kind: KindRole, From: "tamperevident.tamper"},
			{Name: "Verify", Kind: KindRole, From: "tamperevident.verify"},
			{Name: fieldValues, Kind: KindGenerator, From: genValues},
		},
	},

	// The clock family. `timeaware` marks the clock dependency; each claim
	// carries its own quantity, because a lifetime on stored data and a
	// deadline on an operation are different promises that happen to share a
	// clock. Neither rule needs `timeaware` as well: a classification that
	// names a duration has already said a clock is involved.
	// The claim `timeaware` states on its own. Every other rule in this
	// family needs a quantity, and the quantities belong to the
	// classifications layered on top — what is left is the dependency
	// itself, which is the half worth checking first: a subject reading
	// wall time passes every quantity claim whose test never advances far
	// enough, and fails this one at once.
	{
		Law:   lawid.TimeawareMoves,
		Needs: []string{mixinTimeaware},
		Fields: []Field{
			{Name: fieldRead, Kind: KindRole, From: roleSelf},
			{Name: fieldKeys, Kind: KindGenerator, From: genKeys},
			{Name: fieldAdvance, Kind: KindHandle, From: handleClock},
		},
	},

	{
		Law:   lawid.TTLExpiry,
		Needs: []string{mixinTTL},
		Fields: []Field{
			{Name: fieldPut, Kind: KindRole, From: "ttl.put"},
			{Name: fieldRead, Kind: KindRole, From: "ttl.read"},
			{Name: fieldKeys, Kind: KindGenerator, From: genKeys},
			{Name: fieldValues, Kind: KindGenerator, From: genValues},
			{Name: "TTL", Kind: KindConstant, From: paramTTLDuration, PerValue: true},
			{Name: fieldAdvance, Kind: KindHandle, From: handleClock},
			{Name: "NotFound", Kind: KindConstant, From: paramTTLNotFound},
		},
	},
	{
		// `timeout duration=` already states exactly this law's claim — the
		// callable respects a deadline and returns promptly when the context
		// expires — so it selects it rather than a second stamp saying the
		// same thing.
		Law:   lawid.DeadlineRespecting,
		Needs: []string{mixinTimeout},
		Fields: []Field{
			{Name: "Op", Kind: KindRole, From: roleSelf},
			{Name: "Deadline", Kind: KindConstant, From: paramTimeoutDuration},
			{Name: fieldAdvance, Kind: KindHandle, From: handleClock},
			{Name: "AwaitFor", Kind: KindDefault},
			// One row per interface however many methods carry the stamp,
			// so the method is what tells two probes apart in a failure.
			{Name: "Name", Kind: KindMethodName, From: roleSelf},
		},
	},
	{
		Law:   lawid.ScheduledFiresAfterAdvance,
		Needs: []string{mixinScheduled},
		Fields: []Field{
			{Name: "Schedule", Kind: KindRole, From: "scheduled.schedule"},
			// The load-bearing half: a run that cannot count firings reports
			// every scheduler as correct, including one that fires nothing.
			{Name: "FiredCount", Kind: KindRole, From: "scheduled.fired"},
			{Name: "Offsets", Kind: KindGenerator, From: genOffsets},
			{Name: "N", Kind: KindDefault},
			{Name: fieldAdvance, Kind: KindHandle, From: handleClock},
		},
	},

	// The four mixins that completed their families.
	{
		Law:   lawid.StreamNoDuplicates,
		Needs: []string{mixinNoDuplicates},
		Fields: []Field{
			{Name: fieldDrain, Kind: KindRole, From: roleSelf},
			{Name: fieldHash, Kind: KindHandle, From: handleIdentity},
		},
	},
	{
		// Distinct from `cacheable`, which permits a concurrent write to be
		// observed by the second read. This one does not.
		Law:   lawid.PointInTime,
		Needs: []string{mixinPointInTime},
		Fields: []Field{
			{Name: fieldRead, Kind: KindRole, From: roleSelf},
			{Name: fieldKeys, Kind: KindGenerator, From: genKeys},
			{Name: "Disturb", Kind: KindSupplied, From: "disturb", Optional: true},
		},
	},
	perClient(lawid.ReadYourWrites, mixinReadYourWrites),
	{
		Law:   lawid.PoisonConsistent,
		Needs: []string{mixinPoisonable},
		Fields: []Field{
			{Name: "Poison", Kind: KindRole, From: "poisonable.induce"},
			{Name: "Probe", Kind: KindRole, From: roleSelf},
			{Name: "Reads", Kind: KindDefault},
		},
	},

	// The same law by the other road: a closed host is a poisoned host,
	// and `lifecycleafterclose close=` names the callable that gets it
	// there. Two rules for one law rather than a second law, because the
	// claim is the same one — once it reports closed it keeps reporting
	// closed — and the closure shapes are keyed by law.
	//
	// Conditioned on `close=` and not on `sentinel=`. The sentinel is what
	// the law OBSERVES; close is what fills Poison. Selecting on the
	// sentinel earns the law on interfaces where the induction cannot be
	// filled, and a selection one tier makes and the other then refuses is
	// how a harness comes to carry a door for a law nothing binds.
	{
		Law:   lawid.PoisonConsistent,
		Needs: []string{mixinLifecycleAfter},
		When:  []Condition{{Param: paramLifecycleClose}},
		Fields: []Field{
			{Name: "Poison", Kind: KindRole, From: mixinLifecycleAfter + "." + "close"},
			{Name: "Probe", Kind: KindRole, From: roleSelf},
			{Name: "Reads", Kind: KindDefault},
		},
	},

	{
		Law:   lawid.TotalOver,
		Needs: []string{mixinTotal},
		Fields: []Field{
			{Name: fieldCall, Kind: KindRole, From: roleSelf},
			{Name: "Input", Kind: KindGenerator, From: genInputs},
		},
	},

	{
		Law:   lawid.Windowed,
		Needs: []string{mixinWindowed},
		Fields: []Field{
			// Neither the increment nor the count is named by the mixin.
			{Name: "Incr", Kind: KindRole, From: "windowed.incr"},
			{Name: fieldCount, Kind: KindRole, From: "windowed.count"},
			{Name: fieldAdvance, Kind: KindHandle, From: handleClock},
			{Name: fieldKeys, Kind: KindGenerator, From: genKeys},
			{Name: "Window", Kind: KindConstant, From: paramWindowedWindow},
		},
	},

	{
		Law:   lawid.XSSSafe,
		Needs: []string{mixinXSSSafe},
		Fields: []Field{
			{Name: "Render", Kind: KindRole, From: roleSelf},
			{Name: "Payloads", Kind: KindGenerator, From: genPayloads},
			{Name: "Dangerous", Kind: KindDefault},
		},
	},

	// ============================================================ contracts

	{
		Law:   lawid.AppenderMonotonicOffsets,
		Needs: []string{contractAppender},
		Fields: []Field{
			{Name: "Append", Kind: KindRole, From: "appender.fn"},
			// Wide inputs: offset monotonicity holds over any append stream,
			// and nothing here revisits a key.
			{Name: fieldValues, Kind: KindGenerator, From: genInputs},
		},
	},

	// `mode=atomic` is the claim that a failed write leaves observable state
	// unchanged, which is what AUTO-ATOMIC-WRITE already states.
	{
		Law:   lawid.AtomicWrite,
		Needs: []string{contractBatchWriter},
		When:  []Condition{{Param: paramBatchMode, Equals: "atomic"}},
		Fields: []Field{
			{Name: fieldWrite, Kind: KindRole, From: "batch-writer.writer"},
			{Name: fieldValues, Kind: KindGenerator, From: genValues},
			observation(),
		},
	},

	{
		Law:   lawid.Cacheable,
		Needs: []string{contractCache},
		Fields: []Field{
			{Name: fieldRead, Kind: KindRole, From: "cache.cache"},
			{Name: fieldKeys, Kind: KindGenerator, From: genKeys},
		},
	},

	{
		Law:   lawid.CASAtomicOneWinner,
		Needs: []string{contractCAS},
		Fields: []Field{
			{Name: "CAS", Kind: KindRole, From: "cas.writer"},
			{Name: fieldRead, Kind: KindRole, From: familyCell},
			// The law's own pool over the writer's input: the VersionedCell
			// election leaves no shared values pool, and the attempts are
			// the law's to draw anyway — the stamp below makes them land.
			{Name: fieldValues, Kind: KindGenerator, From: genInputs},
			{Name: "Stamp", Kind: KindHandle, From: handleVersionStamp},
			{Name: "Mismatch", Kind: KindConstant, From: paramCASMismatch},
		},
	},

	// The chain family. All four read the replay role; two need the trace the
	// generated file keeps, and the partition set comes from the `partition`
	// mixin where one is declared.
	{
		Law:   lawid.AppendOnlyGrows,
		Needs: []string{contractChain},
		Fields: []Field{
			{Name: fieldReplay, Kind: KindRole, From: "chain.replay"},
			{Name: fieldPartitions, Kind: KindHandle, From: "partitions"},
		},
	},
	{
		Law:   lawid.AppendOnlyNoDrops,
		Needs: []string{contractChain},
		Fields: []Field{
			{Name: fieldReplay, Kind: KindRole, From: "chain.replay"},
			{Name: fieldHistory, Kind: KindHandle, From: handleHistory},
		},
	},
	{
		Law:   lawid.ReplayDeterministic,
		Needs: []string{contractChain},
		Fields: []Field{
			{Name: fieldReplay, Kind: KindRole, From: "chain.replay"},
			{Name: fieldPartitions, Kind: KindHandle, From: "partitions"},
		},
	},
	{
		Law:    lawid.HashChainIntegrityVerify,
		Needs:  []string{contractChain},
		Fields: []Field{{Name: "Verify", Kind: KindRole, From: "chain.verify"}},
	},
	{
		Law:   lawid.ReplayCausalOrdering,
		Needs: []string{contractChain, mixinCausal},
		Fields: []Field{
			{Name: fieldReplay, Kind: KindRole, From: "chain.replay"},
			{Name: fieldPartitions, Kind: KindHandle, From: "partitions"},
			// A dependency graph over entries — the consumer's causality, not
			// a shape's.
			{Name: "EntryID", Kind: KindSupplied, From: "entry-id"},
			{Name: "DependsOn", Kind: KindSupplied, From: "depends-on"},
		},
	},

	// `fidelity` decides which of the two roundtrip laws states the claim:
	// exact is the identity, lossy only requires the second pass to agree with
	// the first. Asserting exactness of a lossy codec fails every correct one.
	{
		Law:   lawid.Roundtrip,
		Needs: []string{contractCodec},
		When:  []Condition{{Param: paramCodecFidelity, NotEquals: "lossy"}},
		Fields: []Field{
			{Name: "Forward", Kind: KindRole, From: "codec.forward"},
			{Name: "Inverse", Kind: KindRole, From: "codec.inverse"},
			// Wide inputs rather than the colliding values pool: a roundtrip
			// is stateless, and the claim is over the domain.
			{Name: fieldValues, Kind: KindGenerator, From: genInputs},
		},
	},
	{
		Law:   lawid.LossyRoundtrip,
		Needs: []string{contractCodec},
		When:  []Condition{{Param: paramCodecFidelity, Equals: "lossy"}},
		Fields: []Field{
			{Name: "Forward", Kind: KindRole, From: "codec.forward"},
			{Name: "Inverse", Kind: KindRole, From: "codec.inverse"},
			// Wide inputs rather than the colliding values pool: a roundtrip
			// is stateless, and the claim is over the domain.
			{Name: fieldValues, Kind: KindGenerator, From: genInputs},
		},
	},

	{
		Law:   lawid.CursorNextAfterClose,
		Needs: []string{contractCursor},
		Fields: []Field{
			{Name: fieldClose, Kind: KindRole, From: "cursor.close"},
			{Name: "Next", Kind: KindRole, From: "cursor.next"},
			{Name: fieldSentinel, Kind: KindConstant, From: paramCursorSentinel},
		},
	},
	{
		Law:    lawid.CursorCloseIdempotent,
		Needs:  []string{contractCursor},
		Fields: []Field{{Name: fieldClose, Kind: KindRole, From: "cursor.close"}},
	},

	{
		Law:   lawid.LeaseDoubleAcquireBlocks,
		Needs: []string{contractLease},
		Fields: []Field{
			{Name: "Acquire", Kind: KindRole, From: "lease.acquire"},
			{Name: "Release", Kind: KindRole, From: "lease.release"},
			{Name: fieldKeys, Kind: KindGenerator, From: genKeys},
			{Name: "Held", Kind: KindConstant, From: paramLeaseHeld},
		},
	},
	{
		Law:   lawid.LeaseReleasedOnCancel,
		Needs: []string{contractLease},
		Fields: []Field{
			{Name: "Acquire", Kind: KindRole, From: "lease.acquire"},
			{Name: "Free", Kind: KindSupplied, From: "free"},
			{Name: fieldKeys, Kind: KindGenerator, From: genKeys},
			{Name: "Timeout", Kind: KindConstant, From: paramLeaseTimeout},
		},
	},

	{
		Law:   lawid.PaginatorNoDuplicates,
		Needs: []string{contractPagination},
		Fields: []Field{
			{Name: "Page", Kind: KindRole, From: "pagination.reader"},
			{Name: "Start", Kind: KindDefault},
			{Name: fieldKeyOf, Kind: KindHandle, From: handleKeyOf},
			{Name: "MaxPages", Kind: KindDefault},
		},
	},
	{
		Law:   lawid.PaginatorResumable,
		Needs: []string{contractPagination},
		Fields: []Field{
			{Name: "Page", Kind: KindRole, From: "pagination.reader"},
			{Name: "Start", Kind: KindDefault},
			{Name: "MaxPages", Kind: KindDefault},
		},
	},

	{
		Law:   lawid.PersisterRetrievable,
		Needs: []string{contractPersister},
		Fields: []Field{
			{Name: "Save", Kind: KindRole, From: "persister.writer"},
			{Name: fieldRead, Kind: KindRole, From: "persister.reader"},
			{Name: fieldValues, Kind: KindGenerator, From: genValues},
		},
	},

	{
		Law:    lawid.PoolBalanced,
		Needs:  []string{contractPool},
		Fields: []Field{{Name: "Stats", Kind: KindSupplied, From: "stats"}},
	},
	{
		Law:    lawid.PoolLeakFree,
		Needs:  []string{contractPool},
		Fields: []Field{{Name: "Balanced", Kind: KindSupplied, From: "balanced"}},
	},

	// A bare publisher claims delivery; `mode` refines it into a bound on how
	// many copies each subscriber sees. Absent means unstated, not a default,
	// so the refined law does not bind without it.
	{
		Law:   lawid.PublisherDelivers,
		Needs: []string{contractPublisher},
		Fields: []Field{
			{Name: "Subscribe", Kind: KindRole, From: "publisher.subscribe"},
			{Name: "Publish", Kind: KindRole, From: "publisher.publish"},
			{Name: fieldDrain, Kind: KindSupplied, From: optDrain},
			{Name: "Messages", Kind: KindGenerator, From: genMessages},
			{Name: "Subscribers", Kind: KindDefault},
		},
	},
	{
		Law:    lawid.PublisherAtLeastOnce,
		Needs:  []string{contractPublisher},
		When:   []Condition{{Param: ParamPublisherMode, Equals: "at-least-once"}},
		Fields: publisherBound(),
	},
	{
		Law:    lawid.PublisherAtMostOnce,
		Needs:  []string{contractPublisher},
		When:   []Condition{{Param: ParamPublisherMode, Equals: "at-most-once"}},
		Fields: publisherBound(),
	},
	{
		Law:    lawid.PublisherExactlyOnce,
		Needs:  []string{contractPublisher},
		When:   []Condition{{Param: ParamPublisherMode, Equals: "exactly-once"}},
		Fields: publisherBound(),
	},

	{
		Law:   lawid.SagaFullCompensation,
		Needs: []string{contractSaga},
		Fields: []Field{
			{Name: "Run", Kind: KindRole, From: "saga.step"},
			observation(),
		},
	},

	{
		Law:   lawid.SingleflightCoalesces,
		Needs: []string{contractSingleflight},
		Fields: []Field{
			{Name: fieldCall, Kind: KindRole, From: "singleflight.fn"},
			// Both are the harness's own instrumentation: the law counts how
			// many times its own compute ran, so the generated file supplies
			// the closure and the counter it increments.
			{Name: "Compute", Kind: KindHandle, From: "coalesce-probe"},
			{Name: "Counter", Kind: KindHandle, From: "coalesce-counter"},
			// The law's own pool rather than the shared keys: the run role
			// takes a callable no pool draws, so no action feeds a keys pool
			// here — the inputs pool ranges over the role's key parameter.
			{Name: fieldKeys, Kind: KindGenerator, From: genInputs},
			{Name: "Parallel", Kind: KindDefault},
		},
	},

	{
		Law:   lawid.TransactionRollback,
		Needs: []string{contractTransaction},
		Fields: []Field{
			// SUTOnly: the law drives Run on the subject with a body that
			// always errors, so the transaction it opens is one the subject
			// discards whole. A derived oracle cannot model a callable-taking
			// method and holds it inert, and without this the whole claim
			// would decline for a divergence a rolled-back run cannot cause.
			// Conduct register: self-cleaning.
			{Name: "Run", Kind: KindRole, From: "transaction.fn", SUTOnly: true},
			observed(fieldRead),
			// The staged mutation the rollback must discard. From the
			// keyed-writer family rather than a contract role: the
			// transaction contract declares only fn, and what a store writes
			// with is its own keyed writer — the same method the sequences
			// drive, so a defect worn on it reaches the law. Keyed rather
			// than plain because the law reads the key back, which means it
			// has to have chosen it.
			//
			// This field is why the law can fail at all. Without a write
			// inside the body there is nothing for an erroring transaction
			// to leave behind, and an interface declaring the contract
			// without a keyed writer now declines the law by name instead of
			// binding a check that always passed.
			{Name: fieldWrite, Kind: KindRole, From: familyKeyedWriter},
			{Name: fieldKeys, Kind: KindGenerator, From: genKeys},
			{Name: fieldValues, Kind: KindGenerator, From: genValues},
			{Name: "NotFound", Kind: KindConstant, From: paramTransactionNotFound},
		},
	},
	{
		// The claim is the tx trio's: what an open transaction staged stays
		// invisible to an outside read until commit. Selection moved from
		// the fn-shaped transaction contract to the trio the fields actually
		// name — the fn contract's run has no handle to stage through.
		Law:   lawid.TransactionNoMidTxVisibility,
		Needs: []string{contractTx},
		Fields: []Field{
			{Name: "Begin", Kind: KindRole, From: "tx.begin"},
			{Name: "TxPut", Kind: KindRole, From: familyHandleWriter},
			{Name: "TxRollback", Kind: KindRole, From: "tx.rollback"},
			observed(fieldRead),
			{Name: fieldKeys, Kind: KindGenerator, From: genKeys},
			// The law's own pool at the read's answer: the trio's shared
			// values pool draws handles, and the staged write's domain is
			// whatever the outside read observes.
			{Name: fieldValues, Kind: KindGenerator, From: genReadback},
		},
	},

	{
		Law:    lawid.TwoPhaseMutex,
		Needs:  []string{contractTx},
		Fields: twoPhase(),
	},
	{
		Law:    lawid.TwoPhaseRollbackAfterCommit,
		Needs:  []string{contractTx},
		Fields: twoPhase(),
	},

	{
		Law:   lawid.UpdaterReplaces,
		Needs: []string{contractUpdater},
		Fields: []Field{
			{Name: "Update", Kind: KindRole, From: "updater.writer"},
			{Name: fieldRead, Kind: KindRole, From: "updater.reader"},
			{Name: fieldValues, Kind: KindGenerator, From: genValues},
			{Name: fieldKeyOf, Kind: KindHandle, From: handleKeyOf},
		},
	},
	{
		Law:   lawid.UpserterIdempotent,
		Needs: []string{contractUpserter},
		Fields: []Field{
			{Name: "Upsert", Kind: KindRole, From: "upserter.writer"},
			{Name: fieldRead, Kind: KindRole, From: "upserter.reader"},
			{Name: fieldValues, Kind: KindGenerator, From: genValues},
			{Name: fieldKeyOf, Kind: KindHandle, From: handleKeyOf},
		},
	},

	{
		Law:   lawid.WatcherReturnsOnChange,
		Needs: []string{contractWatcher},
		Fields: []Field{
			{Name: "Watch", Kind: KindRole, From: "watcher.watch"},
			{Name: "Mutate", Kind: KindRole, From: "watcher.trigger"},
			// Both are methods on the handle Watch returns, which
			// the contract's roles do not reach.
			{Name: "Next", Kind: KindSupplied, From: "next"},
			{Name: "Stop", Kind: KindSupplied, From: "stop"},
			{Name: fieldKeys, Kind: KindGenerator, From: genKeys},
			{Name: fieldValues, Kind: KindGenerator, From: genValues},
			{Name: "Timeout", Kind: KindDefault},
		},
	},

	{
		Law:   lawid.ValidTransition,
		Needs: []string{contractWorkflow},
		Fields: []Field{
			{Name: fieldWrite, Kind: KindRole, From: "workflow.fn"},
			{Name: fieldValues, Kind: KindGenerator, From: genValues},
			observation(),
			// The permitted edges, read from the contract's own parameter —
			// which is the one place the declaration states them.
			{Name: "Allowed", Kind: KindConstant, From: paramWorkflowTransitions},
		},
	},
}

// publisherBound is the manifest the three delivery-mode laws share. They
// differ only in the mode that selects them, which the law reads back as the
// bound it enforces.
func publisherBound() []Field {
	return []Field{
		{Name: "Subscribe", Kind: KindRole, From: "publisher.subscribe"},
		{Name: "Publish", Kind: KindRole, From: "publisher.publish"},
		{Name: "Redeliver", Kind: KindRole, From: "publisher.redeliver", Optional: true},
		{Name: fieldDrain, Kind: KindSupplied, From: optDrain},
		{Name: "Messages", Kind: KindGenerator, From: genMessages},
		{Name: "Mode", Kind: KindConstant, From: ParamPublisherMode},
		{Name: "Subscribers", Kind: KindDefault},
	}
}

// twoPhase is the manifest both `tx` laws share.
func twoPhase() []Field {
	return []Field{
		{Name: "Begin", Kind: KindRole, From: "tx.begin"},
		{Name: "Commit", Kind: KindRole, From: "tx.commit"},
		{Name: "Rollback", Kind: KindRole, From: "tx.rollback"},
		{Name: "Closed", Kind: KindConstant, From: paramTxClosed},
	}
}

// unusedGenerators keeps the offset pool named while no rule draws from it.
//
// AUTO-SCHEDULED-FIRES-AFTER-ADVANCE is the one clock law with no rule: it
// needs a schedule-at method and a fired-count method, and `timeaware` names
// neither. The pool is spelled here so the name exists once when
// that rule lands rather than being invented at the call site.
var _ = genOffsets
