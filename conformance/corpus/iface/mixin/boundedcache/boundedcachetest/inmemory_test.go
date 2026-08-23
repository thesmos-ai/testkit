// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `bounded limit=` declares a ceiling, and the harness hands it to every
// constructor — the real subject's and each planted defect's — so the number
// under test is the declared one rather than a literal repeated here.
package boundedcachetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/boundedcache"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/boundedcache/boundedcachetest"
	"go.thesmos.sh/testkit/engine/suite"
)

// TestMixedContract runs the generated checks and this package's own.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	boundedcachetest.RunMixed(t, inMemory("in-memory"), mixedChecks)
}

// TestMixedChecksCanFail drives each row against its planted defect.
func TestMixedChecksCanFail(t *testing.T) {
	t.Parallel()

	boundedcachetest.ProveMixed(t, inMemory("in-memory"), mixedChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) boundedcachetest.MixedHarness[*boundedcachetest.InMemory] {
	return boundedcachetest.MixedHarness[*boundedcachetest.InMemory]{
		Name: name, New: boundedcachetest.NewInMemory,
	}
}

// --- The checks -----------------------------------------------------------

// None hand-written, and for once that is the finding rather than a gap.
//
// The two claims this declaration makes are both generated. The bound is
// AUTO-AGGREGATOR-BOUNDED, which reads the count the mixin names. That
// what the cache answers is what it was told is the differential, against
// an oracle deliberately built without the bound — a hit has to agree,
// and a miss never disagrees, so eviction stops looking like a wrong
// answer while inventing a hit still is one.
//
// The control below is what a hand-written check here would mostly be
// guarding against, and it does the job from the other side.
var mixedChecks = boundedcachetest.MixedChecks{}

// --- The negative control -----------------------------------------------------

// TestMixedToleratesADifferentVictim is the specificity half of this
// package's evidence.
//
// Every other test here measures sensitivity: a planted defect must be
// caught. None of them can tell a check that is too STRONG from one that
// is right — a check reddening a legal implementation looks exactly like a
// suite working, until somebody's correct code fails it.
//
// The freedom is the reason this fixture exists. `bounded limit=2` says
// the cache holds at most two; it says nothing about WHICH two survive.
// The subject beside it evicts the oldest, so a run that came to require
// that would pass its own tests and fail everybody else's — and the
// asymmetric read comparison is precisely the machinery that must not
// have quietly reintroduced the requirement.
func TestMixedToleratesADifferentVictim(t *testing.T) {
	t.Parallel()

	boundedcachetest.GreenMixed(t, suite.Subject[boundedcachetest.Mixed]{
		Name: "a cache that evicts the newest",
		New: func(testing.TB) boundedcachetest.Mixed {
			return newestOut(boundedcachetest.MixedSuite.DeclaredLimit())
		},
	}, mixedChecks)
}

// newestOutCache evicts the most recently written key rather than the
// oldest, which the bound permits and this package's own subject does not
// do. Same count, different survivors.
type newestOutCache struct {
	capacity int
	order    []boundedcache.Key
	entries  map[boundedcache.Key]boundedcache.Value
}

func newestOut(capacity int) *newestOutCache {
	return &newestOutCache{
		capacity: capacity,
		entries:  map[boundedcache.Key]boundedcache.Value{},
	}
}

func (n *newestOutCache) Put(ctx context.Context, key boundedcache.Key, value boundedcache.Value) error {
	if ctx == nil {
		return context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, held := n.entries[key]; !held {
		if len(n.order) >= n.capacity {
			newest := n.order[len(n.order)-1]
			n.order = n.order[:len(n.order)-1]
			delete(n.entries, newest)
		}
		n.order = append(n.order, key)
	}
	n.entries[key] = value
	return nil
}

func (n *newestOutCache) Get(ctx context.Context, key boundedcache.Key) (boundedcache.Value, bool) {
	if ctx == nil || ctx.Err() != nil {
		return boundedcache.Value{}, false
	}
	v, held := n.entries[key]
	return v, held
}

func (n *newestOutCache) Len(ctx context.Context) (int, error) {
	if ctx == nil {
		return 0, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return len(n.entries), nil
}
