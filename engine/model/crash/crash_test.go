// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package crash_test

import (
	"context"
	"errors"
	"maps"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/model/crash"
)

// scratch moves a test that fails on purpose into a directory of its
// own, so the fail file the property engine writes goes away with it.
//
// Every rejection below drives the schedule against a store built to
// break it, and the engine files a replay seed for each — a genuine
// service to somebody debugging, and litter from a test whose whole
// point is that it failed. The move rules out t.Parallel, which is the
// price: three tests run in sequence rather than leaving three fail
// files behind on every run.
func scratch(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
}

// errMissing is what every store below reports for a key it does not
// hold, so the schedule's absent reading has a sentinel to name.
var errMissing = errors.New("crash_test: no such key")

// errMediumGone is what an induced store reports once its medium has
// failed. Any error refuses the write; the wording carries no weight.
var errMediumGone = errors.New("crash_test: the medium is gone")

// record is what a write installs and a read asks for.
type record struct {
	Key  string
	Body string
}

// written are the keys a drawn record lands on, and read are those plus
// one nothing ever writes — the schedule has to catch a rebuild that
// answers for a key no write acknowledged, and it can only do that by
// asking for one.
var (
	written = []string{"alpha", "beta", "gamma"}
	read    = []string{"alpha", "beta", "gamma", "never-written"}
)

// medium is what survives a crash: the rows a store flushed, held apart
// from any instance so a rebuild can inherit them.
type medium struct{ rows map[string]record }

func newMedium() *medium { return &medium{rows: map[string]record{}} }

// store writes through to its medium and reads back through it, which is
// the correct implementation: nothing is held anywhere a crash can reach.
type store struct{ m *medium }

func (s store) Put(_ context.Context, r record) error {
	s.m.rows[r.Key] = r
	return nil
}

func (s store) Get(_ context.Context, k string) (record, error) {
	r, held := s.m.rows[k]
	if !held {
		return record{}, errMissing
	}
	return r, nil
}

func (store) rebuild(prior store) store { return store{m: prior.m} }

// buffered acknowledges a write into memory and flushes nothing, so a
// rebuild inherits an empty medium — the debt an acknowledgement creates,
// defaulted on.
type buffered struct {
	m   *medium
	buf map[string]record
}

func (s buffered) Put(_ context.Context, r record) error {
	s.buf[r.Key] = r
	return nil
}

func (s buffered) Get(ctx context.Context, k string) (record, error) {
	if r, held := s.buf[k]; held {
		return r, nil
	}
	return store{m: s.m}.Get(ctx, k)
}

func (buffered) rebuild(prior buffered) buffered {
	return buffered{m: prior.m, buf: map[string]record{}}
}

// inventive flushes correctly and then hallucinates one extra row every
// time it is rebuilt.
type inventive struct{ m *medium }

func (s inventive) Put(ctx context.Context, r record) error { return store{m: s.m}.Put(ctx, r) }

func (s inventive) Get(ctx context.Context, k string) (record, error) {
	return store{m: s.m}.Get(ctx, k)
}

func (inventive) rebuild(prior inventive) inventive {
	prior.m.rows["never-written"] = record{Key: "never-written", Body: "from nowhere"}
	return inventive{m: prior.m}
}

// corrupting keeps every key and mangles every body it carries across a
// rebuild.
type corrupting struct{ m *medium }

func (s corrupting) Put(ctx context.Context, r record) error { return store{m: s.m}.Put(ctx, r) }

func (s corrupting) Get(ctx context.Context, k string) (record, error) {
	return store{m: s.m}.Get(ctx, k)
}

func (corrupting) rebuild(prior corrupting) corrupting {
	for k, r := range maps.All(prior.m.rows) {
		r.Body += "-mangled"
		prior.m.rows[k] = r
	}
	return corrupting{m: prior.m}
}

// failing refuses every write once its medium has gone, and writes
// nothing down when it does — the correct behaviour under a failure.
type failing struct {
	m      *medium
	broken bool
}

func (s *failing) Put(ctx context.Context, r record) error {
	if s.broken {
		return errMediumGone
	}
	return store{m: s.m}.Put(ctx, r)
}

func (s *failing) Get(ctx context.Context, k string) (record, error) {
	return store{m: s.m}.Get(ctx, k)
}

// journalling records the write to its medium and only then discovers it
// cannot commit, leaving behind a row nothing acknowledged.
//
// The defect the induced form exists to catch. A double answering the
// error in front of the subject never delivers the write, so this store
// would look identical to the correct one.
type journalling struct {
	m      *medium
	broken bool
}

func (s *journalling) Put(ctx context.Context, r record) error {
	if err := (store{m: s.m}).Put(ctx, r); err != nil {
		return err
	}
	if s.broken {
		return errMediumGone
	}
	return nil
}

func (s *journalling) Get(ctx context.Context, k string) (record, error) {
	return store{m: s.m}.Get(ctx, k)
}

// schedule is the same drawn interleaving for every store below, with
// only the four verbs swapped — so a difference in verdict is a
// difference in the implementation and nothing else.
//
// The verbs arrive receiver-first because that is the shape a method
// expression has, and naming the four methods is what keeps each case to
// one line. The two adapters below are the whole cost of that.
func schedule[S any](
	put func(S, context.Context, record) error,
	get func(S, context.Context, string) (record, error),
	rebuild func(S) S,
) crash.Schedule[S, string, record] {
	return crash.Schedule[S, string, record]{
		Values: model.Map(model.SampledFrom(written), func(k string) record {
			return record{Key: k, Body: k + "-body"}
		}),
		KeyOf: func(r record) string { return r.Key },
		Keys:  model.SampledFrom(read),
		Write: func(ctx context.Context, w S, r record) error { return put(w, ctx, r) },
		Read: func(ctx context.Context, w S, k string) (record, error) {
			return get(w, ctx, k)
		},
		Equal:   func(a, b record) bool { return a == b },
		Absent:  crash.Is(errMissing),
		Rebuild: rebuild,
		Steps:   30,
	}
}

// A store that writes through to its medium takes the schedule.
//
// The green half of the proof, and it has to come first: every rejection
// below is only evidence if the same schedule passes against something
// correct. A schedule that reddened everything would be a schedule that
// asserted nothing about crash recovery.
func TestAWriteThroughStoreSurvivesTheSchedule(t *testing.T) {
	t.Parallel()

	model.Check(t, func(rt *model.T) {
		s := store{m: newMedium()}
		crash.Run(rt, s, schedule(store.Put, store.Get, s.rebuild))
	})
}

// An acknowledged write that a rebuild cannot find is the defect this
// leg exists for, and it is named as a debt rather than as a mismatch.
//
//nolint:paralleltest // scratch moves the cwd, which t.Chdir and t.Parallel refuse together.
func TestABufferedWriteLostToACrashIsRejected(t *testing.T) {
	scratch(t)

	got := testkit.Rejects(t, "a store acknowledging into memory must not pass the crash schedule",
		func(tb testing.TB) {
			tb.Helper()
			model.Check(tb, func(rt *model.T) {
				s := buffered{m: newMedium(), buf: map[string]record{}}
				crash.Run(rt, s, schedule(buffered.Put, buffered.Get, s.rebuild))
			})
		})

	testkit.Contains(t, got, "an acknowledged write is a debt the medium owes across a rebuild",
		"and says which promise was broken, not just that a read disagreed")
}

// The claim is two-sided: a rebuild that answers for a key nothing wrote
// is as wrong as one that forgets a key something did.
//
// Worth stating on its own. An oracle recording attempts rather than
// acknowledgements would pass this defect, and an oracle that only read
// back what it had written would never ask the question.
//
//nolint:paralleltest // scratch moves the cwd, which t.Chdir and t.Parallel refuse together.
func TestARebuildThatInventsStateIsRejected(t *testing.T) {
	scratch(t)

	got := testkit.Rejects(t, "a rebuild seeding a row nobody wrote must not pass",
		func(tb testing.TB) {
			tb.Helper()
			model.Check(tb, func(rt *model.T) {
				s := inventive{m: newMedium()}
				crash.Run(rt, s, schedule(inventive.Put, inventive.Get, s.rebuild))
			})
		})

	testkit.Contains(t, got, "nothing acknowledged this key",
		"the absent half of the claim, named as its own failure")
}

// A rebuild that keeps every key and changes what they hold is caught by
// the comparison rather than by the presence check.
//
//nolint:paralleltest // scratch moves the cwd, which t.Chdir and t.Parallel refuse together.
func TestARebuildThatMangesWhatItKeptIsRejected(t *testing.T) {
	scratch(t)

	got := testkit.Rejects(t, "a rebuild altering the records it kept must not pass",
		func(tb testing.TB) {
			tb.Helper()
			model.Check(tb, func(rt *model.T) {
				s := corrupting{m: newMedium()}
				crash.Run(rt, s, schedule(corrupting.Put, corrupting.Get, s.rebuild))
			})
		})

	testkit.Contains(t, got, "want the acknowledged",
		"and reports both records, so the difference is readable without a rerun")
}

// A schedule with no Absent reads any error as a miss.
//
// The lenient reading is for a contract that stamps no miss sentinel,
// where insisting on one would red every implementation that reports
// absence its own way.
func TestNoAbsentReadsEveryErrorAsAMiss(t *testing.T) {
	t.Parallel()

	model.Check(t, func(rt *model.T) {
		s := store{m: newMedium()}
		sch := schedule(store.Put, store.Get, s.rebuild)
		sch.Absent = nil
		crash.Run(rt, s, sch)
	})
}

// A failure induced in the subject carries across the crash seam.
//
// A medium that has failed does not recover by being reopened, so every
// incarnation a crash produces is put back into the failing state. A
// schedule that induced once and forgot would report a store as having
// survived a failure it stopped having.
func TestAnInducedFailureSurvivesTheRebuild(t *testing.T) {
	t.Parallel()

	model.Check(t, func(rt *model.T) {
		m := newMedium()
		s := &failing{m: m}
		sch := schedule(func(w *failing, ctx context.Context, r record) error {
			return w.Put(ctx, r)
		}, func(w *failing, ctx context.Context, k string) (record, error) {
			return w.Get(ctx, k)
		}, func(prior *failing) *failing { return &failing{m: prior.m} })
		sch.Induce = func(w *failing) { w.broken = true }
		crash.Run(rt, s, sch)
	})
}

// A store that writes to its medium and only then discovers it cannot
// commit is caught, and this is the defect the induced form exists for.
//
// A double in front of the subject cannot find it. That arrangement
// answers the error without calling the subject, so the write never
// arrives, nothing is written down, and the claim has nothing to be
// false about — a green row saying nothing. Here the subject receives
// the write, records it, and then fails.
//
//nolint:paralleltest // scratch moves the cwd, which t.Chdir and t.Parallel refuse together.
func TestAWriteRecordedBeforeItFailedIsRejected(t *testing.T) {
	scratch(t)

	got := testkit.Rejects(t, "a store that journals before it commits must not pass",
		func(tb testing.TB) {
			tb.Helper()
			model.Check(tb, func(rt *model.T) {
				s := &journalling{m: newMedium()}
				sch := schedule(func(w *journalling, ctx context.Context, r record) error {
					return w.Put(ctx, r)
				}, func(w *journalling, ctx context.Context, k string) (record, error) {
					return w.Get(ctx, k)
				}, func(prior *journalling) *journalling { return &journalling{m: prior.m} })
				sch.Induce = func(w *journalling) { w.broken = true }
				crash.Run(rt, s, sch)
			})
		})

	testkit.Contains(t, got, "nothing acknowledged this key",
		"the write was refused, so what it left behind is state nobody asked for")
}
