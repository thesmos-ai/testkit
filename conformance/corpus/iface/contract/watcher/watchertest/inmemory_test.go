// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// watcher is the model tier's under ADR-0028:
// `AUTO-WATCHER-RETURNS-ON-CHANGE` states it, driving the subscription's
// next= and stop= members the directive names.
//
// Every claim below is statable through the interface, so every one is a row
// rather than a test in this package: a row runs against each subject a
// consumer declares and again through the double, and a package test runs
// against the one implementation this package holds.
package watchertest_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/watcher"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/watcher/watchertest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	watchertest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	watchertest.RunContract(t,
		inMemory("in-memory"),
		watchertest.ContractSuite.Without(watchertest.ContractSuite.Checks.Watch.Smoke()),
	)
}

// TestContractChecksCanFail drives every row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	watchertest.ProveContract(t, inMemory("in-memory"), contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

// none is how long a check waits to prove nothing arrives. Short on purpose:
// the claim is absence, and absence does not get truer with time.
const none = 50 * time.Millisecond

func inMemory(name string) watchertest.ContractHarness[*watchertest.InMemory] {
	return watchertest.ContractHarness[*watchertest.InMemory]{
		Name: name, New: watchertest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------
//
// Routing is what every row is about, so every planted defect routes wrongly in
// one nameable way. Each is the near miss its row exists to forbid, which is
// why five rows do not share one broken subject: a watcher that wakes nobody
// reds four of these, and four reds for the same reason are one piece of
// evidence wearing four hats.

var contractChecks = watchertest.ContractChecks{
	{
		Method: "Trigger", Name: "wakes-a-watcher-of-the-changed-key",
		Claim: "Trigger wakes a watcher of the key that changed",
		Run:   wakesAWatcherOfTheChangedKey,
		ProvenBy: watchertest.BrokenContract(
			"a watcher that accepts every change and delivers none", planted(wakesNobody),
		),
		ProvenReason: "reaches the watcher of that key",
	},

	{
		Method: "Watch", Name: "does-not-wake-a-watcher-of-another-key",
		Claim: "Watch does not wake a watcher of another key",
		Run:   doesNotWakeAWatcherOfAnotherKey,
		ProvenBy: watchertest.BrokenContract(
			"a watcher that wakes the whole system on one write", planted(wakesEverybody),
		),
		ProvenReason: "does not reach them",
	},

	{
		Method: "Watch", Name: "wakes-every-watcher-of-one-key",
		Claim: "Watch wakes every watcher of one key",
		Run:   wakesEveryWatcherOfOneKey,
		ProvenBy: watchertest.BrokenContract(
			"a watcher that hands each change to whoever it reached first",
			planted(wakesTheFirst),
		),
		ProvenReason: "and the second",
	},

	{
		Method: "Watch", Name: "delivers-nothing-that-predates-it",
		Claim: "Watch delivers nothing that predates the watch",
		Run:   deliversNothingThatPredatesIt,
		ProvenBy: watchertest.BrokenContract(
			"a watcher that replays what happened before it", planted(keepsABacklog),
		),
		ProvenReason: "handed nothing that predates it",
	},

	{
		Method: "Trigger", Name: "reports-an-unreachable-watcher",
		Claim: "Trigger reports a watcher it can no longer reach",
		Run:   reportsAnUnreachableWatcher,
		ProvenBy: watchertest.BrokenContract(
			"a watcher that drops what it cannot deliver", planted(dropsSilently),
		),
		ProvenReason: "never reported as behind",
	},
}

// --- Bodies -------------------------------------------------------------------

// wakesAWatcherOfTheChangedKey is the contract, stated once.
func wakesAWatcherOfTheChangedKey(
	tb testing.TB, s watcher.Contract, fx watchertest.ContractFixture,
) {
	tb.Helper()
	sub, err := s.Watch(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "a watcher attaches")
	defer sub.Stop()

	testkit.NoError(tb, s.Trigger(tb.Context(), fx.Key(), fx.Value()),
		"the change is recorded")
	got, ok := sub.Next(time.Second)
	testkit.True(tb, ok, "and reaches the watcher of that key")
	testkit.Equal(tb, got, fx.Value(), "carrying what was written")
}

// doesNotWakeAWatcherOfAnotherKey holds the routing narrow.
//
// A subject notifying every watcher on every change satisfies the row above and
// wakes the whole system on one write.
func doesNotWakeAWatcherOfAnotherKey(
	tb testing.TB, s watcher.Contract, fx watchertest.ContractFixture,
) {
	tb.Helper()
	sub, err := s.Watch(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "a watcher attaches to one key")
	defer sub.Stop()

	testkit.NoError(tb,
		s.Trigger(tb.Context(), fx.KeyOther(), watcher.Value{Key: fx.KeyOther()}),
		"a change to another key is recorded")
	_, ok := sub.Next(none)
	testkit.False(tb, ok, "and does not reach them")
}

// wakesEveryWatcherOfOneKey holds the routing wide enough.
//
// A subject handing each change to whichever watcher it reached first satisfies
// a one-watcher check and loses half the wake-ups.
func wakesEveryWatcherOfOneKey(
	tb testing.TB, s watcher.Contract, fx watchertest.ContractFixture,
) {
	tb.Helper()
	first, err := s.Watch(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "the first watcher attaches")
	defer first.Stop()
	second, err := s.Watch(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "and so does the second")
	defer second.Stop()

	testkit.NoError(tb, s.Trigger(tb.Context(), fx.Key(), fx.Value()),
		"the change is recorded")
	_, ok := first.Next(time.Second)
	testkit.True(tb, ok, "reaching the first")
	_, ok = second.Next(time.Second)
	testkit.True(tb, ok, "and the second")
}

// deliversNothingThatPredatesIt keeps this contract out of `outbox`'s.
//
// A subject keeping a backlog would hand this watcher a change from before it
// existed — which is `outbox`, and a different contract.
func deliversNothingThatPredatesIt(
	tb testing.TB, s watcher.Contract, fx watchertest.ContractFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Trigger(tb.Context(), fx.Key(), fx.Value()),
		"a change is recorded with nobody watching")

	sub, err := s.Watch(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "a watcher attaches afterwards")
	defer sub.Stop()
	_, ok := sub.Next(none)
	testkit.False(tb, ok, "and is handed nothing that predates it")
}

// reportsAnUnreachableWatcher makes the bound reachable.
//
// A watcher that never reads eventually cannot be delivered to, and a subject
// that drops the change instead of saying so leaves the caller believing it
// landed.
func reportsAnUnreachableWatcher(
	tb testing.TB, s watcher.Contract, fx watchertest.ContractFixture,
) {
	tb.Helper()
	sub, err := s.Watch(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "a watcher attaches")
	defer sub.Stop()

	for range 32 {
		if err := s.Trigger(tb.Context(), fx.Key(), fx.Value()); err != nil {
			testkit.ErrorIs(tb, err, watchertest.ErrFull,
				"the trigger says why the change could not be taken")
			return
		}
	}
	tb.Fatalf("a watcher that never reads was never reported as behind")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted watcher gets wrong.
//
// One implementation with a routing switch rather than five, because routing is
// the only thing any of them gets wrong: five copies of the subscription
// bookkeeping to vary one branch would bury the difference the rows are about.
type fault int

const (
	// wakesNobody accepts every change and delivers none.
	wakesNobody fault = iota

	// wakesEverybody delivers every change to every watcher, whatever key
	// they asked for.
	wakesEverybody

	// wakesTheFirst delivers to the earliest watcher of a key and no other.
	wakesTheFirst

	// keepsABacklog routes correctly and hands a new watcher the changes
	// that predate it.
	keepsABacklog

	// dropsSilently routes correctly and, when a watcher is too far behind
	// to take the change, discards it rather than saying so.
	dropsSilently
)

// plantedBuffer is small enough that reportsAnUnreachableWatcher's 32 triggers
// overrun it: a bound nothing can reach proves nothing about what happens at it.
const plantedBuffer = 8

// planted builds the constructor for one broken watcher.
func planted(wrong fault) func() *plantedWatcher {
	return func() *plantedWatcher {
		return &plantedWatcher{
			wrong: wrong,
			subs:  map[string][]*plantedSub{},
			past:  map[string][]watcher.Value{},
		}
	}
}

type plantedWatcher struct {
	wrong fault
	mu    sync.Mutex
	subs  map[string][]*plantedSub
	past  map[string][]watcher.Value
}

func (w *plantedWatcher) Watch(_ context.Context, key string) (watcher.Subscription, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	sub := &plantedSub{ch: make(chan watcher.Value, plantedBuffer)}
	if w.wrong == keepsABacklog {
		for _, v := range w.past[key] {
			sub.ch <- v
		}
	}
	w.subs[key] = append(w.subs[key], sub)
	return sub, nil
}

// Trigger never reports a watcher it could not reach, which is dropsSilently's
// defect and harmless to the rest: no other row fills a buffer.
func (w *plantedWatcher) Trigger(_ context.Context, key string, v watcher.Value) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.past[key] = append(w.past[key], v)
	for _, sub := range w.reached(key) {
		select {
		case sub.ch <- v:
		default:
		}
	}
	return nil
}

// reached is the routing, and the one line each fault disagrees about.
func (w *plantedWatcher) reached(key string) []*plantedSub {
	switch w.wrong {
	case wakesNobody:
		return nil
	case wakesEverybody:
		var all []*plantedSub
		for _, subs := range w.subs {
			all = append(all, subs...)
		}
		return all
	case wakesTheFirst:
		if subs := w.subs[key]; len(subs) > 0 {
			return subs[:1]
		}
		return nil
	default:
		return w.subs[key]
	}
}

type plantedSub struct{ ch chan watcher.Value }

func (s *plantedSub) Next(timeout time.Duration) (watcher.Value, bool) {
	select {
	case v := <-s.ch:
		return v, true
	case <-time.After(timeout):
		return watcher.Value{}, false
	}
}

func (*plantedSub) Stop() {}
