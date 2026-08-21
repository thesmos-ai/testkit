// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"fmt"
	"slices"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/accumulates"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/bounded"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/concurrent"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/deprecated"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/errors"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/hooks"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/idempotent"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/indexed"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/integrationonly"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/lifecycleafterclose"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/nilsafe"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/notfound"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/orderafter"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/partition"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/sample"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/scope"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/sideeffect"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/timeaware"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/timeout"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/total"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/ttl"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/validates"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/wrappedvia"
	"go.thesmos.sh/eidos/sdk"
	sdkgolang "go.thesmos.sh/eidos/sdk/golang"

	"go.thesmos.sh/testkit/generator/internal/source"
	"go.thesmos.sh/testkit/generator/internal/subject"
	"go.thesmos.sh/testkit/generator/suite/projection"
)

// Name is the plugin's stable identifier.
const Name = "suite"

// Capability is the label the plugin advertises, so the generators that read
// this one's projection — bench, fuzz, model — can declare the dependency.
const Capability = "suite"

// Version composes into the pipeline's plugin fingerprint. Bump it on any
// change to what this plugin emits, the projection or the templates alike.
const Version = "1.17.0"

// DirectiveName is the bare directive name — without the `//testkit:` prefix —
// that opts an interface in.
const DirectiveName sdk.DirectiveName = "suite"

// SlotName is the [sdk.EmitFile] slot the harness lands in.
const SlotName = "top"

// KindContract is the emit kind for the harness as a whole. The backend
// resolves a template by the kind's string value, so the constant doubles as
// the template's name.
const KindContract sdk.Kind = "suite.contract"

// Plugin is the conformance-suite generator.
//
// The embedded [sdkgolang.Base] answers every declaration the pipeline asks for
// — name, version, priority, capabilities, directives, outputs, templates and
// the template funcmap — so the only method this package writes is
// [Plugin.Generate].
type Plugin struct{ *sdkgolang.Base }

// New returns a fresh plugin instance.
//
// # Placement
//
// [sdk.GeneratorComposition], one bucket after the double's
// [sdk.GeneratorFoundation]. The bucket is what orders the two; a Requires
// naming a plugin in an earlier bucket is silently ignored by eidos's sorter,
// so nothing here relies on one.
//
// # Failure mode
//
// [sdkgolang.Builder.Build] panics on a declaration the pipeline cannot serve —
// a missing template tree, a suffix-less output. Every such mistake is in this
// function rather than in a run's input, so it fires on the first construction
// in any test instead of rendering a short file and failing nowhere.
func New() *Plugin {
	return &Plugin{Base: sdkgolang.NewGenerator(Name, goTemplatesFS, GoOutputs()...).
		// The rewrite's body templates share this FS, so their
		// functions join the merged map the backend parses it with.
		Funcs(renderFuncs()).
		Version(Version).
		Priority(sdk.GeneratorComposition).
		Provides(Capability).
		Directives(directives()...).
		Build()}
}

// directives declares the `//testkit:suite` schema.
//
// No positional argument: the interface the directive sits on is the subject,
// and a name beside it would be a second way to say the same thing. No keys
// either — benchmarks and fuzz targets are their own generators' directives,
// scoped to a method, because a team wants them on the paths that matter rather
// than on all six. Negation is denied because a suite exists exactly where one
// is declared, so deleting the line is the suppression (docs/adr/0016).
func directives() []sdk.DirectiveSchema {
	return []sdk.DirectiveSchema{
		sdk.NewDirective(DirectiveName).
			Describe(
				"Generates a conformance harness for the annotated interface: " +
					"an Assert<Iface>Contract entry point running every check " +
					"the signature and the interface's classifications imply, " +
					"a typed extension point per method, and the inputs those " +
					"checks need, derived. Takes no argument — the interface it " +
					"sits on is the subject. Benchmarks and fuzz targets are " +
					"//testkit:bench and //testkit:fuzz, which scope to a method.",
			).
			On(sdk.NodeKindInterface).
			// The Describe text says it takes no arguments; this is what makes
			// that true. An empty AllowedKeys means unrestricted, so before
			// DenyKeys existed a stray key parsed, validated and stamped
			// nothing, and whoever wrote it believed it had an effect.
			DenyKeys().
			DenyNegation().
			Build(),
	}
}

// Contract is the emit value rendered into the primary output.
type Contract struct {
	sdk.BaseEmit
	subject.Projection

	// EntryName is the identifier a consumer calls — `Assert<Iface>Contract`.
	EntryName string

	// Seed is the write each fresh subject is populated through, nil for an
	// interface declaring no writer.
	Seed *Seed

	// Unseeded says why no seed was derived, empty where one was.
	//
	// A harness that seeds nothing runs every read check against a fresh
	// subject, where a miss and a bug are the same observation. The header
	// carries the reason so the reader knows which of the three exits was
	// taken and therefore what would close it.
	Unseeded string

	// Token qualifies every identifier the file emits, so the templates
	// compose names from one word rather than each lower-casing the
	// interface for itself. Qualifier is the same interface as an ID
	// slug, which is a different grammar — see [Iface].
	Token     string
	Qualifier string

	// Vocab, LawIDs and Prove are the packages the emitted identities
	// and entry points are composed from. Carried rather than spelled
	// in the templates because an import path built inside a template
	// is one the backend cannot register.
	Vocab  string
	LawIDs string
	Prove  string

	// Inventory is every check the derivers licensed, and Index the
	// typed surface a consumer drops one through. Both are projections
	// of the same nodes, which is what keeps the index from naming a
	// check the run does not emit.
	Inventory projection.Inventory
	Index     projection.IndexPlan

	// Pools are the drawn config fields a consumer overrides: one per
	// roled field, three members each. The fixture draws through them
	// and the seeded corpus is zipped from them.
	Pools []projection.PoolPlan

	// Limit is the bound //testkit:mixin bounded declares, empty where
	// none is. Carried as the declared TEXT rather than parsed: it is
	// emitted as a constant's value, and re-spelling a number a
	// declaration already wrote is a second chance to write it
	// differently.
	Limit string

	// Corpus is the seeded corpus a reader-only interface is populated
	// through, and Seeded says the harness takes it.
	//
	// Two fields because they answer different questions: whether this
	// interface must be seeded from outside (nothing on it writes) and
	// whether this run can derive what to seed it with (both roles are
	// stamped). An interface that needs a corpus and cannot derive one
	// keeps the ordinary constructors and says so in the header — a
	// harness demanding docs the suite cannot produce is worse than one
	// that admits the gap.
	Corpus projection.CorpusPlan
	Seeded bool

	// Harness is the run surface this interface's checks can demand —
	// a capability field with no check behind it is a promise the run
	// never collects on.
	Harness projection.HarnessPlan

	// Checks are the derived checks this file can render today; Withheld
	// names the body variants it cannot, so the header says what is
	// missing rather than leaving a reader to infer coverage from a
	// short list.
	Checks   []*CheckEmit
	Withheld []string

	// EmittedIDs is every identity this package declares, sorted, for
	// the listing at the top of the file.
	//
	// Read from the inventory rather than from the index: they are
	// projections of the same plans, and taking it from the index would
	// make the listing agree with the index by construction instead of
	// by both agreeing with what was derived.
	EmittedIDs []string

	// DrawsFixture says the checks builder takes the run's fixture,
	// which it does wherever any of its rows draws — the closures
	// capture it, so a row cannot reach one the builder was not given.
	DrawsFixture bool

	// AnyProven and AnyArgued say which of the two row constructors the
	// table binds, which is only ever the ones its rows call.
	AnyProven, AnyArgued bool

	// SeedsCorpus says the checks builder takes the run's corpus, which
	// it does wherever a seeded probe judges it.
	SeedsCorpus bool

	// Refusals are the checks the rules reached and could not derive.
	// They render into the header: a claim the reader cannot see
	// refused reads as a claim this file checks.
	Refusals []Refusal
}

// Gaps words each refusal as the header states it: what was not
// derived, why, and what closes it.
//
// The three parts stay separate in [Refusal] because attribution and
// remedy are what a census reads; they are joined here because a
// reader of the generated file wants one sentence per gap.
func (c *Contract) Gaps() []string {
	out := make([]string, 0, len(c.Refusals))
	for _, r := range c.Refusals {
		if r.Elsewhere {
			continue
		}
		out = append(out, r.What+" — "+r.Why+". To close it: "+r.Remedy+".")
	}
	slices.Sort(out)
	return out
}

// Elsewhere words the claims another part of testkit owns, which this
// file therefore does not make.
//
// Separate from [Contract.Gaps] because the two ask different things of
// the reader. A gap is something they can close, and every line of it
// ends in an instruction. These end in nothing to do — they are here so
// that a directive the consumer wrote is accounted for rather than
// silently absent, which is the difference between a file that does not
// check something and a file that looks like it does.
func (c *Contract) Elsewhere() []string {
	var out []string
	for _, r := range c.Refusals {
		if !r.Elsewhere {
			continue
		}
		out = append(out, r.What+" — "+r.Why+".")
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// Kind returns [KindContract].
func (*Contract) Kind() sdk.Kind { return KindContract }

// Generate queues one harness per interface carrying the directive.
func (*Plugin) Generate(ctx *sdk.GeneratorContext) error {
	c := sdk.NewProvenance(Name)
	for _, iface := range ctx.Reader.Interfaces().Slice() {
		if !iface.HasPositiveDirective(DirectiveName) {
			continue
		}
		set, complete := source.MethodSet(ctx, iface, Name, harnessConsequence)
		if !complete {
			continue
		}
		if len(set.Methods) == 0 {
			ctx.Diag.Errorf(iface.Pos(),
				"%s: interface %q carries //testkit:%s but declares no method, "+
					"so the harness would assert nothing",
				Name, iface.Name, DirectiveName)
			continue
		}
		// The rewrite's transition state: the projection carrier still
		// queues — the model tier reads Methods, Fixture and Subject
		// from it — and the check emission is the deriver inventory's,
		// rendered once the body and structural templates land. The
		// incumbent's check assembly is deleted, not dormant.
		methods := methodsOf(iface, set)
		// Pools first: a fixture field draws from the pool its role
		// opened, so the fixture cannot be projected before the roles
		// are known. Both come from the same //testkit:default stamps,
		// which is what lets them be matched by name rather than by a
		// correspondence either side would have to invent.
		pools, poolRefusals := poolsOf(ctx.Reader, methods)
		fixture := fixtureOf(ctx, iface, methods, pools)
		for i := range methods {
			// The fixture-field correspondence the derivers draw
			// through — populated here since the incumbent's check
			// assembly, its previous home, is gone.
			methods[i].ArgFields = fixtureArgs(fixture, methods[i], false)
		}
		seed, unseeded := seedOf(fixture, methods)
		corpus, seeded := projection.CorpusOf(pools)
		limit := declaredLimit(methods)
		seeded = seeded && derivedSeeded(methods)

		token, qualifier := projection.Token(iface.Name), projection.IDQualifier(iface.Name)
		derived := Iface{
			Name:      iface.Name,
			Package:   iface.Package,
			Token:     token,
			Qualifier: qualifier,
			Methods:   methods,
			Fixture:   fixture,
			Corpus:    seeded,
		}
		inventory, refusals := InventoryOf(derived)
		refusals = append(refusals, poolRefusals...)
		if err := inventory.Verify(); err != nil {
			// The run's own invariants, held before anything renders. A
			// deriver bug caught here names the check it is about; the
			// same bug reaching a consumer is a compile error in a file
			// they did not write.
			ctx.Diag.Errorf(iface.Pos(), "%s: %s: %v", Name, iface.Name, err)
			continue
		}
		// A generic subject's companion carries a note in place of
		// proofs, so nothing this run emits can drive its claims; the
		// rows say Argued rather than stamping evidence that is not
		// there.
		provable := len(iface.TypeParams) == 0
		checks := checkEmitsOf(sdk.EmitBase(c, iface), derived, inventory, provable, seeded)

		// The index declares what this run EMITS, not what it derived.
		//
		// Built from the whole inventory it named every plan, including
		// the ones whose body no template spells — so a consumer could
		// write `Suite.Checks.Get.Hit()`, compile, and drop nothing. That
		// is the silent drop the typed index exists to prevent, reached
		// through the index itself. The header still names the withheld
		// variants, so an absent entry has a stated reason.
		index, err := projection.IndexOf(projection.Inventory{Checks: emittedPlans(checks)})
		if err != nil {
			ctx.Diag.Errorf(iface.Pos(), "%s: %s: %v", Name, iface.Name, err)
			continue
		}
		ids, err := emittedIDs(projection.Inventory{Checks: emittedPlans(checks)})
		if err != nil {
			ctx.Diag.Errorf(iface.Pos(), "%s: %s: %v", Name, iface.Name, err)
			continue
		}
		base := sdk.EmitBase(c, iface)
		// Before the contract is built: this settles which rows carry
		// evidence, and a row it downgrades must not be counted Proven.
		defects, unproven := proofsOf(base, iface.Package, ctx.Reader, derived, checks)
		proven, argued := stampsUsed(checks)

		contract := &Contract{
			BaseEmit: base,
			Projection: subject.Projection{
				Subject: subjectOf(iface),
				Fixture: fixture,
				Methods: methods,
			},
			EntryName:    "Assert" + iface.Name + "Contract",
			Seed:         seed,
			Unseeded:     unseeded,
			Token:        token,
			Qualifier:    qualifier,
			Vocab:        Vocab,
			LawIDs:       LawIDs,
			Prove:        Prove,
			Inventory:    inventory,
			Index:        index,
			Pools:        pools,
			Limit:        limit,
			Corpus:       corpus,
			Seeded:       seeded,
			Harness:      projection.HarnessOf(iface.Name, inventory.Checks),
			Checks:       checks,
			Withheld:     withheldBodies(inventory),
			EmittedIDs:   ids,
			DrawsFixture: drawsFixture(checks),
			SeedsCorpus:  seedsCorpus(checks),
			AnyProven:    proven,
			AnyArgued:    argued,
			Refusals:     refusals,
		}
		if unseeded != "" {
			// A harness that seeds nothing runs every read check against a
			// fresh subject, where a miss and a bug look identical. A
			// warning, because the consumer's own seed closes it.
			ctx.Diag.Warnf(iface.Pos(),
				"%s: %s derives no seed — %s; supply one with %sSeed",
				Name, iface.Name, unseeded, iface.Name)
		}
		// Queued in one call rather than two. The pair differs only in its
		// emit kind and output tag, and a second append is where the two
		// would drift — which for these two means a harness stamping
		// claims its companion does not prove.
		if err := sdk.QueueEmit(ctx.Store.Emit(), c, SlotName, iface,
			contract,
			&Proofs{
				BaseEmit: sdk.EmitBaseTagged(base, GoTestOutputTag),
				Subject:  subjectOf(iface),
				// Provisional. The harness is routed by Layout, which has
				// not run; [Proofs.SetOutputPackages] corrects this and
				// every defect under it once the target resolves.
				Pkg:          iface.Package,
				Token:        token,
				Vocab:        Vocab,
				Prove:        Prove,
				DrawsFixture: contract.DrawsFixture,
				SeedsCorpus:  contract.SeedsCorpus,
				Fixture:      fixture,
				CorpusFunc:   projection.CorpusName(token),
				Defects:      defects,
				Unproven:     unproven,
				Generic:      !provable,
			},
		); err != nil {
			// Wrapped even though the queue names the plugin and the slot: what
			// it cannot name is which declaration the run was on when it failed,
			// and that is the only part a reader needs to find the source line.
			return fmt.Errorf("%s: queue interface %q: %w", Name, iface.Name, err)
		}
	}
	return nil
}

// harnessConsequence is what this generator loses to an unresolvable
// embed, for the diagnostics [methodset.Resolve] writes.
var harnessConsequence = source.Consequence{
	Partial:    "the harness covers whatever the source had already contributed",
	Incomplete: "a harness over part of a contract passes an implementation that fails the rest",
}

// methodsOf projects every method of the resolved set, with its checks
// selected.
//
// Driven off the resolved set rather than the declarations: an interface that
// embeds another declares none of what it inherits, and a harness reading only
// declarations would cover half a contract without saying it had.
func methodsOf(iface *sdk.Interface, set sdk.MethodSetResult) []subject.Method {
	out := make([]subject.Method, 0, len(set.Methods))
	for _, src := range set.Methods {
		bag := src.Meta()
		roles, partners, params := contractDataOf(bag)
		stamped := shape.Mixins(bag)
		out = append(out, subject.Method{
			Sig:              golang.SigOf(src),
			CheckType:        iface.Name + src.Name + "Check",
			Mixins:           stamped,
			IntegrationOnly:  slices.Contains(stamped, MixinIntegrationOnly),
			Contracts:        shape.Contracts(bag),
			MixinParams:      mixinParamsOf(bag, stamped),
			ContractRoles:    roles,
			ContractPartners: partners,
			ContractParams:   params,
		})
	}
	return out
}

// mixinParamsOf reads the KV arguments of every mixin stamped on a method.
//
// Pulled once, into a map, rather than reached through [shape.MixinParamKey]
// at each use. The projection is what a check is selected from and what a
// template renders, and neither holds the source node by then.
//
// The pairs come from the live registry rather than a list here: a mixin
// declares its own parameters, [mixinParamKeys] reads them, and a hand-kept
// enumeration of twelve was a second answer to a question eidos already
// answers — one that went stale silently the day a mixin grew a parameter.
// Reading every stamped mixin rather than only the ones this generator acts
// on costs a map entry nothing looks up, and buys never having to remember
// this file when a check starts reading a param it did not before.
//
// The stamped set is passed rather than read back off the bag, because the
// caller has already resolved it and a param is only meaningful under a mixin
// the method actually carries.
func mixinParamsOf(bag *sdk.Bag, stamped []string) map[string]string {
	var out map[string]string
	for _, name := range stamped {
		for _, param := range subject.MixinParamKeys(name) {
			v, found := shape.MixinParamKey(name, param).Get(bag)
			if !found {
				continue
			}
			if out == nil {
				out = map[string]string{}
			}
			out[name+"."+param] = v
		}
	}
	return out
}

// The mixins this generator generates a check for, and the parameters it reads.
//
// Named here rather than taken from eidos's own constants at each use so the
// set a reader has to hold is one list. Each is suite-owned under
// docs/adr/0018 — no [engine/model/law] property covers it — and each is
// derivable from the stamp plus the signature, which is what excludes the rest:
// `validates` and `scope` need a value no run can invent, and `sideeffect`,
// `partition`, `hooks` and `sample` name a partner the mixin schema declares
// no parameter for.
//
// The one exception is `ttl`, whose law is the model tier's: its row exists
// because the reader-miss claim speaks the sentinel the ttl declaration
// names, and wording reads the declared home rather than respelling it.
const (
	MixinNilSafe           = nilsafe.Name
	MixinDeprecated        = deprecated.Name
	MixinIntegrationOnly   = integrationonly.Name
	MixinTimeout           = timeout.Name
	MixinTimeoutParam      = timeout.ParamDuration
	MixinOrderAfter        = orderafter.Name
	MixinOrderAfterParam   = orderafter.ParamFn
	MixinOrderAfterUnready = orderafter.ParamUnready
	MixinSideEffect        = sideeffect.Name
	MixinSideEffectParam   = sideeffect.ParamObserve
	MixinPartition         = partition.Name
	MixinPartitionRead     = partition.ParamRead
	MixinPartitionAxis     = partition.ParamAxis
	MixinHooks             = hooks.Name
	MixinHooksParam        = hooks.ParamRegister
	MixinSample            = sample.Name
	MixinSampleParam       = sample.ParamBuilder
	MixinValidates         = validates.Name
	MixinValidatesParam    = validates.ParamFn
	MixinWrappedVia        = wrappedvia.Name
	MixinWrappedViaParam   = wrappedvia.ParamFn

	MixinTimeAware          = timeaware.Name
	MixinIndexed            = indexed.Name
	MixinIndexedBy          = indexed.ParamBy
	MixinIdempotent         = idempotent.Name
	MixinAccumulates        = accumulates.Name
	MixinErrors             = errors.Name
	MixinScope              = scope.Name
	MixinConcurrent         = concurrent.Name
	MixinAfterClose         = lifecycleafterclose.Name
	MixinAfterCloseClose    = lifecycleafterclose.ParamClose
	MixinAfterCloseSentinel = lifecycleafterclose.ParamSentinel
	MixinTTL                = ttl.Name
	MixinTTLNotFound        = ttl.ParamNotFound

	// MixinNotFound is a read declaring its own miss sentinel, and
	// MixinNotFoundSentinel is the error it names.
	//
	// The bare fact, unscoped: every other sentinel in the vocabulary
	// belongs to a condition some other shape owns — expiry, deletion,
	// rollback, close — and a plain reader had no way to say what an
	// absent key reports. Without one it had to claim a TTL to get a
	// correct miss check, and claiming one earned a clocked law it never
	// asked for.
	MixinNotFound         = notfound.Name
	MixinNotFoundSentinel = notfound.ParamSentinel

	// MixinTotal is read as an exclusion, not a check: totality is the
	// declared claim that no input fails, so the zero-on-error family
	// is not emitted against it. The law half is the model tier's.
	MixinTotal = total.Name

	// MixinBounded declares a ceiling the subject holds to, and
	// MixinBoundedLimit is the number. Read here because the harness
	// hands it to every constructor: a bounded subject built at some
	// other capacity is one the bounded law measures against a limit it
	// was not given.
	MixinBounded      = bounded.Name
	MixinBoundedLimit = bounded.ParamLimit
)

// teardownShaped reports the one signature "a second call answers the same"
// can be stated against without a value: context in, error out, nothing else.
func teardownShaped(m subject.Method) bool {
	return m.TakesContext() && m.ReturnsError() &&
		len(m.ValueReturns()) == 0 && !m.HasInput()
}

// subjectOf names the interface every emit value for it is about.
func subjectOf(iface *sdk.Interface) subject.Subject {
	return subject.Subject{
		IfaceName:      iface.Name,
		IfaceRef:       golang.RefFor(iface.Name, iface.Package),
		Runtime:        Module,
		IntegrationEnv: GoIntegrationEnv,
		ClockRef:       golang.RefFor("Clock", Module+"/clock"),
		TypeParams:     golang.TypeParamDecls(iface.TypeParams),
		TypeArgs:       golang.TypeParamNames(iface.TypeParams),
	}
}

// fixtureArgs names the fixture field per parameter the method takes after its
// context, taking the second value of each when alternate is set.
func fixtureArgs(f subject.Fixture, m subject.Method, alternate bool) []string {
	args := m.CallArgs()
	out := make([]string, 0, len(args))
	for _, p := range args {
		// The fixture's own name for the field, not the parameter's: two
		// methods naming one parameter at different types get one field each,
		// and a check has to reach its own.
		name := f.FieldFor(p)
		if alternate {
			name += subject.OtherSuffix
		}
		out = append(out, name)
	}
	return out
}
