// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package batchwriter is the contract-axis fixture for the batch-writer contract:
// a write that takes many values at once.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package batchwriter

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct {
	// Key files the value, and this subject refuses one that is empty —
	// which is what gives `mode=atomic` a failure to be about.
	//
	// Roled, and that is the point: a role opens a pool, and a pool is
	// how a consumer says which inputs their implementation accepts. The
	// adversarial arm draws the empty string among others, and a subject
	// that narrows what it takes has to say so or be red against inputs
	// its own author ruled out.
	//
	//testkit:role key
	//testkit:default "test-key"
	Key string

	// Body is what the value carries.
	//
	//testkit:role payload
	//testkit:default "test-body"
	Body string
}

// Contract is the fixture interface.
//
//testkit:out batchwritertest/ pkg=batchwritertest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Put is the batch-writer contract's writer role, and hosts the directive
	// that names its partners.
	//
	// `mode=atomic` is a claim about state AFTER a failure, so it needs
	// something to observe that state with — which is what the reader role
	// is for. Without it the only statable claim is that a good write
	// succeeds, which holds for a store that also keeps half a refused one.
	//testkit:contract batch-writer role=writer mode=atomic reader=Get
	Put(ctx context.Context, v Value) error

	// Get is the batch-writer contract's reader role: the observation the
	// mode's claim is read through.
	//
	// A role rather than a param, so the pairing is recorded on both members
	// — a consumer holding this method can find the contract it confirms,
	// which a one-directional `read=` could not answer.
	//testkit:contract batch-writer role=reader
	Get(ctx context.Context, key string) (Value, error)
}
