// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // diagnostic, not wrapping
package timeaware

import (
	"context"
	"errors"
	"fmt"
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/engine/model/law"
)

// DeadlineRespecting verifies an operation invoked with a
// deadline-bearing context returns once the deadline fires.
//
// The checker invokes Op in a goroutine with a child context that
// carries a deadline of cfg.Deadline from now, then advances the
// test clock past the deadline and asserts Op returned via the
// supplied done channel. A failure indicates the SUT either
// ignores the context deadline or never checks it.
type DeadlineRespecting[T any] struct {
	// Op is the operation under test. It must respect the context
	// deadline. The implementation receives the ctx and SUT; it
	// returns when the deadline fires or when the operation
	// completes naturally.
	Op func(ctx context.Context, sut T) error

	// Deadline is the relative deadline applied to Op's context.
	Deadline time.Duration

	// Advance advances the test clock by the supplied duration.
	Advance func(time.Duration)

	// AwaitFor is the upper bound on how long the law waits for
	// Op to return after the clock advance. Zero defaults to 1
	// second; the law uses real-time sleep here so a misbehaving
	// SUT can't hang the property suite indefinitely.
	AwaitFor time.Duration

	// Name is the method Op calls, for the failure to say which.
	//
	// An interface declaring several deadline-shaped methods registers
	// this law once per method, and they share a row — one identifier,
	// one claim. Without the name a report says the claim broke and
	// leaves the reader to work out where.
	Name string
}

// named is the method for a message, falling back to a word rather than
// to an empty gap where a binding filled nothing.
func (l DeadlineRespecting[T]) named() string {
	if l.Name == "" {
		return "Op"
	}
	return l.Name
}

// ID returns the stable identifier for this law.
func (DeadlineRespecting[T]) ID() string { return lawid.DeadlineRespecting }

// REQID returns an empty string (auto-derived).
func (DeadlineRespecting[T]) REQID() string { return "" }

// Check verifies that an Op still running when its deadline fires returns a
// context error.
//
// # What the verdict rests on
//
// The returned error, not merely the return — but only once the deadline has
// actually passed. Two ways to get this wrong, and the law has been both:
//
// Discarding the error made it unfalsifiable. An Op that never looked at its
// context and answered nil immediately returned promptly, which was the whole
// of what was checked, so the one implementation the claim exists to catch
// passed it.
//
// Demanding the error unconditionally made it wrong about correct code, which
// is worse. "Returns within a budget" is satisfied — most fully satisfied — by
// an operation that finishes at once, and a subject with nothing to wait for
// answers nil long before any deadline fires. Failing it says the tool is
// broken to the one consumer whose implementation is perfect.
//
// So the deadline having passed is a precondition, not a conclusion. Op
// returning inside its budget refuses it and the run is [law.Vacuous]: this
// sequence engaged nothing, and counting it as a pass would let a subject that
// is merely fast stand in for one that is correct under expiry.
//
// # The clock this law is actually on
//
// The context's deadline is real time, because a [clock.TestClock] hands out
// no context and a fake deadline cannot cancel a real one. Advance releases a
// subject sleeping on the test clock; the context expires on its own. So this
// law is honest about a subject that respects its context and about one that
// ignores it, and it is not a test of the fake clock's plumbing — a subject
// that watches only the injected clock and never the context is outside what
// it can judge.
//
// Concurrency: Op runs on its own goroutine and the channel is buffered, so a
// subject that outlives the wait leaks no goroutine into the next iteration
// beyond the one blocked on its own work — which the AwaitFor branch reports
// rather than hangs on.
func (l DeadlineRespecting[T]) Check(_ *rapid.T, sut, _ T) error {
	ctx, cancel := context.WithTimeout(context.Background(), l.Deadline)
	defer cancel()

	start := time.Now()
	done := make(chan error, 1)
	go func() {
		done <- l.Op(ctx, sut)
	}()

	// Advance the test clock past the deadline.
	l.Advance(l.Deadline + time.Millisecond)

	wait := l.AwaitFor
	if wait <= 0 {
		wait = time.Second
	}
	select {
	case err := <-done:
		if time.Since(start) < l.Deadline {
			// Finished inside the budget: the deadline never fired, so there
			// is no expiry behaviour to judge.
			return law.Vacuous
		}
		if err == nil {
			return fmt.Errorf(
				"timeaware: deadline-respecting law: %s returned nil after its deadline passed, "+
					"so it never saw one", l.named(),
			)
		}
		if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			return fmt.Errorf(
				"deadline-respecting law: %s returned %v after its deadline passed, not a context error",
				l.named(), err,
			)
		}
		return nil
	case <-time.After(wait):
		return fmt.Errorf("deadline-respecting law: %s did not return within %v of deadline advance",
			l.named(), wait)
	}
}
