// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package persister is the contract-axis fixture for the persister contract:
// a write paired with the read that observes it.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package persister

import (
	"context"
	"errors"
)

// ErrMediumGone is what a store reports once its medium has failed.
//
// Named by the fault stamp on Put so a check can ask for that state by
// identity, and answered by whichever harness knows how to produce it.
var ErrMediumGone = errors.New("persister: the medium is gone")

// Value is the payload the contract's roles carry.
//
// Annotated for the builder as well, which nothing else in the corpus
// does: the builder axis and the model axis had never met, so a struct
// that is both a value type and a buildable one was a combination
// nothing exercised. They compose — the builder, the double, the harness
// and the model tier share this type without colliding over a name.
//
// The routing is repeated here because a node's //testkit:out scopes to
// that node, and the interface below carries its own. Without this line
// the builder still emits — beside the source, which is where an output
// with no routing goes, and not where the generated package is.
// Declaring it once above the package clause covers everything beneath
// it and is the other way round; corpus/integration/validated does that.
//
//testkit:out persistertest/ pkg=persistertest
//testkit:builder
type Value struct {
	// Key identifies the record.
	//
	// Roled, which is what opens a pool: a role says what a value IS, and
	// the pool it opens carries a third member hostile to that role. The
	// corpus had one fixture doing this and the adversarial arm is the
	// emitter's most valuable path, so one witness was thin.
	//
	//testkit:role key
	//testkit:default "test-key"
	Key string

	// Body is what the record holds.
	//
	//testkit:role payload
	//testkit:default "test-body"
	Body string
}

// Contract is the fixture interface.
//
//testkit:out persistertest/ pkg=persistertest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Put is the persister contract's writer role, and hosts the directive
	// that names its partners.
	//
	// The fault stamp names the state a caller can put this store into,
	// which is what lets the crash schedule ask its second question: not
	// what an acknowledged write survives, but what a REFUSED one left
	// behind. The store has to receive the write and decline it for that
	// to mean anything.
	//testkit:contract persister role=writer reader=Get
	//testkit:fault ErrMediumGone
	Put(ctx context.Context, v Value) error

	// Get is the persister contract's reader role.
	Get(ctx context.Context, key string) (Value, error)
}
