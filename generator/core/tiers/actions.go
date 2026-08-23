// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers

import (
	"maps"
	"slices"
)

// ActionFor returns the `engine/model/action` constructor that drives a method
// of the given detector shape, and whether the shape names one.
//
// The correspondence is data rather than derivation because the constructor
// names carry word boundaries the shape names flattened — `streamreader` is
// `Stream`, `poisonaccessor` is `PoisonCheck` — and it lives here rather than
// in the model generator because this is the module the conformance gate holds
// to both registries: every detector eidos registers must have a row, and
// every row must name a constructor the engine exports. A mapping the gate
// cannot see is how the last ownership table drifted to naming a third of what
// it claimed.
func ActionFor(shape string) (string, bool) {
	ctor, ok := actionCtors[shape]
	return ctor, ok
}

// actionCtors maps every detector shape to its action constructor.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var actionCtors = map[string]string{
	shapeAggregator:      "Aggregator",
	shapeAnsweringWriter: "AnsweringWriter",
	shapeBatchReader:     "BatchReader",
	// A teardown and a latch probe are one signature — nullary, error only —
	// so they are driven the same way: call both sides, compare whether each
	// refused. The two classifications differ in what they *claim*, which is
	// what selects laws, and nothing about how a sequence drives them. This is
	// also what `Close() error` was already driven by, back when the poison
	// detector claimed the shape.
	shapeCloser:          "PoisonCheck",
	shapeCompositeWriter: "CompositeWriter",
	shapeLifecycle:       "Lifecycle",
	shapeLookup:          "Lookup",
	shapeMultiAggregator: "MultiAggregator",
	shapeMultiArgWriter:  "MultiArgWriter",
	shapeMultiReader:     "MultiReader",
	shapeMutator:         "Mutator",
	shapePointerReader:   "PointerReader",
	shapePoisonAccessor:  "PoisonCheck",
	shapePredicate:       "Predicate",
	shapePure:            "Pure",
	shapeReader:          "Reader",
	shapeReaderNoError:   "ReaderNoError",
	shapeReaderWithBool:  "ReaderWithBool",
	shapeStreamConsumer:  "StreamConsumer",
	shapeStreamReader:    "Stream",
	shapeVoidLifecycle:   "VoidLifecycle",
	shapeWriter:          "Writer",
}

// ContractActionFor returns the constructor that drives a contract role as
// itself rather than as whatever its raw signature shape selected, and
// whether the role names one. The rows are the family members whose fixture
// shapes the shipped constructors actually fit; the absences are verdicts,
// not gaps: AcquireLease renames Lifecycle for a keyless acquire and the
// lease fixture's acquire draws its key; Cursor drains to exhaustion and
// closes, which reddens the second invocation on a correct cursor-shaped
// subject; Persister compares save-answered IDs the persister fixture's
// writer never answers. A recording append keeps its recording closure —
// the rename touches the constructor, never the history log.
func ContractActionFor(contract, role string) (string, bool) {
	ctor, ok := contractActionCtors[contract+"."+role]
	return ctor, ok
}

// ContractActionConsumes returns the sibling roles a composite constructor
// claims: their methods are driven through the primary's action, and a
// standalone action on them would drive the terminal half with handles no
// begin ever minted — the defect the composite exists to retire.
func ContractActionConsumes(contract, role string) []string {
	return contractActionConsumes[contract+"."+role]
}

// RecordingActionFor returns the constructor that drives a shape AND files
// what it wrote into the run's log, and whether the shape has one.
//
// A separate constructor rather than a logging closure passed to the plain
// one: the closure is handed the subject and then the reference, so a log
// filled from inside it holds every write twice. A membership check cannot
// see the difference; a claim about how many elements a drain owes reads it
// as every element having been dropped.
//
// Keyed on the shape the action already drives, because that is what
// decides the constructor's own shape — the chain's takes a partition
// projection beside the log, and a plain write has no partitions to name.
func RecordingActionFor(shape string) (string, bool) {
	ctor, ok := recordingActions[shape]
	return ctor, ok
}

// RecordingActionRows returns the recording table for the census that holds
// each value to a shipped constructor.
func RecordingActionRows() map[string]string {
	return maps.Clone(recordingActions)
}

//nolint:gochecknoglobals // a lookup table, read-only after init.
var recordingActions = map[string]string{
	shapeWriter:                      "WriterRecording",
	contractChain + "." + roleAppend: "ChainAppendRecording",
}

// ContractActionRows returns the whole role table, keyed `contract.role`,
// for the census that holds each value to a shipped constructor in both
// directions.
func ContractActionRows() map[string]string {
	return maps.Clone(contractActionCtors)
}

// The tx trio's role spellings, as the carrier's directive names them.
const (
	roleBegin      = "begin"
	roleTxCommit   = "commit"
	roleTxRollback = "rollback"
)

//nolint:gochecknoglobals // a lookup table, read-only after init.
var contractActionCtors = map[string]string{
	contractUpdater + "." + roleWriter:  "Updater",
	contractUpserter + "." + roleWriter: "Upserter",
	contractCAS + "." + roleWriter:      "CompareAndSwap",
	contractChain + "." + roleAppend:    "ChainAppend",
	contractTx + "." + roleBegin:        "TwoPhase",
	contractPool + "." + roleGet:        "Pool",
}

//nolint:gochecknoglobals // a lookup table, read-only after init.
var contractActionConsumes = map[string][]string{
	contractTx + "." + roleBegin: {roleTxCommit, roleTxRollback},
	contractPool + "." + roleGet: {rolePut},
}

// MapStoreOp returns the [engine/model/ref.MapStore] method a shape delegates
// to, and whether the oracle models that shape at all.
//
// The oracle's methods were written to the shape signatures — `Get(ctx, K) (V,
// error)` is the reader shape verbatim — so a generated adapter forwards its
// parameters in order and changes nothing but the name. A shape with no row
// runs inert in the adapter, and the laws that would compare the subject
// against an inert body are not bound.
func MapStoreOp(shape string) (string, bool) {
	op, ok := mapStoreOps[shape]
	return op, ok
}

// MapStoreShapes returns every shape the oracle models, sorted.
//
// Exported for the two tests that close the table: the tiers side holds every
// key to the detector registry, and the conformance gate holds every value to
// a method the shipped oracle declares. Neither can iterate an unexported map.
func MapStoreShapes() []string {
	out := make([]string, 0, len(mapStoreOps))
	for s := range mapStoreOps {
		out = append(out, s)
	}
	slices.Sort(out)
	return out
}

// mapStoreOps maps the shapes MapStore models to its methods.
//
// The values spell the oracle's method names. Two also appear in the law
// catalogue as field names, and a shared constant would assert a unity that is
// not there — Put the oracle method and Put the law field are different
// things that happen to share a spelling, the catalogue's own stance one file
// over.
//
//nolint:gochecknoglobals,goconst // a lookup table, read-only after init; the
var mapStoreOps = map[string]string{
	shapeReader: "Get",
	// The same read, answering a presence flag instead of an error. The
	// oracle reports a miss as its sentinel, so the adapter folds the two
	// spellings together in one place rather than at every comparison — see
	// AdapterMethod.Folds.
	shapeReaderWithBool: "Get",
	shapeWriter:         "Put",
	shapeAggregator:     "Count",
	shapeStreamReader:   "List",
	ShapeCollector:      "Values",
}

// KeyedStoreOp returns the [engine/model/ref.KeyedStore] method a shape
// delegates to — the keyed-put oracle, chosen where the interface's writer
// takes the key beside the value.
//
// A plain writer has no row: a keyed store cannot place a value whose key is
// not an argument, so a writer-shaped method under this oracle runs inert
// unless a mixin says what it is — see [KeyedStoreMixinOp].
func KeyedStoreOp(shape string) (string, bool) {
	op, ok := keyedStoreOps[shape]
	return op, ok
}

// KeyedStoreShapes returns every shape the keyed oracle models, sorted, for
// the census that holds each value to a method the oracle declares.
func KeyedStoreShapes() []string {
	out := make([]string, 0, len(keyedStoreOps))
	for s := range keyedStoreOps {
		out = append(out, s)
	}
	slices.Sort(out)
	return out
}

// KeyedStoreMixinOp returns the oracle method a mixin assigns to its carrier,
// where the shape alone says too little.
//
// The one row is the delete: Delete(ctx, k) is writer-shaped — one
// non-context parameter — and nothing in the signature distinguishes it from
// a put. The deleteremoves mixin is what states the semantics, and the stamp
// outranks the shape the way a partner reference does.
func KeyedStoreMixinOp(mixin string) (string, bool) {
	op, ok := keyedStoreMixinOps[mixin]
	return op, ok
}

// The keyed oracle's tables, method names spelled here for the reason
// mapStoreOps spells its own.
//
//nolint:gochecknoglobals,goconst // lookup tables, read-only after init; the
var (
	keyedStoreOps = map[string]string{
		shapeReader:          "Get",
		shapeReaderWithBool:  "Get",
		shapeCompositeWriter: "Put",
		shapeAggregator:      "Count",
	}
	keyedStoreMixinOps = map[string]string{
		mixinDeleteRemoves: "Delete",
	}
)

// CollectionOp returns the [engine/model/ref.Collection] method a shape
// delegates to — the append-and-drain oracle behind the stream mixins, chosen
// where a value writer stands beside a collector and no reader keys anything.
//
// The collector spelling is the derived pseudo-shape [ShapeCollector]: an
// aggregator whose result is a slice, which the aggregator constructors
// cannot compare and [engine/model/action.Stream] drains.
func CollectionOp(shape string) (string, bool) {
	op, ok := collectionOps[shape]
	return op, ok
}

// ShapeCollector is the derived spelling for an aggregator-shaped method
// returning a slice.
//
// A pseudo-shape rather than a detector: eidos classifies the signature as an
// aggregator, and the slice result is a fact the generator reads off the
// declaration. It exists so the op tables and the action dispatch can name
// the case without either inventing annotator vocabulary — the census checks
// it never collides with a real detector.
const ShapeCollector = "collector"

//nolint:gochecknoglobals // a lookup table, read-only after init.
var collectionOps = map[string]string{
	shapeWriter:    "Add",
	ShapeCollector: "Items",
}

// CollectionDedupes reports whether the named mixin turns the collection
// oracle into its deduplicating form.
//
// noduplicates is the direct claim: a subject collapsing repeats is its whole
// point, and a plain log diverges from it — by design — at the second
// identical add. The stamp refines the oracle the way it refines delegation.
func CollectionDedupes(mixin string) bool {
	return dedupingMixins[mixin]
}

//nolint:gochecknoglobals // a lookup table, read-only after init.
var dedupingMixins = map[string]bool{
	mixinNoDuplicates: true,
}

// DrainsHistory reports whether the named classification — mixin or contract
// role — marks the drained slice as an event log rather than a store's
// holdings.
//
// The rows are the history vocabularies: snapshotisolation records events,
// and chain replays an append-only log. Identical events repeat, and an
// entry's Key field names the key an operation touched — not the entry's
// identity. The upsert inference reads a conventional ID or Key field as
// identity, which is right for stores and wrong for logs; the claim is what
// tells them apart, and the corpus proved it — the first same-key pair of
// events collapsed the inferred map to one entry while the subject
// faithfully held both.
//
// overmatch and permutation are here for the same reason from the other
// direction. Both are claims about what a drain OWES the writes behind it:
// permutation says exactly what went in comes out, overmatch says at least
// that much. A subject holding a repeat is what each declares, and an
// upsert map inferred from a Key field drops it — so the reference would
// diverge from a correct subject on the second write to one key, and the
// claim would be judged against a model that contradicts it.
func DrainsHistory(classification string) bool {
	return historyDrains[classification]
}

//nolint:gochecknoglobals // a lookup table, read-only after init.
var historyDrains = map[string]bool{
	mixinSnapshotIsolation: true,
	mixinSerializable:      true,
	mixinOverMatch:         true,
	mixinPermutation:       true,
	contractChain:          true,
}

// DefeatsOracles reports whether the named mixin's claim puts the subject
// beyond any immediate store model, with the reason the generated header
// prints.
//
// eventually: a read may lag a write until something forces convergence, and
// every derived oracle answers immediately — the first publish an eventual
// subject had not yet surfaced read as a divergence. crdtmerge: the merge
// relation is the semantics, and every store oracle holds it inert — the
// corpus proved it when the convergence law red-lined the derived adapter,
// whose inert merge can never converge. The twin floor is the honest model
// both times: two instances driven identically diverge identically.
func DefeatsOracles(mixin string) (OracleDefeat, bool) {
	defeat, defeated := oracleDefeats[mixin]
	return defeat, defeated
}

// OracleDefeat is why a claim defeats store modelling, and what lifts it.
type OracleDefeat struct {
	// Why is the reason the header prints beside the twin.
	Why string

	// LiftedByEvictingRead marks a defeat that holds only while every read
	// has to answer for everything the subject is holding.
	//
	// A bound is the case, and it is one claim over two shapes. A bounded
	// COLLECTION answers a list and an unbounded oracle answers a longer
	// one, so the first clamped read is a disagreement and the twin is the
	// only honest floor. A bounded CACHE answers one key and may say no,
	// and a subject miss is what eviction looks like rather than a wrong
	// answer — so comparing hits alone leaves the oracle free to be
	// unbounded, which is what lets it catch a subject that invented one.
	//
	// The defeat is therefore a fact about the stamp and the read shape
	// together. Recorded here rather than at the reader, so the reason and
	// the condition that undoes it stay in one place.
	LiftedByEvictingRead bool
}

//nolint:gochecknoglobals // a lookup table, read-only after init.
var oracleDefeats = map[string]OracleDefeat{
	// The accumulates claim is that a repeated write COMPOUNDS, and every
	// store oracle replaces: the second write to one key leaves the
	// subject holding a sum and the store holding the last argument, so a
	// correct subject diverges on the first re-write. Its reader diverges
	// sooner — an accumulator answers zero for a key nothing added to,
	// where a store answers a miss. Twins compound together.
	mixinAccumulates: {Why: "the accumulates claim is that a repeated write compounds, " +
		"and every store oracle replaces"},
	mixinEventually: {Why: "the eventually claim lets reads lag writes, which no immediate store models"},
	mixinCRDTMerge:  {Why: "the merge relation is the claim, and every store oracle holds it inert"},
	// The atomic claim is about refused writes: the subject rejects by policy
	// and a derived map accepts everything, so the first refusal reads as a
	// semantic disagreement on a correct subject. Twins share the policy.
	mixinAtomic: {Why: "the atomic claim is about refused writes, and a derived store refuses nothing"},
	// The causal claim is an admission policy — an entry lands only after
	// its dependencies — and a derived log admits everything, so a correct
	// subject's first refusal reads as disagreement. Twins share the policy.
	// (The session fixture reaches the twin through its version stamp
	// already; this row covers causal claims with no stamp, like the
	// chain's.)
	mixinCausal: {Why: "the causal claim is an admission policy, and a derived store admits everything"},
	// The bounded claim clamps what its reader answers; a derived
	// collection answers everything it was fed, so a correct subject's
	// first clamped read counts as disagreement. Twins clamp together.
	//
	// Lifted where the reader answers one key and may say no — see
	// [OracleDefeat.LiftedByEvictingRead], which sets out the asymmetry
	// that makes an unbounded oracle usable there.
	mixinBounded: {
		Why: "the bounded claim clamps what the reader answers, and a derived " +
			"collection clamps nothing",
		LiftedByEvictingRead: true,
	},
	// An isolation level is an admission policy before it is anything
	// else: a store claiming it refuses the operation that would put an
	// anomaly in its history, and a derived log records whatever it is
	// handed. So a correct subject's first refusal reads as disagreement,
	// and — worse — a passive log fed arbitrary draws fabricates the very
	// anomalies the laws exist to find, failing a subject that did nothing
	// wrong. Twins share the policy and refuse together.
	mixinSnapshotIsolation: {Why: "the isolation claim is an admission policy, and a derived log records anything"},
	mixinSerializable:      {Why: "the serializable claim is an admission policy, and a derived log records anything"},
}

// MapStorePins reports whether the named mixin turns the map oracle into its
// resolution-pinning form.
//
// The one row is sticky: once a key resolves, it keeps resolving to the same
// value, and a latest-write-wins map diverges from that — by design — at the
// first read after an overwrite. The corpus proved the row the day the pools
// grew wide enough to draw one.
func MapStorePins(mixin string) bool {
	return mixin == mixinSticky
}
