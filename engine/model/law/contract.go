// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Law errors are diagnostic, not wrapped.
package law

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/lawid"
)

// PersisterRetrievable verifies that the value returned by a
// Persister's Save can be looked up via the paired Reader. The
// returned ID is the lookup key. Auto-emitted for methods carrying
// //testkit:contract persister role=writer reader=<M>.
type PersisterRetrievable[T any, V any, ID comparable] struct {
	Save   func(*rapid.T, T, V) (ID, error)
	Read   func(*rapid.T, T, ID) (V, error)
	Values *rapid.Generator[V]
}

// ID returns the stable identifier for this law.
func (PersisterRetrievable[T, V, ID]) ID() string { return lawid.PersisterRetrievable }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (PersisterRetrievable[T, V, ID]) REQID() string { return "" }

// Check verifies Save-then-Read returns the saved value.
func (l PersisterRetrievable[T, V, ID]) Check(rt *rapid.T, sut, ref T) error {
	v := l.Values.Draw(rt, "PersisterRetrievable_value")
	id, err := l.Save(rt, sut, v)
	if err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	// The accepted save lands on both sides — the mirrored half of the [Law]
	// conduct contract. The reference's own id is its own business: the claim
	// reads back through the subject's.
	if mErr := mirror("PersisterRetrievable", func() error {
		_, saveErr := l.Save(rt, ref, v)
		return saveErr
	}); mErr != nil {
		return mErr
	}
	got, err := l.Read(rt, sut, id)
	if err != nil {
		return fmt.Errorf("PersisterRetrievable: saved id=%v, Read errored: %v", id, err)
	}
	if diff := cmp.Diff(v, got); diff != "" {
		return fmt.Errorf("PersisterRetrievable: saved/read mismatch (-saved +read):\n%s", diff)
	}
	return nil
}

// UpdaterReplaces verifies that calling Update twice with values
// that share a key results in only the second value being
// observable via Read. Last-write-wins per key.
//
// The consumer supplies KeyOf to extract the matching key from V.
type UpdaterReplaces[T any, V any, K comparable] struct {
	Update func(*rapid.T, T, V) error
	Read   func(*rapid.T, T, K) (V, error)
	Values *rapid.Generator[V]
	KeyOf  func(V) K
}

// ID returns the stable identifier for this law.
func (UpdaterReplaces[T, V, K]) ID() string { return lawid.UpdaterReplaces }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (UpdaterReplaces[T, V, K]) REQID() string { return "" }

// Check writes v1 then v2 sharing the same key and verifies the
// reader sees v2.
func (l UpdaterReplaces[T, V, K]) Check(rt *rapid.T, sut, ref T) error {
	v1 := l.Values.Draw(rt, "UpdaterReplaces_v1")
	v2 := l.Values.Draw(rt, "UpdaterReplaces_v2")
	if l.KeyOf(v1) != l.KeyOf(v2) {
		return Vacuous // two keys, so neither write replaces the other
	}
	if err := l.Update(rt, sut, v1); err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	// Each accepted update lands on both sides — the mirrored half of the
	// [Law] conduct contract.
	if err := mirror("UpdaterReplaces", func() error { return l.Update(rt, ref, v1) }); err != nil {
		return err
	}
	if err := l.Update(rt, sut, v2); err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	if err := mirror("UpdaterReplaces", func() error { return l.Update(rt, ref, v2) }); err != nil {
		return err
	}
	got, err := l.Read(rt, sut, l.KeyOf(v2))
	if err != nil {
		return fmt.Errorf("UpdaterReplaces: Read after replace errored: %v", err)
	}
	if diff := cmp.Diff(v2, got); diff != "" {
		return fmt.Errorf("UpdaterReplaces: second write not visible (-v2 +read):\n%s", diff)
	}
	return nil
}

// UpserterIdempotent verifies repeated Upserts of the same value
// leave the reader-observed state unchanged after the second call.
type UpserterIdempotent[T any, V any, K comparable] struct {
	Upsert func(*rapid.T, T, V) error
	Read   func(*rapid.T, T, K) (V, error)
	Values *rapid.Generator[V]
	KeyOf  func(V) K
}

// ID returns the stable identifier for this law.
func (UpserterIdempotent[T, V, K]) ID() string { return lawid.UpserterIdempotent }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (UpserterIdempotent[T, V, K]) REQID() string { return "" }

// Check upserts v twice and verifies the read result is stable.
func (l UpserterIdempotent[T, V, K]) Check(rt *rapid.T, sut, ref T) error {
	v := l.Values.Draw(rt, "UpserterIdempotent_value")
	if upsertErr := l.Upsert(rt, sut, v); upsertErr != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	// Each accepted upsert lands on both sides — the mirrored half of the
	// [Law] conduct contract.
	if err := mirror("UpserterIdempotent", func() error { return l.Upsert(rt, ref, v) }); err != nil {
		return err
	}
	first, readErr := l.Read(rt, sut, l.KeyOf(v))
	if readErr != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	if err := l.Upsert(rt, sut, v); err != nil {
		return fmt.Errorf("upserter law: second upsert errored: %v", err)
	}
	if err := mirror("UpserterIdempotent", func() error { return l.Upsert(rt, ref, v) }); err != nil {
		return err
	}
	second, err := l.Read(rt, sut, l.KeyOf(v))
	if err != nil {
		return fmt.Errorf("upserter law: read after second upsert errored: %v", err)
	}
	if diff := cmp.Diff(first, second); diff != "" {
		return fmt.Errorf("upserter law: second upsert changed read (-first +second):\n%s", diff)
	}
	return nil
}

// CASAtomicOneWinner verifies that two concurrent CAS writes with
// the same starting version produce exactly one success and one
// version-mismatch error. Auto-emitted for methods carrying
// //testkit:contract cas role=writer version=<F> mismatch=<E>.
type CASAtomicOneWinner[T any, V any] struct {
	CAS      func(*rapid.T, T, V) error
	Read     func(*rapid.T, T) (V, error)
	Values   *rapid.Generator[V]
	Mismatch error

	// Stamp coerces a drawn attempt to the cell's current version — the
	// version-coherent draw that makes "exactly one winner" a theorem
	// rather than a coin toss, since two attempts at a stale version are
	// two mismatches and no winner. Nil leaves the draws raw, for a caller
	// whose generator is already coherent.
	Stamp func(*rapid.T, T, V) V
}

// ID returns the stable identifier for this law.
func (CASAtomicOneWinner[T, V]) ID() string { return lawid.CASAtomicOneWinner }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (CASAtomicOneWinner[T, V]) REQID() string { return "" }

// Check applies two CAS attempts stamped at one version; expects exactly one
// success and one mismatch.
//
// # Sequential, and why that is still contention
//
// Nothing here runs concurrently. Both attempts are stamped before either
// runs, so both carry the version the cell held at the start — which is the
// whole of what makes them contend. The second arrives stale by
// construction, and a subject that checks versions refuses it. One that
// ignores them accepts both and fails here; one that refuses everything
// accepts neither and fails here too.
//
// So the claim this settles is the version arithmetic, not atomicity under
// real concurrency. Interleaving two callers is the Porcupine leg's, where
// the linearizability check has a history to reason about. The identifier
// says atomic and means this — the sequential half — which is the half a
// subject can be wrong about without any scheduler help.
func (l CASAtomicOneWinner[T, V]) Check(rt *rapid.T, sut, ref T) error {
	v1 := l.Values.Draw(rt, "CASAtomicOneWinner_v1")
	v2 := l.Values.Draw(rt, "CASAtomicOneWinner_v2")
	if l.Stamp != nil {
		// Both attempts stamped before either runs: the point is two writes
		// contending at one version, and stamping the second after the first
		// won would hand it the bumped version and a free win.
		v1, v2 = l.Stamp(rt, sut, v1), l.Stamp(rt, sut, v2)
	}
	err1 := l.CAS(rt, sut, v1)
	err2 := l.CAS(rt, sut, v2)
	// The same pair of attempts lands on both sides — the mirrored half of
	// the [Law] conduct contract. Outcomes are not compared here: on a
	// synchronized pair the cell's own version arithmetic makes them agree,
	// and a divergence is the next action's to find on a pair that at least
	// saw the same calls.
	_ = l.CAS(rt, ref, v1)
	_ = l.CAS(rt, ref, v2)
	successes := 0
	if err1 == nil {
		successes++
	}
	if err2 == nil {
		successes++
	}
	mismatches := 0
	if errors.Is(err1, l.Mismatch) {
		mismatches++
	}
	if errors.Is(err2, l.Mismatch) {
		mismatches++
	}
	if successes != 1 || mismatches != 1 {
		return fmt.Errorf(
			"CASAtomicOneWinner: expected 1 success + 1 mismatch, got successes=%d mismatches=%d (err1=%v err2=%v)",
			successes, mismatches, err1, err2,
		)
	}
	return nil
}

// AppenderMonotonicOffsets verifies the offsets returned by
// successive Appends are strictly increasing. Auto-emitted for
// methods carrying //testkit:contract appender role=fn.
type AppenderMonotonicOffsets[T any, V any, Off interface{ ~int | ~int64 }] struct {
	Append func(*rapid.T, T, V) (Off, error)
	Values *rapid.Generator[V]

	prev    Off
	hasPrev bool
}

// ID returns the stable identifier for this law.
func (*AppenderMonotonicOffsets[T, V, Off]) ID() string { return lawid.AppenderMonotonicOffsets }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (*AppenderMonotonicOffsets[T, V, Off]) REQID() string { return "" }

// Reset clears the prior offset — [Resettable], because the previous
// iteration's log is gone and its offsets order nothing in this one.
func (l *AppenderMonotonicOffsets[T, V, Off]) Reset() {
	var zero Off
	l.prev = zero
	l.hasPrev = false
}

// Check appends a value and verifies the returned offset exceeds the
// previously-observed offset.
func (l *AppenderMonotonicOffsets[T, V, Off]) Check(rt *rapid.T, sut, ref T) error {
	v := l.Values.Draw(rt, "AppenderMonotonicOffsets_value")
	off, err := l.Append(rt, sut, v)
	if err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	// The accepted append lands on both sides — the mirrored half of the
	// [Law] conduct contract. The reference's offset is its own; the claim
	// orders the subject's.
	if mErr := mirror("AppenderMonotonicOffsets", func() error {
		_, appendErr := l.Append(rt, ref, v)
		return appendErr
	}); mErr != nil {
		return mErr
	}
	if l.hasPrev && off <= l.prev {
		return fmt.Errorf("AppenderMonotonicOffsets: offset %v did not exceed previous %v", off, l.prev)
	}
	l.prev = off
	l.hasPrev = true
	return nil
}

// SingleflightCoalesces verifies that N concurrent calls with the
// same key invoke the compute function at most once. Auto-emitted
// for methods carrying //testkit:contract singleflight role=fn.
//
// The consumer threads a shared call counter through Compute; the
// law inspects the counter after running M concurrent calls.
type SingleflightCoalesces[T any, K comparable, V any] struct {
	Call    func(ctx context.Context, sut T, k K, compute func() V) (V, error)
	Compute func() V
	Keys    *rapid.Generator[K]
	// Parallel is how many concurrent calls contend for one key. Zero
	// defaults to 8 — coalescing is only observable under contention, and a
	// pair of callers may serialise by luck.
	Parallel int
	Counter  *int // pointer the Compute closure increments
}

// ID returns the stable identifier for this law.
func (SingleflightCoalesces[T, K, V]) ID() string { return lawid.SingleflightCoalesces }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (SingleflightCoalesces[T, K, V]) REQID() string { return "" }

// Check launches N concurrent calls with the same key and asserts
// the compute counter only advances by 1.
func (l SingleflightCoalesces[T, K, V]) Check(rt *rapid.T, sut, ref T) error {
	k := l.Keys.Draw(rt, "SingleflightCoalesces_key")
	before := *l.Counter
	n := l.Parallel
	if n <= 0 {
		n = 8
	}
	done := make(chan struct{}, n)
	for range n {
		go func() {
			_, _ = l.Call(rt.Context(), sut, k, l.Compute)
			done <- struct{}{}
		}()
	}
	for range n {
		<-done
	}
	got := *l.Counter - before
	if got > 1 {
		return fmt.Errorf("SingleflightCoalesces: %d concurrent calls invoked compute %d times (expected ≤1)", n, got)
	}
	// One call lands on the reference too — the mirrored half of the [Law]
	// conduct contract, for a subject that memoizes what it computed.
	_, _ = l.Call(rt.Context(), ref, k, l.Compute)
	return nil
}

// TransactionRollbackOnError verifies that when the body of a
// TransactionFunc returns an error, no buffered writes are visible
// after the call returns. Auto-emitted for methods carrying
// //testkit:contract transaction role=fn notfound=<E>.
//
// # Why Write is a field and not an optional convenience
//
// The claim is about what an erroring body leaves behind, so the body has to
// leave something behind or there is nothing to be wrong about. Without a
// staged write the check reduced to "an errored run did not change a key
// nobody touched", which every implementation satisfies — including one that
// applies in place and never rolls back at all. Both failure branches were
// unreachable and the law reported a pass on every run it ever made.
//
// Write is therefore the mutation the rollback must discard, invoked inside
// the body against the same key the reads probe.
type TransactionRollbackOnError[T any, K comparable, V any] struct {
	Run  func(*rapid.T, T, func(_ context.Context) error) error
	Read func(*rapid.T, T, K) (V, error)

	// Write stages a mutation from inside the transaction body. It is the
	// subject's own write, not a test double: a closure reaching past the
	// interface to the concrete store would make the law unfalsifiable by
	// construction, because no defect worn on the interface could reach it.
	Write func(*rapid.T, T, K, V) error

	Keys     *rapid.Generator[K]
	Values   *rapid.Generator[V]
	NotFound error
}

// ID returns the stable identifier for this law.
func (TransactionRollbackOnError[T, K, V]) ID() string { return lawid.TransactionRollback }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (TransactionRollbackOnError[T, K, V]) REQID() string { return "" }

// Check stages a write inside a body that always errors, then verifies the
// probe key is observationally what it was before the call.
//
// The staged write is the whole of the law's evidence. A subject that
// discards its staging leaves the probe exactly as it found it; one that
// applies in place, or commits despite the error, shows the staged value
// through the read that follows — which is the defect the identifier names
// and the only one this check can now pass over in silence if Write is
// absent.
func (l TransactionRollbackOnError[T, K, V]) Check(rt *rapid.T, sut, _ T) error {
	if l.Write == nil {
		// Nothing to stage, so nothing an erroring body could fail to
		// discard. Vacuous rather than a pass: the run engaged no claim, and
		// counting it would restore the false green this field exists to end.
		return Vacuous
	}
	probe := l.Keys.Draw(rt, "TransactionRollbackOnError_key")
	staged := l.Values.Draw(rt, "TransactionRollbackOnError_value")
	before, beforeErr := l.Read(rt, sut, probe)
	_ = l.Run(rt, sut, func(_ context.Context) error {
		// The write's own error is not the claim: a store may legitimately
		// refuse a write inside a doomed transaction. What the claim forbids
		// is the write surviving the rollback.
		_ = l.Write(rt, sut, probe, staged)
		return errors.New("law: induced rollback")
	})
	after, afterErr := l.Read(rt, sut, probe)
	// Whether the key existed before or not, the post-error state
	// must equal the pre-call state observationally.
	if (beforeErr == nil) != (afterErr == nil) {
		return fmt.Errorf("TransactionRollbackOnError: key %v: errored body changed presence (before=%v, after=%v)",
			probe, beforeErr, afterErr)
	}
	if beforeErr == nil {
		if diff := cmp.Diff(before, after); diff != "" {
			return fmt.Errorf("TransactionRollbackOnError: key %v: value changed across error (-before +after):\n%s",
				probe, diff)
		}
	}
	return nil
}

// LeaseDoubleAcquireBlocks verifies a second Acquire of an
// already-held lease returns the configured held error. Auto-
// emitted for methods carrying //testkit:contract lease role=acquire
// release=<M>.
type LeaseDoubleAcquireBlocks[T any, K comparable] struct {
	Acquire func(*rapid.T, T, K) error
	Release func(*rapid.T, T, K) error
	Keys    *rapid.Generator[K]
	Held    error
}

// ID returns the stable identifier for this law.
func (LeaseDoubleAcquireBlocks[T, K]) ID() string { return lawid.LeaseDoubleAcquireBlocks }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (LeaseDoubleAcquireBlocks[T, K]) REQID() string { return "" }

// Check acquires a key, attempts to acquire again, verifies the
// held error fires, then releases.
func (l LeaseDoubleAcquireBlocks[T, K]) Check(rt *rapid.T, sut, _ T) error {
	k := l.Keys.Draw(rt, "LeaseDoubleAcquireBlocks_key")
	if err := l.Acquire(rt, sut, k); err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	defer func() { _ = l.Release(rt, sut, k) }()
	err := l.Acquire(rt, sut, k)
	if !errors.Is(err, l.Held) {
		return fmt.Errorf("LeaseDoubleAcquireBlocks: key %v: second acquire returned %v (want held=%v)",
			k, err, l.Held)
	}
	return nil
}

// PaginatorNoDuplicates verifies that a full walk of a paginated
// reader emits every element at most once — no element key appears
// in two pages. Auto-emitted for methods carrying
// //testkit:contract pagination role=reader cursor=<M>.
//
// Page returns one page of items, the cursor to fetch the next page,
// and whether more pages remain. The walk starts at Start and
// follows the returned cursors until Page reports no more pages, or
// until MaxPages is reached — a safety bound so a paginator that
// never signals termination fails the law instead of looping
// forever. MaxPages ≤ 0 defaults to 1000.
type PaginatorNoDuplicates[T any, V any, K comparable, C any] struct {
	Page  func(rt *rapid.T, sut T, cursor C) (items []V, next C, more bool)
	Start C
	KeyOf func(V) K

	// MaxPages bounds the walk so a paginator that never reports
	// exhaustion fails rather than loops. Zero defaults to 1000.
	MaxPages int
}

// ID returns the stable identifier for this law.
func (PaginatorNoDuplicates[T, V, K, C]) ID() string { return lawid.PaginatorNoDuplicates }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (PaginatorNoDuplicates[T, V, K, C]) REQID() string { return "" }

// Check walks every page from Start and fails if any element key is
// observed in two distinct pages, or if the walk fails to terminate
// within MaxPages.
func (l PaginatorNoDuplicates[T, V, K, C]) Check(rt *rapid.T, sut, _ T) error {
	maxPages := l.MaxPages
	if maxPages <= 0 {
		maxPages = 1000
	}
	seen := make(map[K]struct{})
	cursor := l.Start
	for range maxPages {
		items, next, more := l.Page(rt, sut, cursor)
		for _, it := range items {
			k := l.KeyOf(it)
			if _, dup := seen[k]; dup {
				return fmt.Errorf("PaginatorNoDuplicates: key %v appeared in two pages", k)
			}
			seen[k] = struct{}{}
		}
		if !more {
			return nil
		}
		cursor = next
	}
	return fmt.Errorf("PaginatorNoDuplicates: walk did not terminate within %d pages", maxPages)
}

// PaginatorResumable verifies that resuming a walk from any cursor
// observed mid-stream yields exactly the suffix the full walk would
// have produced from that point. Auto-emitted for Paginator methods
// carrying the //testkit:contract pagination role=reader cursor=<M> directive.
//
// Page has the same shape as in [PaginatorNoDuplicates]. The law
// walks once from Start recording the cursor that began each page,
// picks one of those cursors via rapid, re-walks from it, and
// asserts the resumed items equal the corresponding suffix of the
// first walk. MaxPages ≤ 0 defaults to 1000.
type PaginatorResumable[T any, V any, C any] struct {
	Page  func(rt *rapid.T, sut T, cursor C) (items []V, next C, more bool)
	Start C

	// MaxPages bounds both walks, as in PaginatorNoDuplicates.
	// Zero defaults to 1000.
	MaxPages int
}

// ID returns the stable identifier for this law.
func (PaginatorResumable[T, V, C]) ID() string { return lawid.PaginatorResumable }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (PaginatorResumable[T, V, C]) REQID() string { return "" }

// walk pages from the given cursor, returning the per-page items and
// the cursor that began each page. Bounded by maxPages.
func (l PaginatorResumable[T, V, C]) walk(rt *rapid.T, sut T, from C, maxPages int) (pages [][]V, starts []C) {
	cursor := from
	for range maxPages {
		starts = append(starts, cursor)
		items, next, more := l.Page(rt, sut, cursor)
		pages = append(pages, items)
		if !more {
			break
		}
		cursor = next
	}
	return pages, starts
}

// Check walks fully, resumes from a drawn page-start cursor, and
// compares the resumed items against the full-walk suffix.
func (l PaginatorResumable[T, V, C]) Check(rt *rapid.T, sut, _ T) error {
	maxPages := l.MaxPages
	if maxPages <= 0 {
		maxPages = 1000
	}
	full, starts := l.walk(rt, sut, l.Start, maxPages)
	i := rapid.IntRange(0, len(starts)-1).Draw(rt, "PaginatorResumable_page")
	resumed, _ := l.walk(rt, sut, starts[i], maxPages)
	want := slices.Concat(full[i:]...)
	got := slices.Concat(resumed...)
	if diff := cmp.Diff(want, got); diff != "" {
		return fmt.Errorf(
			"PaginatorResumable: resume from page %d diverged from full-walk suffix (-want +got):\n%s",
			i,
			diff,
		)
	}
	return nil
}

// PublisherDelivers verifies that a message published after N
// subscribers have registered reaches every one of them. Auto-
// emitted for methods carrying //testkit:contract publisher role=publish
// subscribe=<M>.
//
// Subscribe registers a subscriber and returns its handle; Publish
// broadcasts a message; Drain returns the messages a subscriber has
// received. The law subscribes Subscribers handles (default 3),
// publishes one drawn message, and asserts each subscriber drained
// it at least once.
type PublisherDelivers[T any, M comparable, Sub any] struct {
	Subscribe func(rt *rapid.T, sut T) (Sub, error)
	Publish   func(rt *rapid.T, sut T, msg M) error
	Drain     func(rt *rapid.T, sut T, sub Sub) ([]M, error)
	Messages  *rapid.Generator[M]

	// Subscribers is how many subscribe before the publish. Zero
	// defaults to 3 — one subscriber cannot distinguish delivery from
	// fan-out, and two cannot distinguish fan-out from a broadcast that
	// happens to reach both.
	Subscribers int
}

// ID returns the stable identifier for this law.
func (PublisherDelivers[T, M, Sub]) ID() string { return lawid.PublisherDelivers }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (PublisherDelivers[T, M, Sub]) REQID() string { return "" }

// Check subscribes N handles, publishes one message, and verifies
// every subscriber received it.
func (l PublisherDelivers[T, M, Sub]) Check(rt *rapid.T, sut, ref T) error {
	n := l.Subscribers
	if n <= 0 {
		n = 3
	}
	subs := make([]Sub, n)
	for i := range n {
		s, err := l.Subscribe(rt, sut)
		if err != nil {
			return Vacuous // a precondition this run supplies was refused
		}
		subs[i] = s
	}
	msg := l.Messages.Draw(rt, "PublisherDelivers_msg")
	if err := l.Publish(rt, sut, msg); err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	// The whole cycle lands on both sides — the mirrored half of the [Law]
	// conduct contract. The reference's subscriber is subscribed, published
	// to and drained the same way, so what residue a cycle leaves stays
	// symmetric across the pair.
	//
	// Mirrored, not compared: what the reference delivered is discarded,
	// and the verdict below reads the subject alone. So which
	// implementation stands behind ref changes nothing this law decides,
	// which is why a publisher with a delivery oracle available still sits
	// on the twin floor. Comparing the two drains — one-sided per mode,
	// the subject free to deliver more under at-least-once and less under
	// at-most-once — is what would change that, and it is a different law
	// from this one.
	if err := mirror("PublisherDelivers", func() error {
		refSub, subErr := l.Subscribe(rt, ref)
		if subErr != nil {
			return subErr
		}
		if pubErr := l.Publish(rt, ref, msg); pubErr != nil {
			return pubErr
		}
		_, drainErr := l.Drain(rt, ref, refSub)
		return drainErr
	}); err != nil {
		return err
	}
	for i, sub := range subs {
		got, err := l.Drain(rt, sut, sub)
		if err != nil {
			return fmt.Errorf("PublisherDelivers: subscriber %d: Drain errored: %v", i, err)
		}
		if !slices.Contains(got, msg) {
			return fmt.Errorf("PublisherDelivers: subscriber %d did not receive published message %v", i, msg)
		}
	}
	return nil
}

// DeliveryMode selects the per-message delivery guarantee a
// [PublisherDeliveryBound] enforces, mirroring the publisher contract
// directive's mode= parameter.
type DeliveryMode int

const (
	// DeliveryAtLeastOnce requires each subscriber to receive a
	// published message one or more times (duplicates permitted).
	DeliveryAtLeastOnce DeliveryMode = iota
	// DeliveryAtMostOnce requires each subscriber to receive a
	// published message zero or one times (loss permitted, duplicates
	// forbidden).
	DeliveryAtMostOnce
	// DeliveryExactlyOnce requires each subscriber to receive a
	// published message exactly once even across a redelivery.
	DeliveryExactlyOnce
)

// PublisherDeliveryBound verifies the per-subscriber delivery count
// of a published message against the bound implied by Mode. Auto-
// emitted for publisher contracts carrying a mode= parameter; the law
// ID is the per-mode variant.
//
// The law publishes one message, optionally triggers a redelivery
// via Redeliver (re-publish for at-least-once, replay for
// exactly-once; nil to skip), then counts how many copies each
// subscriber drained and checks the count against Mode. A refused
// redelivery is a precondition this run supplies, so it holds
// vacuously rather than counting a delivery that never happened.
type PublisherDeliveryBound[T any, M comparable, Sub any] struct {
	Subscribe func(rt *rapid.T, sut T) (Sub, error)
	Publish   func(rt *rapid.T, sut T, msg M) error
	Redeliver func(rt *rapid.T, sut T, msg M) error
	Drain     func(rt *rapid.T, sut T, sub Sub) ([]M, error)
	Messages  *rapid.Generator[M]
	Mode      DeliveryMode

	// Subscribers is how many subscribe before the publish, as in
	// PublisherDelivers. Zero defaults to 3.
	Subscribers int
}

// ID returns the per-mode stable identifier for this law.
func (l PublisherDeliveryBound[T, M, Sub]) ID() string {
	switch l.Mode {
	case DeliveryAtLeastOnce:
		return lawid.PublisherAtLeastOnce
	case DeliveryAtMostOnce:
		return lawid.PublisherAtMostOnce
	case DeliveryExactlyOnce:
		return lawid.PublisherExactlyOnce
	default:
		return lawid.PublisherDelivery
	}
}

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (PublisherDeliveryBound[T, M, Sub]) REQID() string { return "" }

// Check publishes one message (with an optional redelivery) and
// verifies each subscriber's delivery count respects Mode.
func (l PublisherDeliveryBound[T, M, Sub]) Check(rt *rapid.T, sut, ref T) error {
	n := l.Subscribers
	if n <= 0 {
		n = 3
	}
	subs := make([]Sub, n)
	for i := range n {
		s, err := l.Subscribe(rt, sut)
		if err != nil {
			return Vacuous // a precondition this run supplies was refused
		}
		subs[i] = s
	}
	msg := l.Messages.Draw(rt, "PublisherDeliveryBound_msg")
	if err := l.Publish(rt, sut, msg); err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	if l.Redeliver != nil {
		if err := l.Redeliver(rt, sut, msg); err != nil {
			return Vacuous // a precondition this run supplies was refused
		}
	}
	// The whole cycle lands on both sides — the mirrored half of the [Law]
	// conduct contract — redelivery included, so the pair's residue stays
	// symmetric whatever the mode. The reference's own redelivery refusal is
	// swallowed: the count below reads the subject, and a mirror that failed
	// the law for the reference's slack would fail a correct pair.
	if err := mirror("PublisherDeliveryBound", func() error {
		refSub, subErr := l.Subscribe(rt, ref)
		if subErr != nil {
			return subErr
		}
		if pubErr := l.Publish(rt, ref, msg); pubErr != nil {
			return pubErr
		}
		if l.Redeliver != nil {
			_ = l.Redeliver(rt, ref, msg)
		}
		_, drainErr := l.Drain(rt, ref, refSub)
		return drainErr
	}); err != nil {
		return err
	}
	for i, sub := range subs {
		got, err := l.Drain(rt, sut, sub)
		if err != nil {
			return fmt.Errorf("PublisherDeliveryBound: subscriber %d: Drain errored: %v", i, err)
		}
		count := 0
		for _, x := range got {
			if x == msg {
				count++
			}
		}
		if err := checkDeliveryBound(l.Mode, count); err != nil {
			return fmt.Errorf("PublisherDeliveryBound: subscriber %d: %w", i, err)
		}
	}
	return nil
}

// checkDeliveryBound reports whether a per-subscriber delivery count
// satisfies the named mode's bound.
func checkDeliveryBound(mode DeliveryMode, count int) error {
	switch mode {
	case DeliveryAtLeastOnce:
		if count < 1 {
			return fmt.Errorf("at-least-once: message delivered %d times (want ≥1)", count)
		}
	case DeliveryAtMostOnce:
		if count > 1 {
			return fmt.Errorf("at-most-once: message delivered %d times (want ≤1)", count)
		}
	case DeliveryExactlyOnce:
		if count != 1 {
			return fmt.Errorf("exactly-once: message delivered %d times (want 1)", count)
		}
	}
	return nil
}

// TransactionNoMidTxVisibility verifies that a write buffered inside
// an open transaction is invisible to an outside reader until the
// transaction commits. Auto-emitted for TransactionFunc methods with
// a paired Reader.
//
// The law records an outside read of a key, opens a transaction and
// buffers a write to that key, reads the key again from outside
// (which must observe the pre-transaction state), then rolls the
// transaction back. A store that leaks the uncommitted write changes
// the key's observable presence or value mid-transaction and fails.
type TransactionNoMidTxVisibility[T any, Tx any, K comparable, V any] struct {
	Begin func(rt *rapid.T, sut T) (Tx, error)
	// TxPut and TxRollback take the subject beside the handle: a fixture's
	// handle is a token, not a carrier, and the operation that stages
	// through it still belongs to the store that issued it.
	TxPut      func(rt *rapid.T, sut T, tx Tx, k K, v V) error
	TxRollback func(rt *rapid.T, sut T, tx Tx) error
	Read       func(rt *rapid.T, sut T, k K) (V, error)
	Keys       *rapid.Generator[K]
	Values     *rapid.Generator[V]
}

// ID returns the stable identifier for this law.
func (TransactionNoMidTxVisibility[T, Tx, K, V]) ID() string {
	return lawid.TransactionNoMidTxVisibility
}

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (TransactionNoMidTxVisibility[T, Tx, K, V]) REQID() string { return "" }

// Check verifies an outside read taken during an open transaction
// matches the read taken before the transaction began.
func (l TransactionNoMidTxVisibility[T, Tx, K, V]) Check(rt *rapid.T, sut, ref T) error {
	k := l.Keys.Draw(rt, "TransactionNoMidTxVisibility_key")
	v := l.Values.Draw(rt, "TransactionNoMidTxVisibility_value")

	// Levelness first, agreement never: the same begin/stage/rollback lands
	// on the reference, errors swallowed — a begin that advances a counter
	// on one side only reads as divergence at the next compared call. Level
	// twins refuse the same steps, so the guarded replay traces the same
	// shape on both sides.
	defer func() {
		refTx, err := l.Begin(rt, ref)
		if err != nil {
			return
		}
		_ = l.TxPut(rt, ref, refTx, k, v)
		_ = l.TxRollback(rt, ref, refTx)
	}()

	before, beforeErr := l.Read(rt, sut, k)
	tx, err := l.Begin(rt, sut)
	if err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	if putErr := l.TxPut(rt, sut, tx, k, v); putErr != nil {
		_ = l.TxRollback(rt, sut, tx)
		return Vacuous // a precondition this run supplies was refused
	}
	mid, midErr := l.Read(rt, sut, k)
	_ = l.TxRollback(rt, sut, tx)

	if (beforeErr == nil) != (midErr == nil) {
		return fmt.Errorf(
			"TransactionNoMidTxVisibility: key %v: uncommitted write changed presence (before=%v, mid=%v)",
			k,
			beforeErr,
			midErr,
		)
	}
	if beforeErr == nil {
		if diff := cmp.Diff(before, mid); diff != "" {
			return fmt.Errorf(
				"TransactionNoMidTxVisibility: key %v: uncommitted write visible to outside read (-before +mid):\n%s",
				k,
				diff,
			)
		}
	}
	return nil
}

// LeaseReleasedOnCancel verifies that a lease acquired under a
// context is released once that context is cancelled — modelling the
// "release on cancel/panic" half of the AcquireLease contract. Auto-
// emitted for methods carrying //testkit:contract lease role=acquire
// release=<M>.
//
// Acquire takes the governing context directly (not [rapid.T]): the
// law creates a cancellable context, acquires the key under it,
// cancels it, then polls Free until the lease frees or Timeout
// elapses. Release is typically asynchronous (a goroutine observing
// ctx.Done()), so the poll tolerates a brief propagation delay; a
// correct implementation frees within microseconds, while one that
// ignores cancellation never frees and fails at the deadline.
// Timeout ≤ 0 defaults to one second.
type LeaseReleasedOnCancel[T any, K comparable] struct {
	Acquire func(ctx context.Context, sut T, k K) error
	Free    func(rt *rapid.T, sut T, k K) bool
	Keys    *rapid.Generator[K]
	Timeout time.Duration
}

// ID returns the stable identifier for this law.
func (LeaseReleasedOnCancel[T, K]) ID() string { return lawid.LeaseReleasedOnCancel }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (LeaseReleasedOnCancel[T, K]) REQID() string { return "" }

// Check acquires a key under a fresh context, cancels it, and
// verifies the lease becomes free within Timeout.
func (l LeaseReleasedOnCancel[T, K]) Check(rt *rapid.T, sut, _ T) error {
	k := l.Keys.Draw(rt, "LeaseReleasedOnCancel_key")
	ctx, cancel := context.WithCancel(rt.Context())
	if err := l.Acquire(ctx, sut, k); err != nil {
		cancel()
		return Vacuous // a precondition this run supplies was refused
	}
	// Zero tolerance is deliberate: an acquire that returns before the
	// grant is observable is the defect under test, not scheduling
	// noise — the grant happened-before Acquire returned, or Acquire
	// answered a question it had not settled.
	if l.Free(rt, sut, k) {
		cancel()
		return fmt.Errorf("LeaseReleasedOnCancel: key %v reported free immediately after acquire", k)
	}
	cancel()
	timeout := l.Timeout
	if timeout <= 0 {
		timeout = time.Second
	}
	// The declared budget is the contract; the scale is CI headroom. A
	// shared runner under -race can stretch goroutine scheduling past a
	// budget the interface author measured on quiet hardware, and the
	// honest response is a wider budget everywhere, not a flaky red.
	timeout = scaleTimeout(timeout)
	deadline := time.Now().Add(timeout)
	for {
		if l.Free(rt, sut, k) {
			return nil
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf(
				"LeaseReleasedOnCancel: key %v not released within %v of context cancellation",
				k,
				timeout,
			)
		}
		time.Sleep(time.Millisecond)
	}
}

// WatcherReturnsOnChange verifies that a watch established before a
// mutation observes that mutation. Auto-emitted for methods carrying
// //testkit:contract watcher role=watch trigger=<M> next=<M> stop=<M>.
//
// Watch establishes a watch on a key and returns a handle; Mutate
// changes the key; Next blocks for the watch's next event up to the
// given timeout, reporting ok=false on timeout; Stop tears the watch
// down. The law watches a key, mutates it, and asserts the watch
// fired. Timeout ≤ 0 defaults to one second.
type WatcherReturnsOnChange[T any, W any, K comparable, V any] struct {
	Watch   func(rt *rapid.T, sut T, k K) (W, error)
	Mutate  func(rt *rapid.T, sut T, k K, v V) error
	Next    func(w W, timeout time.Duration) (V, bool)
	Stop    func(w W)
	Keys    *rapid.Generator[K]
	Values  *rapid.Generator[V]
	Timeout time.Duration
}

// ID returns the stable identifier for this law.
func (WatcherReturnsOnChange[T, W, K, V]) ID() string { return lawid.WatcherReturnsOnChange }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (WatcherReturnsOnChange[T, W, K, V]) REQID() string { return "" }

// Check establishes a watch, mutates the watched key, and verifies
// the watch fires within Timeout.
func (l WatcherReturnsOnChange[T, W, K, V]) Check(rt *rapid.T, sut, ref T) error {
	k := l.Keys.Draw(rt, "WatcherReturnsOnChange_key")
	v := l.Values.Draw(rt, "WatcherReturnsOnChange_value")
	w, err := l.Watch(rt, sut, k)
	if err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	defer l.Stop(w)
	if mErr := l.Mutate(rt, sut, k, v); mErr != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	// The mutation lands on both sides — the mirrored half of the [Law]
	// conduct contract. The watch itself stays subject-side: it is the
	// observation the claim is about, and the deferred stop cleans it.
	if mErr := mirror("WatcherReturnsOnChange", func() error { return l.Mutate(rt, ref, k, v) }); mErr != nil {
		return mErr
	}
	timeout := l.Timeout
	if timeout <= 0 {
		timeout = time.Second
	}
	if _, ok := l.Next(w, timeout); !ok {
		return fmt.Errorf("WatcherReturnsOnChange: key %v: watch did not fire within %v of a change", k, timeout)
	}
	return nil
}
