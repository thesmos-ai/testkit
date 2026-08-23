// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

// UnarmedDoors registers every generated door, clock, and optional role no
// corpus consumer arms, each with the verdict that keeps the absence
// honest. The census derives the door set from the same in-memory emission
// the assertion gate reads, then looks for the arming in the fixture's own
// hand-written tests — the generated option's spelling, which is the one
// convention the corpus consumers already follow. Both directions are
// enforced by name: a door that is neither armed nor registered is a
// visible skip pretending to be a check, and a row for a door some
// consumer now arms is a stale excuse that must be deleted.
//
// Two rows left, both optional roles a declaration simply does not carry.
// Every capability door the corpus emits is answered: the register held the
// isolation and drain claims for a while, and each of those turned out to
// be a fixture that contradicted its own mixin rather than a door nobody
// could fill.
//
// Keys are `<corpus-dir>/<law-id>.<item>`, where the item is the door's own
// name as the harness's Provide map spells it, the role the header calls
// unarmed, or the literal "clock".
//
//nolint:gochecknoglobals // a debt register, read-only, test-facing.
var UnarmedDoors = map[string]string{
	// The two mode fixtures that keep the redeliver role undeclared prove
	// the role's optional omission — the path every optional role rides —
	// while publisher-redeliver and the exactly-once sibling arm it.
	"iface/contract/publisher-atleastonce/AUTO-PUBLISHER-AT-LEAST-ONCE.Redeliver": "the unarmed sibling proves the role's optional omission; publisher-redeliver arms the duplicate",
	"iface/contract/publisher-atmostonce/AUTO-PUBLISHER-AT-MOST-ONCE.Redeliver":   "a redelivery under at-most-once is the violation itself; the bound is proven on the single publish",
}
