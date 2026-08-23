// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package lawid

import (
	"slices"
	"strings"
)

// DefectClass is a kind of wrongness a law is able to see.
//
// The axis the saturation prover was missing. That prover asks whether *some*
// defect worn on a law's own methods makes it fail, which proves the law is not
// a tautology and nothing more — `AUTO-POINT-IN-TIME` was killed by a
// read-purity flap and `AUTO-CAS-ATOMIC-ONE-WINNER` by a Put that did nothing,
// and both rows were green. A law that only ever fails for a reason its name
// does not mention is a law whose name is a guess.
//
// So each law declares the class its identifier promises, each wear in the
// wardrobe declares the class it expresses, and a kill counts when the two
// intersect. What that turns "bound but unsaturatable" into is "bound, and
// nothing in the wardrobe can produce the defect this law is named for" —
// which is a gap in the wardrobe rather than in the law, and is worth knowing
// apart.
type DefectClass string

// The classes, one per shape of wrongness the wardrobe can express or a law
// can name.
//
// Derived from the two populations rather than invented: every class here is
// both something at least one wear produces and something at least one law
// claims to catch. A class only the wardrobe has is a defect no law is looking
// for; a class only the laws have is a promise nothing can test — and
// [TestEveryClassIsReachable] and the wardrobe's own census hold both ends.
const (
	// ClassNoEffect is an operation that does nothing: the write that does not
	// write, the transform that returns its input.
	ClassNoEffect DefectClass = "no-effect"

	// ClassInstability is an answer that changes when the claim says it holds
	// still — the flickering read, the impure computation.
	ClassInstability DefectClass = "instability"

	// ClassRegression is a value that was supposed to only move one way and
	// moved back.
	ClassRegression DefectClass = "regression"

	// ClassDuplication is one thing arriving twice: the redelivered message,
	// the page that repeats a row, the callback fired again.
	ClassDuplication DefectClass = "duplication"

	// ClassLoss is one thing not arriving at all — the dropped record, the
	// write a later write silently swallowed.
	ClassLoss DefectClass = "loss"

	// ClassOrdering is the right things in the wrong sequence.
	ClassOrdering DefectClass = "ordering"

	// ClassBound is a quantity outside the range something declared for it.
	ClassBound DefectClass = "bound"

	// ClassIsolation is state crossing a boundary that was supposed to hold
	// it: the partition that leaks, the transaction seen mid-flight.
	ClassIsolation DefectClass = "isolation"

	// ClassRepeatability is an operation that works once and will not work
	// again — the close that is not idempotent, the release that refuses a
	// second call.
	ClassRepeatability DefectClass = "repeatability"

	// ClassStaleness is an answer from the past: the read that misses a write
	// it should see, the snapshot that outlives its instant.
	ClassStaleness DefectClass = "staleness"

	// ClassSpuriousFailure is a refusal where the claim says the call must
	// succeed.
	ClassSpuriousFailure DefectClass = "spurious-failure"

	// ClassIntegrity is a value that was altered on the way through, where the
	// claim is that it was not.
	ClassIntegrity DefectClass = "integrity"

	// ClassAtomicity is a partial effect where the claim is all or nothing.
	ClassAtomicity DefectClass = "atomicity"

	// ClassResource is something taken and not given back.
	ClassResource DefectClass = "resource"

	// ClassPermissive is an operation that succeeds where the claim requires a
	// refusal — the read after close that answers, the second acquire that
	// grants, the terminal op a committed transaction accepts.
	//
	// The class the first taxonomy missed, and it was missed for a reason
	// worth recording: every other class here is a *wrong answer*, and this
	// one is a *missing refusal*. `stick` — the wear that makes an operation
	// refuse after its first call — reads as the obvious defect for a
	// close-then-use law and is in fact that law's desired behaviour, which is
	// why four laws were being proved by a wear they should have been immune
	// to. Nothing in the wardrobe produces it.
	ClassPermissive DefectClass = "permissive"
)

// Classes returns every class, for a census that has to be total over them.
func Classes() []DefectClass {
	return []DefectClass{
		ClassNoEffect, ClassInstability, ClassRegression, ClassDuplication,
		ClassLoss, ClassOrdering, ClassBound, ClassIsolation,
		ClassRepeatability, ClassStaleness, ClassSpuriousFailure,
		ClassIntegrity, ClassAtomicity, ClassResource, ClassPermissive,
	}
}

// ClassOf returns the defect classes the named law is able to see, empty for
// an identifier no rule claims.
//
// # Declared rather than derived, and why the cheaper design lost
//
// Reading the class out of the identifier's own words would give one source
// that cannot drift: `NO-DUPLICATES` says duplication, `MONOTONIC` says
// regression, and a law renamed would take its class with it. That was the
// first design, and measuring it is what settled the question — a keyword
// table over every defect-bearing word in the catalogue classifies 54 of the
// 83 laws and cannot reach the other 29.
//
// The 29 are not a tail. `AUTO-READ-AFTER-WRITE`, `AUTO-WRITES-FOLLOW-READS`,
// `AUTO-POINT-IN-TIME`, `AUTO-COMMUTATIVE-WRITE` name a *relation between
// operations* rather than a defect, so no word in them carries a class — and
// the words they do carry belong to their subject, where teaching `READ` to
// mean staleness would misclassify `AUTO-MONOTONIC-READS` next door.
//
// So the classes are declared, and the drift the derivation would have
// prevented is caught a different way: [ClassFromName] is used as a check
// rather than as a source, holding a declaration to the identifier wherever
// the identifier says something. That is 54 of the 83; the other 29 are
// declared against nothing and are
// listed as such, so a reader reviewing this table knows which rows a machine
// checked and which rows a person has to.
func ClassOf(id string) []DefectClass {
	out := slices.Clone(defectClasses[id])
	slices.Sort(out)
	return out
}

// defectClasses is what each law's identifier promises it can see.
//
// Eighty-three judgements, written out rather than generated, because a
// generated table hides which rows anybody looked at. Most carry one class;
// several carry two, and the second is not decoration — `AUTO-PUBLISHER-
// EXACTLY-ONCE` fails for a message that never arrived and for one that
// arrived twice, and a wardrobe offering only the first would prove half of it.
//
// [TestClassesAgreeWithTheName] holds 54 of these against the words in their
// own identifier. The other 29 name a relation rather than a defect and are
// declared against nothing a machine can read, which is the cost of the
// design and is stated in [ClassOf].
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var defectClasses = map[string][]DefectClass{
	AggregatorBounded:            {ClassBound},
	AppendOnlyGrows:              {ClassRegression, ClassLoss},
	AppendOnlyNoDrops:            {ClassLoss},
	AppenderMonotonicOffsets:     {ClassRegression, ClassDuplication},
	Associative:                  {ClassOrdering},
	AtomicWrite:                  {ClassAtomicity},
	Cacheable:                    {ClassInstability, ClassStaleness},
	CASAtomicOneWinner:           {ClassPermissive, ClassSpuriousFailure},
	CausalOrdering:               {ClassOrdering},
	CommutativeWrite:             {ClassOrdering, ClassLoss},
	Conservative:                 {ClassBound},
	CountEqualsReference:         {ClassLoss, ClassDuplication},
	CRDTMerge:                    {ClassLoss},
	CursorCloseIdempotent:        {ClassRepeatability},
	CursorNextAfterClose:         {ClassPermissive},
	DeadlineRespecting:           {ClassBound},
	DefaultOnError:               {ClassIntegrity},
	DeleteReturnsNotFound:        {ClassNoEffect, ClassStaleness},
	EventualConvergence:          {ClassInstability},
	HashChainIntegrityErr:        {ClassIntegrity},
	HashChainIntegrityVerify:     {ClassIntegrity, ClassSpuriousFailure},
	IdempotentLifecycle:          {ClassRepeatability},
	IdempotentWrite:              {ClassRepeatability, ClassDuplication},
	InjectionSafe:                {ClassIntegrity},
	LeakFree:                     {ClassResource},
	LeaseDoubleAcquireBlocks:     {ClassPermissive},
	LeaseReleasedOnCancel:        {ClassResource},
	LifecycleAfterClose:          {ClassPermissive},
	LifecycleRespectsContext:     {ClassPermissive},
	LossyRoundtrip:               {ClassInstability, ClassIntegrity},
	MonotonicNonDecreasing:       {ClassRegression},
	MonotonicReads:               {ClassRegression, ClassStaleness},
	MonotonicWrites:              {ClassRegression, ClassOrdering},
	PaginatorNoDuplicates:        {ClassBound, ClassDuplication},
	PaginatorResumable:           {ClassLoss, ClassRepeatability},
	PersisterRetrievable:         {ClassLoss, ClassIntegrity},
	PointInTime:                  {ClassStaleness, ClassInstability},
	PoisonConsistent:             {ClassInstability},
	PoisonIdempotentRead:         {ClassRepeatability, ClassInstability},
	PoisonNilOnFresh:             {ClassSpuriousFailure},
	PoolBalanced:                 {ClassResource},
	PoolLeakFree:                 {ClassResource},
	PredicateConsistent:          {ClassInstability},
	PublisherAtLeastOnce:         {ClassLoss},
	PublisherAtMostOnce:          {ClassDuplication},
	PublisherDelivers:            {ClassLoss},
	PublisherDelivery:            {ClassLoss, ClassDuplication},
	PublisherExactlyOnce:         {ClassLoss, ClassDuplication},
	PureDeterministic:            {ClassInstability},
	ReadAfterWrite:               {ClassStaleness, ClassNoEffect},
	ReadYourWrites:               {ClassStaleness, ClassIsolation},
	ReplayCausalOrdering:         {ClassOrdering},
	ReplayDeterministic:          {ClassInstability},
	Roundtrip:                    {ClassIntegrity},
	SagaFullCompensation:         {ClassAtomicity},
	ScheduledFiresAfterAdvance:   {ClassLoss},
	TimeawareMoves:               {ClassLoss},
	SingleflightCoalesces:        {ClassDuplication},
	SnapshotIsolationG0:          {ClassIsolation, ClassLoss},
	SnapshotIsolationG1:          {ClassIsolation},
	SnapshotIsolationG2:          {ClassIsolation},
	Sticky:                       {ClassInstability},
	StreamCompletion:             {ClassBound, ClassLoss},
	StreamNoDuplicates:           {ClassDuplication},
	StreamOverMatch:              {ClassLoss},
	StreamPermutation:            {ClassOrdering, ClassLoss},
	StreamReentrant:              {ClassInstability, ClassRepeatability},
	StreamReflectsMutations:      {ClassNoEffect, ClassStaleness},
	StreamStableOrder:            {ClassOrdering},
	TamperEvident:                {ClassNoEffect, ClassIntegrity},
	TotalOver:                    {ClassSpuriousFailure},
	TransactionNoMidTxVisibility: {ClassIsolation},
	TransactionRollback:          {ClassAtomicity},
	TTLExpiry:                    {ClassStaleness},
	TwoPhaseMutex:                {ClassPermissive},
	TwoPhaseRollbackAfterCommit:  {ClassAtomicity},
	UpdaterReplaces:              {ClassNoEffect, ClassStaleness},
	UpserterIdempotent:           {ClassRepeatability, ClassDuplication},
	ValidTransition:              {ClassOrdering},
	WatcherReturnsOnChange:       {ClassLoss, ClassStaleness},
	Windowed:                     {ClassBound},
	WriteObservable:              {ClassNoEffect},
	WritesFollowReads:            {ClassOrdering, ClassStaleness},
	XSSSafe:                      {ClassIntegrity},
}

// nameSays maps the defect-bearing words a law identifier can carry to the
// class each promises.
//
// Only the words that name a *defect*. A law's subject — STREAM, PUBLISHER,
// POOL, CURSOR — says what it is about and nothing about what going wrong
// looks like, and a table that classified those would answer for every law and
// mean nothing.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var nameSays = map[string]DefectClass{
	"DUPLICATES": ClassDuplication, "COALESCES": ClassDuplication,
	"MONOTONIC": ClassRegression, "GROWS": ClassRegression,
	"ORDERING": ClassOrdering, "ORDER": ClassOrdering, "PERMUTATION": ClassOrdering,
	"ISOLATION": ClassIsolation, "VISIBILITY": ClassIsolation,
	"LEAK": ClassResource, "BALANCED": ClassResource,
	"IDEMPOTENT": ClassRepeatability, "RESUMABLE": ClassRepeatability,
	"REENTRANT":     ClassRepeatability,
	"DETERMINISTIC": ClassInstability, "CONSISTENT": ClassInstability,
	"PURE": ClassInstability, "CONVERGENCE": ClassInstability,
	"BOUNDED": ClassBound, "WINDOWED": ClassBound, "CONSERVATIVE": ClassBound,
	"DEADLINE":  ClassBound,
	"ROUNDTRIP": ClassIntegrity, "TAMPER": ClassIntegrity, "INTEGRITY": ClassIntegrity,
	"SAFE": ClassIntegrity,
	// ATOMIC is deliberately absent. In `AUTO-ATOMIC-WRITE` it names the
	// defect; in `AUTO-CAS-ATOMIC-ONE-WINNER` it names the operation, whose
	// defect is two winners. One word, two jobs — the same ambiguity that
	// keeps ONCE out, where AT-LEAST-ONCE is loss and AT-MOST-ONCE is
	// duplication.
	"ROLLBACK": ClassAtomicity, "COMPENSATION": ClassAtomicity,
	"DROPS": ClassLoss, "DELIVERS": ClassLoss, "COMPLETION": ClassLoss,
	"MERGE":  ClassLoss,
	"EXPIRY": ClassStaleness, "REFLECTS": ClassStaleness,
	"TOTAL":       ClassSpuriousFailure,
	"RETRIEVABLE": ClassLoss, "OBSERVABLE": ClassNoEffect,
}

// ClassFromName reads the classes a law's identifier names outright, and
// whether it named any.
//
// The check the declared table is held against, not the source it is built
// from — see [ClassOf] for why that direction lost. Twenty-nine identifiers
// name a relation between operations rather than a defect and are unreadable
// here by construction; `AUTO-READ-AFTER-WRITE` carries no word about what
// going wrong looks like, and the words it does carry belong to its subject.
func ClassFromName(id string) ([]DefectClass, bool) {
	var out []DefectClass
	for word := range strings.SplitSeq(strings.TrimPrefix(id, "AUTO-"), "-") {
		if c, says := nameSays[word]; says && !slices.Contains(out, c) {
			out = append(out, c)
		}
	}
	slices.Sort(out)
	return out, len(out) > 0
}
