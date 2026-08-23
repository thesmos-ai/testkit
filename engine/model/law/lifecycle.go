// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Law errors are diagnostic, not wrapped.
package law

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"runtime"
	"slices"
	"strings"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/lawid"
)

// IdempotentLifecycle verifies that calling the lifecycle method
// twice in succession is observably equivalent to calling it once.
// Auto-emitted for Lifecycle methods carrying //testkit:mixin idempotent.
type IdempotentLifecycle[T any, Obs any] struct {
	Call    func(*rapid.T, T) error
	Observe func(*rapid.T, T) Obs
}

// IsolatedLaw marks the conduct: this Check corrupts its subjects to make
// its observation, and the runner hands it a throwaway pair of its own.
func (IdempotentLifecycle[T, Obs]) IsolatedLaw() {}

// ID returns the stable identifier for this law.
func (IdempotentLifecycle[T, Obs]) ID() string { return lawid.IdempotentLifecycle }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (IdempotentLifecycle[T, Obs]) REQID() string { return "" }

// Check verifies a second Call leaves Observe unchanged and does
// not error.
func (l IdempotentLifecycle[T, Obs]) Check(rt *rapid.T, sut, _ T) error {
	if err := l.Call(rt, sut); err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	before := l.Observe(rt, sut)
	if err := l.Call(rt, sut); err != nil {
		return fmt.Errorf("IdempotentLifecycle: second call errored: %v", err)
	}
	after := l.Observe(rt, sut)
	if fmt.Sprint(before) != fmt.Sprint(after) {
		return fmt.Errorf("IdempotentLifecycle: second call mutated state: before=%v after=%v", before, after)
	}
	return nil
}

// LeakFree verifies a Lifecycle pair (e.g., Open/Close, Acquire/
// Release) does not leak goroutines across N cycles. Auto-emitted
// for Lifecycle methods carrying //testkit:mixin leakfree open=<M> close=<M>.
//
// The cycle count and tolerance are runtime-tuned; the law samples
// runtime.NumGoroutine before and after the cycle and flags a
// growth exceeding Tolerance.
type LeakFree[T any] struct {
	Open  func(*rapid.T, T) error
	Close func(*rapid.T, T) error
	// Cycles is how many open/close rounds to run. Zero defaults to 16 — a
	// leak of one goroutine per cycle needs repetition to rise above the
	// scheduler's own noise.
	Cycles int
	// Tolerance is the goroutine growth treated as noise rather than a leak.
	// Zero defaults to 4, which absorbs the runtime's own background workers
	// without absorbing a per-cycle leak.
	Tolerance int
	// Outstanding is the subject's own census, preferred over the process
	// one where the interface offers it: the goroutine count is global, and
	// a parallel test legitimately parking its own work reads as growth
	// here — the subject's counter is deterministic and its alone.
	Outstanding func(*rapid.T, T) (int, error)
}

// ID returns the stable identifier for this law.
func (LeakFree[T]) ID() string { return lawid.LeakFree }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (LeakFree[T]) REQID() string { return "" }

// Check runs N Open/Close cycles and asserts the goroutine count
// hasn't drifted by more than Tolerance.
func (l LeakFree[T]) Check(rt *rapid.T, sut, _ T) error {
	cycles := l.Cycles
	if cycles <= 0 {
		cycles = 16
	}
	tolerance := l.Tolerance
	if tolerance <= 0 {
		tolerance = 4
	}
	if l.Outstanding != nil {
		before, beforeErr := l.Outstanding(rt, sut)
		if beforeErr != nil {
			return Vacuous // a precondition this run supplies was refused
		}
		for range cycles {
			if err := l.Open(rt, sut); err != nil {
				return Vacuous // a precondition this run supplies was refused
			}
			if err := l.Close(rt, sut); err != nil {
				return Vacuous // a precondition this run supplies was refused
			}
		}
		after, afterErr := l.Outstanding(rt, sut)
		if afterErr != nil {
			return Vacuous // a precondition this run supplies was refused
		}
		if after > before {
			return fmt.Errorf("LeakFree: outstanding grew from %d to %d after %d balanced cycles",
				before, after, cycles)
		}
		return nil
	}
	before := runtime.NumGoroutine()
	for range cycles {
		if err := l.Open(rt, sut); err != nil {
			return Vacuous // a precondition this run supplies was refused
		}
		if err := l.Close(rt, sut); err != nil {
			return Vacuous // a precondition this run supplies was refused
		}
	}
	after := runtime.NumGoroutine()
	if after-before > tolerance {
		// The census is process-wide: a parallel test ramping its own work
		// mid-bracket reads as growth here. A settle and one resample tell
		// the transients apart from a leak — a subject that held its
		// goroutines still holds them after the yield.
		for range 100 {
			runtime.Gosched()
		}
		after = runtime.NumGoroutine()
	}
	if drift := after - before; drift > tolerance {
		return fmt.Errorf("LeakFree: goroutine count grew from %d to %d after %d cycles (tolerance %d)",
			before, after, cycles, tolerance)
	}
	return nil
}

// LifecycleRespectsContext verifies that a Lifecycle method invoked
// with an already-cancelled context returns a context error instead
// of proceeding. Auto-emitted for Lifecycle methods taking a context.
//
// The law calls Op with a pre-cancelled context and requires the
// result to satisfy errors.Is(err, context.Canceled). An
// implementation that ignores the context and returns success fails.
type LifecycleRespectsContext[T any] struct {
	Op func(ctx context.Context, sut T) error

	// Name is the method Op calls, for the failure to say which.
	//
	// An interface with several lifecycle-shaped methods registers this
	// law once per method, and they share a row — one law identifier, one
	// claim. Without a name the report says the claim failed and leaves
	// the reader to find out where: a transaction declaring begin, commit
	// and rollback gave three identical probes and one message.
	Name string
}

// ID returns the stable identifier for this law.
func (LifecycleRespectsContext[T]) ID() string { return lawid.LifecycleRespectsContext }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (LifecycleRespectsContext[T]) REQID() string { return "" }

// Check invokes Op with a cancelled context and verifies it reports
// the cancellation.
func (l LifecycleRespectsContext[T]) Check(rt *rapid.T, sut, _ T) error {
	ctx, cancel := context.WithCancel(rt.Context())
	cancel()
	err := l.Op(ctx, sut)
	if !errors.Is(err, context.Canceled) {
		return fmt.Errorf(
			"LifecycleRespectsContext: %s with a cancelled context returned %v (want context.Canceled)",
			l.named(), err,
		)
	}
	return nil
}

// named is the method for a message, falling back to a word rather than
// to an empty gap where a binding filled nothing.
func (l LifecycleRespectsContext[T]) named() string {
	if l.Name == "" {
		return "the op"
	}
	return l.Name
}

// PoisonNilOnFresh verifies a PoisonAccessor returns nil on a
// freshly-constructed impl. Auto-emitted for PoisonAccessor
// methods.
type PoisonNilOnFresh[T any] struct {
	Factory func() T
	Probe   func(*rapid.T, T) error
}

// ID returns the stable identifier for this law.
func (PoisonNilOnFresh[T]) ID() string { return lawid.PoisonNilOnFresh }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (PoisonNilOnFresh[T]) REQID() string { return "" }

// Check runs Probe on a freshly-constructed impl and verifies
// the result is nil.
func (l PoisonNilOnFresh[T]) Check(rt *rapid.T, _, _ T) error {
	fresh := l.Factory()
	if err := l.Probe(rt, fresh); err != nil {
		return fmt.Errorf("PoisonNilOnFresh: fresh impl reports poison: %v", err)
	}
	return nil
}

// PoisonIdempotentRead verifies the PoisonAccessor is read-only:
// two consecutive reads return the same value (and the same
// error).
type PoisonIdempotentRead[T any] struct {
	Probe func(*rapid.T, T) error
}

// ID returns the stable identifier for this law.
func (PoisonIdempotentRead[T]) ID() string { return lawid.PoisonIdempotentRead }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (PoisonIdempotentRead[T]) REQID() string { return "" }

// Check verifies two consecutive Probe calls agree.
func (l PoisonIdempotentRead[T]) Check(rt *rapid.T, sut, _ T) error {
	a := l.Probe(rt, sut)
	b := l.Probe(rt, sut)
	if (a == nil) != (b == nil) {
		return fmt.Errorf("PoisonIdempotentRead: first=%v, second=%v", a, b)
	}
	return nil
}

// PoisonConsistent verifies that a poison condition is sticky: once
// the accessor reports poison, it keeps reporting it across
// subsequent reads rather than spontaneously healing. Auto-emitted
// for PoisonAccessor methods.
//
// Poison induces the condition; Probe reads it. The law establishes
// poison, confirms it took (a nil probe means the induction was a
// no-op and the law holds vacuously), then probes Reads more times
// (default 3) and fails if any returns nil. Distinct from
// [PoisonIdempotentRead], which checks read purity rather than
// stickiness after a poisoning event.
type PoisonConsistent[T any] struct {
	Poison func(*rapid.T, T)
	Probe  func(*rapid.T, T) error
	// Reads is how many times to probe the poisoned state. Zero defaults to
	// 3: two agreeing reads can agree by accident, three rarely do.
	Reads int
}

// IsolatedLaw marks the conduct: this Check corrupts its subjects to make
// its observation, and the runner hands it a throwaway pair of its own.
func (PoisonConsistent[T]) IsolatedLaw() {}

// ID returns the stable identifier for this law.
func (PoisonConsistent[T]) ID() string { return lawid.PoisonConsistent }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (PoisonConsistent[T]) REQID() string { return "" }

// Check induces poison and verifies it persists across follow-up
// probes.
func (l PoisonConsistent[T]) Check(rt *rapid.T, sut, _ T) error {
	l.Poison(rt, sut)
	if l.Probe(rt, sut) == nil {
		return Vacuous // the poison did not take, so there is no latch to hold
	}
	n := l.Reads
	if n <= 0 {
		n = 3
	}
	for i := range n {
		if err := l.Probe(rt, sut); err == nil {
			return fmt.Errorf("PoisonConsistent: poison healed spontaneously (probe %d after poison returned nil)", i+1)
		}
	}
	return nil
}

// LifecycleAfterCloseSentinel verifies that once the lifecycle's
// Close has run, every stamped method rejects further use with the
// configured sentinel error. Auto-emitted for the
// //testkit:mixin lifecycleafterclose close=<M> sentinel=<E> directive. (The cursor
// composite has its own narrower variant,
// [CursorNextAfterCloseSentinel].)
//
// Ops carries one probe per stamped method, keyed by method name so a
// red names the method that outlived Close. Op is the single-probe
// shorthand; setting both probes both. The claim this law reports is
// only as wide as its probe set — a recorded claim about "every
// method" backed by one probe is the silent-green class this law's
// consumers exist to kill, which is why the probe set is plural.
type LifecycleAfterCloseSentinel[T any] struct {
	Close    func(*rapid.T, T) error
	Op       func(*rapid.T, T) error
	Ops      map[string]func(*rapid.T, T) error
	Sentinel error
}

// IsolatedLaw marks the conduct: this Check corrupts its subjects to make
// its observation, and the runner hands it a throwaway pair of its own.
func (LifecycleAfterCloseSentinel[T]) IsolatedLaw() {}

// ID returns the stable identifier for this law.
func (LifecycleAfterCloseSentinel[T]) ID() string { return lawid.LifecycleAfterClose }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (LifecycleAfterCloseSentinel[T]) REQID() string { return "" }

// Check closes the SUT and verifies every probe reports the sentinel
// afterwards. Probes run in sorted name order so a multi-method red is
// deterministic, and every outliving method is named in one error
// rather than one per rerun.
func (l LifecycleAfterCloseSentinel[T]) Check(rt *rapid.T, sut, _ T) error {
	if l.Op == nil && len(l.Ops) == 0 {
		return errors.New("LifecycleAfterCloseSentinel: no probe configured; set Op or Ops")
	}
	if err := l.Close(rt, sut); err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	var outlived []string
	if l.Op != nil {
		if err := l.Op(rt, sut); !errors.Is(err, l.Sentinel) {
			outlived = append(outlived, fmt.Sprintf("op returned %v", err))
		}
	}
	for _, name := range slices.Sorted(maps.Keys(l.Ops)) {
		if err := l.Ops[name](rt, sut); !errors.Is(err, l.Sentinel) {
			outlived = append(outlived, fmt.Sprintf("%s returned %v", name, err))
		}
	}
	if len(outlived) > 0 {
		return fmt.Errorf("LifecycleAfterCloseSentinel: after Close (want %v): %s",
			l.Sentinel, strings.Join(outlived, "; "))
	}
	return nil
}
