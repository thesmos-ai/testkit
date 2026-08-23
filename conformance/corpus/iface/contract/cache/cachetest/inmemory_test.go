// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// cache is the model tier's under ADR-0028: `AUTO-CACHEABLE` states it.
//
// Neither role writes, so nothing is derived to seed through and the header
// refuses both miss checks. The seeding is the constructor's, which is where a
// seeded subject is built now — a factory may make any starting state, and it
// runs before anything wraps it.
package cachetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cache"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cache/cachetest"
)

// TestContractContract runs the generated checks and this package's own,
// against a cold subject and one whose cache is already warm.
func TestContractContract(t *testing.T) {
	t.Parallel()

	cachetest.RunContract(t, seeded("in-memory"), warmed("in-memory, already warmed"),
		contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	cachetest.RunContract(t,
		seeded("in-memory"),
		cachetest.ContractSuite.Without(cachetest.ContractSuite.Checks.Lookup.Smoke()),
	)
}

// TestContractChecksCanFail drives every row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	cachetest.ProveContract(t, seeded("in-memory"), contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

// seededBody is what the backing store holds under the fixture's key.
const seededBody = "seeded"

func seeded(name string) cachetest.ContractHarness[*cachetest.InMemory] {
	return cachetest.ContractHarness[*cachetest.InMemory]{
		Name: name,
		New: func() *cachetest.InMemory {
			s := cachetest.NewInMemory()
			s.Store(cache.Value{Key: cachetest.DefaultContractFixture().Key(), Body: seededBody})
			return s
		},
	}
}

// warmed reaches the cached read, which is invisible through the interface —
// both reads answer the same thing whether or not anything was cached.
//
// So it is reached by handing the run a subject whose cache is already warm and
// whose backing no longer holds the key. Every read against this one is a hit,
// and a subject with no cache misses. Start rather than New, because warming
// can fail and the failure is the test's to report.
func warmed(name string) cachetest.ContractHarness[*cachetest.InMemory] {
	return cachetest.ContractHarness[*cachetest.InMemory]{Name: name, Start: warmTheCache}
}

func warmTheCache(tb testing.TB) *cachetest.InMemory {
	tb.Helper()
	key := cachetest.DefaultContractFixture().Key()
	s := cachetest.NewInMemory()
	s.Store(cache.Value{Key: key, Body: seededBody})
	if _, err := s.Lookup(tb.Context(), key); err != nil {
		tb.Fatalf("warm the cache the second subject exists for: %v", err)
	}
	s.Forget(key)
	return s
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = cachetest.ContractChecks{
	{
		Method: "Lookup", Name: "answers-from-the-backing-store",
		Claim: "Lookup answers from the backing store on a miss",
		Run:   answersFromTheBackingStore,
		ProvenBy: cachetest.BrokenContract(
			"a cache that answers the zero for a key its backing holds",
			planted(answersTheZero),
		),
		ProvenReason: "carries what was stored",
	},

	{
		Method: "Lookup", Name: "misses-a-key-nothing-holds",
		Claim: "Lookup reports a key neither role holds",
		Run:   missesAKeyNothingHolds,
		ProvenBy: cachetest.BrokenContract(
			"a cache that answers a key neither role holds", planted(answersAnUnheldKey),
		),
		ProvenReason: "a miss rather than an unlabelled failure",
	},
}

// --- Bodies -------------------------------------------------------------------

func answersFromTheBackingStore(
	tb testing.TB, s cache.Contract, fx cachetest.ContractFixture,
) {
	tb.Helper()
	got, err := s.Lookup(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "a seeded key is found")
	testkit.Equal(tb, got.Body, seededBody, "and carries what was stored")
}

// missesAKeyNothingHolds is the claim the generator refuses here — nothing on
// this interface writes, so it cannot tell an unheld key from any other. The
// constructor is what makes the distinction real.
func missesAKeyNothingHolds(
	tb testing.TB, s cache.Contract, fx cachetest.ContractFixture,
) {
	tb.Helper()
	_, err := s.Lookup(tb.Context(), fx.KeyOther())
	testkit.ErrorIs(tb, err, cachetest.ErrNotFound,
		"a key nothing seeded is a miss rather than an unlabelled failure")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted cache gets wrong.
type fault int

const (
	// answersTheZero finds the key and hands back an empty value, which a
	// caller reading only the error takes for a real record.
	answersTheZero fault = iota

	// answersAnUnheldKey manufactures a hit for a key neither role holds,
	// which is a cache with a negative entry it never invalidated.
	answersAnUnheldKey
)

// planted builds the constructor for one broken cache, holding what the
// harness's own subject holds so a read has the same thing to find.
func planted(wrong fault) func() plantedCache {
	return func() plantedCache {
		return plantedCache{wrong: wrong, key: cachetest.DefaultContractFixture().Key()}
	}
}

type plantedCache struct {
	wrong fault
	key   string
}

// Fetch and Lookup answer alike, which is the contract: the cached read is
// invisible through the interface, so a defect that varied them would be
// varying something no row reads.
func (p plantedCache) Fetch(ctx context.Context, key string) (cache.Value, error) {
	return p.Lookup(ctx, key)
}

func (p plantedCache) Lookup(_ context.Context, key string) (cache.Value, error) {
	if key != p.key {
		if p.wrong == answersAnUnheldKey {
			return cache.Value{Key: key, Body: seededBody}, nil
		}
		return cache.Value{}, cachetest.ErrNotFound
	}
	if p.wrong == answersTheZero {
		return cache.Value{}, nil
	}
	return cache.Value{Key: key, Body: seededBody}, nil
}
