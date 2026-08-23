// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// The reader role is what makes `mode=atomic` statable at all: "an error leaves
// observable state unchanged" needs something to observe through, and before
// Get was declared the only statable claim was that a good write succeeds.
package batchwritertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	batchwriter "go.thesmos.sh/testkit/conformance/corpus/iface/contract/batch-writer"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/batch-writer/batchwritertest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	batchwritertest.RunContract(t, inMemory("in-memory"), contractChecks, keyedPool())
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	batchwritertest.RunContract(t,
		inMemory("in-memory"),
		batchwritertest.ContractSuite.Without(batchwritertest.ContractSuite.Checks.Put.Smoke()),
		keyedPool(),
	)
}

// TestContractChecksCanFail drives every row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	batchwritertest.ProveContract(t, inMemory("in-memory"), contractChecks, keyedPool())
}

// keyedPool states what this subject accepts.
//
// It refuses a value with an empty key, which is deliberate — that
// refusal is what gives `mode=atomic` a failure to be about. The
// adversarial arm draws the empty string among others, so without this
// the run would red a subject for declining an input its own author
// ruled out. A pool passed here is that ruling, written down: every tier
// draws it verbatim and the widening is dropped.
func keyedPool() batchwritertest.ContractConfig {
	return batchwritertest.ContractConfig{
		KeyPool:  []string{"test-key", "other-key"},
		BodyPool: []string{"test-body", "other-body"},
	}
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) batchwritertest.ContractHarness[*batchwritertest.InMemory] {
	return batchwritertest.ContractHarness[*batchwritertest.InMemory]{
		Name: name, New: batchwritertest.NewInMemory,
		// The crash seam. The map outlives the instance holding it, which
		// is what makes a rebuild over it mean anything: an acknowledged
		// write is still there when the process that took it is not.
		Recover: batchwritertest.Reopen,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = batchwritertest.ContractChecks{
	{
		Method: "Put", Name: "refuses-unkeyed",
		Claim: "Put refuses a value with nothing to file it under",
		Run:   refusesUnkeyed,
		ProvenBy: batchwritertest.BrokenContract(
			"a store that files an unkeyed value under nothing",
			planted(takesTheUnkeyed),
		),
		ProvenReason: "an unkeyed value is refused",
	},

	{
		Method: "Put", Name: "a-refused-write-lands-nowhere",
		Claim: "Put leaves the reader answering as it did when it refuses",
		Run:   aRefusedWriteLandsNowhere,
		ProvenBy: batchwritertest.BrokenContract(
			"a store that applies the batch before it validates it",
			planted(appliesBeforeItValidates),
		),
		ProvenReason: "unchanged by the write that failed",
	},
}

// --- Bodies -------------------------------------------------------------------

// refusesUnkeyed is the subject's one way to fail, and `mode=atomic` needs one:
// "an error leaves observable state unchanged" has no case to observe against a
// write that always succeeds.
func refusesUnkeyed(
	tb testing.TB, s batchwriter.Contract, fx batchwritertest.ContractFixture,
) {
	tb.Helper()
	testkit.Error(tb, s.Put(tb.Context(), batchwriter.Value{Body: fx.Value().Body}),
		"an unkeyed value is refused")
	testkit.NoError(tb, s.Put(tb.Context(), fx.Value()),
		"and the store still takes a keyed one")
}

// aRefusedWriteLandsNowhere is `mode=atomic` read through the role that now
// exists to read it.
func aRefusedWriteLandsNowhere(
	tb testing.TB, s batchwriter.Contract, fx batchwritertest.ContractFixture,
) {
	tb.Helper()
	held := fx.Value()
	testkit.NoError(tb, s.Put(tb.Context(), held), "a keyed value lands")

	before, err := s.Get(tb.Context(), held.Key)
	testkit.NoError(tb, err, "and reads back")

	testkit.Error(tb, s.Put(tb.Context(), batchwriter.Value{Body: "unkeyed"}),
		"the unkeyed write is refused")

	after, err := s.Get(tb.Context(), held.Key)
	testkit.NoError(tb, err, "the earlier value is still there")
	testkit.Equal(tb, after, before, "unchanged by the write that failed")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted store gets wrong.
type fault int

const (
	// takesTheUnkeyed files a value with no key under the empty one, which
	// is a store with no guard rather than a store with a broken one.
	takesTheUnkeyed fault = iota

	// appliesBeforeItValidates writes the refused value over everything it
	// held and then reports the refusal, which is the shape a batch applied
	// before it was checked has — and the reason the row reads the store
	// after the failure rather than trusting the error.
	//
	// It overwrites rather than clearing, so the earlier key is still
	// readable: a store that emptied itself would red the row on the read
	// before it, which is a different claim.
	appliesBeforeItValidates
)

// planted builds the constructor for one broken store.
func planted(wrong fault) func() *plantedStore {
	return func() *plantedStore {
		return &plantedStore{wrong: wrong, held: map[string]batchwriter.Value{}}
	}
}

type plantedStore struct {
	wrong fault
	held  map[string]batchwriter.Value
}

func (p *plantedStore) Put(_ context.Context, v batchwriter.Value) error {
	if v.Key == "" {
		if p.wrong == takesTheUnkeyed {
			p.held[v.Key] = v
			return nil
		}
		if p.wrong == appliesBeforeItValidates {
			for key, held := range p.held {
				held.Body = v.Body
				p.held[key] = held
			}
		}
		return batchwritertest.ErrUnkeyed
	}
	p.held[v.Key] = v
	return nil
}

func (p *plantedStore) Get(_ context.Context, key string) (batchwriter.Value, error) {
	v, held := p.held[key]
	if !held {
		return batchwriter.Value{}, batchwritertest.ErrNotFound
	}
	return v, nil
}
