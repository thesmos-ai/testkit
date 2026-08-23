# Model

> **Status: shipped.**
> [RFC-0003](../../rfc/0003-the-projection-consumers.md) fixes this
> generator's design. What ships today: the directive, the differential
> property (actions, pools, derived references with their mixin
> refinements, law bindings), the isolated-law walk, the test clock and
> the clocked law family, generic interfaces at their witnesses, the
> concurrent Porcupine leg, the fuzz target, the generated companion that
> proves the emission — the reference's self-conformance, the inert-body
> probes, and the mutation kill matrix — and the consumer options. Where
> this page and the RFC differ, this page reflects what shipped: the RFC
> records the design, including the bounded exhaustive search that was
> since deleted as dead capability.

The `model` generator binds the classifications
[ADR-0018](../../adr/0018-one-tier-owns-each-classification.md) assigns to
the model tier onto the shipped `engine/model` runtime: property-based
state-machine testing on [`pgregory.net/rapid`](https://pgregory.net/rapid),
linearizability checking on
[`anishathalye/porcupine`](https://github.com/anishathalye/porcupine),
and a mutation self-check that proves the bound laws can kill injected
bugs. Consumers get law-backed conformance for one
directive; everything below is derived from the shape stamps and the
[suite generator](suite.md)'s projection.

## The directive

```go
//testkit:model
type Store interface { ... }
```

Interface-scoped, negation denied — the tier exists where one is declared,
and deleting the line is the suppression, the same shape as
`//testkit:stub` and `//testkit:suite`. Three keys: `ref=` names a
reference constructor where no shipped oracle models the shape, `gen=` a
generator constructor for a value type reflection cannot draw, and
`witness=` the concrete types a generic interface's property runs at —
comma-separated, one per type parameter, in declaration order. The witness
list is required exactly where the interface is generic: the pools, the
reference and every law assert *through* those types, so the source names
them or the generator refuses with the key that would fix it. Everything
the file emits then lands at the witnesses — the instantiated subject
type, the pools, the derived oracle — and the option plugs into the
consumer's matching instantiation of the harness. The generator emits where the
directive stands and `suite` queued a projection; a directive on an
interface `suite` never touched, or one where no classification maps to a
law, is a diagnostic at the directive.

The directive is what admits the dependency: the primary output imports the
`engine` module, and through it rapid and Porcupine — a requirement a
classification line alone must not impose. On an interface carrying
model-owned classifications and no `//testkit:model`, the suite harness
header names the tier as unarmed and the line that arms it, so an owed
assertion is visibly waiting rather than silently absent
([ADR-0017](../../adr/0017-every-classification-owes-an-assertion.md)).

Once armed, the laws run inside the ordinary `Assert<Iface>Contract` entry
against each subject's plain form; drop by path with
`<Iface>Without("model/AUTO-…")`, or delete the directive to shed the
emission and the dependency together.

A second directive is declared here: `//testkit:domain-gen <Type> <Func>`
registers a [`rapid.Generator`](https://pgregory.net/rapid) for an opaque
domain type the generator cannot synthesize from reflection. An opaque
parameter with no hint is a diagnostic at the parameter.

## What one interface gets

- **A law registry** — one binding per mapped classification, instantiated
  at concrete types. Law fields fill by a fixed taxonomy: role closures
  from the stamped method, generators from the shared derivation, constants
  from the classification's own parameter stamps, trace handles from the
  runner.
- **An action set** — one `engine/model/action` constructor call per
  method, matching its detector or contract; partner-role methods are
  excluded (the suite tier owns their checks).
- **Legs, one per claim shape** — the differential over random sequences,
  a law leg per bundled law, a row of its own for every law the catalogue
  words, the linearizability leg for whichever of five concurrency
  families the shape selects, and the crash schedule where an
  acknowledged write means "this key holds this record until something
  else writes it". Each states one sentence and reports under its own ID.
- **Generators, derived once** — keys from a small sampled set, values
  blending the fixture pair with arbitrary `model.Make` draws, the value
  type's key field pinned to the key pool, shared by every path.
  Collisions are what make the laws fire; the wide bodies are what makes
  a same-key overwrite carry a body no fixture spells. A claim that
  narrows the accepted value domain (`validates`, `sample`) keeps the
  pool to the proven fixture pair, and the header says so.
- **A derived reference** — an adapter over the matching
  `engine/model/ref` oracle: the map for value-carries-key stores, the
  keyed store for key-beside-value writers, the collection for
  append-and-drain — each refined by claim (`noduplicates` and `crdtmerge`
  dedupe, `sticky` pins resolutions, `snapshotisolation` and `chain` force
  the log over the upsert inference). A contract claim outranks the
  shapes: where its role vocabulary resolves completely — the carrier's
  `role=`, the partner keys — the contract's own store stands in
  (`lease` → `LeaseTracker`), its constructor sentinels minted or
  lenified per the claims, and an oracle whose every sentinel lenifies
  away falls to the twin, because the kill matrix proved a
  never-disagreeing store checks nothing. Where no store models the shape, the
  **twin floor** stands in: a second instance from the subject's own
  factory, which catches nondeterminism and hidden shared state but not a
  subject wrong the same way twice — the header says why the floor was
  reached, and `ref=` raises it. The sequences drive only what the oracle
  models; a method the adapter holds inert is skipped by name.
- **A clock, where a claim reads time** — the `ttl`, `windowed`,
  `timeout` and `scheduled` families bind laws that age entries, roll
  windows and fire schedules. They arm only under
  `<Iface>ModelClocked(func(clk *clock.TestClock) T)`, which builds the
  subject on the run's own `TestClock`; the clocked laws advance it and
  nothing else does. A clocked run forces the twin reference even where a
  map oracle derives — under the clock the derived oracle lies, because
  mirrored writes age on the subject alone — so twins on one clock age
  and fire together. The scheduled law mirrors every accepted schedule
  onto the reference and asserts *at least* its batch fired after the
  advance: the ambient action stream schedules beside it, so exact-count
  is the quiescent claim the fixture's unit tests keep.
- **An isolated walk, where a law corrupts** — the close/poison/tamper
  family (`CursorCloseIdempotent`, `IdempotentLifecycle`,
  `PoisonConsistent`, `TamperEvident` and kin) runs once per iteration
  against a throwaway pair from the subject's factory; the shared pair
  never meets them. A law whose precondition the run's draws cannot
  satisfy reports vacuously, counted apart from a pass — a law vacuous on
  every check past the census floor is named in the run's log, because
  sixty vacuous returns are sixty times a binding asserted nothing.
- **Per-client session laws, where a version is named** — the five session
  mixins (the four read/write-ordering guarantees, and `causal` since its
  param landed upstream) accept `version=<member>`, naming the field of
  the value that carries the store-assigned ordering stamp. The causal law
  additionally binds through its supplied `HappensBefore` door — the
  ordering is the consumer's domain relation, never a shape's. The generator emits one
  classifier per interface — trace event in, per-client read or write out —
  and binds the read-ordering law over it; the laws run over the sequential
  trace and, with real client IDs, over the concurrent leg's multi-client
  interleaving, where Porcupine stays out (a store-assigned stamp defeats
  value equality — the model is stepless and the trace laws are the check).
  A version-stamped fixture forces the twin reference for the same reason.
  The write-ordering laws bind beside an answering writer — the
  `(ctx, V) (V, error)` shape whose answered state carries the stamp — and
  refuse by name on a writer answering only an error, which hides the
  version the store assigned from the trace.
- **A typed door for every supplied law field** — a law whose closure is a
  domain fact (a merge algebra, an ordering, a transaction history) cannot
  be derived, so the generator builds the door instead: one
  `<Iface>Model<Field>` option per supplied field, spelled at the
  fixture's own types, with the law registering only when armed and the
  header marking it `(supplied: field)`. The corpus arms the stable-order
  `Less`, the lease `Free` and the pool `Stats` doors end to end — the
  lease door's first run caught the reference subject holding keys past
  their cancelled context.
- **A derived subscription drain, where a publisher claims delivery** — the
  publisher contract's laws (delivers, and the `mode=` bounds at-least-,
  at-most- and exactly-once) bind through one generated sweep over the
  subscription channel: a non-blocking read of everything Publish already
  delivered. That is the synchronous floor — an asynchronous publisher
  supplies `<Iface>ModelDrain`, which outranks the derivation, or the floor
  reads its in-flight deliveries as loss; the header states which drain is
  in play. The redelivery arm binds where the directive names a redeliver
  role — an error-returning method the law re-offers the published message
  through, a refusal holding vacuously — and where nothing declares one
  the header's `(unarmed: Redeliver)` annotation says the bound was
  exercised on the single publish alone.
- **The contract-shape closures** — laws over roles whose signatures carry
  a handle, a callable or a cursor bind through closures the shared pools
  cannot spell. A `Begin(ctx) (Tx, error)` threads its handle into the
  commit/rollback pair for both two-phase laws; a saga's step role earns a
  coordinating run that steps drawn values, unwinds the committed prefix in
  reverse through the pinned compensation, and restores everything it
  committed on every path (the mirrored re-run draws fresh values, so a run
  that left state behind would desynchronize the twin pair); a singleflight
  `Run(ctx, key, compute)` is counted through the generator's own locked
  probe; a transaction's body-taking run receives the law's induced
  failure; and a `Page(ctx, cursor) (items, next, more, error)` walk pays
  both paginator laws, keyed by identity where no projection derives.
  Each shape refuses by name where the fixture's role does not carry it — a
  flat begin, a computeless run, a keyed read with no cursor to resume
  from. The atomic mixin joins the oracle-defeating claims: its law is
  about refused writes, a derived map refuses nothing, and the twin shares
  the policy.
- **Multi-replica closures, where convergence is the claim** — the
  eventually mixin's `settle=`/`sync=` members compose into closures over
  the whole replica set: settle runs per replica, and the pairwise sync
  becomes the star round (the hub absorbs every spoke, then every spoke
  absorbs the hub — the minimal exchange that provably reaches a lattice's
  join). The snapshot is the shared whole-state observation, and the merge
  stays the consumer's door: the join is the domain's algebra. The
  replay-causality law binds the same way on a chain co-stamped `causal`,
  through its supplied identifier and dependency doors — and causal joins
  the oracle-defeating claims, because an admission policy is a refusal a
  derived log never makes.
- **Member closures, where a directive names a handle's methods** — the
  watcher contract's `next=`/`stop=` params resolve against the
  subscription the watch role answers (the resolver's member scope), and
  the generator derives the bounded read and the teardown as calls on the
  handle — the stamp names the method, the law's field declares its shape,
  and the compile gate in the armed package holds the two together. A
  handle-answering read drops from the drawn sequences: an interface
  compares by identity, and two runs' handles never share one.
- **The mid-transaction door, at the trio's own types** — the no-mid-tx
  law binds on the begin/commit/rollback trio beside its keyed read: the
  generated closures thread the handle, the outside read observes committed
  state, and the staged write itself is the `TxPut` door — spelled at the
  handle, key and read-back types, because how a store stages is its own
  business and the consumer's closure reaches its subject's staging API
  directly. The law's value pool draws the read's answer (`readback`),
  since no role input names what the store holds.
- **Version-coherent CAS draws, and an append log the runner clears** —
  the cas contract's one-winner law stamps both drawn attempts at the
  cell's current version before racing them (the VersionedCell dialect:
  the seen version advanced by one, zero for an unreadable cell), because
  two attempts at a stale version are two mismatches and no winner. The
  chain contract's no-drops law reads a generated append history: the
  append action logs every success into it, the law checks membership
  against the replay, and the runner clears it each iteration through
  `WithHistoryReset`.
- **A concurrent path** — where the unrefined map pair derives,
  `<Iface>ModelConcurrent` runs four workers interleaving the reader and
  writer over the same shared pools, Porcupine-checking the history
  against `linearize.KV` per key. It registers beside the sequential leg
  as `model/concurrent`; the laws stay sequential, whose step boundary
  they need, and the companion holds the leg to the derived reference.
- **Contract-role sequences** — a role method joins the sequences as
  itself where a constructor fits its fixture shape: the tx trio is driven
  as one `action.TwoPhase` cycle threading each begin's own handle into
  its drawn terminal (the standalone terminals are dropped, because a
  commit drawn from a value pool operates on handles no begin minted),
  and updater, upserter, cas and chain writes carry their role's own
  constructor name. The family members whose shapes the constructors do
  not fit are argued refusals in the tiers table, not gaps.
- **The miss identity in the sequences** — where the declaration stamps a
  sentinel, every error-answering reader action carries
  `action.WithSentinel`: a pair agreeing a read fails must also agree on
  whether the failure is the declared identity, so a subject missing
  under a private error stops reading as agreement.
- **The versioned-cell and append-log legs** — a `cas` contract on the
  shipped VersionedCell oracle earns a Porcupine leg against
  `linearize.CASCell` in the oracle's own dialect (stamp is seen+1, an
  empty cell matches only the zero version, the stamped mismatch is the
  identity matched); an `appender` contract whose method answers `int64`
  offsets earns one against the shared `linearize.AppendLog`, where a
  torn append hides from the per-client law. A chain append answering no
  offset and a keyless fold derive no leg, for reasons recorded at the
  derivation.
- **`version=` refused by name** — the ordering stamp is read and
  assigned as a field selector, so a `version=` naming a zero-arg method
  or nothing at all dies at the directive with a diagnostic instead of
  surfacing as an unattributed build error in the consumer's package.
- **The saturation prover** — binding a law is necessary;
  `<Iface>ModelSaturation(t, factory, opts...)` is what makes it
  sufficient. Per bound law, a defect is worn on each method the law's
  closures reach — zeros, the fixture pair alternated, a waning or waxing
  count, a sputtered refusal, a fading replay, an echoed page — and at
  least one worn run must fail naming the law's own identifier, with the
  clean factory standing as reference so the twin floor has real teeth.
  A law every defect survives is bound but unsaturatable and fails by
  name; a law behind an unarmed door or the clocked factory is skipped
  visibly. The derived-tier companions run it automatically; the corpus's
  consumers run it against their subjects, and so can any consumer.
- **A report header** — the generated docblock is a per-method table of
  what the run derived: actions, law IDs, the cluster map, what was skipped
  and the option that arms it.

## The self-checks

`_model.gen_test.go` carries the emission's own proofs:

- **`Fuzz<Iface>Model`** — the interface's *sequence space* as one fuzz
  target: `model.MakeFuzz` lets the coverage-guided engine drive rapid's
  choice stream, hunting for the action ordering that breaks a law. One
  target per interface.
- **The mutation kill matrix** — `Test<Iface>ModelKillsInertMutants`:
  one mutant per driven method, each a reference whose one method answers
  zeros and forwards nothing, and the property must fail every one. An
  unkilled mutant means that method's participation checks nothing — a
  hole in the derivation, named by method, asserted at a total kill rate.
  Each probe runs under a named failure surrogate and removes the
  failfiles and artifacts its expected failure provokes.

The last two are emitted only where the reference is derivable — both
certify subject-versus-reference agreement. All three, plus the concurrent
path, honour `testing.Short()`; the sequential law run does not skip,
because it is the assertion a declared classification owes.

## What a consumer writes

Nothing beyond the suite wiring: pass `<Iface>Model()` to the contract
entry. The options are the escape hatches: `<Iface>ModelReference(factory)`
replaces the derived oracle, `<Iface>ModelValues(gen)` replaces the values
pool wholesale, `<Iface>ModelClocked(build)` hands the subject the run's
test clock where a claim reads time, the `gen=` directive key names a
generator constructor in the routed package for a value type reflection
cannot draw (a pointer payload, an invariant-carrying domain type), the
`ref=` key names a reference constructor where no shipped oracle models
the shape, and `<Iface>Without("model")` declines the tier. `<Iface>ModelFuzz(f, factory)`
is the one-line fuzz wiring: the fuzzer's bytes replay as rapid's choice
stream over the subject's own branches.

Beyond the options, this tier puts two surfaces on the harness a consumer
does write against. `Prop` and `Prop<Method>` are drawn-input check bodies
— the argument arrives from the same pool the generated checks draw from,
and a failure reported through the `PropT` shrinks. And the doors: a
clocked claim asks for `OnClock`, a poison claim for `Induce`, and the
crash schedule for `Recover`. A door appears only where some check needs
it, so a field you are given is a field something reads.

## Layout conventions

This tier emits no file of its own. It contributes into the regions the
harness generator hands out, so everything below lands in
`<source>_suite.gen.go` beside the checks it belongs with:

| Region | Contents |
|---|---|
| `suite.checks` | one expression per contributed row, appended to the run's check list |
| `suite.decls` | the rows function, the generators, the derived reference, the action set, the leg bodies, and the `PropT` alias |
| `harness.fields` | the doors a contributed row needs — `OnClock`, `Induce`, `Recover` |
| `harness.lowering` | the line carrying each door onto the runtime subject |
| `check.fields` | `Prop` and `Prop<Method>` on the row type |
| `check.bodies` | the dispatch turning one of those into a body the runner calls |

One file rather than two, and the reason is the manifest: a package with
a model tier used to carry two check sets that had to be kept in step by
hand. A row is a row wherever it was derived, and a consumer reading the
report cannot tell which generator wrote which.

## See also

- [Suite](suite.md) — the projection and registry this generator reads, and
  the tier that owns the non-law checks.
- [Stub](stub.md) — the double; the model tier runs subjects plain, never
  wrapped.
- [RFC-0003](../../rfc/0003-the-projection-consumers.md) — the design
  record: the cluster rule, the law-field taxonomy, the probes, the
  integration matrix over every `engine/model` subpackage.
