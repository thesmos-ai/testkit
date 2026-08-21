// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

// LawFieldKindPrefix composes each law field's emit kind —
// `model.lawfield.<Shape>` — which is the template that renders it.
//
// Dispatch is on the closure's shape, not the field's name: the catalogue
// spells Read on a keyed store and Read on a version cell with one word, and
// the two closures could not be less alike. The shape table below is the
// transcription of each law struct's field types, held to the shipped structs
// the same way the binding rows are — a wrong shape renders a closure that
// fails to compile in whichever corpus package arms it.
const LawFieldKindPrefix = "model.lawfield."

// The law-declared pool names.
const (
	poolInputs   = "inputs"
	poolPayloads = "payloads"
	poolOffsets  = "offsets"
)

// builtinString and builtinInt are the builtins several derivations spell.
const (
	builtinString = "string"
	builtinInt    = "int"
)

// The closure vocabulary, one per distinct field type across the law structs.
const (
	shapeKeyedRead  lawShape = "Read"        // func(rt, T, K) (V, error)
	shapeValueOp    lawShape = "Write"       // func(rt, T, V) error
	shapeDrainSlice lawShape = "Drain"       // func(rt, T) ([]V, error), slice role
	shapeDrainSeq   lawShape = "DrainSeq"    // same field over an iterator role
	shapeScalar     lawShape = "Scalar"      // func(rt, T) (R, error)
	shapeScalarLen  lawShape = "ScalarLen"   // same, R = len of a returned slice
	shapeBoolCall   lawShape = "BoolCall"    // func(rt, T) bool
	shapeResultCall lawShape = "ResultCall"  // func(rt, T) R
	shapeInputCall  lawShape = "InputCall"   // func(rt, T, X) (R, error)
	shapeCtxOp      lawShape = "CtxOp"       // func(ctx, T) error
	shapeErrOp      lawShape = "ErrOp"       // func(rt, T) error
	shapeKeyedOp    lawShape = "KeyedOp"     // func(rt, T, K) error
	shapeKVOp       lawShape = "KVOp"        // func(rt, T, k, v string) error
	shapeSum        lawShape = "Sum"         // func(rt, T) int64
	shapeMerge      lawShape = "Merge"       // func(rt, dst, src T) error
	shapeSave       lawShape = "Save"        // func(rt, T, V) (K, error), K synthesized
	shapeAppendOff  lawShape = "Append"      // func(rt, T, V) (Off, error)
	shapeReplay     lawShape = "ReplaySlice" // func(rt, T, K) iter.Seq2[E, error]

	shapeOkOp        lawShape = "OkOp"        // func(rt, T) bool — err-op success
	shapeNextOp      lawShape = "NextOp"      // func(rt, T) (V, bool, error)
	shapeDoOp        lawShape = "DoOp"        // func(rt, T) — fire and forget
	shapePinnedWrite lawShape = "PinnedWrite" // func(rt, T, K, V) error — pin then put
	shapeCtxOpFixed  lawShape = "CtxOpFixed"  // func(ctx, T) error — fixed fixture arg
	shapeScheduleAt  lawShape = "ScheduleAt"  // func(rt, T, at time.Duration) error
	shapeCountObs    lawShape = "CountObs"    // func(rt, T) int — loud on error
	shapeSubscribe   lawShape = "Subscribe"   // func(rt, T) (Sub, error) — the handle kept, never compared
	shapeCtxKeyedOp  lawShape = "CtxKeyedOp"  // func(ctx, T, K) error — the law draws the key

	shapeHandleCall  lawShape = "HandleCall"  // func(rt, T) (H, error) — the handle a terminal pair threads
	shapeHandleOp    lawShape = "HandleOp"    // func(rt, T, H) error — one terminal operation on the handle
	shapeSagaRun     lawShape = "SagaRun"     // func(rt, T) error — step drawn values, unwind what committed
	shapeComputeCall lawShape = "ComputeCall" // func(ctx, T, K, compute) (V, error) — the deduplicated call
	shapeBodyRun     lawShape = "BodyRun"     // func(rt, T, body) error — the transactional scope
	shapePageRead    lawShape = "PageRead"    // func(rt, T, C) ([]V, C, bool) — one page, loud on error

	shapeKeyedHandle lawShape = "KeyedHandle" // func(rt, T, K) (W, error) — the handle kept, never compared
	shapeKeyedWrite  lawShape = "KeyedWrite"  // func(rt, T, K, V) error — a keyed write at both pools
	shapeHandleWrite lawShape = "HandleWrite" // func(rt, T, H, K, V) error — a keyed write inside an open handle

	shapePeerSync   lawShape = "PeerSync"   // func(rt, []T) error — the star round over pairwise sync
	shapeEachSettle lawShape = "EachSettle" // func(rt, []T) — one settle per replica
)

// lawRoleShapes transcribes each rowed law's role-field closure types from
// the engine structs — the second half of what the binding row transcribes.
// A rowed law's role field missing here is a build refusal by name, never a
// wrong guess.
//
// The field and role spellings this file repeats — each names one concept.
const (
	fRead    = "Read"
	fWrite   = "Write"
	fTxPut   = "TxPut"
	fDrain   = "Drain"
	fCall    = "Call"
	fCount   = "Count"
	fProbe   = "Probe"
	fClose   = "Close"
	fromSelf = "self"

	// handleClassifier is the manifest spelling of the per-client
	// trace-classifier handle.
	handleClassifier = "trace-classifier"

	// The publisher rows' role spellings, and the drain option's manifest
	// name.
	fSubscribe = "Subscribe"
	fPublish   = "Publish"
	fRedeliver = "Redeliver"
	optDrain   = "drain"
	builtin64  = "int64"
	fMerge     = "Merge"
	fHistory   = "History"
	fEntryID   = "EntryID"

	// The contract-shape family's field and handle spellings.
	fBegin          = "Begin"
	fCommit         = "Commit"
	fRollback       = "Rollback"
	fRun            = "Run"
	fPage           = "Page"
	fReplay         = "Replay"
	fCAS            = "CAS"
	fStore          = "Store"
	kindInert       = "inert"
	kindSputter     = "sputter"
	kindFlap        = "flap"
	kindOvershoot   = "overshoot"
	kindFlicker     = "flicker"
	kindFadeSeq     = "fadeseq"
	kindDupSeq      = "dupseq"
	kindJumble      = "jumble"
	kindDupDrain    = "dupdrain"
	kindFlood       = "flood"
	kindRegress     = "regress"
	kindPassthrough = "passthrough"
	kindStick       = "stick"
	kindSpill       = "spill"
	kindLatch       = "latch"
	kindGreedy      = "greedy"

	// satFloodItems is what a flooding drain answers, and it is pinned to
	// [law.StreamCompletion]'s own default limit rather than derived: the
	// field is KindDefault everywhere, so the law's constant is the only
	// number in play. A wear yielding fewer would prove nothing, and one
	// yielding without bound is what took 30 GB of a machine — every drain
	// the generator emits ranges to exhaustion before its limit is read.
	satFloodItems       = 10000
	fieldMax            = "Max"
	fromFamilyCell      = "family.cell"
	fromFamilyKeyedWr   = "family.keyedwriter"
	fromFamilyHandleWr  = "family.handlewriter"
	handleKeyProjection = "key-projection"
	handleCoalesce      = "coalesce-probe"
	handleVersionStamp  = "version-stamp"
	handleHistoryLog    = "history"

	// paramCASVersion is the cas contract's version member stamp — the
	// field of the attempt the compare-and-set guards by.
	paramCASVersion = "shape.contract.cas.param.version"

	// The watcher contract's member params: methods of the handle the
	// watch role answers, resolved by the member scope and read here as
	// qualified stamps.
	memberNext      = "next"
	memberStop      = "stop"
	poolReadback    = "readback"
	paramWatcherKey = "shape.contract.watcher.param."
	fNext           = "Next"
	fWatch          = "Watch"
)

// builtinInts are the signed integer builtins a sum or an offset totals;
// builtinOrdered is everything `<` orders.
//
//nolint:gochecknoglobals // vocabulary tables, read-only after init.
var (
	builtinInts    = []string{builtinInt, "int8", "int16", "int32", builtin64}
	builtinOrdered = append(append([]string{}, builtinInts...),
		"uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64", builtinString)
)

// The supplied-shape vocabulary — each names one closure type arm in the
// option templates.
const (
	supClientOpPred = "ClientOpPred" // func(a, b law.ClientOp[K]) bool
	supElemPred     = "ElemPred"     // func(a, b V) bool
	supElemList     = "ElemList"     // func(*model.T, T) []V
	supTxnHistory   = "TxnHistory"   // func(*model.T, T) []law.Txn[K]
	supMerge        = "Merge"        // func(a, b S) S
	supKeyPred      = "KeyPred"      // func(*model.T, T, K) bool
	supSubjPred     = "SubjPred"     // func(*model.T, T) bool
	supStats        = "Stats"        // func(*model.T, T) (int, int, int)
	supEntryID      = "EntryID"      // func(E) string
	supDependsOn    = "DependsOn"    // func(E) []string
)

// The classification and When-parameter composition lives on the suite
// projection ([subject.Method.Classifications], [subject.LawParams]) — the one
// home both tiers select from, so they cannot disagree about what the run
// classified.
