// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package law defines type-parametric algebraic invariants for
// property-based state-machine testing.
//
// Each [Law] encodes a single algebraic property. Most check it
// observationally; a law that must write to state its claim keeps the
// differential pair synchronized — the conduct contract on [Law] names
// the four ways, and [mirror] is the one write-through laws share.
//
// Laws receive [rapid.T] to draw fresh samples per check, integrating
// with rapid's shrinking and generation.
package law

import (
	"errors"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/lawid"
)

// Law is a type-parametric invariant checked after every action.
//
// CONTRACT: a law must leave the (sut, ref) pair as synchronized as it
// found it. The runner interleaves laws with a differential action
// stream over one shared pair, so a mutation reaching only one side
// surfaces as the next action's false divergence — a failure naming
// the subject for a state the law created. Four conducts satisfy the
// contract, and every law in this package is one of them:
//
//   - observational: reads only. Most laws.
//   - mirrored: every mutation the subject accepts lands on the
//     reference too, through [mirror], and a refusal is reported as
//     the divergence it is.
//   - self-cleaning: mutates and restores within one Check — an
//     acquire released, a put deleted, a transaction rolled back.
//   - isolated: builds its own subjects through a Factory field and
//     never touches the pair.
//
// A law that can satisfy none of these — one that corrupts or kills
// the subject to make its observation — must not be registered against
// a shared pair at all; the conformance gate's conduct census is what
// keeps the generator from binding one.
//
// Check receives [rapid.T] so laws can draw fresh samples per
// invocation, integrating with rapid's shrinking.
type Law[T any] interface {
	// ID returns a stable identifier for this law.
	ID() string

	// REQID returns a requirement tag (e.g., "REQ-PKG-FOO-001").
	// Empty for auto-derived laws unless tagged.
	REQID() string

	// Check verifies the law holds. Must not mutate sut or ref.
	Check(rt *rapid.T, sut, ref T) error
}

// Vacuous is the sentinel a law returns where a precondition this run
// supplies was refused — the subject declined the draw, so the claim was
// never engaged. The runner counts it apart from a pass: sixty vacuous
// returns are sixty times a law asserted nothing, and a rate of one hundred
// percent is a binding that reads as coverage while checking nothing.
//
//nolint:revive,errname,staticcheck // an io.EOF-style sentinel: the vocabulary reads law.Vacuous, law.Holds
var Vacuous = errors.New("law: vacuously holds — a precondition this run supplies was refused")

// Holds reports a check that found no violation: a clean pass, or the
// vacuous sentinel a refused precondition returns. The spelling tests reach
// for — a law that "holds" includes one whose claim this draw never engaged,
// and the runner's census is where the difference is counted.
func Holds(err error) bool { return err == nil || errors.Is(err, Vacuous) }

// Isolated marks a law whose Check corrupts its subjects — closing,
// poisoning, tampering — and therefore runs against a throwaway pair of its
// own, once per iteration. No mirror repairs a closed subject; the shared
// pair must never meet one. The conformance conduct census holds the marker
// and the verdict to each other.
type Isolated interface{ IsolatedLaw() }

// mirror applies to the reference a mutation the subject accepted.
//
// The mirrored-conduct half of the [Law] contract, spelled once: a
// write-through law calls this after every mutation the subject took,
// and a reference that refuses is reported by the law — not left for
// the next action to misattribute to the subject.
func mirror(law string, apply func() error) error {
	if err := apply(); err != nil {
		return fmt.Errorf("%s: the reference refused what the subject accepted: %w", law, err)
	}
	return nil
}

// StatefulLaw extends [Law] with access to the current step count.
// Use this for laws that need cross-action state tracking, such as
// chain-growth monotonicity (AppendOnlyHistoryGrows) or
// frozen-after-poison invariants. The runner detects StatefulLaw
// via interface assertion and passes the step number.
type StatefulLaw[T any] interface {
	Law[T]

	// CheckWithStep is called instead of Check when the law implements
	// StatefulLaw. Step is the 0-based action count within the current
	// rapid iteration.
	CheckWithStep(rt *rapid.T, sut, ref T, step int) error
}

// Resettable is implemented by every law carrying cross-action state. The
// runner resets each one at the start of every property iteration: the pair
// is rebuilt fresh through the factories, and state observed against the
// previous iteration's stores is a memory of nothing that still exists.
//
// The rule is not optional for a stateful law. One that keeps its state
// across iterations false-fails the moment two iterations draw different
// values for the same key — which the fixture-pair pools never did, and the
// first wide pool did on its first run.
type Resettable interface {
	// Reset clears the cross-action state.
	Reset()
}

// ReadAfterWrite checks that every key in a sample pool is consistent
// between SUT and reference. Observational — never writes.
//
// The generator populates Keys with the same pool the Put/Get actions
// draw from. For any key, SUT.Read(key) must equal ref.Read(key).
type ReadAfterWrite[T any, K comparable, V any] struct {
	Read func(*rapid.T, T, K) (V, error)
	Keys *rapid.Generator[K]
}

// ID returns the stable identifier for this law.
func (ReadAfterWrite[T, K, V]) ID() string { return lawid.ReadAfterWrite }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (ReadAfterWrite[T, K, V]) REQID() string { return "" }

// Check verifies the law holds for the given SUT and reference.
func (l ReadAfterWrite[T, K, V]) Check(rt *rapid.T, sut, ref T) error {
	k := l.Keys.Draw(rt, "ReadAfterWrite_key")
	sutGot, sutErr := l.Read(rt, sut, k)
	refGot, refErr := l.Read(rt, ref, k)
	if (sutErr == nil) != (refErr == nil) {
		//nolint:errorlint // diagnostic message, not wrapping
		return fmt.Errorf("ReadAfterWrite: key %v: SUT err=%v, ref err=%v",
			k, sutErr, refErr)
	}
	if sutErr != nil {
		return nil //nolint:nilerr // both errored — agreement, not a bug
	}
	if diff := cmp.Diff(refGot, sutGot); diff != "" {
		return fmt.Errorf("ReadAfterWrite: key %v: SUT/ref disagree (-ref +sut):\n%s", k, diff)
	}
	return nil
}

// DeleteReturnsNotFound checks that where the reference returns the
// sentinel error, the SUT also returns it. Observational — never writes.
type DeleteReturnsNotFound[T any, K comparable, V any] struct {
	Read     func(*rapid.T, T, K) (V, error)
	Keys     *rapid.Generator[K]
	Sentinel error

	// RefMiss is what the REFERENCE reports for a key it does not hold,
	// which is what decides whether this check has anything to ask.
	//
	// Apart from Sentinel because the two are different errors whenever a
	// store keeps a tombstone: Sentinel is what the SUBJECT owes for a
	// key its delete removed, and the reference is a plain map that never
	// heard of it. Read through one field, the guard compared the
	// reference's miss against the subject's tombstone, never matched,
	// and the law held vacuously for every subject.
	//
	// Empty falls back to Sentinel, which is right where a declaration
	// names no separate tombstone: the reference is then built with that
	// identity and the two are one error.
	RefMiss error
}

// absent is [DeleteReturnsNotFound.RefMiss], or Sentinel where the
// declaration names no separate tombstone.
func (l DeleteReturnsNotFound[T, K, V]) absent() error {
	if l.RefMiss != nil {
		return l.RefMiss
	}
	return l.Sentinel
}

// ID returns the stable identifier for this law.
func (DeleteReturnsNotFound[T, K, V]) ID() string { return lawid.DeleteReturnsNotFound }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (DeleteReturnsNotFound[T, K, V]) REQID() string { return "" }

// Check verifies the law holds for the given SUT and reference.
func (l DeleteReturnsNotFound[T, K, V]) Check(rt *rapid.T, sut, ref T) error {
	k := l.Keys.Draw(rt, "DeleteReturnsNotFound_key")
	_, refErr := l.Read(rt, ref, k)
	if !errors.Is(refErr, l.absent()) {
		return Vacuous // the key the draw picked exists, so nothing was deleted
	}
	_, sutErr := l.Read(rt, sut, k)
	if !errors.Is(sutErr, l.Sentinel) {
		//nolint:errorlint // diagnostic message, not wrapping
		return fmt.Errorf(
			"DeleteReturnsNotFound: key %v: ref returned sentinel %v but SUT returned %v",
			k, l.Sentinel, sutErr,
		)
	}
	return nil
}

// CountEqualsReference checks that the SUT's count matches the
// reference's count. Purely observational.
type CountEqualsReference[T any, R comparable] struct {
	Count func(*rapid.T, T) (R, error)
}

// ID returns the stable identifier for this law.
func (CountEqualsReference[T, R]) ID() string { return lawid.CountEqualsReference }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (CountEqualsReference[T, R]) REQID() string { return "" }

// Check verifies the law holds for the given SUT and reference.
//
// Error presence is compared, never demanded: a pool both sides have drained
// refuses both counts identically, and the law holds vacuously — the corpus
// caught the earlier spelling failing exactly that agreement.
func (l CountEqualsReference[T, R]) Check(rt *rapid.T, sut, ref T) error {
	sutN, sutErr := l.Count(rt, sut)
	refN, refErr := l.Count(rt, ref)
	if (sutErr != nil) != (refErr != nil) {
		//nolint:errorlint // diagnostic message, not wrapping
		return fmt.Errorf("CountEqualsReference: SUT err=%v, ref err=%v",
			sutErr, refErr)
	}
	if sutErr != nil {
		return Vacuous // both sides refused, so there are no counts to compare
	}
	if sutN != refN {
		return fmt.Errorf("CountEqualsReference: SUT=%v, ref=%v", sutN, refN)
	}
	return nil
}

// CRDTMerge verifies the convergence contract of a state-based CRDT
// merge: two replicas receiving disjoint slices of the same write
// stream converge to the same observable state after a bidirectional
// merge, and re-merging is idempotent. Auto-emitted for the
// //testkit:mixin crdtmerge directive; runs under AcrossImpls
// with two replica factories.
//
// The law constructs two fresh replicas via Factory, routes each
// drawn value to one of them (rapid picks the split), merges A←B
// then B←A, and compares Observe on both. Observe must be
// deterministic (e.g., sorted keys) for the comparison to be
// meaningful.
type CRDTMerge[T any, V any, Obs any] struct {
	Factory func() T
	Write   func(*rapid.T, T, V) error
	Merge   func(rt *rapid.T, dst, src T) error
	Values  *rapid.Generator[V]
	Observe func(*rapid.T, T) Obs
}

// ID returns the stable identifier for this law.
func (CRDTMerge[T, V, Obs]) ID() string { return lawid.CRDTMerge }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (CRDTMerge[T, V, Obs]) REQID() string { return "" }

// Check splits writes across two replicas, merges both directions,
// and verifies convergence plus idempotent re-merge.
func (l CRDTMerge[T, V, Obs]) Check(rt *rapid.T, _, _ T) error {
	a := l.Factory()
	b := l.Factory()
	n := rapid.IntRange(1, 6).Draw(rt, "CRDTMerge_writes")
	for i := range n {
		v := l.Values.Draw(rt, fmt.Sprintf("CRDTMerge_v%d", i))
		dst := a
		if rapid.Bool().Draw(rt, fmt.Sprintf("CRDTMerge_toB%d", i)) {
			dst = b
		}
		if err := l.Write(rt, dst, v); err != nil {
			return Vacuous // a precondition this run supplies was refused
		}
	}
	if err := l.Merge(rt, a, b); err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	if err := l.Merge(rt, b, a); err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	obsA := l.Observe(rt, a)
	obsB := l.Observe(rt, b)
	if diff := cmp.Diff(obsA, obsB); diff != "" {
		return fmt.Errorf("CRDTMerge: replicas did not converge after bidirectional merge (-a +b):\n%s", diff)
	}
	if err := l.Merge(rt, a, b); err != nil {
		return fmt.Errorf("CRDTMerge: re-merge errored: %w", err)
	}
	again := l.Observe(rt, a)
	if diff := cmp.Diff(obsA, again); diff != "" {
		return fmt.Errorf("CRDTMerge: re-merge was not idempotent (-before +after):\n%s", diff)
	}
	return nil
}
