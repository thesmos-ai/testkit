// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

import (
	"reflect"

	"go.thesmos.sh/testkit/engine/model/action"
)

// ActionCtors maps every constructor name the tiers action table may answer to
// the shipped function it names.
//
// The values are the functions themselves, instantiated at throwaway types:
// referencing them is the existence check, done by the compiler rather than by
// a string comparison nothing anchors. A constructor renamed in `engine`
// breaks this file in the same build; a row in tiers naming anything absent
// here fails the census beside it.
//
// This is the only module that can hold the two ends together — tiers must not
// depend on `engine` (docs/adr/0005), and `engine` has never heard of tiers —
// which is the same standing the law census has, one table over.
//
//nolint:gochecknoglobals // a census table, read-only, test-facing.
var ActionCtors = map[string]any{
	"Aggregator":      action.Aggregator[any, int],
	"AnsweringWriter": action.AnsweringWriter[any, string],
	"BatchReader":     action.BatchReader[any, string, string],
	"CompositeWriter": action.CompositeWriter[any, string, string],
	"Lifecycle":       action.Lifecycle[any],
	"Lookup":          action.Lookup[any, string, string, string],
	"MultiAggregator": action.MultiAggregator[any, int, int],
	"MultiArgWriter":  action.MultiArgWriter[any],
	"MultiReader":     action.MultiReader[any, string, string, string],
	"Mutator":         action.Mutator[any, string],
	"PointerReader":   action.PointerReader[any, string, string],
	"PoisonCheck":     action.PoisonCheck[any],
	"Predicate":       action.Predicate[any],
	"Pure":            action.Pure[any, string],
	"Reader":          action.Reader[any, string, string],
	"ReaderNoError":   action.ReaderNoError[any, string, string],
	"ReaderWithBool":  action.ReaderWithBool[any, string, string],
	"Stream":          action.Stream[any, string],
	"StreamConsumer":  action.StreamConsumer[any, string, string],
	"VoidLifecycle":   action.VoidLifecycle[any],
	"Writer":          action.Writer[any, string],
	"WriterRecording": action.WriterRecording[any, string],

	// The contract-role rows: constructors the role table re-points to,
	// held here the same way the shape rows are. The family members with
	// no row are argued refusals in the tiers table's own comment.
	"Updater":              action.Updater[any, string],
	"Upserter":             action.Upserter[any, string],
	"CompareAndSwap":       action.CompareAndSwap[any, string],
	"ChainAppend":          action.ChainAppend[any, string],
	"ChainAppendRecording": action.ChainAppendRecording[any, string, string],
	"TwoPhase":             action.TwoPhase[any, string],
	"Pool":                 action.Pool[any, string],
}

// HasMethod reports whether the named method exists on the given type,
// resolved through the pointer method set the generated adapter holds.
func HasMethod(t reflect.Type, name string) bool {
	_, ok := reflect.PointerTo(t).MethodByName(name)
	return ok
}
