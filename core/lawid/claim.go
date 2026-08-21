// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package lawid

import (
	"fmt"
	"slices"
	"strings"
)

// Claim is one law's human claim: the sentence a lock row, a report,
// and a skipped subtest speak for it. It lives beside the identifier
// for the reason the identifier lives here at all — the generator
// writes the sentence into manifests and the engine reports outcomes
// under it, and a wording spelled in two modules drifts where no
// compiler can see.
//
// Parametric where the wording names something only a declaration
// knows. The placeholders are the closed vocabulary below; a claim
// interpolating anything else fails this package's own census, and
// [Claim.Fill] refuses a sentence left half-filled rather than
// publishing a bracket into a manifest.
type Claim string

// The placeholder vocabulary a claim may interpolate. Each names a
// fact the selecting declaration carries; the consumer filling a
// claim resolves them from its own stamps, and over-supplying is
// free — an absent placeholder ignores its pair.
const (
	// PlaceClose is the close method the selecting declaration names:
	// the after-close teardown, or a produced handle's release.
	PlaceClose = "{close}"
	// PlaceNext is the produced handle's reader.
	PlaceNext = "{next}"
	// PlaceProduced is the contract's own word for the produced
	// handle.
	PlaceProduced = "{produced}"
	// PlaceSubject is the subject interface's token.
	PlaceSubject = "{subject}"
)

// Placeholders enumerates the vocabulary, for the census that holds
// every claim's tokens to it.
func Placeholders() []string {
	return []string{PlaceClose, PlaceNext, PlaceProduced, PlaceSubject}
}

// Fill interpolates placeholder/value pairs and refuses a claim left
// unfilled: a leftover bracket in a manifest row would read as prose
// and diff forever after.
func (c Claim) Fill(pairs ...string) (string, error) {
	if len(pairs)%2 != 0 {
		return "", fmt.Errorf("lawid: Fill takes placeholder/value pairs, got %d values", len(pairs))
	}
	out := string(c)
	for i := 0; i < len(pairs); i += 2 {
		out = strings.ReplaceAll(out, pairs[i], pairs[i+1])
	}
	for _, p := range Placeholders() {
		if strings.Contains(out, p) {
			return "", fmt.Errorf("lawid: claim %q left %s unfilled", string(c), p)
		}
	}
	return out, nil
}

// ClaimOf returns the law's claim, false for an identifier this
// package does not word yet — the consumer's signal to refuse the
// row by name rather than invent a sentence. Wordings accrete toward
// the full registry under the conformance corpus, which surfaces
// every unworded law the day a fixture stamps its classification.
func ClaimOf(id string) (Claim, bool) {
	w, ok := worded()[id]
	return w.claim, ok
}

// AccessorOf returns the law's spelling in a generated check index —
// the method a consumer calls to name the check, as in
// `ix.Model.CloseIdempotent()`.
//
// Here rather than in the emitter because it is a fact about the law,
// and the law's facts have one home. The identifier cannot be derived
// from the constant's name either: the contract word that qualifies a
// law globally (Cursor, Lease) is redundant inside an index already
// scoped to one interface, and which part is redundant is a judgment
// per law rather than a prefix rule.
func AccessorOf(id string) (string, bool) {
	w, ok := worded()[id]
	return w.accessor, ok
}

// IsLaw reports whether the identifier is one this package registers.
//
// The question a caller holding a bare identity segment has to ask
// before deciding what an unworded one means: a registered law with no
// wording is a gap to refuse by name, while a segment that was never a
// law is somebody else's vocabulary and none of this package's
// business. Prefix-sniffing for AUTO- cannot tell them apart — the
// runtime spells segments that way too.
func IsLaw(id string) bool { return slices.Contains(All(), id) }

// ConstOf returns the Go identifier this package declares the law
// under, for emitted code that must name the law rather than repeat
// its text — `lawid.CursorCloseIdempotent`, not the AUTO- string.
//
// Carried as data because Go cannot ask a constant for its own name,
// and a generated file spelling the literal would be the one place
// the identifier is not the single home.
func ConstOf(id string) (string, bool) {
	w, ok := worded()[id]
	return w.constant, ok
}

// law is what this package knows about one identifier.
//
// One row per law rather than a map per fact: a claim and an accessor
// added in separate places is a law worded in one and unspellable in
// the other, which no census catches until a fixture stamps it.
type law struct {
	claim Claim

	// accessor is the law's spelling in a generated index; constant
	// is the identifier this package declares it under.
	accessor, constant string
}

// worded is the laws the proof-of-concept corpus pinned. The claim
// spellings are its manifests', verbatim; the accessors are its
// generated indexes'.
func worded() map[string]law {
	return map[string]law{
		TTLExpiry: {
			"an entry stops being readable once its lifetime has run out",
			"Expiry", "TTLExpiry",
		},
		// The two comparison laws. Both word what they actually do —
		// compare the subject against the reference — rather than the
		// property a reader might assume from the name: ReadAfterWrite
		// never writes, and neither says anything about a subject taken
		// on its own.
		ReadAfterWrite: {
			"every key reads the same on the subject as on the reference",
			"ReadsAgree", "ReadAfterWrite",
		},
		CountEqualsReference: {
			"the subject counts what the reference counts",
			"Counts", "CountEqualsReference",
		},
		// The publisher pair, worded as the proof-of-concept corpus
		// spelled them.
		PublisherDelivers: {
			"a message published after subscribers registered reaches every one of them",
			"Delivers", "PublisherDelivers",
		},
		PublisherAtLeastOnce: {
			"every subscriber's delivery count for a published message is one or more",
			"AtLeastOnce", "PublisherAtLeastOnce",
		},
		DeadlineRespecting: {
			"an operation given a deadline returns once that deadline fires",
			"Deadline", "DeadlineRespecting",
		},
		ScheduledFiresAfterAdvance: {
			// At least, not exactly: the law shares its subject with an
			// action stream scheduling work of its own, and whatever of
			// that is pending fires inside the same advance. The wording
			// says what the law checks, which is the whole reason it is
			// written here rather than inferred from the identifier.
			"work scheduled for a time already passed has fired",
			"Scheduled", "ScheduledFiresAfterAdvance",
		},
		LifecycleAfterClose: {
			"once {close} has run, every method reports the closed sentinel",
			"AfterClose", "LifecycleAfterClose",
		},
		PoisonConsistent: {
			"once the {subject} reports it is closed, it keeps reporting it",
			"PoisonConsistent", "PoisonConsistent",
		},
		CursorCloseIdempotent: {
			"a second {close} on a {produced} changes nothing",
			"CloseIdempotent", "CursorCloseIdempotent",
		},
		CursorNextAfterClose: {
			"once a {produced} is closed, {next} reports the closed sentinel",
			"NextAfterClose", "CursorNextAfterClose",
		},
		AppenderMonotonicOffsets: {
			"offsets of successive appends strictly increase",
			"MonotonicOffsets", "AppenderMonotonicOffsets",
		},
		LeaseDoubleAcquireBlocks: {
			"a second acquire of a held key reports the held sentinel",
			"DoubleAcquire", "LeaseDoubleAcquireBlocks",
		},
		LeaseReleasedOnCancel: {
			"a held lease frees once its acquiring context is cancelled",
			"ReleasedOnCancel", "LeaseReleasedOnCancel",
		},
	}
}
