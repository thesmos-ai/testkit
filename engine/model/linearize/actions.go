// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize

import (
	"context"
	"fmt"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model"
)

// ConcurrentReader creates a ConcurrentAction for a Reader-shaped method.
// Draws a key, calls impl.Read, records (key, ReaderResult{V, error}).
func ConcurrentReader[T any, K comparable, V any](
	name string,
	keys *rapid.Generator[K],
	read func(context.Context, T, K) (V, error),
) model.ConcurrentAction[T] {
	return model.ConcurrentAction[T]{
		Name: name,
		Gen: func(rt *rapid.T) any {
			return keys.Draw(rt, name+"_key")
		},
		Apply: func(ctx context.Context, impl T, input any) any {
			k := input.(K)
			v, err := read(ctx, impl, k)
			return ReaderResult[V]{Value: v, Err: err}
		},
		PartitionKey: func(input any) string {
			return fmt.Sprint(input)
		},
	}
}

// ConcurrentReaderWithBool creates a ConcurrentAction for a
// ReaderWithBool-shaped method: func(ctx, K) (V, bool).
func ConcurrentReaderWithBool[T any, K comparable, V any](
	name string,
	keys *rapid.Generator[K],
	read func(context.Context, T, K) (V, bool),
) model.ConcurrentAction[T] {
	return model.ConcurrentAction[T]{
		Name: name,
		Gen: func(rt *rapid.T) any {
			return keys.Draw(rt, name+"_key")
		},
		Apply: func(ctx context.Context, impl T, input any) any {
			k := input.(K)
			v, ok := read(ctx, impl, k)
			return ReaderBoolResult[V]{Value: v, OK: ok}
		},
		PartitionKey: func(input any) string {
			return fmt.Sprint(input)
		},
	}
}

// ConcurrentWriter creates a ConcurrentAction for a Writer-shaped method.
// Draws a value, calls impl.Write, partitions by keyOf(value).
func ConcurrentWriter[T any, K comparable, V any](
	name string,
	values *rapid.Generator[V],
	write func(context.Context, T, V) error,
	keyOf func(V) K,
) model.ConcurrentAction[T] {
	return model.ConcurrentAction[T]{
		Name: name,
		Gen: func(rt *rapid.T) any {
			return values.Draw(rt, name+"_value")
		},
		Apply: func(ctx context.Context, impl T, input any) any {
			v := input.(V)
			err := write(ctx, impl, v)
			return WriterResult{Err: err}
		},
		PartitionKey: func(input any) string {
			v := input.(V)
			return fmt.Sprint(keyOf(v))
		},
	}
}

// ConcurrentDeleter creates a ConcurrentAction for a Deleter-shaped method.
func ConcurrentDeleter[T any, K comparable](
	name string,
	keys *rapid.Generator[K],
	del func(context.Context, T, K) error,
) model.ConcurrentAction[T] {
	return model.ConcurrentAction[T]{
		Name: name,
		Gen: func(rt *rapid.T) any {
			return keys.Draw(rt, name+"_key")
		},
		Apply: func(ctx context.Context, impl T, input any) any {
			k := input.(K)
			err := del(ctx, impl, k)
			return WriterResult{Err: err}
		},
		PartitionKey: func(input any) string {
			return fmt.Sprint(input)
		},
	}
}

// ConcurrentLookup creates a ConcurrentAction for a Lookup-shaped method:
// func(T, K) (R1, R2, bool).
func ConcurrentLookup[T any, K comparable, R1, R2 any](
	name string,
	keys *rapid.Generator[K],
	lookup func(T, K) (R1, R2, bool),
) model.ConcurrentAction[T] {
	return model.ConcurrentAction[T]{
		Name: name,
		Gen: func(rt *rapid.T) any {
			return keys.Draw(rt, name+"_key")
		},
		Apply: func(_ context.Context, impl T, input any) any {
			k := input.(K)
			r1, r2, ok := lookup(impl, k)
			return LookupResult[R1, R2]{R1: r1, R2: r2, OK: ok}
		},
		PartitionKey: func(input any) string {
			return fmt.Sprint(input)
		},
	}
}

// ConcurrentAnsweringWriter creates a ConcurrentAction for an
// answeringwriter-shaped method: func(ctx, V) (V, error). The answered
// state carries the store-assigned stamp the per-client laws order writes
// by, so the result reaches the trace whole.
func ConcurrentAnsweringWriter[T any, K comparable, V any](
	name string,
	values *rapid.Generator[V],
	write func(context.Context, T, V) (V, error),
	keyOf func(V) K,
) model.ConcurrentAction[T] {
	return model.ConcurrentAction[T]{
		Name: name,
		Gen: func(rt *rapid.T) any {
			return values.Draw(rt, name+"_value")
		},
		Apply: func(ctx context.Context, impl T, input any) any {
			v := input.(V)
			got, err := write(ctx, impl, v)
			return AnsweringResult[V]{Value: got, Err: err}
		},
		PartitionKey: func(input any) string {
			return fmt.Sprint(keyOf(input.(V)))
		},
	}
}

// ConcurrentCAS creates a ConcurrentAction for a version-guarded write at a
// single cell. One partition, deliberately: the cell is the unit the
// compare-and-set guards, and a drawn version that misses is itself a
// modeled outcome rather than noise.
func ConcurrentCAS[T, V any](
	name string,
	values *rapid.Generator[V],
	cas func(context.Context, T, V) error,
) model.ConcurrentAction[T] {
	return model.ConcurrentAction[T]{
		Name: name,
		Gen: func(rt *rapid.T) any {
			return values.Draw(rt, name+"_value")
		},
		Apply: func(ctx context.Context, impl T, input any) any {
			err := cas(ctx, impl, input.(V))
			return WriterResult{Err: err}
		},
		PartitionKey: func(any) string { return "" },
	}
}

// ConcurrentWrite creates a ConcurrentAction for a Writer-shaped method
// whose history a STEPLESS model carries: one partition, because nothing
// partitions a history no model steps.
//
// The keyed [ConcurrentWriter] cannot serve it. That one partitions by the
// key a write lands on, and a store assigning its own version stamp
// derives no key projection at all — which is the same fact that put it on
// a stepless model, since a stamp the subject chooses defeats the value
// equality every model compares by. The verdict comes from the per-client
// laws reading the multi-client trace, and those read the method name.
func ConcurrentWrite[T, V any](
	name string,
	values *rapid.Generator[V],
	write func(context.Context, T, V) error,
) model.ConcurrentAction[T] {
	return model.ConcurrentAction[T]{
		Name: name,
		Gen: func(rt *rapid.T) any {
			return values.Draw(rt, name+"_value")
		},
		Apply: func(ctx context.Context, impl T, input any) any {
			err := write(ctx, impl, input.(V))
			return WriterResult{Err: err}
		},
		PartitionKey: func(any) string { return "" },
	}
}

// ConcurrentAnsweringWrite is [ConcurrentWrite] for a write that answers
// the state it stored, which is how a version-stamping store hands its
// stamp back — the value the per-client laws order reads against.
func ConcurrentAnsweringWrite[T, V any](
	name string,
	values *rapid.Generator[V],
	write func(context.Context, T, V) (V, error),
) model.ConcurrentAction[T] {
	return model.ConcurrentAction[T]{
		Name: name,
		Gen: func(rt *rapid.T) any {
			return values.Draw(rt, name+"_value")
		},
		Apply: func(ctx context.Context, impl T, input any) any {
			v, err := write(ctx, impl, input.(V))
			return AnsweringResult[V]{Value: v, Err: err}
		},
		PartitionKey: func(any) string { return "" },
	}
}

// ConcurrentCellReader creates a ConcurrentAction for a keyless read of the
// whole cell — nothing to draw, and the one partition is the cell itself.
func ConcurrentCellReader[T, V any](
	name string,
	read func(context.Context, T) (V, error),
) model.ConcurrentAction[T] {
	return model.ConcurrentAction[T]{
		Name: name,
		Gen:  func(*rapid.T) any { return nil },
		Apply: func(ctx context.Context, impl T, _ any) any {
			v, err := read(ctx, impl)
			return ReaderResult[V]{Value: v, Err: err}
		},
		PartitionKey: func(any) string { return "" },
	}
}

// ConcurrentAppend creates a ConcurrentAction for an offset-answering
// append into the single shared history the [AppendLog] model checks.
func ConcurrentAppend[T, V any](
	name string,
	values *rapid.Generator[V],
	appendFn func(context.Context, T, V) (int64, error),
) model.ConcurrentAction[T] {
	return model.ConcurrentAction[T]{
		Name: name,
		Gen: func(rt *rapid.T) any {
			return values.Draw(rt, name+"_value")
		},
		Apply: func(ctx context.Context, impl T, input any) any {
			off, err := appendFn(ctx, impl, input.(V))
			return AppendResult{Off: off, Err: err}
		},
		PartitionKey: func(any) string { return "" },
	}
}
