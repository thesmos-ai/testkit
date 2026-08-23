// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package restrictedpool is the mixin-axis fixture for pool provenance: a
// declaration whose input domain a run can narrow, and a model tier that
// has to respect the narrowing.
//
// Every other fixture draws its sequences from a pool the generator
// derived, and widening a derived pool costs nothing — nobody said what
// the implementation accepts, so arbitrary draws are fair. This one says
// it. The roles below carry pools a run replaces by passing a config, and
// a run that replaced them has stated what the subject takes. A tier that
// blended arbitrary or hostile draws past that statement would red correct
// code against inputs its owner ruled out, which is worse than not
// probing: the red is real, the defect is not, and the fix is to weaken a
// check that was right.
//
// The pair is the smallest shape that carries both halves. Roles on named
// types are what produce the config a run overrides; the keyed read and
// write are what give the model tier sequences to draw for.
package restrictedpool

import (
	"context"
	"errors"
)

// ErrNotFound is what Get reports for a key nothing wrote.
var ErrNotFound = errors.New("restrictedpool: not found")

// Key identifies one record. Named rather than a bare string because a
// role is written on a declaration, and a bare parameter has none.
//
//testkit:role key
//testkit:default "test-key"
type Key string

// Body is what a record holds.
//
//testkit:role payload
//testkit:default "test-body"
type Body string

// Store is the fixture interface.
//
//testkit:out restrictedpooltest/ pkg=restrictedpooltest
//testkit:stub
//testkit:suite
//testkit:model
type Store interface {
	// Put writes a body under a key, drawing both roles.
	Put(ctx context.Context, key Key, body Body) error

	// Get reads one back, and reports a miss with the declared sentinel.
	Get(ctx context.Context, key Key) (Body, error)
}
