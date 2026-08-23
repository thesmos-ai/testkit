// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // diagnostic, not wrapping
package timeaware

import (
	"errors"
	"fmt"
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/engine/model/law"
)

// TTLExpiryAfterAdvance verifies a TTL-bound store removes its
// entries after the clock advances past the TTL. The checker
// drives the SUT through a single Put → Read → Advance → Read
// sequence drawn from the consumer-supplied generators.
//
// The reference T parameter is ignored — TTL is a self-consistency
// property of the SUT.
type TTLExpiryAfterAdvance[T any, K comparable, V any] struct {
	// Put stores the value at key k. Errors abort the law as
	// vacuous (the precondition was not met).
	Put func(rt *rapid.T, sut T, k K, v V) error

	// Read returns the stored value or [NotFound] when absent.
	Read func(rt *rapid.T, sut T, k K) (V, error)

	// Keys and Values supply test inputs.
	Keys   *rapid.Generator[K]
	Values *rapid.Generator[V]

	// TTL is the time-to-live the run holds entries to.
	//
	// One duration for the whole law, whichever way the declaration
	// spelled it: a directive that fixes it once, or a field on the value
	// whose declared default the fixture carries. The second shape is
	// what gives a defect a field to reach for, and the law cannot tell
	// them apart — nor should it, since the claim is the same sentence.
	TTL time.Duration

	// Advance advances the test clock by the supplied duration.
	// Implementations typically wrap [clock.TestClock.Advance].
	Advance func(time.Duration)

	// NotFound is the sentinel Read returns when the key has
	// expired or never existed. Compared via errors.Is.
	NotFound error
}

// ID returns the stable identifier for this law.
func (TTLExpiryAfterAdvance[T, K, V]) ID() string { return lawid.TTLExpiry }

// REQID returns an empty string (auto-derived).
func (TTLExpiryAfterAdvance[T, K, V]) REQID() string { return "" }

// Check verifies post-TTL expiry of a freshly-stored entry.
func (l TTLExpiryAfterAdvance[T, K, V]) Check(rt *rapid.T, sut, _ T) error {
	k := l.Keys.Draw(rt, "ttl_key")
	v := l.Values.Draw(rt, "ttl_value")

	if err := l.Put(rt, sut, k, v); err != nil {
		return law.Vacuous // a precondition this run supplies was refused
	}
	if _, err := l.Read(rt, sut, k); err != nil {
		return fmt.Errorf("ttl-expiry law: Read pre-advance errored: %v", err)
	}

	// Advance past the TTL with a small epsilon so the entry is
	// strictly past expiry, regardless of how the SUT rounds.
	l.Advance(l.TTL + time.Millisecond)

	_, err := l.Read(rt, sut, k)
	if !errors.Is(err, l.NotFound) {
		return fmt.Errorf("ttl-expiry law: post-advance Read returned %v, want %v", err, l.NotFound)
	}
	return nil
}
