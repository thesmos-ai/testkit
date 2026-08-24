// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package persistertest_test

import (
	"context"
	"sync"
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/persister"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/persister/persistertest"
	"go.thesmos.sh/testkit/engine/suite"
)

// TestContractAcrossImplementations runs two correct implementations in
// one pass, which is the only arrangement where three of the harness's
// fields mean anything.
//
// Oracle names the one to compare against and at most one run may set
// it; Serial holds an implementation to itself, for something that
// cannot take being built many times at once; and Excuse is how an
// implementation says a check is asking the impossible of it rather than
// failing it. Each was inert while every consumer test here ran a single
// store.
func TestContractAcrossImplementations(t *testing.T) {
	t.Parallel()

	persistertest.RunContract(t, inMemory("in-memory"), volatile("volatile"), contractChecks)
}

// TestContractGreen runs every check against a correct implementation
// that is not the one the rows were written against.
//
// The negative control. A check that reds here is one pinned to how the
// in-memory store happens to work rather than to what the contract says,
// and nothing else in this file can tell the two apart.
func TestContractGreen(t *testing.T) {
	t.Parallel()

	persistertest.GreenContract(t, suite.Subject[persistertest.Contract]{
		Name: "a store that keeps nothing past itself",
		New: func(tb testing.TB) persistertest.Contract {
			tb.Helper()
			return startVolatile(tb)
		},
		// The two sim rows ask what survives a rebuild, and this control
		// has nothing to rebuild over. Excused rather than answered: an
		// unarmed control reds on wiring, and a wiring red says nothing
		// about whether the checks are pinned to one store's internals,
		// which is the only question this run is here to settle.
		Excused: suite.ExcuseSet([]suite.ID{
			persistertest.ContractSuite.Checks.Sim.Recovery(),
			persistertest.ContractSuite.Checks.Sim.Fault(),
		}),
	}, contractChecks)
}

// --- The volatile harness ------------------------------------------------------

// volatile describes a store that keeps everything in the instance.
//
// Correct, and durable in no sense: the two sim rows ask what survives
// the process being rebuilt over its medium, and this one has no medium
// to rebuild over. Excusing them reports exactly that. Answering with a
// Recover that hands back a fresh empty store would report the claim as
// PASSED, which is the lie the field's own docblock warns about.
func volatile(name string) persistertest.ContractHarness[*volatileStore] {
	return persistertest.ContractHarness[*volatileStore]{
		Name: name,
		// Start rather than New: the store registers its own teardown on
		// the test, which is the shape a consumer holding a temp
		// directory or a connection needs.
		Start: startVolatile,
		Excuse: []suite.ID{
			persistertest.ContractSuite.Checks.Sim.Recovery(),
			persistertest.ContractSuite.Checks.Sim.Fault(),
		},
		// One at a time. Nothing here needs it — the point is that a
		// consumer with a fixed port or a real database does, and this is
		// where the field is exercised.
		Serial: true,
	}
}

// startVolatile builds a store and registers the teardown on the test.
func startVolatile(tb testing.TB) *volatileStore {
	tb.Helper()
	s := &volatileStore{values: map[string]persister.Value{}}
	tb.Cleanup(s.drop)
	return s
}

// ctxErr reports a cancelled context and tolerates a nil one, which the
// generated nil-context check hands over on purpose: a method reached
// through an interface must answer rather than panic.
func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return context.Canceled
	}
	return ctx.Err()
}

// volatileStore is a correct persister that keeps nothing past itself.
//
// Different from InMemory in the way that matters to the negative
// control: it stores a copy keyed by the value's own key and hands a
// copy back, where InMemory stores the value as given. A check pinned to
// either one's internals rather than to the contract reds here.
type volatileStore struct {
	mu      sync.Mutex
	values  map[string]persister.Value
	dropped bool
}

func (s *volatileStore) drop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values, s.dropped = nil, true
}

func (s *volatileStore) Put(ctx context.Context, v persister.Value) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dropped {
		return persister.ErrMediumGone
	}
	stored := v
	s.values[v.Key] = stored
	return nil
}

func (s *volatileStore) Get(ctx context.Context, key string) (persister.Value, error) {
	if err := ctxErr(ctx); err != nil {
		return persister.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	held, found := s.values[key]
	if !found {
		return persister.Value{}, persistertest.ErrNotFound
	}
	return held, nil
}
