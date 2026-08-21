// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"slices"
	"strings"

	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"
	sdkgolang "go.thesmos.sh/eidos/sdk/golang"

	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/projection"
	"go.thesmos.sh/testkit/generator/internal/subject"
	"go.thesmos.sh/testkit/generator/suite"
)

// Name is the plugin's stable identifier.
const Name = "model"

// Capability is the label the plugin advertises.
const Capability = "model"

// Version composes into the pipeline's plugin fingerprint. Bump it on any
// change to what this plugin emits, the projection or the templates alike.
const Version = "0.65.0"

// DirectiveName is the bare directive name — without the `//testkit:` prefix —
// that opts an interface in.
const DirectiveName sdk.DirectiveName = "model"

// RefKey names a constructor in the source package that builds the reference,
// for an interface whose shape no shipped oracle models.
const RefKey = "ref"

// GenKey names a generator constructor in the routed output package —
// `func() *model.Generator[V]` over the values pool's type — for a value the
// wide pools cannot draw by reflection, or one whose domain the consumer
// knows better than any reflection walk.
const GenKey = "gen"

// WitnessKey names the concrete types a generic interface's property runs at,
// comma-separated in declaration order — `witness=string,int`. The same key
// the stub directive reads, because both answer the same question: a Go test
// cannot carry type parameters, so the source names the types or the tier
// declines. Required here where the stub derives: the pools, the reference
// and every law land at these types, and a silently guessed palette would
// change what the property asserts.
const WitnessKey = "witness"

// TierName is the path the model run reports under inside the contract entry,
// and the one `<Iface>Without` drops it by.
const TierName = "model"

// SlotName is the [sdk.EmitFile] slot the bindings land in.
const SlotName = "top"

// KindBindings is the emit kind of the one value queued per interface, and
// therefore the template that renders the file.
const KindBindings sdk.Kind = "model.bindings"

// ActionKindPrefix composes each action's emit kind — `model.action.<shape>` —
// which is the template that renders its constructor call.
const ActionKindPrefix = "model.action."

// TracePkg is the trace vocabulary's import path — the event type the
// generated session classifier reads.
const TracePkg = "go.thesmos.sh/testkit/core/trace"

// sessionMixins are the five per-client guarantees, each carrying the
// version= param that names the ordering stamp on the value.
//
// mixinMonotonicReads is the read-ordering session mixin's spelling — the
// one the tests and the vocabulary list both name.
const mixinMonotonicReads = "monotonicreads"

//nolint:gochecknoglobals // a vocabulary list, read-only after init.
var sessionMixins = []string{
	mixinMonotonicReads, "monotonicwrites", "readyourwrites", "writesfollowreads",
	"causal",
}

// PublisherSpec is the derived subscription drain: the file-level sweep's
// identifier and the types its closure is spelled at.
type PublisherSpec struct {
	// DrainName is the generated sweep's identifier.
	DrainName string

	// Sub is the subscription handle's type — the receive channel — and Msg
	// the element it carries.
	Sub, Msg sdk.Ref
}

// SuppliedOption is one consumer-supplied door: the law field it fills, the
// config field the guarded registration reads, and the closure type spelled
// at the fixture's own instantiation.
type SuppliedOption struct {
	// Field is the law struct's field — the option is <Iface>Model<Field>.
	Field string

	// Config is the config struct's field — the field's name with its first
	// rune lowered, which the law literal and the guard both read.
	Config string

	// Shape selects the closure type's template arm; Iface, Key, Elem and
	// Out are the refs each arm spells.
	Shape                 string
	Iface, Key, Elem, Out sdk.Ref
}

// addSuppliedOption records a door once: two laws reading one field — the
// three isolation levels share History — get one option, and a second law
// asking the same name at a different shape is a refusal, not a shadow.
func (b *Bindings) addSuppliedOption(o *SuppliedOption) string {
	for _, have := range b.SuppliedOptions {
		if have.Config != o.Config {
			continue
		}
		if have.Shape != o.Shape {
			return "asks the " + o.Config + " option at a second shape"
		}
		return ""
	}
	b.SuppliedOptions = append(b.SuppliedOptions, o)
	return ""
}

// The two shared pool locals every generated leg declares. Every draw in
// the file goes through one of them, which is what keeps a law's values
// colliding with the sequences it runs beside.
const (
	poolKeys   = "keys"
	poolValues = "values"
)

// factoryFieldKind is the law field naming the subject factory, and
// factoryIdent the parameter it reads.
//
// Spelled here because two places have to agree: the template that
// renders the field, and the signature that declares what it names.
const (
	factoryFieldKind = sdk.Kind(LawFieldKindPrefix + "Factory")
	factoryIdent     = "factory"
)

// The detector spellings this plugin branches on beyond its template
// dispatch: the keyed put draws from both pools and selects the keyed
// oracle, and the reader and writer are the canonical store pair.
const (
	shapeCompositeWriter = "compositewriter"
	shapeReader          = "reader"
	shapeWriter          = "writer"
	shapeAnsweringWriter = "answeringwriter"
	shapeAggregator      = "aggregator"
	shapeMultiReader     = "multireader"
	shapeBatchReader     = "batchreader"
)

// Plugin is the model-tier generator: it turns an interface's classifications
// into a property-based state-machine run inside the generated contract entry.
type Plugin struct{ *sdkgolang.Base }

// New returns a fresh plugin instance.
//
// [sdk.GeneratorCrossCutting], one bucket after the harness's
// [sdk.GeneratorComposition] — the cross-bucket ordering is what guarantees
// the projection is queued before this reads it. The Requires is documentary:
// eidos's sorter ignores a dependency naming an earlier bucket.
func New() *Plugin {
	return &Plugin{Base: sdkgolang.NewGenerator(Name, goTemplatesFS, GoOutputs()...).
		Version(Version).
		Priority(sdk.GeneratorCrossCutting).
		Provides(Capability).
		Requires(suite.Capability).
		Directives(directives()...).
		Build()}
}

// directives declares the `//testkit:model` schema.
//
// Negation is denied because the tier exists exactly where one is declared,
// so deleting the line is the suppression (docs/adr/0016) — and deleting it
// removes the emission, the file and the `engine` module requirement together.
func directives() []sdk.DirectiveSchema {
	return []sdk.DirectiveSchema{
		sdk.NewDirective(DirectiveName).
			Describe(
				"Generates property-based state-machine tests for the annotated "+
					"interface: random sequences of its methods run against the "+
					"subject and a known-good in-memory reference side by side, "+
					"compared after every call. "+RefKey+" names a constructor in "+
					"the source package returning the interface, for a shape no "+
					"shipped reference models. "+GenKey+" names a generator "+
					"constructor in the routed output package, for a value type "+
					"the wide pools cannot draw by reflection. "+WitnessKey+
					" names the concrete types a generic interface's property "+
					"runs at, comma-separated in declaration order; required "+
					"exactly where the interface is generic.",
			).
			AllowedKeys(RefKey, GenKey, WitnessKey).
			On(sdk.NodeKindInterface).
			DenyNegation().
			Build(),
	}
}

// fieldKey is the identity convention's canonical member spelling.
const fieldKey = "Key"

// keyFieldConventions are the field names read as a value's identity, in
// preference order — the upsert inference's one convention.
//
//nolint:gochecknoglobals // a vocabulary table, read-only after init.
var keyFieldConventions = []string{"ID", fieldKey}

// Bindings is the value queued once per interface carrying the directive.
type Bindings struct {
	sdk.BaseEmit
	subject.Subject

	// Witnesses are the concrete types a generic interface's property runs
	// at, in declaration order — empty in the ordinary non-generic case. The
	// template renders them wherever the file names one of the harness's own
	// generic types, and [Bindings.IfaceRef] arrives already instantiated.
	Witnesses []sdk.Ref

	// witnessQ maps each type parameter's name to its witness's stamp
	// spelling. The annotator spells a parameter's key or value type by its
	// bare name, so every stamp read routes through [Bindings.substQ].
	witnessQ map[string]string

	// Session is the derived per-client classification, nil where no session
	// mixin stamps a version. One derivation, referenced by the sequential
	// registry and the concurrent leg alike. sessionKeyField is the value's
	// identity member, derived beside the twin decision where the reader is
	// in hand.
	Session         *SessionSpec
	sessionKeyField string

	// Publisher is the derived subscription sweep, nil where no publisher
	// law binds through a channel-answering subscribe. One derivation: the
	// file-level sweep, the option that outranks it, and the property local
	// every drain field reads.
	Publisher *PublisherSpec

	// SuppliedOptions are the typed doors this file generates: one option
	// and one config field per supplied law field, spelled at the fixture's
	// own types. A law reading one registers only when it is set.
	SuppliedOptions []*SuppliedOption

	// OptionName is `<Iface>Model` — the option a consumer passes to the
	// contract entry to run this tier. PropertyName is `<Iface>ModelProperty`,
	// the composition point it and any bespoke harness share. OptionTypeName
	// and ConfigName carry the tier's own option surface.
	OptionName, PropertyName, OptionTypeName, ConfigName string

	// SatLaws and SatMutants are the saturation surface: per bound law, the
	// methods its closures reach; per reached method, the mutant wrappers
	// the generated prover wears. Binding a law is necessary; this is what
	// makes it sufficient — a law no mutant of its own methods can redden
	// is bound but unsaturatable, and the prover says so by name.
	SatLaws    []SatLaw
	SatMutants []SatMutant

	// EntryName is the harness entry the option is passed to, for the header.
	EntryName string

	// FixtureCtor is the harness's derived-input constructor. The pools draw
	// from its fields rather than from generators of their own, because the
	// harness's checks, the seeds and these sequences must keep hitting the
	// same keys — a pool of fresh random keys never revisits a written one,
	// and every comparison passes over a history with no collisions in it.
	FixtureCtor string

	// Keys and Values are the two shared pools. Values.Field is empty for an
	// interface with no writer-shaped method, and the template then declares
	// no pool — an unused local is a compile error in a generated file.
	Keys, Values Pool

	// Reference is the known-good implementation every action compares
	// against.
	Reference Reference

	// Actions is one per driven method, in declaration order.
	Actions []*Action

	// Adapter is every interface method, delegated to the oracle or inert,
	// in declaration order. Empty when the reference was supplied.
	Adapter []AdapterMethod

	// Skipped names the methods no action drives, each with why. Rendered
	// into the header: a method absent from the sequences without a stated
	// reason is indistinguishable from a generator that forgot it.
	Skipped []Skip

	// Laws is every law the classifications earned and this build could fill,
	// registered on the run through the same registry a consumer's own laws
	// join. Unbound is the rest — selected, and waiting on something the
	// header names, because a law that quietly failed to bind reads as a
	// claim the run checks.
	Laws    []*LawBinding
	Unbound []Skip

	// Legs are the bodies the rows run, one per row and in plan order.
	//
	// Emit values rather than a shape this template branches on: a leg is
	// a different body, not a different setting, and rendering each
	// through its own template is what keeps the reason for the split
	// beside the code it produced.
	Legs []sdk.EmitNode

	// SubjectSpelling is the interface as the harness file spells it, and
	// FixtureTypeName its sample inputs. Taken from the harness rather
	// than derived, because the qualified form compiles beside the local
	// one — two spellings of one type in one file, and nothing complains.
	SubjectSpelling, FixtureTypeName string

	// VeneerVar is the harness's naming surface, for the prose that tells
	// a reader how to decline these rows.
	VeneerVar string

	// Rows are the check plans this tier renders — the ones the harness
	// generator planned and listed under Withheld because no template of
	// its own spells them.
	//
	// Read rather than re-derived, which is the rule RFC-0002 and
	// RFC-0003 state: two derivations of one row disagree silently, and
	// the generated file is where that turns up. It also cannot be
	// re-derived correctly here — the row's Needs is what put the
	// capability field on the harness, and the harness was projected
	// before this generator ran.
	Rows []projection.CheckPlan

	// LawPools are the pools the laws declare beyond the shared pair: a wide
	// input domain for a stateless claim, the adversarial strings a safety
	// claim needs. LawsUseFixture marks a law closure anchored on a fixture
	// key, which obliges the property to construct the fixture.
	LawPools       []LawPool
	LawsUseFixture bool

	// UsesClock marks a clock-bound law in the set, which obliges the
	// property to offer the ModelClocked option and guard those laws on it.
	UsesClock bool

	// Coalesced marks a bound coalescing law, which obliges the property to
	// declare the locked call counter its compute probe increments.
	Coalesced bool

	// RecordsHistory marks a bound no-drops law: the property declares the
	// append log at HistoryElem, the append action records into it, and the
	// runner clears it each iteration.
	RecordsHistory bool
	HistoryElem    sdk.Ref

	// ConcEntry is the append leg's entry type — the appender method's own
	// argument, because the writer action's value stamp is the answered
	// offset there, not the entry the log holds.
	ConcEntry sdk.Ref

	// ConcFamily picks the concurrent leg's model: empty for none, "kv" for
	// the keyed-store pair, "lease" for the acquire/release table.
	ConcFamily string

	// ConcReader and ConcWriter are the actions the kv leg drives against
	// the Porcupine keyed-store model; ConcAcquire and ConcRelease the
	// lease leg's two. All point into Actions, so the closures the legs
	// spell agree with the sequential ones about every method and type.
	ConcReader, ConcWriter   *Action
	ConcAcquire, ConcRelease *Action

	// PkgName is where Layout routed the file — see [Bindings.SetOutputPackages].
	PkgName string

	// contractKeySrc is the contract-oracle role method whose argument is the
	// store's key domain, and contractKeyedRoles the role methods that draw
	// it — a lease's acquire and release take the key, and a reader-less
	// keyed contract would otherwise never declare the pool its laws draw.
	contractKeySrc     *subject.Method
	contractKeyedRoles map[string]bool

	// concAcquireName and concReleaseName are the lease leg's role methods,
	// recorded by the derivation for the action lookup the leg wires.
	concAcquireName, concReleaseName string
}

// The lease contract's role spellings, as the directives stamp them, and
// the concurrent-leg families the template branches on.
const (
	roleLeaseAcquire = "acquire"
	roleLeaseRelease = "release"

	concFamilyKV      = "kv"
	concFamilyLease   = "lease"
	concFamilySession = "session"
	concFamilyCAS     = "cas"
	concFamilyAppend  = "append"

	// shapeCASWriter is the re-pointed shape the contract-role pass spells
	// for the cas write, matched here when the cell leg derives.
	shapeCASWriter = "cas.writer"
)

// Kind returns [KindBindings].
func (*Bindings) Kind() sdk.Kind { return KindBindings }

// SetOutputPackages records where Layout routed the file, which is not known
// during Generate: `out=`/`pkg=` directives move it after this plugin ran.
//
// The one consumer is the miss sentinel's message, whose text must open with
// the package that declares it — the convention every hand-written error in
// this repository is held to, and a generated file is not excused from.
func (b *Bindings) SetOutputPackages(byTag map[string]string) {
	if path := byTag[""]; path != "" {
		b.PkgName = path[strings.LastIndexByte(path, '/')+1:]
	}
}

// MissPrefix is what the sentinel's message opens with: the routed package
// where Layout resolved one, the interface's own spelling where a run never
// reached Layout — still a plausible message rather than an empty prefix.
func (b *Bindings) MissPrefix() string {
	if b.PkgName != "" {
		return b.PkgName
	}
	return strings.ToLower(b.IfaceName)
}

// KeyOfName is the shared key projection's identifier — one derivation used
// by the reference constructor and every law field that keys a value, so the
// two cannot disagree about which field is the identity.
func (b *Bindings) KeyOfName() string { return b.declName("KeyOf") }

// ActionsFuncName is the operation vocabulary both legs drive.
func (b *Bindings) ActionsFuncName() string { return b.declName("Actions") }

// LawsFuncName is the bundled laws the shared sequences carry.
func (b *Bindings) LawsFuncName() string { return b.declName("Laws") }

// KeysFuncName and ValuesFuncName are the two shared pools.
//
// Functions rather than the locals they used to be: both legs draw from
// them, and a pool declared inside one leg is one the other cannot reach.
func (b *Bindings) KeysFuncName() string { return b.declName("Keys") }

// ValuesFuncName is [Bindings.KeysFuncName] for the values pool.
func (b *Bindings) ValuesFuncName() string { return b.declName("Values") }

// declName composes one identifier this tier declares inside the harness's
// own file — `<iface>Model<What>`.
//
// One prefix for all of them, because they land beside the harness
// generator's declarations and a reader has to be able to tell whose is
// whose at a glance.
func (b *Bindings) declName(what string) string {
	return strings.ToLower(b.IfaceName[:1]) + b.IfaceName[1:] + "Model" + what
}

// Concurrent reports whether a linearizability leg derives: the map-shaped
// pair the keyed-store model speaks, or a contract family whose own model
// ships — each unrefined by a claim that changes what its operations mean.
func (b *Bindings) Concurrent() bool { return b.ConcFamily != "" }

// LeaseHeld is the held sentinel the lease leg's model matches — the same
// identity the oracle constructor renders.
func (b *Bindings) LeaseHeld() CtorErr {
	for _, e := range b.Reference.CtorErrs {
		if e.Sym != nil || e.Name != "" {
			return e
		}
	}
	return CtorErr{}
}

// CasMismatch is the stale-version sentinel the cas leg's model matches —
// the spec's first error row, the same identity the sequential oracle's
// constructor consumes.
func (b *Bindings) CasMismatch() CtorErr {
	if len(b.Reference.CtorErrs) == 0 {
		return CtorErr{}
	}
	return b.Reference.CtorErrs[0]
}

// LinearizePkg surfaces the Porcupine wiring's import path to the templates.
func (*Bindings) LinearizePkg() string { return LinearizePkg }

// HistoryPkg surfaces the append log's import path to the template that
// declares the recording local.
func (*Bindings) HistoryPkg() string { return HistoryPkg }

// RootPkg surfaces the runtime module's import path — the prover's
// FailableTB lives there.
func (*Bindings) RootPkg() string { return RootPkg }

// ModelPkg surfaces the runner's import path to the templates, which can
// reach a method and not a const.
func (*Bindings) ModelPkg() string { return ModelPkg }

// RefPkg returns the reference package's import path.
func (*Bindings) RefPkg() string { return RefPkg }

// ClockPkg surfaces the test clock's import path to the templates.
func (*Bindings) ClockPkg() string { return ClockPkg }

// TracePath surfaces the trace vocabulary's import path to the templates.
func (*Bindings) TracePath() string { return TracePkg }

// LawPath surfaces the law package's import path to the templates.
func (*Bindings) LawPath() string { return LawPkg }

// TierName returns the tier's base path.
func (*Bindings) TierName() string { return TierName }

// TierPath is the path this interface's run reports under: "model" where an
// independent oracle stands opposite the subject, "model/twin" where the
// subject's own factory is the floor — in the test output, because a weaker
// claim a reader has to open a generated file to learn about is a claim
// most readers hold wrong.
func (b *Bindings) TierPath() string {
	if b.Reference.Twin() {
		return TierName + "/twin"
	}
	return TierName
}

// NeedsFixture reports whether anything in the property reads the fixture —
// a pool, a fixture-anchored law closure, or a multi-argument writer's
// per-position pairs. An unused local is a compile error in a generated file.
func (b *Bindings) NeedsFixture() bool {
	if b.UsesKeys() || b.UsesValues() || b.LawsUseFixture {
		return true
	}
	for _, a := range b.Actions {
		if len(a.Args) > 0 {
			return true
		}
	}
	return false
}

// UsesValues reports whether any action draws from the values pool.
func (b *Bindings) UsesValues() bool {
	for _, a := range b.Actions {
		if a.Pool == poolValues {
			return true
		}
	}
	return false
}

// UsesKeys reports whether anything draws from the keys pool. A composite
// writer draws from both, whatever its Pool says, and a pinned values pool
// draws a key for every value.
func (b *Bindings) UsesKeys() bool {
	if b.Values.Pin != "" {
		return true
	}
	for _, a := range b.Actions {
		if a.Pool == poolKeys || a.Shape == shapeCompositeWriter {
			return true
		}
	}
	return false
}

// LawsUseKeys and LawsUseValues report whether a bundled law draws from a
// shared pool.
//
// Asked separately from the actions because the two now live in separate
// functions. A pool the actions draw and the laws do not is a local the
// laws' function must not declare — an unused one does not compile, and
// this file's output is not something a consumer can edit around.
func (b *Bindings) LawsUseKeys() bool { return b.lawsDraw(poolKeys) }

// LawsUseValues is [Bindings.LawsUseKeys] for the values pool.
func (b *Bindings) LawsUseValues() bool { return b.lawsDraw(poolValues) }

func (b *Bindings) lawsDraw(pool string) bool {
	return b.lawFields(func(f *LawField) bool { return f.Pool == pool })
}

// LawsDraw reports whether any of the given laws draws from the named
// shared pool.
//
// Exported for a leg carrying one law of its own: that leg declares the
// pool locals its law reads and no others, and asking the whole bound set
// would declare locals nothing there uses.
func LawsDraw(laws []*LawBinding, pool string) bool {
	for _, l := range laws {
		if slices.ContainsFunc(l.Fields, func(f *LawField) bool { return f.Pool == pool }) {
			return true
		}
	}
	return false
}

// PoolsFor are the law-declared pools the given laws actually name, in
// the order the derivation found them.
//
// Filtered rather than taken whole for the same reason: the pools were
// derived for every selected law, and a leg carrying one of them declares
// a local for that one alone.
func (b *Bindings) PoolsFor(laws []*LawBinding) []LawPool {
	named := map[string]bool{}
	for _, l := range laws {
		for _, f := range l.Fields {
			named[f.Pool] = true
		}
	}
	out := make([]LawPool, 0, len(b.LawPools))
	for _, p := range b.LawPools {
		if named[p.Name] {
			out = append(out, p)
		}
	}
	return out
}

// OwnLegLaws are the laws that ride a leg of their own — the ones the
// bundled observational run cannot carry, because each needs something
// it does not provide: a clock to move, a subject in a failure state,
// writes of its own.
func (b *Bindings) OwnLegLaws() []*LawBinding {
	var out []*LawBinding
	for _, l := range b.Laws {
		if len(l.Supplied) > 0 {
			continue
		}
		if _, own := tiers.LegOf(l.ID); own {
			out = append(out, l)
		}
	}
	return out
}

// LawsNeedFactory reports whether a bundled law builds instances of its
// own — a merge claim compares two, and no observation over one states it.
//
// The leg has the subject in hand and this function does not, so the
// factory arrives as a parameter rather than as a local. That is the whole
// reason to ask: a parameter nothing reads does not compile.
func (b *Bindings) LawsNeedFactory() bool {
	return b.lawFields(func(f *LawField) bool { return f.KindName == factoryFieldKind })
}

func (b *Bindings) lawFields(match func(*LawField) bool) bool {
	for _, l := range b.LegLaws() {
		if slices.ContainsFunc(l.Fields, match) {
			return true
		}
	}
	return false
}

// LegLawPools are the law-declared pools the bundled laws name.
func (b *Bindings) LegLawPools() []LawPool { return b.PoolsFor(b.LegLaws()) }

// ActionsUseKeys reports whether an action draws from the keys pool.
//
// Narrower than [Bindings.UsesKeys], which also answers true for a pinned
// values pool — the pin is spelled inside the values constructor now, so
// the actions never see the keys it draws.
func (b *Bindings) ActionsUseKeys() bool {
	for _, a := range b.Actions {
		if a.Pool == poolKeys || a.Shape == shapeCompositeWriter {
			return true
		}
	}
	return false
}

// NeedsKeysPool and NeedsValuesPool report whether the pool's constructor
// is declared at all. A function nothing calls is dead code the linter
// refuses, and this tier emits into somebody else's file.
func (b *Bindings) NeedsKeysPool() bool { return b.UsesKeys() || b.LawsUseKeys() }

// NeedsValuesPool is [Bindings.NeedsKeysPool] for the values pool.
func (b *Bindings) NeedsValuesPool() bool { return b.UsesValues() || b.LawsUseValues() }

// LegLaws are the laws the shared sequences carry: every bound law with no
// leg of its own, and none whose fields wait on a value only the consumer
// has.
//
// The second exclusion is what the config used to carry. A supplied field
// arrived through an option on a surface this tier no longer emits, so the
// law that reads it cannot be filled — and a law literal with a nil
// closure in it panics on the first draw rather than reporting anything.
// The header names each one and what it is waiting for.
func (b *Bindings) LegLaws() []*LawBinding {
	var out []*LawBinding
	for _, l := range b.Laws {
		if l.Clocked || len(l.Supplied) > 0 {
			continue
		}
		if _, own := tiers.LegOf(l.ID); own {
			continue
		}
		out = append(out, l)
	}
	return out
}

// The three store models Go interfaces declare — a value that carries its own
// key, a key passed beside the value, and an append-and-drain collection with
// no keys at all — plus the twin floor beneath them: a second instance from
// the subject's own factory. Two twins driven identically must agree, which
// catches nondeterminism, hidden shared state and races without an
// independent model, and misses what an independent model exists to catch —
// a subject that is systematically wrong agrees with itself. The header says
// which floor was reached and why, and `ref=` raises it.
const (
	OracleMap        Oracle = "map"
	OracleKeyed      Oracle = "keyed"
	OracleCollection Oracle = "collection"
	OracleContract   Oracle = "contract"
	OracleTwin       Oracle = "twin"
)

// KindCompanion is the companion's emit kind and template.
const KindCompanion sdk.Kind = "model.companion"

// substQ rewrites a classification stamp's spelling at the witnesses: a
// stamp naming a type parameter answers the concrete type the property runs
// at, and every other spelling passes through untouched.
func (b *Bindings) substQ(q string) string {
	if w, bound := b.witnessQ[q]; bound {
		return w
	}
	return q
}

// actionFor answers the driven action on the named method, nil where the
// method drives nothing.
func (b *Bindings) actionFor(method string) *Action {
	for _, a := range b.Actions {
		if a.Method == method {
			return a
		}
	}
	return nil
}

// dropAction removes the named method's action and records why in the
// header's not-driven block.
func (b *Bindings) dropAction(method, reason string) {
	kept := b.Actions[:0]
	for _, a := range b.Actions {
		if a.Method == method {
			b.Skipped = append(b.Skipped, Skip{Method: method, Reason: reason})
			continue
		}
		kept = append(kept, a)
	}
	b.Actions = kept
}

// keyQOf reads a method's key-type stamp through the witness substitution:
// the annotator spells a type parameter by its bare name, and the property
// runs at the concrete type the source pinned. The stamped flag mirrors the
// meta read for the sites that distinguish absent from empty.
//
//nolint:unparam // mirrors valueQOf; callers today read only the spelling
func (b *Bindings) keyQOf(m *subject.Method) (string, bool) {
	q, stamped := shape.MetaKeyType.Get(m.Source.Meta())
	return b.substQ(q), stamped
}

// valueQOf is [Bindings.keyQOf] for the value-type stamp.
func (b *Bindings) valueQOf(m *subject.Method) (string, bool) {
	q, stamped := shape.MetaValueType.Get(m.Source.Meta())
	return b.substQ(q), stamped
}
