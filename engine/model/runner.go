// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anishathalye/porcupine"
	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/coverage"
	"go.thesmos.sh/testkit/core/trace"
	"go.thesmos.sh/testkit/engine/model/law"
)

// OpInput is recorded as porcupine.Operation.Input for concurrent
// linearizability checking. Used by both the concurrent runner and
// the linearize package's model builders.
type OpInput struct {
	Name         string // action name ("Get", "Put", "Delete")
	PartitionKey string // for Porcupine partitioning; "" for unpartitioned
	Args         any    // shape-specific: K for Reader/Deleter, V for Writer
}

// OpOutput is recorded as porcupine.Operation.Output.
type OpOutput struct {
	Result any // shape-specific typed result
}

// Action is a named command that runs against both the SUT and
// reference. The runner dispatches actions randomly via rapid's
// state-machine orchestration.
type Action[T any] struct {
	// Name is the action name (e.g., "Get", "Put", "Delete").
	Name string

	// Run executes the action against both SUT and reference.
	// Returns an ActionResult with the comparison outcome and
	// structured I/O for trace recording.
	Run func(rt *rapid.T, sut, ref T) ActionResult

	// Reset clears whatever this action carries between iterations, and is
	// called before each one. Nil for the actions that carry nothing,
	// which is nearly all of them.
	//
	// The action slice is built once and every iteration reuses it, so an
	// action holding state — a subscription opened at one step and read at
	// a later one — would carry the previous iteration's into the next.
	// That makes the run unreplayable: rapid shrinks by replaying draws and
	// takes the verdict to be a function of those draws, and state from an
	// iteration the replay does not perform breaks it.
	//
	// The laws beside these have had [law.Resettable] for the same reason.
	Reset func()

	// Kind classifies the failure type when this action's Run
	// returns a non-nil Err. Set explicitly by every framework
	// helper; defaults to FailureUnclassified for consumer actions.
	Kind FailureKind
}

// ActionResult is the return value from Action[T].Run.
type ActionResult struct {
	// Err is non-nil when the action detected a divergence.
	Err error
	// CallErr is the SUT's own error for this call — a miss, a refusal —
	// recorded into the trace so a trace-scanning law can tell an errored
	// read's zero value from a read that answered zero. Divergence lives in
	// Err; agreement-in-error lives here.
	CallErr error
	// Input is the drawn value(s) for this action.
	Input any
	// Output is the SUT's result.
	Output any
}

// TraceResult lets a concurrent action's typed result speak the trace's
// vocabulary: the raw output a trace-scanning law reads, and the call's own
// error. A result that does not implement it lands in the trace whole.
type TraceResult interface {
	TraceOutput() (output any, err error)
}

// ConcurrentAction is an action that records structured I/O for
// Porcupine linearizability checking. Unlike [Action], it operates
// on the SUT only (no reference) and captures typed input/output.
type ConcurrentAction[T any] struct {
	// Name is the action name (e.g., "Get", "Put").
	Name string

	// Gen draws a random input using rapid and returns it.
	Gen func(rt *rapid.T) any

	// Apply runs the operation against an impl and returns the result.
	Apply func(ctx context.Context, impl T, input any) any

	// PartitionKey extracts the string partition key from the input.
	// Return "" for unpartitioned operations.
	PartitionKey func(input any) string
}

// ConcurrentConfig configures concurrent linearizability testing.
type ConcurrentConfig[T any] struct {
	// Workers is the number of concurrent goroutines. Default 4.
	Workers int
	// OpsPerWorker is the number of operations each worker performs. Default 50.
	OpsPerWorker int
	// Timeout for the Porcupine check. Default 10s. Zero means unlimited.
	Timeout time.Duration
	// Model is the Porcupine linearizability model. Use [linearize.KV]
	// for CRUD interfaces or [linearize.NewModelBuilder] for custom specs.
	Model porcupine.Model
	// Actions are linearizability-checked via Porcupine.
	Actions []ConcurrentAction[T]
	// StressActions run concurrently alongside linearizability workers
	// but are NOT recorded to Porcupine. Purpose: race detection under -race.
	StressActions []Action[T]
}

// Config holds the configuration for a model-based test run.
type Config[T any] struct {
	// SUTFactory creates a fresh SUT per test run. Required.
	SUTFactory func() T

	// RefFactory creates a fresh reference model per test run. Required.
	RefFactory func() T

	// Actions are the commands the runner randomly selects from.
	Actions []Action[T]

	// Laws are the invariants checked after every action.
	Laws *Registry[T]

	// LawsOnly makes the laws the run's only oracle: an action's
	// SUT-versus-reference disagreement stops being a failure, and
	// nothing but a law's own verdict ends an iteration.
	//
	// For asking whether a law can fail. The two oracles compete: the
	// actions compare every call against the reference and abort on
	// divergence, so a subject broken on purpose almost always dies of
	// the differential at step 0 and the laws are never reached. The
	// question "can this law catch a defect" then has no way to be
	// answered — every defect is caught by something else first.
	//
	// The reference is still built and still driven, so a law that
	// compares the two sides has both, and the subject's state stays
	// the state a real sequence would produce.
	//
	// Not for a conformance run. The differential is the strongest
	// oracle the tier has, and a run that silences it asserts only
	// what the laws happen to cover.
	LawsOnly bool

	// Cleanup is called on SUT and ref after each iteration.
	// Optional. Use for impls that hold resources (connections,
	// goroutines, file handles).
	Cleanup func(T)

	// HistoryResetters are called at the start of each rapid iteration
	// to reset per-iteration chain history traces. Wired by
	// [WithHistoryReset].
	HistoryResetters []func()

	// Concurrent enables concurrent linearizability testing.
	// When set, the runner spawns workers and validates via Porcupine
	// instead of running the sequential property.
	Concurrent *ConcurrentConfig[T]

	// DisableTrace skips per-action trace recording when true.
	// Set via [WithoutTrace]. Default false (trace enabled).
	DisableTrace bool

	// ArtifactDir overrides the directory for failure artifacts
	// (Porcupine HTML, goroutine stacks). Resolved via:
	// 1. This field (WithArtifactDir option)
	// 2. .testkit.yaml artifacts.dir
	// 3. Fallback: .testkit/artifacts/
	ArtifactDir string

	// StateHash hashes T's observable state for state-space coverage.
	// Set via [WithStateHash]. When nil, state-space coverage is not
	// tracked. Applied to the reference model each iteration when a
	// RefFactory is set (the canonical deterministic state), otherwise
	// to the SUT.
	StateHash func(T) uint64

	// SaturationThreshold is the number of consecutive iterations
	// without a new state after which the state space is reported
	// saturated. Zero uses the default. Set via
	// [WithSaturationThreshold].
	SaturationThreshold int

	// Coverage, when non-nil, receives the run's coverage signals —
	// state-space metrics and the REQ-to-law matrix — accumulated
	// across iterations. Set via [WithCoverageSink].
	Coverage *coverage.ComponentCoverage
}

// Run executes a model-based test. For each rapid iteration, it
// creates fresh SUT and reference instances, runs a random sequence
// of actions, and checks all registered laws after every action.
// When [Config.Concurrent] is non-nil, dispatches to the concurrent
// linearizability runner instead of the sequential property.
//
// Use -rapid.checks=N to control iteration count (default 100).
func Run[T any](t rapid.TB, cfg Config[T]) Outcome {
	t.Helper()
	return dispatch(t, cfg)
}

// dispatch is the shared entry point for [Run] and [Assert]. Routes
// to the sequential or concurrent runner based on [Config.Concurrent]
// and rejects unsupported combinations rather than silently dropping
// configuration.
func dispatch[T any](t rapid.TB, cfg Config[T]) Outcome {
	t.Helper()
	if cfg.Concurrent != nil {
		// Laws + concurrent execution is currently unsupported: laws
		// expect sequential SUT/ref comparison, but the concurrent
		// runner has no reference under linearizability and no
		// well-defined "after every action" boundary across workers.
		// Trace-scanning laws ride the concurrent history; everything else
		// needs the sequential step boundary and is rejected loud rather
		// than silently dropped.
		if cfg.Laws != nil {
			for _, l := range cfg.Laws.laws {
				if _, scans := l.(law.TraceBinder); !scans {
					t.Fatal("model: law " + l.ID() + " is unsupported with " +
						"Concurrent — it compares SUT and ref at a step " +
						"boundary the interleaving does not have. Only " +
						"trace-scanning laws (law.TraceBinder) run here; use " +
						"the sequential runner for the rest.")
				}
			}
		}
		runConcurrent(t, cfg)
		return outcomeOf(cfg.Laws)
	}
	rapid.Check(t, propertyFromConfig(cfg))
	return outcomeOf(cfg.Laws)
}

// Property builds the rapid property function from a factory and options
// without running it. Use this to obtain the property for [rapid.MakeFuzz]:
//
//	prop := model.Property(factory, model.WithReference(ref), ...)
//	f.Fuzz(rapid.MakeFuzz(prop))
func Property[T any](sutFactory func() T, opts ...Option[T]) func(*rapid.T) {
	cfg := Config[T]{SUTFactory: sutFactory}
	for _, opt := range opts {
		opt(&cfg)
	}
	return propertyFromConfig(cfg)
}

// propertyFromConfig builds the rapid property function from an
// already-built Config. Shared between [Property] (which builds cfg
// from options) and [dispatch] (which receives cfg directly via
// [Run]).
func propertyFromConfig[T any](cfg Config[T]) func(*rapid.T) {
	if cfg.Laws == nil {
		cfg.Laws = NewRegistry[T]()
	}

	// Coverage accumulators persist across rapid iterations (the
	// returned closure is called once per iteration). The REQ-to-law
	// matrix is static; the state-space tracker grows per iteration.
	var tracker *stateSpaceTracker
	if cfg.Coverage != nil {
		cfg.Coverage.REQToLaw = buildREQToLaw(cfg.Laws)
		if cfg.StateHash != nil {
			tracker = newStateSpaceTracker(cfg.SaturationThreshold)
		}
	}

	return func(rt *rapid.T) {
		if cfg.SUTFactory == nil {
			rt.Fatal("model.Property: SUTFactory is required")
		}
		if len(cfg.Actions) == 0 {
			rt.Fatal("model.Property: at least one Action is required")
		}

		// Reset per-iteration history traces (chain laws).
		for _, reset := range cfg.HistoryResetters {
			reset()
		}

		// And whatever an action carries across steps of one iteration but
		// must not carry between them — see [Action.Reset].
		for _, a := range cfg.Actions {
			if a.Reset != nil {
				a.Reset()
			}
		}

		// Per-iteration trace buffer.
		var iterTrace trace.Trace
		iterTrace.Reset()

		// Bind trace to laws that implement TraceBinder (trace combinators),
		// and clear cross-action law state: the pair is about to be rebuilt,
		// and state observed against the previous iteration's stores would
		// false-fail the first draw that differs from that iteration's.
		for _, l := range cfg.Laws.laws {
			if binder, ok := l.(law.TraceBinder); ok {
				binder.BindTrace(&iterTrace)
			}
			if r, ok := l.(law.Resettable); ok {
				r.Reset()
			}
		}

		sut := cfg.SUTFactory()
		var ref T
		if cfg.RefFactory != nil {
			ref = cfg.RefFactory()
		}

		// Isolated laws run once per iteration, each against a throwaway
		// pair of its own: their Checks corrupt what they observe — closing,
		// poisoning, tampering — and the shared pair must never meet one.
		// Once, not per step, because their claims are self-contained
		// rituals rather than observations of the accumulated history.
		for _, l := range cfg.Laws.laws {
			if _, isolated := l.(law.Isolated); !isolated {
				continue
			}
			cfg.Laws.ran[l.ID()]++
			isoSut := cfg.SUTFactory()
			var isoRef T
			if cfg.RefFactory != nil {
				isoRef = cfg.RefFactory()
			}
			err := l.Check(rt, isoSut, isoRef)
			if errors.Is(err, law.Vacuous) {
				cfg.Laws.vacuous[l.ID()]++
				continue
			}
			if err != nil {
				f := &Failure{
					Kind:         FailureInvariant,
					LawID:        l.ID(),
					REQID:        l.REQID(),
					StepRan:      StepID{WorkerID: -1, Index: 0},
					StepReported: StepID{WorkerID: -1, Index: 0},
					Err:          err,
					SUTState:     fmt.Sprintf("%+v", isoSut),
					RefState:     fmt.Sprintf("%+v", isoRef),
				}
				if jsonPath := emitClassifiedJSON(rt, cfg.ArtifactDir, f); jsonPath != "" {
					f.ArtifactPaths = append(f.ArtifactPaths, "json: "+jsonPath)
				}
				rt.Fatalf("%s", formatFailure(f))
			}
		}
		step := 0

		if cfg.Cleanup != nil {
			defer cfg.Cleanup(sut)
			if cfg.RefFactory != nil {
				defer cfg.Cleanup(ref)
			}
		}

		actionMap := make(map[string]func(*rapid.T), len(cfg.Actions)+1)
		for _, a := range cfg.Actions {
			actionMap[a.Name] = func(rt *rapid.T) {
				startNs := time.Now().UnixNano()
				result := a.Run(rt, sut, ref)
				endNs := time.Now().UnixNano()

				if !cfg.DisableTrace {
					var inputs []any
					if result.Input != nil {
						inputs = []any{result.Input}
					}
					evErr := result.CallErr
					if result.Err != nil {
						evErr = result.Err
					}
					iterTrace.Record(trace.Event{
						StartNs:  startNs,
						EndNs:    endNs,
						Method:   a.Name,
						ClientID: -1, // sequential
						Inputs:   inputs,
						Output:   result.Output,
						Err:      evErr,
					})
				}

				// Under LawsOnly the differential is not an oracle, so the
				// disagreement it reports is not a failure. Only the
				// semantic kind is silenced: a structural or liveness
				// result is the action saying it could not run at all,
				// which no law can speak for.
				if result.Err != nil && (!cfg.LawsOnly || a.Kind != FailureSemantic) {
					f := &Failure{
						Kind:         a.Kind,
						StepRan:      StepID{WorkerID: -1, Index: step},
						StepReported: StepID{WorkerID: -1, Index: step},
						Err:          result.Err,
					}
					// Attach trace per Kind policy.
					if shouldAttachTrace(a.Kind) {
						f.Trace = iterTrace.Snapshot()
					}
					if jsonPath := emitClassifiedJSON(rt, cfg.ArtifactDir, f); jsonPath != "" {
						f.ArtifactPaths = append(f.ArtifactPaths, "json: "+jsonPath)
					}
					rt.Fatalf("%s", formatFailure(f))
				}
				step++
			}
		}

		// Empty-string key: rapid's check action, after every command.
		actionMap[""] = func(rt *rapid.T) {
			for _, l := range cfg.Laws.laws {
				if _, isolated := l.(law.Isolated); isolated {
					// An isolated law corrupts its subjects; it ran once at
					// the top of the iteration against a pair of its own,
					// and the shared pair must never meet it.
					continue
				}
				cfg.Laws.ran[l.ID()]++ // vacuity is counted apart, in noteVacuous
				var err error
				if sl, ok := l.(law.StatefulLaw[T]); ok {
					err = sl.CheckWithStep(rt, sut, ref, step)
				} else {
					err = l.Check(rt, sut, ref)
				}
				if errors.Is(err, law.Vacuous) {
					// The subject declined the draw, so the claim was never
					// engaged. Counted apart from a pass: a law vacuous on
					// every check reads as coverage while checking nothing,
					// and the registry's census says so once per run.
					cfg.Laws.noteVacuous(rt, l.ID())
					continue
				}
				if err != nil {
					f := &Failure{
						Kind:         FailureInvariant,
						LawID:        l.ID(),
						REQID:        l.REQID(),
						StepRan:      StepID{WorkerID: -1, Index: step},
						StepReported: StepID{WorkerID: -1, Index: step},
						Err:          err,
						SUTState:     fmt.Sprintf("%+v", sut),
						RefState:     fmt.Sprintf("%+v", ref),
					}
					if shouldAttachTrace(FailureInvariant) && !cfg.DisableTrace {
						f.Trace = iterTrace.Snapshot()
					}
					if jsonPath := emitClassifiedJSON(rt, cfg.ArtifactDir, f); jsonPath != "" {
						f.ArtifactPaths = append(f.ArtifactPaths, "json: "+jsonPath)
					}
					rt.Fatalf("%s", formatFailure(f))
				}
			}
		}

		rt.Repeat(actionMap)

		// State-space coverage: hash the end-of-iteration state and
		// fold it into the run's exploration footprint. Prefer the
		// reference model (deterministic, canonical) when present.
		if tracker != nil {
			subject := sut
			if cfg.RefFactory != nil {
				subject = ref
			}
			cfg.Coverage.StateSpace = tracker.observe(cfg.StateHash(subject))
		}
	}
}

// Option configures a model-based test run.
type Option[T any] func(*Config[T])

// WithReference sets the reference model factory.
func WithReference[T any](factory func() T) Option[T] {
	return func(c *Config[T]) { c.RefFactory = factory }
}

// WithActions sets the action list.
func WithActions[T any](actions ...Action[T]) Option[T] {
	return func(c *Config[T]) { c.Actions = actions }
}

// WithLaws sets the entire law registry. Used by the generator to pass
// the pre-built auto-law registry.
func WithLaws[T any](r *Registry[T]) Option[T] {
	return func(c *Config[T]) { c.Laws = r }
}

// WithLaw adds a law to the registry.
func WithLaw[T any](l law.Law[T]) Option[T] {
	return func(c *Config[T]) {
		if c.Laws == nil {
			c.Laws = NewRegistry[T]()
		}
		c.Laws.Add(l)
	}
}

// WithLawREQ adds a law with a REQ tag for traceability.
func WithLawREQ[T any](reqID string, l law.Law[T]) Option[T] {
	return func(c *Config[T]) {
		if c.Laws == nil {
			c.Laws = NewRegistry[T]()
		}
		c.Laws.Add(tagLaw(l, reqID))
	}
}

// WithCleanup sets a cleanup function called on SUT and ref after
// each iteration.
func WithCleanup[T any](fn func(T)) Option[T] {
	return func(c *Config[T]) { c.Cleanup = fn }
}

// WithConcurrent enables concurrent linearizability testing.
func WithConcurrent[T any](cfg ConcurrentConfig[T]) Option[T] {
	return func(c *Config[T]) { c.Concurrent = &cfg }
}

// WithHistoryReset registers a reset function called at the start of
// each rapid iteration. Used by chain action helpers to clear the
// per-iteration append history.
func WithHistoryReset[T any](reset func()) Option[T] {
	return func(c *Config[T]) {
		c.HistoryResetters = append(c.HistoryResetters, reset)
	}
}

// WithStateHash enables state-space coverage tracking. The runner
// hashes the end-of-iteration state (the reference model when a
// reference is set, otherwise the SUT) and accumulates the
// exploration footprint into the [WithCoverageSink] sink. No-op
// without a sink.
func WithStateHash[T any](hash func(T) uint64) Option[T] {
	return func(c *Config[T]) { c.StateHash = hash }
}

// WithSaturationThreshold sets how many consecutive iterations
// without a new state mark the state space saturated. Zero uses the
// default.
func WithSaturationThreshold[T any](n int) Option[T] {
	return func(c *Config[T]) { c.SaturationThreshold = n }
}

// WithCoverageSink directs the run's coverage signals — state-space
// metrics and the REQ-to-law matrix — into sink. The sink is filled
// in place across iterations; read it after the run returns.
func WithCoverageSink[T any](sink *coverage.ComponentCoverage) Option[T] {
	return func(c *Config[T]) { c.Coverage = sink }
}

// WithLawsOnly makes the laws the run's only oracle — see
// [Config.LawsOnly] for what that silences and why it is not a
// conformance setting.
//
// Takes the flag rather than being nullary because the one caller that
// wants it decides per run: a generated saturation prover isolates one
// law and asks whether it can fail, and the same generated property
// serves the ordinary conformance run where the differential must
// stay armed.
func WithLawsOnly[T any](on bool) Option[T] {
	return func(c *Config[T]) { c.LawsOnly = on }
}

// SkipLaw removes an auto-derived law by ID.
func SkipLaw[T any](id string) Option[T] {
	return func(c *Config[T]) {
		if c.Laws != nil {
			c.Laws.SkipByID(id)
		}
	}
}

// WithoutTrace disables per-action trace recording. When set,
// failures will not include operation history context. Use for
// performance-sensitive property tests where trace overhead matters.
func WithoutTrace[T any]() Option[T] {
	return func(c *Config[T]) { c.DisableTrace = true }
}

// WithArtifactDir sets the directory for failure artifacts. Overrides
// the .testkit.yaml artifacts.dir setting and the default .testkit/artifacts/.
func WithArtifactDir[T any](dir string) Option[T] {
	return func(c *Config[T]) { c.ArtifactDir = dir }
}

// shouldAttachTrace returns true if the Kind policy says trace
// should be attached to a failure. Liveness failures (goroutine
// leaks, deadlocks) carry their own artifact (goroutine stacks)
// and don't benefit from the operation trace; every other kind
// gets the trace because the operation history is what diagnoses
// the failure — especially Semantic (SUT vs reference divergence)
// where the sequence of preceding operations is the bug context.
func shouldAttachTrace(k FailureKind) bool {
	return k != FailureLiveness
}

// Assert is the convenience entry point. Builds a [Config] from
// options and delegates to [Run] — the dispatcher is shared so the
// two entry points are observably identical (Concurrent dispatch,
// Laws validation, defaulting).
func Assert[T any](t rapid.TB, sutFactory func() T, opts ...Option[T]) Outcome {
	t.Helper()
	cfg := Config[T]{SUTFactory: sutFactory}
	for _, opt := range opts {
		opt(&cfg)
	}
	return dispatch(t, cfg)
}

// tagLaw wraps a law with a REQ ID override, keeping whatever the wrapped
// law is besides a law.
//
// A law may also be [law.TraceBinder], [law.Resettable], [law.StatefulLaw]
// or [law.Isolated], and the runner asks by type assertion. An embedding
// wrapper satisfies none of them, so tagging a law with a REQ identifier
// silently turned off everything the runner does for it: no trace bound
// (which a trace-scanning law then dereferences as nil), no reset between
// iterations (the cross-iteration leak an earlier fix already paid for
// once), and an isolated law let loose on the shared pair it corrupts.
//
// Three of the four are behaviours and forward at run time. Isolation is a
// marker, and a type either carries it or does not — so it takes a second
// wrapper, chosen here, rather than a method that would make every tagged
// law isolated.
func tagLaw[T any](l law.Law[T], reqID string) law.Law[T] {
	tagged := &taggedLaw[T]{Law: l, reqID: reqID}
	if _, isolated := l.(law.Isolated); isolated {
		return &taggedIsolatedLaw[T]{taggedLaw: tagged}
	}
	return tagged
}

// taggedLaw wraps a law with a REQ ID override.
type taggedLaw[T any] struct {
	law.Law[T]
	reqID string
}

func (t *taggedLaw[T]) REQID() string { return t.reqID }

// BindTrace forwards to the wrapped law where it scans a trace, and is a
// no-op otherwise — the runner binds unconditionally and a law that does not
// read one has nothing to bind.
func (t *taggedLaw[T]) BindTrace(tr *trace.Trace) {
	if binder, ok := t.Law.(law.TraceBinder); ok {
		binder.BindTrace(tr)
	}
}

// Reset forwards to the wrapped law where it carries cross-action state.
func (t *taggedLaw[T]) Reset() {
	if resettable, ok := t.Law.(law.Resettable); ok {
		resettable.Reset()
	}
}

// CheckWithStep forwards to the wrapped law where the step matters, and
// falls back to the step-free check where it does not.
func (t *taggedLaw[T]) CheckWithStep(rt *rapid.T, sut, ref T, step int) error {
	if stateful, ok := t.Law.(law.StatefulLaw[T]); ok {
		return stateful.CheckWithStep(rt, sut, ref, step)
	}
	return t.Check(rt, sut, ref)
}

// taggedIsolatedLaw is [taggedLaw] for a law that corrupts its subjects, so
// the marker the runner reads survives the tagging.
type taggedIsolatedLaw[T any] struct{ *taggedLaw[T] }

// IsolatedLaw marks the wrapper as the wrapped law is marked.
func (*taggedIsolatedLaw[T]) IsolatedLaw() {}
