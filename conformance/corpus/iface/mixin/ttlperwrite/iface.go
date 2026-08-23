// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package ttlperwrite is the mixin-axis fixture for a lifetime carried on
// the value rather than fixed by the directive.
//
// The sibling fixture `ttl` stamps `duration=1m`: one lifetime for every
// write, declared once. This one stamps no duration at all, and each
// write says how long its own entry lasts. Both are `ttl`, and the claim
// is the same sentence — an entry stops being readable once its lifetime
// has run out — but only this shape has a field a defect can reach for,
// which is what lets the claim be proven rather than argued.
package ttlperwrite

import (
	"context"
	"errors"
	"time"
)

// ErrExpired is what a read past the lifetime reports.
//
// Named by the notfound= stamp, so the law compares the identity the
// declaration gave rather than merely noticing that something failed.
var ErrExpired = errors.New("ttlperwrite: entry expired")

// Entry is the payload, carrying its own lifetime.
type Entry struct {
	// Key identifies the entry.
	Key string

	// Body is what a read answers with.
	Body string

	// Lifetime is how long the entry stays readable. Zero means forever,
	// which is what makes stripping it a defect rather than a refusal:
	// the write still lands and the read still answers, and the only
	// thing that changed is that nothing ever expires.
	Lifetime time.Duration
}

// Mixed is the fixture interface.
//
//testkit:out ttlperwritetest/ pkg=ttlperwritetest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Put stores an entry, starting the lifetime it carries.
	Put(ctx context.Context, e Entry) error

	// Read returns it while that lifetime holds, and the sentinel after.
	//testkit:mixin ttl put=Put read=Read notfound=ErrExpired
	Read(ctx context.Context, key string) (Entry, error)
}
