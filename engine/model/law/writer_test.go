// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
	"html"
	"slices"
	"strconv"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/law"
)

var xssVectors = rapid.SampledFrom([]string{
	"<script>alert(1)</script>",
	"<img src=x onerror=alert(1)>",
	"<svg onload=alert(1)>",
	"<iframe src=javascript:alert(1)>",
})

type wkv struct {
	data map[string]string
}

func (s *wkv) put(_ *rapid.T, v string) error {
	if s.data == nil {
		s.data = make(map[string]string)
	}
	s.data[v] = v
	return nil
}

func (s *wkv) observe(_ *rapid.T) string {
	var b strings.Builder
	for _, v := range s.data {
		b.WriteString(v)
	}
	return b.String()
}

func TestIdempotentWrite(t *testing.T) {
	t.Parallel()

	t.Run("repeating same Write leaves Observe unchanged", func(t *testing.T) {
		t.Parallel()
		s := &wkv{}
		l := law.IdempotentWrite[*wkv, string, string]{
			Write:   func(rt *rapid.T, w *wkv, v string) error { return w.put(rt, v) },
			Values:  rapid.Just("v"),
			Observe: func(rt *rapid.T, w *wkv) string { return w.observe(rt) },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("non-idempotent second write flagged", func(t *testing.T) {
		t.Parallel()
		i := 0
		l := law.IdempotentWrite[*wkv, string, string]{
			Write: func(_ *rapid.T, w *wkv, v string) error {
				if w.data == nil {
					w.data = make(map[string]string)
				}
				i++
				w.data[v] = v + " " + string(rune('A'+(i%26)))
				return nil
			},
			Values:  rapid.Just("v"),
			Observe: func(_ *rapid.T, w *wkv) string { return w.data["v"] },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &wkv{}, &wkv{}); err == nil {
				rt.Fatal("expected idempotence to be flagged")
			}
		})
	})

	t.Run("both writes land on both sides of a real pair", func(t *testing.T) {
		t.Parallel()
		// The reason is [WriteObservable]'s: the runner interleaves laws with
		// differential actions over one shared pair, and a write reaching
		// only the subject is the next action's false divergence.
		l := law.IdempotentWrite[*wkv, string, string]{
			Write:   func(rt *rapid.T, w *wkv, v string) error { return w.put(rt, v) },
			Values:  rapid.Just("v"),
			Observe: func(rt *rapid.T, w *wkv) string { return w.observe(rt) },
		}
		rapid.Check(t, func(rt *rapid.T) {
			sut, ref := &wkv{}, &wkv{}
			if err := l.Check(rt, sut, ref); err != nil {
				rt.Fatal(err)
			}
			if _, ok := ref.data["v"]; !ok {
				rt.Fatal("the reference never saw the write: the pair has diverged")
			}
		})
	})

	t.Run("a subject refusing its repeat is reported", func(t *testing.T) {
		t.Parallel()
		// The first write is the precondition; the second is the claim. A
		// subject that errors on the repeat has not absorbed it.
		rapid.Check(t, func(rt *rapid.T) {
			sut, sutCalls := &wkv{}, 0
			l := law.IdempotentWrite[*wkv, string, string]{
				Write: func(rt *rapid.T, w *wkv, v string) error {
					if w == sut {
						if sutCalls++; sutCalls == 2 {
							return errors.New("refused")
						}
					}
					return w.put(rt, v)
				},
				Values:  rapid.Just("v"),
				Observe: func(rt *rapid.T, w *wkv) string { return w.observe(rt) },
			}
			if err := l.Check(rt, sut, &wkv{}); err == nil {
				rt.Fatal("expected the refused repeat to be reported")
			}
		})
	})

	t.Run("a reference refusing either mirror is reported", func(t *testing.T) {
		t.Parallel()
		// Both refusal arms, because they are different accusations: the
		// first says the reference rejects what the subject accepted, the
		// second that it will not absorb the repeat idempotence is about.
		for refuseAt := 1; refuseAt <= 2; refuseAt++ {
			rapid.Check(t, func(rt *rapid.T) {
				ref, refCalls := &wkv{}, 0
				l := law.IdempotentWrite[*wkv, string, string]{
					Write: func(rt *rapid.T, w *wkv, v string) error {
						if w == ref {
							if refCalls++; refCalls == refuseAt {
								return errors.New("refused")
							}
						}
						return w.put(rt, v)
					},
					Values:  rapid.Just("v"),
					Observe: func(rt *rapid.T, w *wkv) string { return w.observe(rt) },
				}
				if err := l.Check(rt, &wkv{}, ref); err == nil {
					rt.Fatalf("expected the reference's refusal #%d to be reported", refuseAt)
				}
			})
		}
	})
}

func TestAtomicWrite(t *testing.T) {
	t.Parallel()

	t.Run("erroring write that mutates is flagged", func(t *testing.T) {
		t.Parallel()
		l := law.AtomicWrite[*wkv, string, string]{
			Write: func(_ *rapid.T, w *wkv, v string) error {
				if w.data == nil {
					w.data = make(map[string]string)
				}
				w.data[v] = v
				return errors.New("boom")
			},
			Values:  rapid.Just("v"),
			Observe: func(_ *rapid.T, w *wkv) string { return w.data["v"] },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &wkv{}, &wkv{}); err == nil {
				rt.Fatal("expected atomicity to be flagged")
			}
		})
	})

	t.Run("erroring write that does not mutate passes", func(t *testing.T) {
		t.Parallel()
		l := law.AtomicWrite[*wkv, string, string]{
			Write:   func(_ *rapid.T, _ *wkv, _ string) error { return errors.New("boom") },
			Values:  rapid.Just("v"),
			Observe: func(_ *rapid.T, w *wkv) string { return w.observe(nil) },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &wkv{}, &wkv{}); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

func TestCommutativeWrite(t *testing.T) {
	t.Parallel()

	t.Run("a;b == b;a for set-style writes", func(t *testing.T) {
		t.Parallel()
		l := law.CommutativeWrite[*wkv, string, string]{
			Factory: func() *wkv { return &wkv{data: make(map[string]string)} },
			Write:   func(_ *rapid.T, w *wkv, v string) error { w.data[v] = v; return nil },
			Values:  rapid.SampledFrom([]string{"a", "b", "c"}),
			Observe: func(_ *rapid.T, w *wkv) string {
				// map iteration → sort via observe()'s deterministic
				// concat path; commutative writes converge here.
				keys := make([]byte, 0, len(w.data))
				for k := range w.data {
					if len(k) > 0 {
						keys = append(keys, k[0])
					}
				}
				// Sort.
				for i := 1; i < len(keys); i++ {
					for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
						keys[j], keys[j-1] = keys[j-1], keys[j]
					}
				}
				return string(keys)
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &wkv{}, &wkv{}); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

func TestValidTransition(t *testing.T) {
	t.Parallel()

	t.Run("allowed transitions pass", func(t *testing.T) {
		t.Parallel()
		l := law.ValidTransition[*wkv, string, int]{
			Write:   func(_ *rapid.T, w *wkv, _ string) error { w.data = map[string]string{"x": "y"}; return nil },
			Values:  rapid.Just("v"),
			Observe: func(_ *rapid.T, w *wkv) int { return len(w.data) },
			Allowed: func(from, to int) bool { return to >= from },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &wkv{}, &wkv{}); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("disallowed transition flagged", func(t *testing.T) {
		t.Parallel()
		l := law.ValidTransition[*wkv, string, int]{
			Write:   func(_ *rapid.T, w *wkv, _ string) error { w.data = map[string]string{"x": "y"}; return nil },
			Values:  rapid.Just("v"),
			Observe: func(_ *rapid.T, w *wkv) int { return len(w.data) },
			Allowed: func(_, _ int) bool { return false },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &wkv{}, &wkv{}); err == nil {
				rt.Fatal("expected disallowed transition to be flagged")
			}
		})
	})
}

// wstore is a keyed store for the write-observable law.
type wstore struct {
	data map[string]string
	drop bool // when set, writes are silently dropped (the bug)
}

func (s *wstore) write(v string) error {
	if s.data == nil {
		s.data = make(map[string]string)
	}
	if s.drop {
		return nil
	}
	s.data[v] = v
	return nil
}

func (s *wstore) read(k string) (string, error) {
	v, ok := s.data[k]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

// wslot keeps one value and ignores the key it was handed — the subject
// [law.WriteObservable]'s claim is about.
type wslot struct{ held string }

func TestWriteObservable(t *testing.T) {
	t.Parallel()

	t.Run("written value is observable via read", func(t *testing.T) {
		t.Parallel()
		l := law.WriteObservable[*wstore, string, string]{
			Write:  func(_ *rapid.T, s *wstore, v string) error { return s.write(v) },
			Read:   func(_ *rapid.T, s *wstore, k string) (string, error) { return s.read(k) },
			Values: rapid.SampledFrom([]string{"a", "b", "c"}),
			KeyOf:  func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := &wstore{}
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("store that drops writes is caught", func(t *testing.T) {
		t.Parallel()
		l := law.WriteObservable[*wstore, string, string]{
			Write:  func(_ *rapid.T, s *wstore, v string) error { return s.write(v) },
			Read:   func(_ *rapid.T, s *wstore, k string) (string, error) { return s.read(k) },
			Values: rapid.SampledFrom([]string{"a", "b", "c"}),
			KeyOf:  func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := &wstore{drop: true}
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected dropped write to be caught")
			}
		})
	})

	t.Run("the write lands on both sides of a real pair", func(t *testing.T) {
		t.Parallel()
		// The runner interleaves laws with differential actions over one
		// shared (sut, ref) pair, so a write reaching only the subject is
		// the next action's false divergence. Every other subtest here
		// passes the same store twice, which is exactly the arrangement
		// that cannot see this.
		l := law.WriteObservable[*wstore, string, string]{
			Write:  func(_ *rapid.T, s *wstore, v string) error { return s.write(v) },
			Read:   func(_ *rapid.T, s *wstore, k string) (string, error) { return s.read(k) },
			Values: rapid.SampledFrom([]string{"a", "b", "c"}),
			KeyOf:  func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			sut, ref := &wstore{}, &wstore{}
			if err := l.Check(rt, sut, ref); err != nil {
				rt.Fatal(err)
			}
			for k := range sut.data {
				if _, err := ref.read(k); err != nil {
					rt.Fatalf("the reference never saw %q: the pair has diverged", k)
				}
			}
		})
	})

	t.Run("a subject that ignores its key is caught", func(t *testing.T) {
		t.Parallel()
		// The claim names the key, so this is the subject it is really
		// about: one that takes the key, throws it away, and answers
		// whatever it last stored. It passes a check that reads back
		// immediately, because the value it just kept IS the value asked
		// for. Two writes before either read is what separates them.
		l := law.WriteObservable[*wslot, string, string]{
			Write:  func(_ *rapid.T, s *wslot, v string) error { s.held = v; return nil },
			Read:   func(_ *rapid.T, s *wslot, _ string) (string, error) { return s.held, nil },
			Values: rapid.SampledFrom([]string{"a", "b", "c"}),
			KeyOf:  func(v string) string { return v },
		}
		caught := false
		rapid.Check(t, func(rt *rapid.T) {
			s := &wslot{}
			if err := l.Check(rt, s, s); err != nil {
				caught = true
			}
		})
		if !caught {
			t.Fatal("a one-slot store passed a law whose claim names the key")
		}
	})

	t.Run("a projection answering one key checks nothing about keys", func(t *testing.T) {
		t.Parallel()
		// The same subject, under a projection that answers the same key
		// for every value. Both writes go to one key, so the first is gone
		// by the store's own rules and the law asks only about the second
		// — which a one-slot store answers correctly.
		//
		// This is the shape the generator refuses to bind, and this is why:
		// the row would claim the key half of the claim over a binding that
		// structurally cannot reach it.
		l := law.WriteObservable[*wslot, string, string]{
			Write:  func(_ *rapid.T, s *wslot, v string) error { s.held = v; return nil },
			Read:   func(_ *rapid.T, s *wslot, _ string) (string, error) { return s.held, nil },
			Values: rapid.SampledFrom([]string{"a", "b", "c"}),
			KeyOf:  func(string) string { return "the one key" },
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := &wslot{}
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatalf("a collapsed projection has no key claim to fail: %v", err)
			}
		})
	})

	t.Run("a reference refusing the mirrored write is reported", func(t *testing.T) {
		t.Parallel()
		l := law.WriteObservable[*wstore, string, string]{
			Write: func(_ *rapid.T, s *wstore, v string) error {
				if s.drop {
					return errors.New("refused")
				}
				return s.write(v)
			},
			Read:   func(_ *rapid.T, s *wstore, k string) (string, error) { return s.read(k) },
			Values: rapid.SampledFrom([]string{"a", "b", "c"}),
			KeyOf:  func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			// drop doubles as a refusal switch here: the subject accepts,
			// the reference refuses, and the divergence is the law's to
			// report rather than the next action's to misattribute.
			if err := l.Check(rt, &wstore{}, &wstore{drop: true}); err == nil {
				rt.Fatal("expected the refusing reference to be reported")
			}
		})
	})
}

// tamperStore keeps a length checksum over its data. verify
// recomputes and compares — unless blind is set, the bug in which
// integrity is never checked. tamper corrupts data without updating
// the checksum.
type tamperStore struct {
	data  []string
	sum   int
	blind bool
}

func (s *tamperStore) write(v string) error {
	s.data = append(s.data, v)
	s.sum += len(v)
	return nil
}

func (s *tamperStore) verify() error {
	if s.blind {
		return nil
	}
	got := 0
	for _, v := range s.data {
		got += len(v)
	}
	if got != s.sum {
		return errors.New("integrity check failed")
	}
	return nil
}

func (s *tamperStore) tamper() bool {
	if len(s.data) == 0 {
		return false
	}
	s.data[0] += "X" // corrupt content without updating the checksum
	return true
}

func TestTamperEvident(t *testing.T) {
	t.Parallel()

	// The marker is the runner's dispatch, on the value receiver: a registry
	// holding the law by value must still route it to a throwaway pair.
	var iso law.Isolated = law.TamperEvident[*tamperStore, string]{}
	iso.IsolatedLaw()

	t.Run("checksum store detects post-write tampering", func(t *testing.T) {
		t.Parallel()
		l := law.TamperEvident[*tamperStore, string]{
			Write:  func(_ *rapid.T, s *tamperStore, v string) error { return s.write(v) },
			Tamper: func(_ *rapid.T, s *tamperStore) bool { return s.tamper() },
			Verify: func(_ *rapid.T, s *tamperStore) error { return s.verify() },
			Values: rapid.SampledFrom([]string{"a", "bb", "ccc"}),
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := &tamperStore{}
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("store with no integrity check is caught", func(t *testing.T) {
		t.Parallel()
		l := law.TamperEvident[*tamperStore, string]{
			Write:  func(_ *rapid.T, s *tamperStore, v string) error { return s.write(v) },
			Tamper: func(_ *rapid.T, s *tamperStore) bool { return s.tamper() },
			Verify: func(_ *rapid.T, s *tamperStore) error { return s.verify() },
			Values: rapid.SampledFrom([]string{"a", "bb"}),
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := &tamperStore{blind: true}
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected undetected tampering to be caught")
			}
		})
	})
}

func TestXSSSafe(t *testing.T) {
	t.Parallel()

	t.Run("HTML-escaping renderer neutralizes XSS vectors", func(t *testing.T) {
		t.Parallel()
		l := law.XSSSafe[struct{}]{
			Render:   func(_ *rapid.T, _ struct{}, raw string) (string, error) { return html.EscapeString(raw), nil },
			Payloads: xssVectors,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("renderer that passes markup through verbatim is caught", func(t *testing.T) {
		t.Parallel()
		l := law.XSSSafe[struct{}]{
			Render:   func(_ *rapid.T, _ struct{}, raw string) (string, error) { return raw, nil }, // no escaping
			Payloads: xssVectors,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err == nil {
				rt.Fatal("expected unescaped markup to be caught")
			}
		})
	})
}

var injectionVectors = rapid.SampledFrom([]string{
	"' OR '1'='1",
	"'; DROP TABLE users; --",
	"admin'--",
	"$(rm -rf /)",
	"`whoami`",
})

// injStore stores values under keys. When vulnerable, a value
// containing shell/SQL metacharacters corrupts the canary — the
// injection "breaks out" of its parameter.
type injStore struct {
	data       map[string]string
	vulnerable bool
}

func (s *injStore) store(k, v string) error {
	if s.data == nil {
		s.data = make(map[string]string)
	}
	if s.vulnerable && strings.ContainsAny(v, "'\"`;$") {
		s.data["canary"] = "HACKED"
	}
	s.data[k] = v
	return nil
}

func (s *injStore) load(k string) (string, error) {
	v, ok := s.data[k]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func TestInjectionSafe(t *testing.T) {
	t.Parallel()

	t.Run("parameterized store keeps payloads as literal data", func(t *testing.T) {
		t.Parallel()
		l := law.InjectionSafe[*injStore]{
			Store:       func(_ *rapid.T, s *injStore, k, v string) error { return s.store(k, v) },
			Load:        func(_ *rapid.T, s *injStore, k string) (string, error) { return s.load(k) },
			Payloads:    injectionVectors,
			CanaryKey:   "canary",
			CanaryValue: "safe",
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := &injStore{}
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("store where injection corrupts other data is caught", func(t *testing.T) {
		t.Parallel()
		l := law.InjectionSafe[*injStore]{
			Store:       func(_ *rapid.T, s *injStore, k, v string) error { return s.store(k, v) },
			Load:        func(_ *rapid.T, s *injStore, k string) (string, error) { return s.load(k) },
			Payloads:    injectionVectors,
			CanaryKey:   "canary",
			CanaryValue: "safe",
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := &injStore{vulnerable: true}
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected injection breakout to be caught")
			}
		})
	})
}

// kvStore is a minimal string→string store for driving the writer laws'
// precondition and violation branches.
type kvStore struct {
	data     map[string]string
	writeErr error
	readErr  error
	tampered bool
}

func newKVStore() *kvStore { return &kvStore{data: map[string]string{}} }

func (s *kvStore) put(k, v string) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	s.data[k] = v
	return nil
}

func (s *kvStore) get(k string) (string, error) {
	if s.readErr != nil {
		return "", s.readErr
	}
	v, ok := s.data[k]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func TestWriteObservablePreconditionsAndViolations(t *testing.T) {
	t.Parallel()

	mk := func(s *kvStore, read func(*rapid.T, *kvStore, string) (string, error)) law.WriteObservable[*kvStore, string, string] {
		return law.WriteObservable[*kvStore, string, string]{
			Write:  func(_ *rapid.T, st *kvStore, v string) error { return st.put(v[:1], v) },
			Read:   read,
			Values: rapid.Just("ab"),
			KeyOf:  func(v string) string { return v[:1] },
		}
	}
	okRead := func(_ *rapid.T, st *kvStore, k string) (string, error) { return st.get(k) }

	t.Run("a refused write holds vacuously", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		s.writeErr = errors.New("read-only")
		l := mk(s, okRead)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); !law.Holds(err) {
				rt.Fatalf("a refused write is a precondition, not a violation: %v", err)
			}
		})
	})

	t.Run("a written value that cannot be read is a violation", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		l := mk(s, func(*rapid.T, *kvStore, string) (string, error) {
			return "", errors.New("gone")
		})
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("an unreadable write is a violation")
			}
		})
	})

	t.Run("a read returning a different value is a violation", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		l := mk(s, func(*rapid.T, *kvStore, string) (string, error) { return "other", nil })
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a mismatched read is a violation")
			}
		})
	})
}

func TestTamperEvidentPreconditionsAndViolations(t *testing.T) {
	t.Parallel()

	t.Run("a refused write holds vacuously", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		s.writeErr = errors.New("read-only")
		l := law.TamperEvident[*kvStore, string]{
			Write:  func(_ *rapid.T, st *kvStore, v string) error { return st.put(v, v) },
			Tamper: func(*rapid.T, *kvStore) bool { return true },
			Verify: func(*rapid.T, *kvStore) error { return nil },
			Values: rapid.Just("v"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); !law.Holds(err) {
				rt.Fatalf("a refused write is a precondition, not a violation: %v", err)
			}
		})
	})

	t.Run("Verify failing on intact data is a violation", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		l := law.TamperEvident[*kvStore, string]{
			Write:  func(_ *rapid.T, st *kvStore, v string) error { return st.put(v, v) },
			Tamper: func(*rapid.T, *kvStore) bool { return true },
			Verify: func(*rapid.T, *kvStore) error { return errors.New("corrupt") },
			Values: rapid.Just("v"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("Verify failing before any tampering is a violation")
			}
		})
	})

	// Tamper returning false means the subject cannot be corrupted in a
	// meaningful way — an empty store, say — so there is nothing to detect and
	// the law must skip rather than fail.
	t.Run("inapplicable tampering holds vacuously", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		l := law.TamperEvident[*kvStore, string]{
			Write:  func(_ *rapid.T, st *kvStore, v string) error { return st.put(v, v) },
			Tamper: func(*rapid.T, *kvStore) bool { return false },
			Verify: func(*rapid.T, *kvStore) error { return nil },
			Values: rapid.Just("v"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); !errors.Is(err, law.Vacuous) {
				rt.Fatalf("a tamper that refused leaves nothing to detect: %v", err)
			}
		})
	})

	t.Run("a store with no integrity mechanism is flagged", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		l := law.TamperEvident[*kvStore, string]{
			Write:  func(_ *rapid.T, st *kvStore, v string) error { return st.put(v, v) },
			Tamper: func(_ *rapid.T, st *kvStore) bool { st.tampered = true; return true },
			Verify: func(*rapid.T, *kvStore) error { return nil }, // never detects
			Values: rapid.Just("v"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a store that verifies clean after tampering is a violation")
			}
		})
	})

	t.Run("a store that detects tampering passes", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		l := law.TamperEvident[*kvStore, string]{
			Write:  func(_ *rapid.T, st *kvStore, v string) error { return st.put(v, v) },
			Tamper: func(_ *rapid.T, st *kvStore) bool { st.tampered = true; return true },
			Verify: func(_ *rapid.T, st *kvStore) error {
				if st.tampered {
					return errors.New("integrity violation")
				}
				return nil
			},
			Values: rapid.Just("v"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			s.tampered = false
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatalf("a store that detects tampering must pass: %v", err)
			}
		})
	})
}

// InjectionSafe stores a canary alongside a hostile payload and checks the
// payload came back byte-identical and left the canary intact. Both a mangled
// payload and a damaged canary are violations; a store that refuses either
// write has simply failed a precondition.
func TestInjectionSafePreconditionsAndViolations(t *testing.T) {
	t.Parallel()

	mk := func(
		store func(*rapid.T, *kvStore, string, string) error,
		load func(*rapid.T, *kvStore, string) (string, error),
	) law.InjectionSafe[*kvStore] {
		return law.InjectionSafe[*kvStore]{
			Store: store, Load: load,
			CanaryKey:   "canary",
			CanaryValue: "intact",
			Payloads:    rapid.Just("'; DROP TABLE users; --"),
		}
	}
	okStore := func(_ *rapid.T, s *kvStore, k, v string) error { return s.put(k, v) }
	okLoad := func(_ *rapid.T, s *kvStore, k string) (string, error) { return s.get(k) }

	t.Run("a refused write holds vacuously", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		s.writeErr = errors.New("read-only")
		l := mk(okStore, okLoad)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); !law.Holds(err) {
				rt.Fatalf("a refused write is a precondition, not a violation: %v", err)
			}
		})
	})

	t.Run("an unretrievable payload is a violation", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		l := mk(okStore, func(_ *rapid.T, st *kvStore, k string) (string, error) {
			if k == "canary" {
				return st.get(k)
			}
			return "", errors.New("gone")
		})
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a payload that cannot be read back is a violation")
			}
		})
	})

	t.Run("a payload altered in storage is a violation", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		l := mk(func(_ *rapid.T, st *kvStore, k, v string) error {
			return st.put(k, strings.ReplaceAll(v, "'", "")) // naive sanitiser
		}, okLoad)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("silently rewriting the stored payload is a violation")
			}
		})
	})

	t.Run("a lost canary is a violation", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		l := mk(okStore, func(_ *rapid.T, st *kvStore, k string) (string, error) {
			if k == "canary" {
				return "", errors.New("canary gone")
			}
			return st.get(k)
		})
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a canary lost after storing the payload is a violation")
			}
		})
	})

	t.Run("a corrupted canary is a violation", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		l := mk(okStore, func(_ *rapid.T, st *kvStore, k string) (string, error) {
			if k == "canary" {
				return "clobbered", nil
			}
			return st.get(k)
		})
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a payload that changes the canary's value is a violation")
			}
		})
	})

	t.Run("a faithful store passes", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		l := mk(okStore, okLoad)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatalf("a store that round-trips payloads must pass: %v", err)
			}
		})
	})
}

// AtomicWrite only has something to say about failed writes: a successful
// write is expected to mutate state, and its correctness is another law's
// business. The interesting case is an error that left a partial mutation
// behind.
func TestAtomicWriteBranches(t *testing.T) {
	t.Parallel()

	observe := func(_ *rapid.T, s *kvStore) int { return len(s.data) }

	t.Run("a successful write is not the law's concern", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		l := law.AtomicWrite[*kvStore, string, int]{
			Write:   func(_ *rapid.T, st *kvStore, v string) error { return st.put(v, v) },
			Values:  rapid.Just("v"),
			Observe: observe,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatalf("a successful write must be skipped, not judged: %v", err)
			}
		})
	})

	t.Run("an errored write that changed nothing passes", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		l := law.AtomicWrite[*kvStore, string, int]{
			Write:   func(*rapid.T, *kvStore, string) error { return errors.New("rejected") },
			Values:  rapid.Just("v"),
			Observe: observe,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatalf("a clean rejection must pass: %v", err)
			}
		})
	})

	t.Run("an errored write that mutated state is a violation", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		n := 0
		l := law.AtomicWrite[*kvStore, string, int]{
			Write: func(_ *rapid.T, st *kvStore, v string) error {
				// A unique key every call, so the partial mutation is always
				// observable — a wrapping key would stop growing the store and
				// the law would rightly see no change.
				n++
				st.data[v+strconv.Itoa(n)] = v // partial mutation, then fail
				return errors.New("rejected after writing")
			},
			Values:  rapid.Just("v"),
			Observe: observe,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a failed write that left a mutation behind is a violation")
			}
		})
	})
}

func TestCommutativeWriteBranches(t *testing.T) {
	t.Parallel()

	observe := func(_ *rapid.T, s *kvStore) string { return s.data["log"] }
	vals := rapid.SampledFrom([]string{"a", "b"})

	t.Run("order-independent writes pass", func(t *testing.T) {
		t.Parallel()
		l := law.CommutativeWrite[*kvStore, string, string]{
			Factory: newKVStore,
			Write: func(_ *rapid.T, st *kvStore, v string) error {
				st.data["log"] += v
				// Sort so the observation cannot depend on arrival order.
				b := []byte(st.data["log"])
				slices.Sort(b)
				st.data["log"] = string(b)
				return nil
			},
			Values:  vals,
			Observe: observe,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err != nil {
				rt.Fatalf("commutative writes must pass: %v", err)
			}
		})
	})

	t.Run("order-dependent writes are flagged", func(t *testing.T) {
		t.Parallel()
		l := law.CommutativeWrite[*kvStore, string, string]{
			Factory: newKVStore,
			Write: func(_ *rapid.T, st *kvStore, v string) error {
				st.data["log"] += v // append order is observable
				return nil
			},
			Values:  vals,
			Observe: observe,
		}
		// rapid draws both values from the same generator, so it will
		// sometimes draw a pair where a == b — and there, order genuinely
		// does not matter and the law is right to pass. The assertion is
		// therefore that a distinct pair is caught at least once, not that
		// every draw fails.
		flagged := false
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err != nil {
				flagged = true
			}
		})
		if !flagged {
			t.Fatal("append-order-sensitive writes must be flagged for a distinct pair")
		}
	})

	t.Run("a refused write holds vacuously", func(t *testing.T) {
		t.Parallel()
		for _, nth := range []int{1, 2, 3, 4} {
			rapid.Check(t, func(rt *rapid.T) {
				fail := failOnNth(nth)
				l := law.CommutativeWrite[*kvStore, string, string]{
					Factory: newKVStore,
					Write: func(_ *rapid.T, st *kvStore, v string) error {
						if err := fail(); err != nil {
							return err
						}
						st.data["log"] += v
						return nil
					},
					Values:  rapid.Just("a"),
					Observe: observe,
				}
				if err := l.Check(rt, nil, nil); !law.Holds(err) {
					rt.Fatalf("a refused write is a precondition, not a violation: %v", err)
				}
			})
		}
	})
}

func TestXSSSafeBranches(t *testing.T) {
	t.Parallel()

	t.Run("a refused render holds vacuously", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		l := law.XSSSafe[*kvStore]{
			Render:   func(*rapid.T, *kvStore, string) (string, error) { return "", errors.New("no") },
			Payloads: xssVectors,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); !law.Holds(err) {
				rt.Fatalf("a refused render is a precondition, not a violation: %v", err)
			}
		})
	})

	// An explicit Dangerous list replaces the built-in token set, so a
	// renderer that escapes the defaults still fails against a custom token.
	t.Run("a custom Dangerous list overrides the defaults", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		l := law.XSSSafe[*kvStore]{
			Render: func(_ *rapid.T, _ *kvStore, raw string) (string, error) {
				return html.EscapeString(raw) + "<custom>", nil
			},
			Payloads:  xssVectors,
			Dangerous: []string{"<custom>"},
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a token from the custom list must be flagged")
			}
		})
	})
}

// Every law splits its inputs into preconditions — a refused operation says
// nothing about the property — and violations. These cover the arms the
// happy-path tests never reach.
func TestWriterLawPreconditionsAndViolations(t *testing.T) {
	t.Parallel()

	boom := errors.New("refused")

	t.Run("IdempotentWrite holds vacuously when the first write is refused", func(t *testing.T) {
		t.Parallel()
		l := law.IdempotentWrite[*wkv, string, string]{
			Write:   func(*rapid.T, *wkv, string) error { return boom },
			Values:  rapid.Just("v"),
			Observe: func(rt *rapid.T, w *wkv) string { return w.observe(rt) },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &wkv{}, &wkv{}); !law.Holds(err) {
				rt.Fatalf("a refused first write is a precondition: %v", err)
			}
		})
	})

	// The second write is different: the first one succeeded, so refusing the
	// repeat is the subject contradicting itself, not declining the input.
	t.Run("IdempotentWrite flags a refused repeat", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			calls := 0
			l := law.IdempotentWrite[*wkv, string, string]{
				Write: func(rt *rapid.T, w *wkv, v string) error {
					calls++
					if calls > 1 {
						return boom
					}
					return w.put(rt, v)
				},
				Values:  rapid.Just("v"),
				Observe: func(rt *rapid.T, w *wkv) string { return w.observe(rt) },
			}
			if err := l.Check(rt, &wkv{}, &wkv{}); err == nil {
				rt.Fatal("repeating an accepted write must not error")
			}
		})
	})

	t.Run("InjectionSafe holds vacuously when the payload is refused", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			store := map[string]string{}
			l := law.InjectionSafe[*wkv]{
				Store: func(_ *rapid.T, _ *wkv, key, value string) error {
					if key != "canary" {
						return boom
					}
					store[key] = value
					return nil
				},
				Load:        func(_ *rapid.T, _ *wkv, key string) (string, error) { return store[key], nil },
				Payloads:    xssVectors,
				CanaryKey:   "canary",
				CanaryValue: "intact",
			}
			if err := l.Check(rt, &wkv{}, &wkv{}); !law.Holds(err) {
				rt.Fatalf("a store that refuses the payload is a precondition: %v", err)
			}
		})
	})

	t.Run("ValidTransition holds vacuously when the write is refused", func(t *testing.T) {
		t.Parallel()
		l := law.ValidTransition[*wkv, string, string]{
			Write:   func(*rapid.T, *wkv, string) error { return boom },
			Values:  rapid.Just("v"),
			Observe: func(rt *rapid.T, w *wkv) string { return w.observe(rt) },
			Allowed: func(string, string) bool { return false },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &wkv{}, &wkv{}); !law.Holds(err) {
				rt.Fatalf("a refused write is a precondition: %v", err)
			}
		})
	})

	// A write that changes nothing is not a transition, so the predicate is
	// never consulted — even one that rejects everything.
	t.Run("ValidTransition ignores a write that does not move the state", func(t *testing.T) {
		t.Parallel()
		l := law.ValidTransition[*wkv, string, string]{
			Write:   func(*rapid.T, *wkv, string) error { return nil },
			Values:  rapid.Just("v"),
			Observe: func(*rapid.T, *wkv) string { return "steady" },
			Allowed: func(string, string) bool { return false },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &wkv{}, &wkv{}); err != nil {
				rt.Fatalf("an unchanged state is not a transition: %v", err)
			}
		})
	})
}

// The pair tests below hold the three mirrored writer laws to the conduct
// contract on [law.Law]: every mutation the subject accepts lands on the
// reference, and a refusal is the law's to report. Each law's other subtests
// pass one store as both sides, which is exactly the arrangement that cannot
// see a desynchronized pair.

func TestAtomicWritePair(t *testing.T) {
	t.Parallel()

	t.Run("the accepted write lands on both sides", func(t *testing.T) {
		t.Parallel()
		l := law.AtomicWrite[*wkv, string, string]{
			Write:   func(rt *rapid.T, w *wkv, v string) error { return w.put(rt, v) },
			Values:  rapid.Just("v"),
			Observe: func(rt *rapid.T, w *wkv) string { return w.observe(rt) },
		}
		rapid.Check(t, func(rt *rapid.T) {
			sut, ref := &wkv{}, &wkv{}
			if err := l.Check(rt, sut, ref); err != nil {
				rt.Fatal(err)
			}
			if _, ok := ref.data["v"]; !ok {
				rt.Fatal("the reference never saw the write: the pair has diverged")
			}
		})
	})

	t.Run("a refusing reference is reported", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			ref := &wkv{}
			l := law.AtomicWrite[*wkv, string, string]{
				Write: func(rt *rapid.T, w *wkv, v string) error {
					if w == ref {
						return errors.New("refused")
					}
					return w.put(rt, v)
				},
				Values:  rapid.Just("v"),
				Observe: func(rt *rapid.T, w *wkv) string { return w.observe(rt) },
			}
			if err := l.Check(rt, &wkv{}, ref); err == nil {
				rt.Fatal("expected the refusal to be reported")
			}
		})
	})
}

func TestValidTransitionPair(t *testing.T) {
	t.Parallel()

	t.Run("the accepted write lands on both sides", func(t *testing.T) {
		t.Parallel()
		l := law.ValidTransition[*wkv, string, int]{
			Write:   func(rt *rapid.T, w *wkv, v string) error { return w.put(rt, v) },
			Values:  rapid.Just("v"),
			Observe: func(_ *rapid.T, w *wkv) int { return len(w.data) },
			Allowed: func(from, to int) bool { return to >= from },
		}
		rapid.Check(t, func(rt *rapid.T) {
			sut, ref := &wkv{}, &wkv{}
			if err := l.Check(rt, sut, ref); err != nil {
				rt.Fatal(err)
			}
			if len(ref.data) == 0 {
				rt.Fatal("the reference never saw the write: the pair has diverged")
			}
		})
	})

	t.Run("a refusing reference is reported", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			ref := &wkv{}
			l := law.ValidTransition[*wkv, string, int]{
				Write: func(rt *rapid.T, w *wkv, v string) error {
					if w == ref {
						return errors.New("refused")
					}
					return w.put(rt, v)
				},
				Values:  rapid.Just("v"),
				Observe: func(_ *rapid.T, w *wkv) int { return len(w.data) },
				Allowed: func(from, to int) bool { return to >= from },
			}
			if err := l.Check(rt, &wkv{}, ref); err == nil {
				rt.Fatal("expected the refusal to be reported")
			}
		})
	})
}

func TestInjectionSafePair(t *testing.T) {
	t.Parallel()

	t.Run("both stores land on both sides", func(t *testing.T) {
		t.Parallel()
		l := law.InjectionSafe[*injStore]{
			Store:       func(_ *rapid.T, s *injStore, k, v string) error { return s.store(k, v) },
			Load:        func(_ *rapid.T, s *injStore, k string) (string, error) { return s.load(k) },
			Payloads:    injectionVectors,
			CanaryKey:   "canary",
			CanaryValue: "safe",
		}
		rapid.Check(t, func(rt *rapid.T) {
			sut, ref := &injStore{}, &injStore{}
			if err := l.Check(rt, sut, ref); err != nil {
				rt.Fatal(err)
			}
			if _, err := ref.load("canary"); err != nil {
				rt.Fatal("the reference never saw the canary: the pair has diverged")
			}
			if _, err := ref.load("injectionsafe_target"); err != nil {
				rt.Fatal("the reference never saw the payload: the pair has diverged")
			}
		})
	})

	t.Run("a reference refusing either mirror is reported", func(t *testing.T) {
		t.Parallel()
		// Both arms: the canary mirror and the payload mirror are separate
		// stores, and each refusal must surface as the law's own report.
		for refuseAt := 1; refuseAt <= 2; refuseAt++ {
			rapid.Check(t, func(rt *rapid.T) {
				ref, refCalls := &injStore{}, 0
				l := law.InjectionSafe[*injStore]{
					Store: func(_ *rapid.T, s *injStore, k, v string) error {
						if s == ref {
							if refCalls++; refCalls == refuseAt {
								return errors.New("refused")
							}
						}
						return s.store(k, v)
					},
					Load:        func(_ *rapid.T, s *injStore, k string) (string, error) { return s.load(k) },
					Payloads:    injectionVectors,
					CanaryKey:   "canary",
					CanaryValue: "safe",
				}
				if err := l.Check(rt, &injStore{}, ref); err == nil {
					rt.Fatalf("expected the reference's refusal #%d to be reported", refuseAt)
				}
			})
		}
	})
}
