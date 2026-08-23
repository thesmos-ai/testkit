// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// restrictedpool is the pool-provenance axis: what a tier may draw turns
// on who chose the pool.
//
// Two runs below, and the pair is the whole fixture. The first takes the
// DERIVED pool — nobody said what this store accepts, so the transforms'
// hostile member is in it and a faithful store meets a control sequence
// and a broken rune. The second passes a pool of its own, which is a
// statement about what the store takes, and the subject beside it is
// written to match: it refuses anything outside the narrowing. That
// subject survives only because the narrowing is respected, which is what
// makes it evidence rather than decoration.
package restrictedpooltest_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/restrictedpool"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/restrictedpool/restrictedpooltest"
)

// TestStoreContract runs every check against the derived pool, hostile
// member and all.
func TestStoreContract(t *testing.T) {
	t.Parallel()

	restrictedpooltest.RunStore(t, inMemory("in-memory"), storeChecks)
}

// TestStoreChecksCanFail drives every planted defect through the check it
// is evidence for.
func TestStoreChecksCanFail(t *testing.T) {
	t.Parallel()

	restrictedpooltest.ProveStore(t, inMemory("in-memory"), storeChecks)
}

// TestStoreContractUnderANarrowedPool is the other half of the axis.
//
// The config says what this store takes, and plainOnly refuses everything
// else. Both are true statements about one implementation, and the run
// passes only because the tier draws the pool it was given rather than the
// one it would have derived: a hostile member reaching Put here would be
// refused, the reference would accept it, and the differential would red a
// store that had done nothing wrong.
func TestStoreContractUnderANarrowedPool(t *testing.T) {
	t.Parallel()

	restrictedpooltest.RunStore(t,
		restrictedpooltest.StoreHarness[*plainOnly]{Name: "plain-only", New: newPlainOnly},
		restrictedpooltest.StoreConfig{
			KeyPool:  []restrictedpool.Key{"first-key", "second-key"},
			BodyPool: []restrictedpool.Body{"first-body", "second-body"},
		},
	)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) restrictedpooltest.StoreHarness[*restrictedpooltest.InMemory] {
	return restrictedpooltest.StoreHarness[*restrictedpooltest.InMemory]{
		Name: name, New: restrictedpooltest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var storeChecks = restrictedpooltest.StoreChecks{
	{
		Method: "Put", Name: "keeps-a-hostile-body-verbatim",
		Claim: "Put keeps a body carrying a control sequence as data",
		Run:   keepsAHostileBodyVerbatim,
		ProvenBy: restrictedpooltest.BrokenStore(
			"a store that trims what it is given", newTrimsTheBody,
		),
		ProvenReason: "the body comes back as it went in",
	},
}

// --- Bodies -------------------------------------------------------------------

// keepsAHostileBodyVerbatim states what the derived pool exists to probe.
//
// The generated sequences draw the hostile member on their own; this says
// what the answer must be when they do, which a drawn sequence cannot —
// it compares against a reference, and a store and a reference that
// mangle a body identically agree.
func keepsAHostileBodyVerbatim(
	tb testing.TB, s restrictedpool.Store, fx restrictedpooltest.StoreFixture,
) {
	tb.Helper()
	hostile := fx.BodyPool()[len(fx.BodyPool())-1]
	testkit.NoError(tb, s.Put(tb.Context(), fx.Key(), hostile), "the body is accepted")

	got, err := s.Get(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "and the key reads")
	testkit.Equal(tb, got, hostile, "and the body comes back as it went in")
}

// --- Planted defects ----------------------------------------------------------

// trimsTheBody strips what it does not recognise, which is a store that
// interprets its input instead of holding it.
type trimsTheBody struct {
	mu     sync.Mutex
	bodies map[restrictedpool.Key]restrictedpool.Body
}

func newTrimsTheBody() *trimsTheBody {
	return &trimsTheBody{bodies: map[restrictedpool.Key]restrictedpool.Body{}}
}

func (t *trimsTheBody) Put(_ context.Context, key restrictedpool.Key, body restrictedpool.Body) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.bodies[key] = restrictedpool.Body(strings.TrimFunc(string(body), func(r rune) bool {
		return r < ' ' || r > '~'
	}))
	return nil
}

func (t *trimsTheBody) Get(_ context.Context, key restrictedpool.Key) (restrictedpool.Body, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	body, ok := t.bodies[key]
	if !ok {
		return "", restrictedpool.ErrNotFound
	}
	return body, nil
}

// plainOnly takes printable ASCII and refuses the rest — a store with a
// domain, which is ordinary: a column width, a charset, a schema.
//
// It is correct under the narrowing the run beside it passes, and only
// under that narrowing. Run against the derived pool it would refuse the
// hostile member and diverge from a reference that accepts everything,
// which is exactly the red a tier drawing past a consumer's statement
// would produce on code that was right.
type plainOnly struct {
	mu     sync.Mutex
	bodies map[restrictedpool.Key]restrictedpool.Body
}

func newPlainOnly() *plainOnly {
	return &plainOnly{bodies: map[restrictedpool.Key]restrictedpool.Body{}}
}

func (p *plainOnly) Put(ctx context.Context, key restrictedpool.Key, body restrictedpool.Body) error {
	if ctx == nil {
		return restrictedpool.ErrNotFound
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.IndexFunc(string(body), func(r rune) bool { return r < ' ' || r > '~' }) >= 0 {
		return errUnacceptable
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.bodies[key] = body
	return nil
}

func (p *plainOnly) Get(ctx context.Context, key restrictedpool.Key) (restrictedpool.Body, error) {
	if ctx == nil {
		return "", restrictedpool.ErrNotFound
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	body, ok := p.bodies[key]
	if !ok {
		return "", restrictedpool.ErrNotFound
	}
	return body, nil
}

// errUnacceptable is plainOnly's refusal for a body outside its domain.
var errUnacceptable = errors.New("restrictedpooltest: this store takes plain bodies only")
