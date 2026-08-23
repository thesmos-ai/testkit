// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/suite"
)

// The emit kinds and template names for the bodies this tier's rows run.
//
// A kind per leg, the way the doors are a kind per capability. The two
// legs share their inputs and almost nothing else: the differential runs
// with no law registered, so a disagreement is what ends the run; the
// laws leg runs the same sequences with that verdict switched off, so the
// laws are the only thing that can. One template carrying both behind a
// conditional would put the reason for the split where nobody reading
// either body can see it.
const (
	KindDifferentialLeg sdk.Kind = "model.leg.differential"
	KindLawsLeg         sdk.Kind = "model.leg.laws"
	KindOwnLeg          sdk.Kind = "model.leg.own"
	KindClockedLeg      sdk.Kind = "model.leg.clocked"
)

// leg is what every body needs and no template can work out for itself.
type leg struct {
	sdk.BaseEmit

	// Assert is the function this leg declares — the one the row's
	// RunWith calls.
	Assert string

	// Subject is the interface as the harness file spells it, for the
	// runtime Subject the body receives.
	Subject string

	// Draws says the leg takes the run's sample inputs; Fixture is the
	// parameter's name and FixtureType its type. False where nothing in
	// the tier reads the fixture, in which case a parameter for it would
	// not compile.
	Draws                bool
	Fixture, FixtureType string

	// Actions names the function yielding the operation vocabulary both
	// legs drive, and Laws the one yielding the bound laws.
	Actions, Laws string

	// Ref is the derived reference's constructor and RefExpr the one the
	// directive named; both are empty for the twin floor, where the
	// subject's own factory stands in and the body spells it.
	Ref     string
	RefExpr *sdk.Expr

	// History is the append log the recording actions fill, empty where no
	// law reads one, and HistoryElem what it holds. The leg declares it,
	// hands it to the actions and resets it each iteration: a law comparing
	// against a history the actions never wrote into is a law over an empty
	// record.
	History     string
	HistoryElem sdk.Ref

	// Factory says a bundled law builds instances of its own — a merge
	// claim compares two replicas, and no observation over one states it.
	// The leg has the subject and hands the factory over.
	Factory bool

	// The packages a body reaches, so a template asks rather than spells.
	Vocab, LegsPkg, ModelPkg, HistoryPkg, LawPath string
}

// Twin reports the floor where no independent reference exists: the leg
// builds its comparison instance from the subject's own factory.
func (l leg) Twin() bool { return l.Ref == "" && l.RefExpr == nil }

// DifferentialLeg is the body of the row claiming the subject and the
// reference agree over every sequence.
type DifferentialLeg struct{ leg }

// Kind returns the template this body renders through.
func (*DifferentialLeg) Kind() sdk.Kind { return KindDifferentialLeg }

// LawsLeg is the body of the row claiming every bundled law holds over
// the same sequences.
type LawsLeg struct{ leg }

// Kind returns the template this body renders through.
func (*LawsLeg) Kind() sdk.Kind { return KindLawsLeg }

// OwnLeg is the body of a row carrying one law the shared sequences
// cannot: it needs a lifecycle probe, a poisoned subject, or writes of
// its own, and the bundle provides none of those.
type OwnLeg struct {
	leg

	// Laws are the bindings this leg registers — every binding of the
	// one law its row claims. Rendered inline rather than through the
	// bundle's function, because a clocked law's Advance reads a local
	// the leg declares and a literal spelled outside that function names
	// nothing.
	Laws []*LawBinding

	// Drain is the publisher sweep a delivery law reads, empty for every
	// other law. Declared by whichever body registers the law, which is
	// this one now that a worded law has a leg of its own.
	Drain string

	// Builds says the law builds subjects of its own — an algebraic
	// claim comparing two folds, a freshness claim wanting an untouched
	// one — and Coalesces that it counts its own invocations under a
	// lock. Both are locals the bundle declared and this leg now owes.
	Builds, Coalesces bool

	// Keys and Values say which shared pools the law draws, and Pools the
	// ones it brings of its own. A local for a pool this law does not
	// read would not compile.
	Keys, Values bool
	Pools        []LawPool

	// KeysFunc and ValuesFunc name the pool constructors the locals call.
	KeysFunc, ValuesFunc string
}

// Kind returns the template this body renders through.
func (*OwnLeg) Kind() sdk.Kind { return KindOwnLeg }

// ClockedLeg is [OwnLeg] for a law that moves time: the subject is built
// on a clock the law advances, which is the one capability this tier asks
// a consumer for.
type ClockedLeg struct {
	OwnLeg

	// Clock is the controllable clock's package, whose test clock the
	// factory builds and the law's Advance moves.
	Clock string
}

// Kind returns the template this body renders through.
func (*ClockedLeg) Kind() sdk.Kind { return KindClockedLeg }

// KindDeclined is the emit kind and template for a refusal: the interface
// asked for this tier and the harness it has cannot carry it.
const KindDeclined sdk.Kind = "model.declined"

// Declined is that refusal, rendered where the rows would have been.
//
// In the generated file rather than in the generator's diagnostics,
// because the question a reader asks is "why does this package have no
// model checks", and they ask it while reading the package. A directive
// that was honoured everywhere else and silently dropped here is the one
// absence nothing else in the output would show.
type Declined struct {
	sdk.BaseEmit

	// Iface is the interface that asked.
	Iface string

	// Why is the reason it cannot be given, one comment line per element.
	//
	// Wrapped at the source rather than here, because the only correct
	// place to break a sentence is where its author would break it, and a
	// column count in a template does not know that.
	Why []string
}

// Kind returns the template this refusal renders through.
func (*Declined) Kind() sdk.Kind { return KindDeclined }

// KindConcurrent is the emit kind and template for the linearizability
// leg: workers driving one instance at once, the recorded history
// checked against the model this interface's shape selects.
const KindConcurrent sdk.Kind = "model.leg.concurrent"

// KindSimLeg is the body behind a row claiming the subject's state
// survives the crash seam.
const KindSimLeg sdk.Kind = "model.leg.sim"

// SimLeg is that body: a drawn interleaving of writes, reads and crash
// points, every read judged against what the writes acknowledged.
//
// Far less to carry than the concurrent leg, and for a reason worth
// stating: there is no model to select and no reference to pick. The
// subject is judged against its own acknowledgements, so the leg needs
// only the pair of verbs, the pools they draw from, and the projection
// saying which key a write lands on.
type SimLeg struct {
	leg

	// Reader and Writer are the two ops the schedule drives, pointing at
	// the same actions the sequential legs use.
	Reader, Writer *Action

	// Key and Value are the store's two types, for the schedule's own type
	// arguments — the oracle is a map from one to the other, and Go will
	// not infer either from a closure.
	Key, Value sdk.Ref

	// Miss is the declaration's stamped sentinel for a read of what is not
	// there, nil where it stamps none — which the schedule reads as "any
	// error means absent", the lenient reading for a contract that names
	// no miss.
	Miss *sdk.Expr

	// CrashPkg is where the schedule and its oracle live, named rather than
	// spelled because an import path built inside a template is one the
	// backend cannot register.
	CrashPkg string

	// KeysFunc and ValuesFunc are the shared pools the schedule draws from,
	// and KeyOfName the projection a write is filed under — the same three
	// the sequential legs use.
	KeysFunc, ValuesFunc, KeyOfName string

	// Iface is the subject's own interface type, for the closures' second
	// parameter: the schedule speaks the declaration's type where the leg
	// speaks the harness's.
	Iface sdk.Ref
}

// Kind returns the template this leg renders through.
func (*SimLeg) Kind() sdk.Kind { return KindSimLeg }

// ConcurrentLeg is that body.
//
// Family names which model steps the history, spelled as the tiers
// vocabulary spells it. The rest is what each family's constructor takes
// and no other's does, which is why they sit beside each other rather
// than behind one field: the lease model takes two sentinels and no type
// arguments, the cell model takes a version projection, and a template
// arm that shared a slot between them would render the wrong one without
// saying so.
type ConcurrentLeg struct {
	leg

	// Family selects the arm: "kv", "lease", "cas", "append", "session".
	Family string

	// Reader and Writer are the two ops the history records, as the
	// canonical names the shipped models switch on rather than the
	// subject's own — the models dispatch by name, and a mismatch steps
	// nothing rather than erroring.
	Reader, Writer *Action

	// Acquire and Release are the lease family's pair, in its place.
	Acquire, Release *Action

	// Key and Value are the keyed store's two types, Entry the append
	// log's, and Version the cell's stamp.
	Key, Value, Entry, Version sdk.Ref

	// Miss is the declaration's stamped sentinel for a read of what is not
	// there, nil where it stamps none — which the shipped models read as
	// "any error means absent" rather than as a match on nothing.
	Miss *sdk.Expr

	// Errs are the contract oracle's constructor sentinels, in the order
	// its spec declares them: the lease's held then free, the cell's
	// mismatch then empty. The arms index by position because the position
	// IS the contract — the spec's own slice is what the live oracle's
	// constructor is called with, and a second ordering here would be a
	// second chance to get it wrong.
	Errs []CtorErr

	// VersionField is the member the cell's stamp lives on, for the
	// projection the cas model steps with.
	VersionField string

	// LinearizePkg is where the shipped models and the concurrent action
	// constructors live, and PorcupinePkg where the stepless model's own
	// type is declared — named because the session arm renders a zero of
	// it, which is how the runner is told to skip the search and let the
	// per-client laws carry the verdict.
	LinearizePkg, PorcupinePkg string

	// KeysFunc and ValuesFunc are the shared pools the recorded ops draw
	// from, and KeyOfName the projection a keyed write partitions by —
	// the same three the sequential legs use, so the two halves of a run
	// cannot disagree about what a key is.
	KeysFunc, ValuesFunc, KeyOfName string

	// DrawsKeys and DrawsValues say which of the two pools this leg's ops
	// declare, because a local nothing reads does not compile. ReaderPool
	// and WriterPool name the one each op draws from.
	//
	// Read off the actions rather than fixed per family: a compare-and-set
	// and an offset-answering append both draw their argument from the
	// KEYS pool, because that is the only pool their declaration produced,
	// and a leg that assumed values would name a function nothing
	// declares.
	DrawsKeys, DrawsValues bool
	ReaderPool, WriterPool string

	// SessionLaws are the per-client laws a stepless family's verdict
	// comes from, empty for a family whose model steps the history itself.
	// See [Bindings.SessionLaws].
	SessionLaws []*LawBinding

	// WriterAnswers marks a write that hands the stored state back — how a
	// version-stamping store returns its stamp, and the value the
	// per-client laws order reads against.
	WriterAnswers bool

	// Iface is the subject's own interface type, for the closures' second
	// parameter: the actions speak the declaration's type where the
	// config speaks the harness's.
	Iface sdk.Ref
}

// Kind returns the template this leg renders through.
func (*ConcurrentLeg) Kind() sdk.Kind { return KindConcurrent }

// KindCompat is the emit kind and template for the legs witness.
const KindCompat sdk.Kind = "model.compat"

// Compat is the version-skew witness for the leg package this tier's
// declarations ride.
//
// The harness's own witness covers the check vocabulary, and nothing
// covered the legs: a generated file calls legs.Law and legs.Differential
// on every model row it declares, and a breaking change to either would
// have compiled against a file generated before it and run the wrong
// idiom quietly. The suite's CompatV2 says the check format matches; this
// says the leg contract does.
//
// Emitted only where this tier contributed rows, because a file that
// names no leg has no skew to guard against and importing one for the
// witness alone would be an import nothing uses.
type Compat struct {
	sdk.BaseEmit

	// LegsPkg is the bridge package whose witness this names, carried
	// rather than spelled in the template because an import path built
	// inside a template is one the backend cannot register.
	LegsPkg string
}

// Kind returns the template the witness renders through.
func (*Compat) Kind() sdk.Kind { return KindCompat }

// drawsFixture reports whether this tier's declarations take the run's
// sample inputs.
//
// Both halves, and one answer for the row, the declaration and every leg.
// The tier has to want them — a parameter nothing reads does not compile —
// and the harness has to have them, because the appended expression is
// spelled inside that generator's function and can only name what it
// declares. Asking the two questions separately is how a call and a
// signature come to disagree in a file nobody may edit.
func drawsFixture(b *Bindings, harness *suite.Contract) bool {
	return b.NeedsFixture() && harness.DrawsFixture
}

// historyIdent is the append log's name inside a leg, spelled to match
// what the recording action's own template reads. The action renders
// inside the function this hands it to, so the two agree here or the
// generated file names an undeclared identifier.
const historyIdent = "appendHist"

// legsFor is the body each planned row runs, in plan order.
//
// Driven off the rows rather than off the bindings, so a row without a
// body and a body without a row are both impossible: the runtime refuses
// a check setting neither Run nor RunWith by name, and a function nothing
// calls does not compile in a file the consumer may not edit.
func legsFor(b *Bindings, harness *suite.Contract) []sdk.EmitNode {
	base := leg{
		BaseEmit:    b.BaseEmit,
		Subject:     harness.SubjectType(),
		Draws:       drawsFixture(b, harness),
		Fixture:     fixtureIdent,
		FixtureType: harness.Fixture.TypeName,
		Actions:     b.ActionsFuncName(),
		Laws:        b.LawsFuncName(),
		Factory:     b.LawsNeedFactory(),
		Vocab:       VocabPkg,
		LegsPkg:     LegsPkg,
		ModelPkg:    ModelPkg,
		HistoryPkg:  HistoryPkg,
		LawPath:     LawPkg,
	}
	switch {
	case b.Reference.Supplied():
		base.RefExpr = b.Reference.SuppliedCtor
	case !b.Reference.Twin():
		base.Ref = b.Reference.CtorName
	}
	if b.RecordsHistory {
		base.History, base.HistoryElem = historyIdent, b.HistoryElem
	}

	// The own-leg laws by identity, because a row for one names it in the
	// segment of its own ID — which is how a body finds the law it is the
	// body of without re-running the selection.
	own := map[string]*LawBinding{}
	for _, l := range b.RowedLaws() {
		own[l.ID] = l
	}

	out := make([]sdk.EmitNode, 0, len(b.Rows))
	for _, p := range b.Rows {
		l := base
		l.Assert = rowAssertName(projection.Token(b.IfaceName), p.ID)
		law, single := own[p.ID.Seg]
		switch {
		case single:
			out = append(out, ownLegFor(b, l, law))
		case p.Body.BodyKind() == projection.DifferentialLeg{}.BodyKind():
			out = append(out, &DifferentialLeg{leg: l})
		case p.Body.BodyKind() == projection.ConcurrentLeg{}.BodyKind():
			out = append(out, concurrentLegFor(b, l))
		case p.Body.BodyKind() == projection.SimLeg{}.BodyKind():
			out = append(out, simLegFor(b, l))
		default:
			out = append(out, &LawsLeg{leg: l})
		}
	}
	return out
}

// simLegFor is the crash schedule's body.
func simLegFor(b *Bindings, base leg) sdk.EmitNode {
	l := &SimLeg{
		leg:        base,
		Reader:     b.SimReader,
		Writer:     b.SimWriter,
		Miss:       b.Reference.MissSym,
		CrashPkg:   CrashPkg,
		KeysFunc:   b.KeysFuncName(),
		ValuesFunc: b.ValuesFuncName(),
		KeyOfName:  b.KeyOfName(),
	}
	if b.Keys.Type != nil {
		l.Key = b.Keys.Type
	}
	if b.Values.Type != nil {
		l.Value = b.Values.Type
	}
	for _, a := range []*Action{l.Reader, l.Writer} {
		if a != nil && a.Iface != nil {
			l.Iface = a.Iface
			break
		}
	}
	return l
}

// Sim reports whether the crash-recovery pair wired.
func (b *Bindings) Sim() bool { return b.SimReader != nil && b.SimWriter != nil }

// simLegReason is why this interface's crash claim cannot be spelled as
// a leg, empty where it can.
//
// The schedule holds every acknowledged write to a later read, so it can
// only run where an acknowledgement means "this key now holds this
// record until something else writes it". Each refusal below is a store
// that breaks that sentence honestly, and running the schedule against
// one would red correct code rather than find a lost write — which is
// worse than not stating the claim, because a red nobody can fix is a
// red everybody learns to ignore.
func simLegReason(b *Bindings) string {
	switch {
	case b.Reference.Supplied():
		return "the directive names the reference, and a crash claim needs to know what a " +
			"write installs — which is exactly what a supplied constructor keeps to itself"
	case b.Reference.Oracle != OracleMap:
		return "an acknowledged write here does not simply sit at its key until something " +
			"overwrites it, and a schedule holding it to that would red correct code"
	case b.Reference.Pins:
		return "the first write to a key pins it, so a later acknowledged write is one the " +
			"store never promised to answer with"
	case b.Reference.Dedupe:
		return "a duplicate write is acknowledged and not installed, and the schedule holds " +
			"every acknowledgement to a read"
	case b.Session != nil || b.Reference.VersionField != "":
		return "the store stamps what it stores, so a read answers a record the write did " +
			"not hand it and no comparison of the two states the claim"
	case !b.NeedsKeysPool() || !b.NeedsValuesPool():
		return "the schedule draws a key to read and a record to write, and this run " +
			"declares no pool for one of them"
	case b.Reference.KeyField == "":
		return "a write is filed under the key it lands on, and no projection derives " +
			"which member that is"
	}
	return ""
}

// concPoolOf names the shared pool an op draws its argument from, empty
// where the op draws nothing — the cell's read takes no key.
func concPoolOf(a *Action) string {
	if a == nil {
		return ""
	}
	return a.Pool
}

// concLegReason is why this interface's concurrency family cannot be
// spelled as a leg, empty where it can.
//
// A family is CLASSIFIED from the shape; whether the leg can be written
// is a second question, and the two used to be one because nothing
// rendered the answer. The keyed models compare a read's value against
// what a write installed, so they need the pair to draw from and the
// projection saying which key a write lands on; the cell guards by a
// version member; the append log holds the entry its appender takes.
// Where one is missing the row is not planned and the header names it,
// which is the same refusal every other underivable claim takes.
func concLegReason(b *Bindings) string {
	switch b.ConcFamily {
	case concFamilyKV:
		switch {
		case !b.NeedsKeysPool() || !b.NeedsValuesPool():
			return "the recorded pair draws a key and a value, and this run declares no pool for one of them"
		case b.Reference.KeyField == "":
			return "a recorded write is partitioned by the key it lands on, and no projection derives which member that is"
		}
	case concFamilySession:
		// No key projection is asked for here, and that is the point: the
		// stamp the subject assigns is what put this family on a stepless
		// model, and a model that steps nothing partitions nothing.
		if !b.NeedsKeysPool() || !b.NeedsValuesPool() {
			return "the recorded pair draws a key and a value, and this run declares no pool for one of them"
		}
	case concFamilyLease:
		if !b.NeedsKeysPool() {
			return "the acquire and release draw a key, and this run declares no pool for one"
		}
	case concFamilyCAS:
		switch {
		case b.ConcWriter == nil || b.ConcWriter.Pool == "":
			return "the guarded write draws its argument from no shared pool"
		case b.Reference.VersionField == "":
			return "the cell guards by a version member, and no stamp names which one"
		}
	case concFamilyAppend:
		if b.ConcWriter == nil || b.ConcWriter.Pool == "" || b.ConcEntry == nil {
			return "the log holds the entry its appender takes, and this run spells no pool for one"
		}
	}
	return ""
}

// concurrentLegFor is the body of the linearizability row: the family the
// shape selected, and whatever that family's model takes.
//
// Everything here was already derived and thrown away. The classification
// resolved the pair of ops, the entry type and the two lease sentinels
// and then rendered nothing, so the work reached no generated file and
// the claim it would have stated went unmade.
func concurrentLegFor(b *Bindings, base leg) sdk.EmitNode {
	l := &ConcurrentLeg{
		leg:          base,
		Family:       b.ConcFamily,
		Reader:       b.ConcReader,
		Writer:       b.ConcWriter,
		Acquire:      b.ConcAcquire,
		Release:      b.ConcRelease,
		Entry:        b.ConcEntry,
		LinearizePkg: LinearizePkg,
		PorcupinePkg: PorcupinePkg,
		VersionField: b.Reference.VersionField,
		KeysFunc:     b.KeysFuncName(),
		ValuesFunc:   b.ValuesFuncName(),
		KeyOfName:    b.KeyOfName(),
	}
	if b.Keys.Type != nil {
		l.Key = b.Keys.Type
	}
	if b.Values.Type != nil {
		l.Value = b.Values.Type
	}
	// The miss identity the model validates a read of an absent key with.
	// Nil where the declaration stamps none, which the shipped models read
	// as "any error means absent" rather than as a match on nothing.
	l.Miss = b.Reference.MissSym
	l.Errs = b.Reference.CtorErrs
	// The pool each op draws from, and the locals that owes. The lease's
	// two ops both take the key, which no Action records as a pool, so the
	// family answers for them.
	l.ReaderPool, l.WriterPool = concPoolOf(l.Reader), concPoolOf(l.Writer)
	if b.ConcFamily == concFamilyLease {
		l.ReaderPool, l.WriterPool = poolKeys, poolKeys
	}
	l.DrawsKeys = l.ReaderPool == poolKeys || l.WriterPool == poolKeys
	l.DrawsValues = l.ReaderPool == poolValues || l.WriterPool == poolValues
	l.WriterAnswers = l.Writer != nil && l.Writer.Shape == shapeAnsweringWriter
	if b.ConcFamily == concFamilySession {
		// Only here. The stepped families decide from the model alone, and
		// a law bound beside one would judge the same history twice.
		l.SessionLaws = b.SessionLaws()
	}
	// The declaration's own interface type, off whichever op this family
	// drives: the config is typed at the harness's subject and the
	// closures speak the interface, exactly as the sequential actions do.
	for _, a := range []*Action{l.Reader, l.Writer, l.Acquire, l.Release} {
		if a != nil && a.Iface != nil {
			l.Iface = a.Iface
			break
		}
	}
	return l
}

// ownLegFor is the body of a row carrying one law: the clocked shape
// where the law moves time, the plain one otherwise.
func ownLegFor(b *Bindings, base leg, law *LawBinding) sdk.EmitNode {
	one := b.BindingsOf(law.ID)
	body := OwnLeg{
		leg:        base,
		Laws:       one,
		Keys:       LawsDraw(one, poolKeys),
		Values:     LawsDraw(one, poolValues),
		Pools:      b.PoolsFor(one),
		KeysFunc:   b.KeysFuncName(),
		ValuesFunc: b.ValuesFuncName(),
	}
	if b.Publisher != nil && LawsDrawDrain(one) {
		body.Drain = b.Publisher.DrainName
	}
	body.Builds = LawsNeed(one, factoryFieldKind)
	body.Coalesces = LawsNeed(one, sdk.Kind(LawFieldKindPrefix+"Compute")) ||
		LawsNeed(one, sdk.Kind(LawFieldKindPrefix+"Counter"))
	if law.Clocked {
		return &ClockedLeg{OwnLeg: body, Clock: ClockPkg}
	}
	return &body
}
