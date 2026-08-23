# Derivation rules: source → generated conformance

What the suite/model/sim generators infer, from which input, with which
rule — written down before the generator exists, from what the `gen/`
corpus proved by hand. Every rule here is demonstrated in
`gen/example/kvtest` and litigated in an RFC-0004 amendment; the
amendment is the rationale, this file is the lookup table.

## Inputs

- **Directives** (`//testkit:*`): the only licensed claims. `suite`,
  `model`, `stub`, `builder` opt a surface in; `ctx`, `appender`,
  `mixin`, `role`, `default` state claims the checks enforce.
- **Shapes**: method signatures classify (Reader, Writer, Aggregator,
  Lifecycle, Appender, Stream) exactly as `engine/model`'s tiers do.
- **Prose claims**: a documented sentence becomes a check only when a
  directive or a sentinel carries it (ErrClosed's "returned by every
  method after Close" → the after-close law). Prose alone licenses
  nothing.

## Per-tier rules

The validation packs amended this table; the A/B/C/D series below are
the litigated deltas, and where a row here conflicts with an amendment,
THE AMENDMENT WINS. The rows most affected carry their pointers inline.

| Input | Rule | Output |
|---|---|---|
| a `context.Context` first parameter | the context families derive from the shape — SUPERSEDES A13's directive gate and B4's opt-out row. Close carries cancel and nilcontext but never deadline (A5); zero-on-error rides the same shape, its error source being the context the method already takes | signature/cancel, /deadline, /nilcontext, /zero-on-error families |
| method shape | one smoke per method, always | signature/smoke |
| `//testkit:role key\|payload` + `//testkit:default` | pool[0] = the stamp verbatim; pool[1] = the distinctness transform; pool[2] = the hostile member (A14, A16) | derived config pools |
| pool provenance | derived pools blend `AdversarialStrings`; a supplied pool reaches every tier verbatim (panel 1.2) | model draw generators |
| Reader+Writer over one key type | derived reference = `ref` map oracle wrapped to the interface. NOT universal: an evicting/bounded reader takes the asymmetric hit-subset comparison instead (B3), and a channel-returning method compares the open path only (A8) | model/<iface>/differential |
| Append+Replay | derived reference = plain append log (hash-less); `ref.AppendOnly` when a chain hash is declared | model/journal/differential |
| written type ≠ read type | the Porcupine spec is built with `linearize.NewModelBuilder` per-op steps; stock `KV` only when they unify (A14) | the concurrent leg's model |
| "safe for concurrent use" sentence | linearizable leg; lifecycle methods excluded from the recorded set; global reads ride as stress | model/<iface>/AUTO-LINEARIZABLE |
| `//testkit:mixin ttl` | bind `timeaware.TTLExpiryAfterAdvance` on a per-iteration clock, LawsOnly leg (A16). The lifetime comes from `duration=`, or where the declaration stamps none, from the drawn value's own `time.Duration` field — the largest the pool carries, since the law advances once and holds every value it drew to that. Two duration fields on the value declines: which one the claim is about is the declaration's to say | model/<iface>/AUTO-TTL-EXPIRY |
| sentinel + Induce seam | bind `law.PoisonConsistent`; probe maps the sentinel onto the read surface | AUTO-POISON-CONSISTENT |
| `//testkit:mixin lifecycleafterclose close=<M> sentinel=<E>` per sentinel-reporting operation | bind `law.LifecycleAfterCloseSentinel` — RATIFIED: the mixin is the licensed home (the earlier kv prose-derivation contradicted the "prose licenses nothing" input rule) and the gen sources now carry it: kv Put/Get/Len, cache Put/Len. A method without an error channel (cache Get) has nothing to speak the sentinel through and is not stamped | AUTO-LIFECYCLE-AFTER-CLOSE |
| `//testkit:appender` | bind `law.AppenderMonotonicOffsets` on its own leg (it writes) | AUTO-APPENDER-MONOTONIC-OFFSETS |
| chain shape | HistoryGrows + NoDrops (over `history.History` the append action records) + ReplayDeterminism, bundled (A18) | model/<iface>/laws |
| law IDs | always `"model/<iface>/" + lawid.X` — never invented (A16) | index + lock rows |
| ≥2 model-bearing interfaces in a package | family scopes carry the interface segment (A18) | ID grammar |
| Recover seam + durable medium | recovery check + crash-durability + faulty-medium sims; `Excuse` for memory-only subjects (A15, A19) | sim family |
| reader-only interface | the harness constructor RECEIVES the corpus zipped from the pools; hit/miss/size checks read against exactly it (§7.1) | seed seam |

## Defect-emitter rules (proofs)

Five mechanical rules cover most law-family proofs — see the table in
`kv_proofs.gen_test.go`: discard-write, freeze-return, fresh-medium,
sentinel-once, strip-role-field. Signature-tier proofs are stub-transform
one-liners (panic, ctx-swap, nil-deref, echo-beside-error). A family
with no rule ships `Argued`, never an underived Proven.

## Ownership split (per the A20 rulings)

The GENERATOR owns and regenerates: checks, model/sim bindings, index,
lock, stub, builder, doc, self-checks, package proofs. `testkit scaffold`
emits ONCE, consumer-owned: the wiring file (harness literal with
discovered constructors as TODOs), the checks table, and the proofs test
(`TestChecksCanFail`) — scaffolding is the proofs-enforcement mechanism.
The alternative (scaffolded consumer-owned model bindings) was considered
and not taken: bindings derive fully from the rules above, and what
derives fully must regenerate, or it drifts.

**Negative controls are CONSUMER-AUTHORED, never emitted.** A control is
a legal-but-different implementation the whole suite must pass — it
measures specificity, where the proofs measure sensitivity — and
different-but-legal is invented, not derived: the generator can derive
THAT a freedom exists (`mixin bounded` names no victim, a contract mode
permits duplicates), but writing an implementation that exercises the
freedom is creative work. Corpus evidence: the five packs' controls
(MRU eviction, FIFO recycling, bounded dawdling, lazy-behind-a-boundary,
benign undefined-input readings) each embody an insight no rule spells.
They live in the consumer package (`example/*_control_test.go`) on the
seams every consumer has: `prove.Green` over the suite's checks, or the
run's own harness where a fixture reaches a subject only through the
seed seam (Catalog's corpus). The generator's remaining duties, both
PROPOSED: `testkit scaffold` emits the control slot beside the proofs
test, and the report carries a note when a policy-freedom-bearing
mixin has no control registered — absence must be visible, the same
contract the vacuity notes keep.

## Extension seam (per interface, derived)

Every `//testkit:suite` interface gets the full extension surface —
checks table, `Broken<I>` defect sugar, `Prove<I>` — because uniform
wiring must not imply Store-only extension, and the sim lifecycle
(consumer-authored crash/fault checks first, promoted later) needs the
seam on every interface. The row shape is DERIVED, never copied:

- **A method gets a `Prop<Method>` sugar field only if it has a
  drawable domain input.** `Append(ctx, Entry)` earns `PropAppend`;
  `Replay(ctx)` and `Size(ctx)` earn nothing. The sugar draws from the
  same source the model tier uses (pools with provenance for Store,
  the Entry blend for Journal, seeded keys plus the miss key for
  Catalog), so a row's draws and the generated tier's draws cannot
  disagree.
- **The bodies' extra parameter is the interface's draw source, only
  where one exists.** Store rows get the fixture, Counter rows the
  normalized config, Catalog rows the seeded corpus; Journal rows get
  nothing, because Journal has nothing a run overrides. `Prove<I>`
  takes a config parameter exactly where `Run<I>` accepts one, so
  proofs always bind to what the run binds to.
- **Defect sugar follows the constructor seam.** `BrokenCatalog` takes
  a seed function because Catalog's constructor is the seed seam — a
  planted defect there is a loader that betrays its corpus.

## Class derivation

- **Class defaults to the ID family; divergence is a deliberate source
  annotation.** The generator derives `Class` from the check's family
  (`signature/cancel`, `model/laws`, ...) with zero authoring burden;
  only an annotation overrides it, the way `own/durability` classes a
  Put-scoped check by regime. The axis exists for the report's by-class
  histogram — the one view that answers "how much of my confidence
  comes from simulation vs. signatures" — and its real customer is the
  sim tier, whose classes (`sim/crash`, `sim/fault`, `sim/recovery`)
  are inherently cross-cutting. `checks.lock` rows carry Class, so
  removing the axis later is a manifest-format break; keeping it costs
  this rule.

## Lint posture the generated code assumes

The corpus is lint-clean under the repo config except four classes that
are findings about the CONFIG, not the code. A generator emitting at
scale must resolve each at the config level — per-site `//nolint` in
generated or exemplary consumer code would teach consumers noise:

- **thelper** — the vocabulary's callback signatures (check bodies,
  harness `Start`/`Recover`, planted-defect closures) take `testing.TB`
  as the CHECK'S parameter, not as a helper's. `tb.Helper()` there
  would be wrong: a check body's failure must point at its own
  assertion line, not the runner's call site. The config needs the
  callback shapes exempted.
- **forbidigo** — the root pattern intended for `fmt.Print*` logging
  also matches `fmt.Fprintf` into a `strings.Builder`, which is
  rendering. The pattern should stop at writers it cannot name.
- **tparallel** — the runner owns leg parallelism (the `Serial` field,
  excused legs); a static heuristic cannot see a conditional
  `t.Parallel` and flags every driver test.
- **depguard** — `PropT = model.T = rapid.T` is a type ALIAS: rapid is
  part of the vocabulary's property surface by construction, so
  consumer test files must be allowed to import it. The runtime
  deny-list needs a test-file carve-out.

Point exemptions that ARE per-site stay in the code with reasons:
gosec G204/G703 and musttag-on-`time.Time`, each `//nolint` explaining
itself.

## Amendments — validation pack 1 (bus, `gen/example/bustest`)

Rows the second domain forced; each verdict is litigated in
`poc-validation.md`, which cites the emission that proves it. Rules
above stay as written; a future consolidation folds these in.

| Input | Rule | Output |
|---|---|---|
| `//testkit:contract publisher role=publish subscribe=<M> [redeliver=<M>] mode=<mode>` (A1) | bind `law.PublisherDelivers` plus the per-mode `law.PublisherDeliveryBound`; redeliver names the redelivery method (Publish itself for a plain bus); each on its own leg (delivery laws write). The engine law docblocks spell this directive bare (`//testkit:publisher <Subscribe>`) — DRIFT against the conformance corpus's grammar; upstream docblock fix owed | model/AUTO-PUBLISHER-DELIVERS, model/AUTO-PUBLISHER-&lt;MODE&gt; |
| roled bare parameters (A2) | `role`/`default` stamps live on the named TYPE declaration when no request-struct field exists to carry them | derived config pools |
| Publisher+Subscriber over one key type (A3) | derived reference = never-dropping fan-out (key → open subscriptions), written against the shape | model/differential |
| a domain whose laws all write (A4) | no observational bundle: each law rides its own leg, and no `model/laws` bundle leg exists | per-law legs |
| a context-taking teardown (A5) | Close carries cancel and nilcontext, never deadline — unchanged by the directive's removal, since it was always a fact about the shape | signature families |
| "safe for concurrent use" on async fan-out (A6) | the Porcupine rule DOES NOT APPLY: no per-key register models async delivery; the claim lowers to delivery laws under concurrent load (engine work, deferred) | no AUTO-LINEARIZABLE leg |
| delivery defect rules (A7) | D1 partial-fanout (one subscriber fed, the bound reds alone); the differential's defect is a REFUSAL, which every delivery law reads as Vacuous — single-claim by construction | proofs |
| channel-returning method (A8) | lowers to Subscriber/Watcher: the differential compares the open path only; deliveries are the delivery laws' observation — never compare channels | model/differential scope |
| law-driven cycles on drawn-pool keys (A9) | delivery laws run on a law-private key outside the pools, or ambient action traffic lands in the law's drain | law leg wiring |
| capability fields (A10) | the harness carries only the fields its emitted check set can demand — no OnClock without a clocked check, no Recover without a sim family | harness typed surface |

Open, not derivable from shape: the quiescence seam an ASYNC bus needs
before delivery laws can drain honestly (the bus contract is synchronous,
which licenses the non-blocking drain this pack uses).

## Amendments — validation pack 2 (cache, `gen/example/cachetest`)

The bounded-nondeterminism rows; litigated in `poc-validation.md`
pack 2. The directive spellings follow the corpus grammar — mixin for
single-method claims, contract for multi-method protocols — which is
the vocabulary's one home; engine law docblocks that spell directives
bare are drift, and their fix is owed upstream.

| Input | Rule | Output |
|---|---|---|
| `//testkit:mixin bounded limit=N` (B1) | `AggregatorBounded[0..N]` in the observational bundle; the literal is the bound, the suite owns the number, and every harness constructor RECEIVES it — the seed seam generalized to a scalar. N must sit within the drawn-sequence budget's reach, or the eviction path ships untested behind a law that reads as coverage | model/laws + the harness ctor shape |
| `//testkit:mixin cacheable` on an errorless reader (B2) | `Cacheable` over the packed (value, ok) observation | model/laws |
| bounded/evicting reader (B3) | the stock differential DOES NOT APPLY: a subject hit must agree with the unbounded reference, an unexplained hit is invention, a miss is always legal; Len is excluded from the differential — the bounded law owns it. DELIVERED: `action.EvictingReader` carries the asymmetry, and the binding is one constructor call | model/differential |
| ~~no `//testkit:ctx`~~ (B4) | SUPERSEDED: the families derive from the parameter, so a context-taking method carries them whether or not anything is declared. A subject that legitimately never observes cancellation drops the check by ID rather than by withholding a directive | signature tier |
| law-leg tiers (B5) | a laws leg notes a reference tier only when a bundled law actually reads the reference; a bundle of subject-only observations notes none | tier honesty |
| errorless method on a stub (B6) | an injected fault answers the zero return — the signature has no error slot; the hazard is documented at the stub | stub emission |
| bounded defect rules (B7) | I1 invent-hit (fabricated presence; differential-only by construction), G1 exceed-bound (a count past the limit — deterministic under the proof budget, where a never-evicting twin is hostage to sequence length and slipped a first-draft proof) | proofs |

## Amendments — validation pack 3 (pool + lease, `gen/example/pooltest`)

The resource-lifecycle rows; litigated in `poc-validation.md` pack 3.

| Input | Rule | Output |
|---|---|---|
| `//testkit:contract pool role=get put=<M> stats=<M>` (C1 — base spelling `role=get put=Put` is corpus-canonical; `stats=` is the PROPOSED extension wiring the balanced laws' observation, which the corpus fixture leaves unwired) | the cycle action plus a Stats aggregator in the differential; balanced-accounting and leak-free bundle reading stats= | model/&lt;iface&gt;/differential, model/&lt;iface&gt;/laws |
| cycle-shaped resource pair (C2) | quiescence laws bind ONLY over cycle-shaped actions — get-then-put per invocation, quiescent between actions, which is where laws run; a bare borrow action reds correct code | action lowering |
| contract-owned context semantics (C3) | a contract may claim ctx behaviour (release-on-cancel) with NO `//testkit:ctx` declared; the signature families and a contract's ctx claim are independent axes | check emission |
| contract key parameters (C4) | a contract role's key parameter derives its pool from the contract itself — plain types, no role stamp | derived config pools |
| unobserved freedom (C5) | absent an observer method, lease freedom probes through the acquire/release pair, self-cleaning | law closures |
| polling laws (C6) | bind on the STANDARD law leg under `law.Budget(1, …)` — DELIVERED: one polled invocation per property iteration, refilled by the runner's per-iteration reset, which is also what keeps rapid's shrinking deterministic. A red still short-circuits at its first failing iteration (~2.5s measured for a broken subject); the earlier bare-loop-with-manual-census binding is retired | law leg shape |
| pool-produced inputs (C7) | a method whose input a sibling produces (Put's Conn comes from Get) has no drawable domain input: no Prop* sugar, and its smoke borrows first | row shape + smokes |
| resource defect rules (C8) | A1 lying-accounting (numbers the cycle count contradicts), L1 grant-always, L2 deaf-watcher (a correct lease under a context nobody watches) | proofs |

## Amendments — validation pack 4 (cursor log, `gen/example/scantest`)

The produced-secondary rows; litigated in `poc-validation.md` pack 4.

| Input | Rule | Output |
|---|---|---|
| `//testkit:contract cursor role=open next=<M> close=<M> sentinel=<E>` on a PRODUCER (D1, PROPOSED — the corpus spells role=next on a standalone cursor) | the produced-cursor arm: the directive sits on the producing method and names the cursor's roles | cursor law legs |
| produced secondary interfaces (D2) | no sub-harness exists: laws instantiate at the SECONDARY's type and lift onto the producer with `law.Produced` on the standard leg — DELIVERED. The producer is the constructor, one fresh secondary per check, a refused open Vacuous through the runner's own census; the direct-binding pattern is retired | law leg shape |
| cursor-shaped replay (D3) | lowers to the Stream action by drain composition — open, drain via next, close, compare the slice; no new action shape | model/differential |
| produced-interface surface (D4) | no smokes, no index entries, no stub for the produced type; its contract laws are its coverage, the opener's smoke closes what it opens, and stub emission follows `//testkit:stub` on the suite-bearing interface | signature tier + stub emission |
| cursor defect rules (D5) | K1 refuse-teardown (second close errors), K2 outlive-close (close acknowledged and ignored) — hand types over a real cursor, returned through the producer's stub override | proofs |

## Amendments — stdlib promotion (the emission diet)

The DRY/SOLID pass over the five packs promoted every idiom the
generator would otherwise re-emit per package. The emitter now TARGETS
these surfaces instead of expanding their bodies:

| Idiom the emitter used to expand | Emit instead | Home |
|---|---|---|
| reference pick, law leg, differential leg, adversarial blend, vacuity note | `legs.Reference` / `legs.Law` / `legs.Differential` / `legs.Blend` / `legs.NoteVacuity`, plus one `var _ = legs.CompatV1` witness per model file | `gen/legs` |
| the per-check `sig` closure | `sig := suite.ProvenCheck[Iface]` | `gen/suite` |
| the harness excuse/induction lowering loops | `suite.ExcuseSet(h.Excuse)`, `suite.LowerInductions[Iface](h.Name, h.Induce)`; the package aliases `type Inductions[T any] = suite.Inductions[T]` | `gen/suite` |
| the duplicate-config refusal in `applyTo` | `if !rc.ConfigOnce("XConfig") { return }` on the embedded Bundle | `gen/suite` |
| the `orDefault` panic wrapper | `suite.Must(c.normalize())` | `gen/suite` |
| the drop-hint function per interface | `var xDropHint = suite.DropHinter("XSuite", xIndexPath)` | `gen/suite` |
| the tb-shaped proof-defect constructor and reason pin | `one := prove.One[Iface]`; `.Reasoned(reason)` on the defect | `gen/suite/prove` |
| the stub `all []stub.Configurable` fan-out loops and cleanup-verify tail | `s.group = stub.NewGroup(members...)`; option one-liners delegate to `s.group`; `s.group.Bind(tb)`; `ResetCalls` is `s.group.Reset()` | `stub` (root module — rides the release train) |

What stays emitted per package, deliberately: tb-LESS `one` adapters
(they adapt `func() J` builders the package owns), capability-bearing
subject wiring (Recover's typed downcast), and every body that names
package types. The bridge doctrine: generated model files import
`suite` AND `legs`; `suite` still imports only `testing` and `clock`.

## Migration deltas from the incumbent

- **The per-run stub double-pass is dropped, deliberately.** The
  incumbent drove each generated suite against the stub double as an
  extra subject every run. Stub fidelity now lives in the generated
  companion self-tests (`stub.Behaviour` per method) — asserted once
  where the double is defined, instead of re-proved per consumer run.
  This is an improvement (the double-pass measured the double, not the
  subject), recorded here so it reads as a decision, not lost coverage.
- Doubles and builders emit into the generated sibling package, never
  the source package (panel 1.1).
- File spelling follows the incumbent: `<source>_<plugin>.gen.go`.

## Amendments — the second emission diet (references and actions)

Two rules replace bodies the emitter previously expanded:

- **A derived reference is a `ref` primitive plus signature adaptation,
  chosen by the same role classification that lowers the actions.**
  Keyed writer + reader → `ref.KeyedStore` (a (V, bool) read folds the
  not-found sentinel to absence); append + offset → `ref.MonotonicLog`;
  append + produced cursor → `ref.MonotonicLog` with `ref.BoundedCursor`
  over its snapshot (the cursor satisfies the produced interface as-is);
  get/put cycle → `ref.BalancedPool`; keyed acquire/release →
  `ref.LeaseTracker` (a nil free-release error spells release-of-unheld
  as a no-op); extractor-keyed writes → `ref.MapStore` (already the kv
  spelling). What stays in the emitted wrapper is exactly the
  signature-derived part: closed-state guards, error-channel folding,
  result-shape adaptation. A shape with no matching primitive is emitted
  whole and named as such — bustest's channel-shaped topic fan-out is
  the corpus's one instance; a primitive earns its place in `ref` on the
  second consumer, not the first.
- **A delegation closure emits as the `*Of` constructor with the
  interface method expression.** `action.WriterOf(id, gen, Log.Append)`
  replaces the closure restating the call — removing the one place an
  emitted action can call the wrong method and still compile. A closure
  that does more than delegate (binds an argument, records history,
  drains a produced value) keeps the closure-shaped constructor; the
  corpus keeps three: the topic-binding subscribe, the history-recording
  chain append, and the drain composition.

`suite.LowerRecover` joins `LowerInductions` for the same downcast in
the recover arm; the harness lowering is now loop-free end to end.

## Amendments — the remediation sweep (post-audit)

- **Family-scoped IDs carry the interface segment unconditionally.**
  Reopens the A18 conditional: an ID that renames when a second
  interface arrives breaks every manifest, dashboard and drop that
  recorded it. Applied corpus-wide while zero consumers hold locks.
  The unexported-name analog follows the same rule (kvtest's
  first-interface types take the store prefix). OPEN: signature-tier
  IDs stay method-scoped; two interfaces sharing a method name in one
  package need the same ruling before the emitter ships.
- **The lock is v2**: `ID<TAB>class<TAB>claim<TAB>binds`, where binds
  names the law IDs and probe sets a check delegates to — the column
  that makes an assertion-body change diff. The emitter fills it
  mechanically for every law-binding check; the corpus carries it on
  the structural-class checks that motivated it.
- **A lifecycle-after-close claim is exactly as wide as its probe
  set**: the law takes one probe per stamped method, the emitter emits
  the full stamped set, and the probe set appears in the lock's binds
  column. A claim string wider than the probe set is the silent-green
  class and must not be emitted.
- **Emitter duties recorded, not yet built**: nested request builders;
  a `signature/cancel-inflight` family for methods that block;
  generic-interface suites; per-method zero-on-error opt-out; the
  expected-artifact manifest behind absence-as-signal; claims derived
  from probe data (the full structural fix); a clocked-oracle bundle
  should a clocked differential leg ever exist; `testkit vet` for
  directive-prefix typos.
