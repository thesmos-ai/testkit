// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/history"
	"go.thesmos.sh/testkit/engine/model/law"
)

func TestStreamReentrancy(t *testing.T) {
	t.Parallel()

	t.Run("passes for reentrant stream", func(t *testing.T) {
		t.Parallel()
		items := []string{"a", "b", "c"}
		l := law.StreamReentrancy[[]string, string]{
			Collect: func(_ *rapid.T, s []string) ([]string, error) {
				return s, nil
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, items, nil)
			if err != nil {
				rt.Fatalf("unexpected error: %v", err)
			}
		})
	})

	t.Run("detects one-shot iterator", func(t *testing.T) {
		t.Parallel()
		l := law.StreamReentrancy[*atomic.Int64, string]{
			Collect: func(_ *rapid.T, counter *atomic.Int64) ([]string, error) {
				n := counter.Add(1)
				if n > 1 {
					// BUG: second iteration returns empty.
					return nil, nil
				}
				return []string{"item"}, nil
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			counter := &atomic.Int64{}
			err := l.Check(rt, counter, nil)
			if err == nil {
				rt.Fatal("should have detected one-shot iterator")
			}
		})
	})

	t.Run("passes for empty stream", func(t *testing.T) {
		t.Parallel()
		l := law.StreamReentrancy[string, int]{
			Collect: func(_ *rapid.T, _ string) ([]int, error) {
				return nil, nil
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, "x", "x")
			if err != nil {
				rt.Fatalf("unexpected error: %v", err)
			}
		})
	})

	t.Run("detects error on second iteration", func(t *testing.T) {
		t.Parallel()
		l := law.StreamReentrancy[*atomic.Int64, string]{
			Collect: func(_ *rapid.T, counter *atomic.Int64) ([]string, error) {
				n := counter.Add(1)
				if n > 1 {
					return nil, errors.New("second iteration fails")
				}
				return []string{"item"}, nil
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			counter := &atomic.Int64{}
			err := l.Check(rt, counter, nil)
			if err == nil {
				rt.Fatal("should have detected error on second iteration")
			}
		})
	})
}

func TestStreamCompletion(t *testing.T) {
	t.Parallel()

	t.Run("drain under limit passes", func(t *testing.T) {
		t.Parallel()
		l := law.StreamCompletion[[]string, string]{
			Drain: func(_ *rapid.T, s []string) ([]string, error) { return s, nil },
			Limit: 100,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []string{"a", "b"}, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("drain at limit flagged", func(t *testing.T) {
		t.Parallel()
		l := law.StreamCompletion[[]string, string]{
			Drain: func(_ *rapid.T, s []string) ([]string, error) { return s, nil },
			Limit: 3,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []string{"a", "b", "c", "d"}, nil); err == nil {
				rt.Fatal("expected limit reached")
			}
		})
	})
}

func TestStreamNoDuplicates(t *testing.T) {
	t.Parallel()

	t.Run("unique drain passes", func(t *testing.T) {
		t.Parallel()
		l := law.StreamNoDuplicates[[]string, string, string]{
			Drain: func(_ *rapid.T, s []string) ([]string, error) { return s, nil },
			Hash:  func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []string{"a", "b", "c"}, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("duplicate drain flagged", func(t *testing.T) {
		t.Parallel()
		l := law.StreamNoDuplicates[[]string, string, string]{
			Drain: func(_ *rapid.T, s []string) ([]string, error) { return s, nil },
			Hash:  func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []string{"a", "b", "a"}, nil); err == nil {
				rt.Fatal("expected duplicate")
			}
		})
	})
}

func TestStreamStableOrder(t *testing.T) {
	t.Parallel()

	t.Run("sorted drain passes", func(t *testing.T) {
		t.Parallel()
		l := law.StreamStableOrder[[]int, int]{
			Drain: func(_ *rapid.T, s []int) ([]int, error) { return s, nil },
			Less:  func(a, b int) bool { return a < b },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []int{1, 2, 3}, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("out-of-order drain flagged", func(t *testing.T) {
		t.Parallel()
		l := law.StreamStableOrder[[]int, int]{
			Drain: func(_ *rapid.T, s []int) ([]int, error) { return s, nil },
			Less:  func(a, b int) bool { return a < b },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []int{3, 1, 2}, nil); err == nil {
				rt.Fatal("expected out-of-order")
			}
		})
	})
}

// writeLog is the run's record of what went in, which is what the two
// drain laws judge a drain against. Unpartitioned, under the empty key
// the generated recording write records against.
func writeLog(written ...string) *history.History[string, string] {
	h := history.New[string, string]()
	for _, w := range written {
		h.Record("", w)
	}
	return h
}

// intLog is [writeLog] over the int element the streamSUT cases draw.
func intLog(written ...int) *history.History[string, int] {
	h := history.New[string, int]()
	for _, w := range written {
		h.Record("", w)
	}
	return h
}

func TestStreamPermutation(t *testing.T) {
	t.Parallel()

	t.Run("drain that permutes expected passes", func(t *testing.T) {
		t.Parallel()
		l := law.StreamPermutation[[]string, string, string]{
			Drain:   func(_ *rapid.T, s []string) ([]string, error) { return s, nil },
			History: writeLog("a", "b", "c"),
			Hash:    func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []string{"c", "a", "b"}, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("length mismatch flagged", func(t *testing.T) {
		t.Parallel()
		l := law.StreamPermutation[[]string, string, string]{
			Drain:   func(_ *rapid.T, s []string) ([]string, error) { return s, nil },
			History: writeLog("a", "b"),
			Hash:    func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []string{"a"}, nil); err == nil {
				rt.Fatal("expected length mismatch")
			}
		})
	})

	t.Run("element mismatch flagged", func(t *testing.T) {
		t.Parallel()
		l := law.StreamPermutation[[]string, string, string]{
			Drain:   func(_ *rapid.T, s []string) ([]string, error) { return s, nil },
			History: writeLog("a", "b"),
			Hash:    func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []string{"a", "x"}, nil); err == nil {
				rt.Fatal("expected element mismatch")
			}
		})
	})
}

func TestStreamOverMatch(t *testing.T) {
	t.Parallel()

	t.Run("drain containing required passes", func(t *testing.T) {
		t.Parallel()
		l := law.StreamOverMatch[[]string, string, string]{
			Drain:   func(_ *rapid.T, s []string) ([]string, error) { return s, nil },
			History: writeLog("a", "b"),
			Hash:    func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []string{"a", "b", "c", "d"}, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("missing required element flagged", func(t *testing.T) {
		t.Parallel()
		l := law.StreamOverMatch[[]string, string, string]{
			Drain:   func(_ *rapid.T, s []string) ([]string, error) { return s, nil },
			History: writeLog("a", "b"),
			Hash:    func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []string{"a", "c"}, nil); err == nil {
				rt.Fatal("expected missing element")
			}
		})
	})

	t.Run("drain error is vacuous", func(t *testing.T) {
		t.Parallel()
		l := law.StreamOverMatch[[]string, string, string]{
			Drain:   func(_ *rapid.T, _ []string) ([]string, error) { return nil, errors.New("nope") },
			History: writeLog("a"),
			Hash:    func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); !law.Holds(err) {
				rt.Fatal(err)
			}
		})
	})
}

// snapStore is a keyed store whose stream either reflects live state
// or (when stale is set) a snapshot frozen at construction — the bug
// where mutations never show up in the streamed view.
type snapStore struct {
	live  map[string]string
	snap  map[string]string
	stale bool
}

func newSnapStore(stale bool) *snapStore {
	return &snapStore{live: map[string]string{}, snap: map[string]string{}, stale: stale}
}

func (s *snapStore) put(v string) error {
	s.live[v] = v
	return nil
}

func (s *snapStore) del(v string) error {
	delete(s.live, v)
	return nil
}

func (s *snapStore) stream() ([]string, error) {
	src := s.live
	if s.stale {
		src = s.snap
	}
	out := make([]string, 0, len(src))
	for v := range src {
		out = append(out, v)
	}
	return out, nil
}

func TestStreamReflectsMutations(t *testing.T) {
	t.Parallel()

	t.Run("live stream reflects puts and deletes", func(t *testing.T) {
		t.Parallel()
		l := law.StreamReflectsMutations[*snapStore, string, string]{
			Put:    func(_ *rapid.T, s *snapStore, v string) error { return s.put(v) },
			Delete: func(_ *rapid.T, s *snapStore, v string) error { return s.del(v) },
			Drain:  func(_ *rapid.T, s *snapStore) ([]string, error) { return s.stream() },
			Values: rapid.SampledFrom([]string{"a", "b", "c"}),
			Hash:   func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := newSnapStore(false)
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("stale-snapshot stream is caught", func(t *testing.T) {
		t.Parallel()
		l := law.StreamReflectsMutations[*snapStore, string, string]{
			Put:    func(_ *rapid.T, s *snapStore, v string) error { return s.put(v) },
			Delete: func(_ *rapid.T, s *snapStore, v string) error { return s.del(v) },
			Drain:  func(_ *rapid.T, s *snapStore) ([]string, error) { return s.stream() },
			Values: rapid.SampledFrom([]string{"a", "b", "c"}),
			Hash:   func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := newSnapStore(true)
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected stale stream to be caught")
			}
		})
	})

	t.Run("delete omitted checks only the put direction", func(t *testing.T) {
		t.Parallel()
		l := law.StreamReflectsMutations[*snapStore, string, string]{
			Put:    func(_ *rapid.T, s *snapStore, v string) error { return s.put(v) },
			Drain:  func(_ *rapid.T, s *snapStore) ([]string, error) { return s.stream() },
			Values: rapid.SampledFrom([]string{"a", "b"}),
			Hash:   func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := newSnapStore(false)
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

// streamSUT drains a fixed slice, optionally failing the drain. It exists to
// drive the stream laws' precondition and violation branches without a real
// iterator.
type streamSUT struct {
	items    []int
	drainErr error
	drains   int
}

func (s *streamSUT) drain(*rapid.T, *streamSUT) ([]int, error) {
	s.drains++
	if s.drainErr != nil {
		return nil, s.drainErr
	}
	return append([]int(nil), s.items...), nil
}

// Every stream law treats a failed drain as a precondition — the subject could
// not produce a stream, so there is nothing to judge — except StreamReentrancy,
// which is specifically about a stream that works once and then does not.
func TestStreamLawDrainFailures(t *testing.T) {
	t.Parallel()

	t.Run("StreamCompletion holds vacuously", func(t *testing.T) {
		t.Parallel()
		s := &streamSUT{drainErr: errors.New("closed")}
		l := law.StreamCompletion[*streamSUT, int]{Drain: s.drain}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); !law.Holds(err) {
				rt.Fatalf("a failed drain is a precondition: %v", err)
			}
		})
	})

	t.Run("StreamNoDuplicates holds vacuously", func(t *testing.T) {
		t.Parallel()
		s := &streamSUT{drainErr: errors.New("closed")}
		l := law.StreamNoDuplicates[*streamSUT, int, int]{
			Drain: s.drain, Hash: func(v int) int { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); !law.Holds(err) {
				rt.Fatalf("a failed drain is a precondition: %v", err)
			}
		})
	})

	t.Run("StreamStableOrder holds vacuously", func(t *testing.T) {
		t.Parallel()
		s := &streamSUT{drainErr: errors.New("closed")}
		l := law.StreamStableOrder[*streamSUT, int]{
			Drain: s.drain, Less: func(a, b int) bool { return a < b },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); !law.Holds(err) {
				rt.Fatalf("a failed drain is a precondition: %v", err)
			}
		})
	})

	t.Run("StreamPermutation holds vacuously", func(t *testing.T) {
		t.Parallel()
		s := &streamSUT{drainErr: errors.New("closed")}
		l := law.StreamPermutation[*streamSUT, int, int]{
			Drain:   s.drain,
			History: intLog(1),
			Hash:    func(v int) int { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); !law.Holds(err) {
				rt.Fatalf("a failed drain is a precondition: %v", err)
			}
		})
	})

	t.Run("StreamReentrancy reports a failed drain instead", func(t *testing.T) {
		t.Parallel()
		s := &streamSUT{drainErr: errors.New("closed")}
		l := law.StreamReentrancy[*streamSUT, int]{Collect: s.drain}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("reentrancy is precisely about drains that fail")
			}
		})
	})

	t.Run("StreamReentrancy flags a second drain that fails", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := &streamSUT{items: []int{1, 2}}
			l := law.StreamReentrancy[*streamSUT, int]{
				Collect: func(rt *rapid.T, sut *streamSUT) ([]int, error) {
					sut.drains++
					if sut.drains > 1 {
						return nil, errors.New("single-use iterator")
					}
					return sut.items, nil
				},
			}
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a single-use iterator is a reentrancy violation")
			}
		})
	})
}

func TestStreamLawViolations(t *testing.T) {
	t.Parallel()

	t.Run("StreamCompletion flags a stream that hits the limit", func(t *testing.T) {
		t.Parallel()
		s := &streamSUT{items: []int{1, 2, 3, 4, 5}}
		l := law.StreamCompletion[*streamSUT, int]{Drain: s.drain, Limit: 3}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("reaching the limit means the stream did not terminate")
			}
		})
	})

	// Limit <= 0 selects the runner's default rather than failing every
	// stream immediately.
	t.Run("StreamCompletion falls back to a default limit", func(t *testing.T) {
		t.Parallel()
		s := &streamSUT{items: []int{1, 2, 3}}
		l := law.StreamCompletion[*streamSUT, int]{Drain: s.drain, Limit: 0}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatalf("a short stream must pass under the default limit: %v", err)
			}
		})
	})

	t.Run("StreamNoDuplicates flags a repeated hash", func(t *testing.T) {
		t.Parallel()
		s := &streamSUT{items: []int{1, 2, 1}}
		l := law.StreamNoDuplicates[*streamSUT, int, int]{
			Drain: s.drain, Hash: func(v int) int { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a repeated element is a duplicate violation")
			}
		})
	})

	t.Run("StreamStableOrder flags an out-of-order pair", func(t *testing.T) {
		t.Parallel()
		s := &streamSUT{items: []int{1, 3, 2}}
		l := law.StreamStableOrder[*streamSUT, int]{
			Drain: s.drain, Less: func(a, b int) bool { return a < b },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a descending pair violates the declared order")
			}
		})
	})

	t.Run("StreamPermutation flags a length mismatch", func(t *testing.T) {
		t.Parallel()
		s := &streamSUT{items: []int{1, 2}}
		l := law.StreamPermutation[*streamSUT, int, int]{
			Drain:   s.drain,
			History: intLog(1, 2, 3),
			Hash:    func(v int) int { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a different element count is not a permutation")
			}
		})
	})

	t.Run("StreamPermutation flags same-length different-elements", func(t *testing.T) {
		t.Parallel()
		s := &streamSUT{items: []int{1, 2}}
		l := law.StreamPermutation[*streamSUT, int, int]{
			Drain:   s.drain,
			History: intLog(1, 3),
			Hash:    func(v int) int { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("same length with different elements is not a permutation")
			}
		})
	})

	t.Run("StreamPermutation accepts a reordering", func(t *testing.T) {
		t.Parallel()
		s := &streamSUT{items: []int{2, 1}}
		l := law.StreamPermutation[*streamSUT, int, int]{
			Drain:   s.drain,
			History: intLog(1, 2),
			Hash:    func(v int) int { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatalf("a reordering is a valid permutation: %v", err)
			}
		})
	})
}

// StreamReflectsMutations is the only stream law that writes: a put must
// become visible and a delete must become invisible. Delete is optional, so
// the law must stop cleanly when it is absent rather than treating a
// still-present value as a violation.
func TestStreamReflectsMutationsBranches(t *testing.T) {
	t.Parallel()

	type setSUT struct{ items []int }
	drain := func(_ *rapid.T, s *setSUT) ([]int, error) { return append([]int(nil), s.items...), nil }
	put := func(_ *rapid.T, s *setSUT, v int) error { s.items = append(s.items, v); return nil }
	del := func(_ *rapid.T, s *setSUT, v int) error {
		out := s.items[:0]
		for _, it := range s.items {
			if it != v {
				out = append(out, it)
			}
		}
		s.items = out
		return nil
	}
	hash := func(v int) int { return v }

	t.Run("a refused put holds vacuously", func(t *testing.T) {
		t.Parallel()
		s := &setSUT{}
		l := law.StreamReflectsMutations[*setSUT, int, int]{
			Put:    func(*rapid.T, *setSUT, int) error { return errors.New("read-only") },
			Drain:  drain,
			Values: rapid.Just(1),
			Hash:   hash,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); !law.Holds(err) {
				rt.Fatalf("a refused put is a precondition: %v", err)
			}
		})
	})

	t.Run("a put that does not appear is a violation", func(t *testing.T) {
		t.Parallel()
		s := &setSUT{}
		l := law.StreamReflectsMutations[*setSUT, int, int]{
			Put:    func(*rapid.T, *setSUT, int) error { return nil }, // silently drops
			Drain:  drain,
			Values: rapid.Just(1),
			Hash:   hash,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a put that never appears in the stream is a violation")
			}
		})
	})

	t.Run("a failed drain after put is a violation", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			drains := 0
			s := &setSUT{}
			l := law.StreamReflectsMutations[*setSUT, int, int]{
				Put: put,
				// The baseline count answered, so refusing after the put is
				// the subject breaking rather than a precondition.
				Drain: func(rt *rapid.T, s *setSUT) ([]int, error) {
					drains++
					if drains > 1 {
						return nil, errors.New("closed")
					}
					return drain(rt, s)
				},
				Values: rapid.Just(1),
				Hash:   hash,
			}
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a drain that errors after a successful put is a violation")
			}
		})
	})

	t.Run("a drain refused before anything ran holds vacuously", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := &setSUT{}
			l := law.StreamReflectsMutations[*setSUT, int, int]{
				Put:    put,
				Drain:  func(*rapid.T, *setSUT) ([]int, error) { return nil, errors.New("closed") },
				Values: rapid.Just(1),
				Hash:   hash,
			}
			if err := l.Check(rt, s, s); !law.Holds(err) {
				rt.Fatalf("no baseline count means no claim to compare: %v", err)
			}
		})
	})

	t.Run("without Delete the law stops after the put check", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := &setSUT{}
			l := law.StreamReflectsMutations[*setSUT, int, int]{
				Put: put, Drain: drain, Values: rapid.Just(1), Hash: hash,
			}
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatalf("an absent Delete must end the law, not fail it: %v", err)
			}
		})
	})

	t.Run("a refused delete holds vacuously", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := &setSUT{}
			l := law.StreamReflectsMutations[*setSUT, int, int]{
				Put:    put,
				Delete: func(*rapid.T, *setSUT, int) error { return errors.New("no") },
				Drain:  drain, Values: rapid.Just(1), Hash: hash,
			}
			if err := l.Check(rt, s, s); !law.Holds(err) {
				rt.Fatalf("a refused delete is a precondition: %v", err)
			}
		})
	})

	t.Run("a delete that leaves the value visible is a violation", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := &setSUT{}
			l := law.StreamReflectsMutations[*setSUT, int, int]{
				Put:    put,
				Delete: func(*rapid.T, *setSUT, int) error { return nil }, // no-op delete
				Drain:  drain, Values: rapid.Just(1), Hash: hash,
			}
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a delete that does not remove the value is a violation")
			}
		})
	})

	t.Run("a faithful put and delete pass", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := &setSUT{}
			l := law.StreamReflectsMutations[*setSUT, int, int]{
				Put: put, Delete: del, Drain: drain, Values: rapid.Just(1), Hash: hash,
			}
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatalf("a store that reflects both mutations must pass: %v", err)
			}
		})
	})

	// The drain answered before the delete, so refusing to answer after it is
	// the subject breaking, not a precondition.
	t.Run("a drain that fails after the delete is a violation", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			drains := 0
			s := &setSUT{}
			l := law.StreamReflectsMutations[*setSUT, int, int]{
				Put: put,
				Drain: func(rt *rapid.T, s *setSUT) ([]int, error) {
					drains++
					if drains > 2 {
						return nil, errors.New("stream closed")
					}
					return drain(rt, s)
				},
				Delete: del,
				Values: rapid.Just(1),
				Hash:   hash,
			}
			err := l.Check(rt, s, s)
			if err == nil || !strings.Contains(err.Error(), "drain after delete errored") {
				rt.Fatalf("a stream that stops draining must be reported, got: %v", err)
			}
		})
	})
}
