// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// An interface's declarations are not its method set. Composed declares Get and
// inherits Ping and Close, so a harness reading declarations alone would hold an
// implementation to a third of its contract and report success.
package embeddedtest_test

import (
	"context"
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/embedded"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/embedded/embeddedtest"
)

// TestComposedContract runs the generated checks and this package's own.
func TestComposedContract(t *testing.T) {
	t.Parallel()

	embeddedtest.RunComposed(t, composed("in-memory"), composedChecks)
}

// TestComposedChecksCanFail drives the Composed row against its planted defect.
func TestComposedChecksCanFail(t *testing.T) {
	t.Parallel()

	embeddedtest.ProveComposed(t, composed("in-memory"), composedChecks)
}

// TestBaseContract runs the first embedded interface, which is a contract in
// its own right — one implementation answers to all three.
func TestBaseContract(t *testing.T) {
	t.Parallel()

	embeddedtest.RunBase(t, base("in-memory"), baseChecks)
}

// TestBaseContractWithoutSmoke drops a check per contract rather than per
// package: three interfaces share this file and each has its own index.
func TestBaseContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	embeddedtest.RunBase(t,
		base("in-memory"),
		embeddedtest.BaseSuite.Without(embeddedtest.BaseSuite.Checks.Ping.Smoke()),
	)
}

// TestBaseChecksCanFail drives the Base row against its planted defect.
func TestBaseChecksCanFail(t *testing.T) {
	t.Parallel()

	embeddedtest.ProveBase(t, base("in-memory"), baseChecks)
}

// TestCloserContract runs the second embedded interface.
func TestCloserContract(t *testing.T) {
	t.Parallel()

	embeddedtest.RunCloser(t, closer("in-memory"), closerChecks)
}

// TestCloserContractWithoutSmoke is the same drop, for Closer.
func TestCloserContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	embeddedtest.RunCloser(t,
		closer("in-memory"),
		embeddedtest.CloserSuite.Without(embeddedtest.CloserSuite.Checks.Close.Smoke()),
	)
}

// TestCloserChecksCanFail drives the Closer row against its planted defect.
func TestCloserChecksCanFail(t *testing.T) {
	t.Parallel()

	embeddedtest.ProveCloser(t, closer("in-memory"), closerChecks)
}

// --- Harnesses ---------------------------------------------------------------

// seededBody is what the seeded key holds.
const seededBody = "seeded"

// composed seeds the reader, because Composed declares no writer and the hit
// path is unreachable without a seeded constructor.
func composed(name string) embeddedtest.ComposedHarness[*embeddedtest.InMemory] {
	return embeddedtest.ComposedHarness[*embeddedtest.InMemory]{Name: name, New: seeded}
}

func seeded() *embeddedtest.InMemory {
	s := embeddedtest.NewInMemory()
	s.Put(embeddedtest.DefaultComposedFixture().Key(), seededBody)
	return s
}

// base and closer take the bare constructor: neither interface exposes state a
// reader observes, so neither has anything to seed.
func base(name string) embeddedtest.BaseHarness[*embeddedtest.InMemory] {
	return embeddedtest.BaseHarness[*embeddedtest.InMemory]{
		Name: name, New: embeddedtest.NewInMemory,
	}
}

func closer(name string) embeddedtest.CloserHarness[*embeddedtest.InMemory] {
	return embeddedtest.CloserHarness[*embeddedtest.InMemory]{
		Name: name, New: embeddedtest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------
//
// One row per interface, and one planted subject behind all three: what a
// defect here has to be wrong about is a METHOD, and the three interfaces
// between them declare three.

var composedChecks = embeddedtest.ComposedChecks{
	{
		Method: "Get", Name: "returns-what-was-seeded",
		Claim: "Get returns what was seeded",
		Run:   returnsWhatWasSeeded,
		ProvenBy: embeddedtest.BrokenComposed(
			"a subject whose reads answer the zero value", planted(answersTheZero),
		),
		ProvenReason: "carries what was written",
	},
}

var baseChecks = embeddedtest.BaseChecks{
	{
		Method: "Ping", Name: "succeeds-on-a-fresh-subject",
		Claim: "Ping succeeds on a fresh subject",
		Run:   succeedsOnAFreshSubject,
		ProvenBy: embeddedtest.BrokenBase(
			"a subject that is unwell before anything happened to it",
			planted(refusesAFreshPing),
		),
		ProvenReason: "an open subject answers a ping",
	},
}

var closerChecks = embeddedtest.CloserChecks{
	{
		Method: "Close", Name: "second-close-succeeds",
		Claim: "Close is idempotent",
		Run:   secondCloseSucceeds,
		ProvenBy: embeddedtest.BrokenCloser(
			"a subject that refuses to be closed twice", planted(refusesTheSecondClose),
		),
		ProvenReason: "and so does the second",
	},
}

// --- Bodies -------------------------------------------------------------------

func returnsWhatWasSeeded(
	tb testing.TB, s embedded.Composed, fx embeddedtest.ComposedFixture,
) {
	tb.Helper()
	got, err := s.Get(tb.Context(), fx.Key())
	testkit.NoError(tb, err, "a seeded identifier is found")
	testkit.Equal(tb, got, seededBody, "and carries what was written")
}

func succeedsOnAFreshSubject(
	tb testing.TB, s embedded.Base, _ embeddedtest.BaseFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Ping(tb.Context()), "an open subject answers a ping")
}

func secondCloseSucceeds(
	tb testing.TB, s embedded.Closer, _ embeddedtest.CloserFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Close(tb.Context()), "the first close succeeds")
	testkit.NoError(tb, s.Close(tb.Context()), "and so does the second")
}

// --- Planted defects ----------------------------------------------------------

// errPlanted is what a planted refusal answers with. Its identity does not
// matter to any row here — each asserts only that there was no error — so one
// sentinel serves all three.
var errPlanted = errors.New("embeddedtest_test: the planted defect refuses")

// fault names which of the three methods one planted subject gets wrong.
type fault int

const (
	// answersTheZero finds the key and hands back nothing, which is the
	// bug a check reading only the error cannot see.
	answersTheZero fault = iota

	// refusesAFreshPing reports a subject unwell before anything has
	// happened to it.
	refusesAFreshPing

	// refusesTheSecondClose treats shutdown as a state change rather than a
	// state, which takes down every caller with a deferred Close.
	refusesTheSecondClose
)

// planted builds the constructor for one broken subject.
//
// One type behind all three rows, because it satisfies all three interfaces —
// which is this fixture's whole point: an implementation answers to the
// embedded contracts and the composed one at once.
func planted(wrong fault) func() *plantedSubject {
	return func() *plantedSubject { return &plantedSubject{wrong: wrong} }
}

type plantedSubject struct {
	wrong  fault
	closed bool
}

func (p *plantedSubject) Get(context.Context, string) (string, error) {
	if p.wrong == answersTheZero {
		return "", nil
	}
	return seededBody, nil
}

func (p *plantedSubject) Ping(context.Context) error {
	if p.wrong == refusesAFreshPing {
		return errPlanted
	}
	return nil
}

func (p *plantedSubject) Close(context.Context) error {
	if p.wrong == refusesTheSecondClose && p.closed {
		return errPlanted
	}
	p.closed = true
	return nil
}
