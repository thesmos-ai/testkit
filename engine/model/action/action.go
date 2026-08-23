// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package action provides shape-typed Action helpers for model-based
// testing. Each helper eliminates the per-method boilerplate of
// drawing a sample, calling both SUT and reference, and comparing
// results. The generator emits one call per detected method.
//
//nolint:errorlint // Action errors are diagnostic (SUT vs ref comparison), not wrapped.
package action

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/model/history"
)

// Reader creates an action for a Reader-shaped method: func(ctx, K) (V, error).
// Draws a key from the provided generator, calls both SUT and ref, and
// compares results — presence always, identity where [WithSentinel] arms it.
func Reader[T any, K comparable, V any](
	name string,
	keys *rapid.Generator[K],
	read func(context.Context, T, K) (V, error),
	options ...Opt,
) model.Action[T] {
	o := optsOf(options)
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			k := keys.Draw(rt, name+"_key")
			sutGot, sutErr := read(rt.Context(), sut, k)
			refGot, refErr := read(rt.Context(), ref, k)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:    fmt.Errorf("%s(%v): SUT err=%v, ref err=%v", name, k, sutErr, refErr),
					Input:  k,
					Output: sutGot,
				}
			}
			if sutErr != nil {
				if err := o.identity(name, k, sutErr, refErr); err != nil {
					return model.ActionResult{Err: err, Input: k, Output: sutGot}
				}
			}
			if sutErr == nil {
				if diff := cmp.Diff(refGot, sutGot); diff != "" {
					return model.ActionResult{
						Err:    fmt.Errorf("%s(%v): SUT/ref disagree:\n%s", name, k, diff),
						Input:  k,
						Output: sutGot,
					}
				}
			}
			return model.ActionResult{Input: k, Output: sutGot, CallErr: sutErr}
		},
	}
}

// ReaderWithBool creates an action for a ReaderWithBool-shaped method:
// func(ctx, K) (V, bool) or func(K) (V, bool). Draws a key, calls both
// SUT and ref, compares value and ok flag.
func ReaderWithBool[T any, K comparable, V any](
	name string,
	keys *rapid.Generator[K],
	read func(context.Context, T, K) (V, bool),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			k := keys.Draw(rt, name+"_key")
			sutGot, sutOK := read(rt.Context(), sut, k)
			refGot, refOK := read(rt.Context(), ref, k)
			out := ReaderWithBoolOutput{V: sutGot, OK: sutOK}
			if sutOK != refOK {
				return model.ActionResult{
					Err:    fmt.Errorf("%s(%v): SUT ok=%v, ref ok=%v", name, k, sutOK, refOK),
					Input:  k,
					Output: out,
				}
			}
			if sutOK {
				if diff := cmp.Diff(refGot, sutGot); diff != "" {
					return model.ActionResult{
						Err:    fmt.Errorf("%s(%v): SUT/ref disagree:\n%s", name, k, diff),
						Input:  k,
						Output: out,
					}
				}
			}
			return model.ActionResult{Input: k, Output: out}
		},
	}
}

// Lookup creates an action for a Lookup-shaped method:
// func(T, K) (R1, R2, bool). Draws a key, calls both SUT and ref,
// compares ok flag and R1 when present. R2 is compared via
// cmpOpts if provided (needed for uncomparable types like functions);
// otherwise skipped.
func Lookup[T any, K comparable, R1, R2 any](
	name string,
	keys *rapid.Generator[K],
	lookup func(T, K) (R1, R2, bool),
	cmpOpts ...cmp.Option,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			k := keys.Draw(rt, name+"_key")
			sutR1, sutR2, sutOK := lookup(sut, k)
			refR1, refR2, refOK := lookup(ref, k)
			out := LookupOutput{R1: sutR1, R2: sutR2, OK: sutOK}
			if sutOK != refOK {
				return model.ActionResult{
					Err:    fmt.Errorf("%s(%v): SUT ok=%v, ref ok=%v", name, k, sutOK, refOK),
					Input:  k,
					Output: out,
				}
			}
			if sutOK {
				if diff := cmp.Diff(refR1, sutR1); diff != "" {
					return model.ActionResult{
						Err:    fmt.Errorf("%s(%v) R1: SUT/ref disagree:\n%s", name, k, diff),
						Input:  k,
						Output: out,
					}
				}
				if len(cmpOpts) > 0 {
					if diff := cmp.Diff(refR2, sutR2, cmpOpts...); diff != "" {
						return model.ActionResult{
							Err:    fmt.Errorf("%s(%v) R2: SUT/ref disagree:\n%s", name, k, diff),
							Input:  k,
							Output: out,
						}
					}
				}
			}
			return model.ActionResult{Input: k, Output: out}
		},
	}
}

// Mutator creates an action for a Mutator-shaped method: func(ctx, V)
// with no return. Calls both SUT and ref with the same drawn value.
// Divergence is detected by laws (not return-value comparison).
func Mutator[T, V any](
	name string,
	values *rapid.Generator[V],
	mutate func(context.Context, T, V),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			v := values.Draw(rt, name+"_value")
			mutate(rt.Context(), sut, v)
			mutate(rt.Context(), ref, v)
			return model.ActionResult{Input: v}
		},
	}
}

// PoisonCheck creates an action for a PoisonAccessor-shaped method:
// func() error. Calls both SUT and ref, compares error states.
func PoisonCheck[T any](
	name string,
	check func(T) error,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			sutErr := check(sut)
			refErr := check(ref)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:    fmt.Errorf("%s: SUT err=%v, ref err=%v", name, sutErr, refErr),
					Output: sutErr,
				}
			}
			return model.ActionResult{Output: sutErr}
		},
	}
}

// Writer creates an action for a Writer-shaped method: func(ctx, V) error.
// Draws a value from the provided generator, calls both SUT and ref.
func Writer[T, V any](
	name string,
	values *rapid.Generator[V],
	write func(context.Context, T, V) error,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			v := values.Draw(rt, name+"_value")
			sutErr := write(rt.Context(), sut, v)
			refErr := write(rt.Context(), ref, v)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:   fmt.Errorf("%s(%v): SUT err=%v, ref err=%v", name, v, sutErr, refErr),
					Input: v,
				}
			}
			return model.ActionResult{Input: v, CallErr: sutErr}
		},
	}
}

// WriterRecording is [Writer] with the run's record of what went in: a
// write both sides accepted is logged once, into the [history.History] a
// drain claim is judged against.
//
// Once, not once per side. That is what a closure inside the write cannot
// do — it is handed the subject, then handed the reference, and a log
// filled from inside it says twice as much went in as did. A membership
// check is indifferent to the difference; a claim about HOW MANY elements
// a drain owes reads it as every element having been dropped.
//
// Unpartitioned, under the empty key: a collection mixin declares no
// partitioning, and the history's key exists for chains that do.
func WriterRecording[T, V any](
	name string,
	values *rapid.Generator[V],
	hist *history.History[string, V],
	write func(context.Context, T, V) error,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			v := values.Draw(rt, name+"_value")
			sutErr := write(rt.Context(), sut, v)
			refErr := write(rt.Context(), ref, v)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:   fmt.Errorf("%s(%v): SUT err=%v, ref err=%v", name, v, sutErr, refErr),
					Input: v,
				}
			}
			if sutErr == nil {
				hist.Record("", v)
			}
			return model.ActionResult{Input: v, CallErr: sutErr}
		},
	}
}

// Deleter creates an action for a Deleter-shaped method: func(ctx, K) error.
// Draws a key from the provided generator, calls both SUT and ref.
func Deleter[T any, K comparable](
	name string,
	keys *rapid.Generator[K],
	del func(context.Context, T, K) error,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			k := keys.Draw(rt, name+"_key")
			sutErr := del(rt.Context(), sut, k)
			refErr := del(rt.Context(), ref, k)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:   fmt.Errorf("%s(%v): SUT err=%v, ref err=%v", name, k, sutErr, refErr),
					Input: k,
				}
			}
			return model.ActionResult{Input: k, CallErr: sutErr}
		},
	}
}

// Aggregator creates an action for an Aggregator-shaped method: func(ctx) (R, error).
// Calls both SUT and ref, compares results.
func Aggregator[T any, R comparable](
	name string,
	agg func(context.Context, T) (R, error),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			sutGot, sutErr := agg(rt.Context(), sut)
			refGot, refErr := agg(rt.Context(), ref)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:    fmt.Errorf("%s: SUT err=%v, ref err=%v", name, sutErr, refErr),
					Output: sutGot,
				}
			}
			if sutErr == nil && sutGot != refGot {
				return model.ActionResult{
					Err:    fmt.Errorf("%s: SUT=%v, ref=%v", name, sutGot, refGot),
					Output: sutGot,
				}
			}
			return model.ActionResult{Output: sutGot}
		},
	}
}

// Lifecycle creates an action for a Lifecycle-shaped method: func(ctx) error.
// Calls both SUT and ref, compares error outcomes.
func Lifecycle[T any](
	name string,
	call func(context.Context, T) error,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			sutErr := call(rt.Context(), sut)
			refErr := call(rt.Context(), ref)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err: fmt.Errorf("%s: SUT err=%v, ref err=%v", name, sutErr, refErr),
				}
			}
			return model.ActionResult{}
		},
	}
}

// Pure creates an action for a Pure-shaped method: func(T) R.
// Calls both SUT and ref, compares results. No context.
func Pure[T, R any](
	name string,
	call func(T) R,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			sutGot := call(sut)
			refGot := call(ref)
			if diff := cmp.Diff(refGot, sutGot); diff != "" {
				return model.ActionResult{
					Err:    fmt.Errorf("%s: SUT/ref disagree:\n%s", name, diff),
					Output: sutGot,
				}
			}
			return model.ActionResult{Output: sutGot}
		},
	}
}

// Predicate creates an action for a Predicate-shaped method: func(T) bool.
// Calls both SUT and ref, compares results. No context.
func Predicate[T any](
	name string,
	call func(T) bool,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			sutGot := call(sut)
			refGot := call(ref)
			if sutGot != refGot {
				return model.ActionResult{
					Err:    fmt.Errorf("%s: SUT=%v, ref=%v", name, sutGot, refGot),
					Output: sutGot,
				}
			}
			return model.ActionResult{Output: sutGot}
		},
	}
}

// Stream creates an action for a StreamReader-shaped method that
// returns all items. Calls both SUT and ref, collects results,
// compares. Order-insensitive: results are sorted by string
// representation before comparison since map-backed stores produce
// non-deterministic iteration order.
func Stream[T, V any](
	name string,
	collect func(context.Context, T) ([]V, error),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			sutItems, sutErr := collect(rt.Context(), sut)
			refItems, refErr := collect(rt.Context(), ref)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:    fmt.Errorf("%s: SUT err=%v, ref err=%v", name, sutErr, refErr),
					Output: sutItems,
				}
			}
			if sutErr == nil {
				sortByString(sutItems)
				sortByString(refItems)
				// EquateEmpty: a nil drain and an empty one are the same
				// answer to "what do you hold". Which of the two a subject
				// spells is an allocation detail no stream claim is about,
				// and without this the first empty drain of any run fails
				// on it.
				if diff := cmp.Diff(refItems, sutItems, cmpopts.EquateEmpty()); diff != "" {
					return model.ActionResult{
						Err:    fmt.Errorf("%s: SUT/ref disagree:\n%s", name, diff),
						Output: sutItems,
					}
				}
			}
			return model.ActionResult{Output: sutItems}
		},
	}
}

// Stress creates an action that calls the SUT without comparing
// against the reference. Used for concurrent StressActions where
// only race detection matters.
func Stress[T any](
	name string,
	call func(T),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(_ *rapid.T, sut, _ T) model.ActionResult {
			call(sut)
			return model.ActionResult{}
		},
	}
}

// Unknown creates an action for an Unknown-shaped method.
// Consumer provides the full comparison logic.
func Unknown[T any](
	name string,
	run func(rt *rapid.T, sut, ref T) model.ActionResult,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Run:  run,
	}
}

// EvictingReader creates an action for a bounded reader whose misses are
// LEGAL: `func(ctx, K) (V, bool)` on a subject that may evict. The
// comparison is ASYMMETRIC — a subject hit must agree with the
// reference's value, and a hit the reference cannot explain is
// invention; a subject miss is never a divergence, because eviction is
// the contract. Pair it with an UNBOUNDED reference, and keep any count
// observation out of the differential: the unbounded reference legally
// disagrees there, and the bounded law owns the count.
//
// The symmetric [ReaderWithBool] is for readers whose presence IS the
// claim; this one is for readers whose absence is the policy's business.
func EvictingReader[T any, K, V comparable](
	name string,
	keys *rapid.Generator[K],
	read func(context.Context, T, K) (V, bool),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			k := keys.Draw(rt, name+"_key")
			sv, sok := read(rt.Context(), sut, k)
			rv, rok := read(rt.Context(), ref, k)
			switch {
			case sok && !rok:
				return model.ActionResult{
					Err:   fmt.Errorf("%s(%v): subject invented a hit (%v) the reference cannot explain", name, k, sv),
					Input: k,
				}
			case sok && sv != rv:
				return model.ActionResult{
					Err:   fmt.Errorf("%s(%v): hit disagrees: subject %v, reference %v", name, k, sv, rv),
					Input: k,
				}
			default:
				// A subject miss is legal whatever the reference holds.
				return model.ActionResult{Input: k}
			}
		},
	}
}

// sortByString sorts a slice by the Sprint representation of each element.
func sortByString[V any](s []V) {
	sort.Slice(s, func(i, j int) bool {
		return fmt.Sprint(s[i]) < fmt.Sprint(s[j])
	})
}

// AnsweringWriter creates an action for an answeringwriter-shaped method:
// func(ctx, V) (V, error) — a write that answers the stored state. Draws a
// value, calls both SUT and ref, and compares the answered values the way a
// read would: the answer is an observation, and a pair that stores alike
// must answer alike.
func AnsweringWriter[T, V any](
	name string,
	values *rapid.Generator[V],
	write func(context.Context, T, V) (V, error),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			v := values.Draw(rt, name+"_value")
			sutGot, sutErr := write(rt.Context(), sut, v)
			refGot, refErr := write(rt.Context(), ref, v)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:    fmt.Errorf("%s(%v): SUT err=%v, ref err=%v", name, v, sutErr, refErr),
					Input:  v,
					Output: sutGot,
				}
			}
			if sutErr == nil {
				if diff := cmp.Diff(refGot, sutGot); diff != "" {
					return model.ActionResult{
						Err:    fmt.Errorf("%s(%v): SUT/ref disagree on the answered state:\n%s", name, v, diff),
						Input:  v,
						Output: sutGot,
					}
				}
			}
			return model.ActionResult{Input: v, Output: sutGot, CallErr: sutErr}
		},
	}
}
