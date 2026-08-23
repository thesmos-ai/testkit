// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action

import (
	"context"
	"fmt"
	"sort"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model"
)

// DeliveryMode is the per-subscriber guarantee a [Delivery] holds a subject
// to, mirroring the publisher contract's mode= parameter.
//
// It decides which direction a disagreement is allowed to run, and nothing
// else: the same three actions drive every mode.
type DeliveryMode int

const (
	// AtLeastOnce permits the subject to deliver a message more often than
	// the reference and never less. Duplicates are the guarantee's price.
	AtLeastOnce DeliveryMode = iota

	// AtMostOnce permits the subject to deliver less often and never more.
	// Loss is the price; a duplicate is a violation.
	AtMostOnce

	// ExactlyOnce permits neither. The two multisets must match.
	ExactlyOnce
)

// Delivery drives a subscription held across a whole sequence, so what a
// subject delivered can be compared against what a reference delivered.
//
// This exists because the delivery laws cannot state the claim it states.
// Each of those runs a self-contained cycle — subscribe, publish one drawn
// message, drain — where the expected answer is known before the reference
// is consulted, so the reference is mirrored and its answer discarded.
// What no single cycle can ask is whether a subject's deliveries over a
// RUN match the publishes over that run, and that is the question a
// reference answers: it accumulated the same publishes, so its drain IS
// the expectation.
//
// The three actions have to be separate for the same reason. A subscribe
// that drained immediately would be a cycle again; the value is in the
// steps that fall between one action and the next, which the run's own
// interleaving supplies.
type Delivery[T any, Msg comparable] struct {
	subscribe func(context.Context, T) (<-chan Msg, error)
	publish   func(context.Context, T, Msg) error
	messages  *rapid.Generator[Msg]
	mode      DeliveryMode
	name      string

	// The two open subscriptions and what has been taken from each. Held
	// across the steps of one iteration and cleared between them — see
	// [model.Action.Reset], which is what makes that safe.
	sut, ref     <-chan Msg
	sutOut       []Msg
	refOut       []Msg
	subscribed   bool
	publishCount int
}

// NewDelivery builds the action set for one subscription pair.
//
// name prefixes the three actions, so a report says which subscription a
// disagreement was on where an interface carries more than one.
func NewDelivery[T any, Msg comparable](
	name string,
	subscribe func(context.Context, T) (<-chan Msg, error),
	publish func(context.Context, T, Msg) error,
	messages *rapid.Generator[Msg],
	mode DeliveryMode,
) *Delivery[T, Msg] {
	return &Delivery[T, Msg]{
		subscribe: subscribe, publish: publish,
		messages: messages, mode: mode, name: name,
	}
}

// Actions are the three the run draws from: open the pair, publish to both,
// compare what each has delivered.
//
// Every one is a no-op before the subscription is open, rather than an
// error. Which order the run draws them in is the run's business, and a
// publish that happened before anyone subscribed is a legal history — just
// not one this claim is about.
func (d *Delivery[T, Msg]) Actions() []model.Action[T] {
	return []model.Action[T]{
		{
			Name:  d.name + "Subscribe",
			Kind:  model.FailureSemantic,
			Reset: d.reset,
			Run:   d.runSubscribe,
		},
		{
			Name: d.name + "Publish",
			Kind: model.FailureSemantic,
			Run:  d.runPublish,
		},
		{
			Name: d.name + "Delivered",
			Kind: model.FailureSemantic,
			Run:  d.runDelivered,
		},
	}
}

// reset drops both subscriptions between iterations. Carried on the
// subscribe action alone, because one call clears the whole set.
func (d *Delivery[T, Msg]) reset() {
	d.sut, d.ref = nil, nil
	d.sutOut, d.refOut = nil, nil
	d.subscribed = false
	d.publishCount = 0
}

// runSubscribe opens a subscription on each side, once per iteration.
//
// A second subscribe is a no-op rather than a second pair. Two pairs would
// need the run to say which one a later publish was for, and a
// fan-out claim over several subscribers is what the delivery laws already
// state — this one is about one subscriber over a long history.
func (d *Delivery[T, Msg]) runSubscribe(rt *rapid.T, sut, ref T) model.ActionResult {
	if d.subscribed {
		return model.ActionResult{}
	}
	sutCh, err := d.subscribe(rt.Context(), sut)
	if err != nil {
		return model.ActionResult{CallErr: err}
	}
	refCh, refErr := d.subscribe(rt.Context(), ref)
	if refErr != nil {
		return model.ActionResult{
			Err: fmt.Errorf("%sSubscribe: the reference refused a subscription the subject accepted: %w",
				d.name, refErr),
		}
	}
	d.sut, d.ref, d.subscribed = sutCh, refCh, true
	return model.ActionResult{}
}

// runPublish sends one drawn message to both sides.
//
// A refusal by the subject is its own answer and is mirrored: the
// reference is not published to either, so the two stay level and a later
// comparison is about delivery rather than about a write one side never
// took.
func (d *Delivery[T, Msg]) runPublish(rt *rapid.T, sut, ref T) model.ActionResult {
	msg := d.messages.Draw(rt, d.name+"_msg")
	if err := d.publish(rt.Context(), sut, msg); err != nil {
		return model.ActionResult{CallErr: err, Input: msg}
	}
	if err := d.publish(rt.Context(), ref, msg); err != nil {
		return model.ActionResult{
			Err: fmt.Errorf("%sPublish(%v): the reference refused a publish the subject took: %w",
				d.name, msg, err),
			Input: msg,
		}
	}
	d.publishCount++
	return model.ActionResult{Input: msg}
}

// runDelivered takes what is waiting on each side and holds the totals to
// the mode.
//
// Both drains are non-blocking, and an empty channel is read as delivered
// rather than as pending. That is the same floor the delivery laws stand
// on: this contract's publish returns once the message is buffered, so a
// subject that delivers asynchronously reads as loss here. It is a floor
// stated out loud rather than a race.
func (d *Delivery[T, Msg]) runDelivered(_ *rapid.T, _, _ T) model.ActionResult {
	if !d.subscribed {
		return model.ActionResult{}
	}
	d.sutOut = append(d.sutOut, drain(d.sut)...)
	d.refOut = append(d.refOut, drain(d.ref)...)
	if err := d.compare(); err != nil {
		return model.ActionResult{Err: err, Input: d.publishCount}
	}
	return model.ActionResult{Output: len(d.sutOut)}
}

// compare holds the two multisets to the mode's direction.
//
// Counted per message rather than compared as sequences. Order is a claim
// of its own and not every publisher makes it; counting states the part
// every mode agrees on, which is how many copies of each message reached
// the subscriber.
func (d *Delivery[T, Msg]) compare() error {
	sut, ref := counts(d.sutOut), counts(d.refOut)
	for _, msg := range keysOf(sut, ref) {
		got, want := sut[msg], ref[msg]
		switch d.mode {
		case AtLeastOnce:
			if got < want {
				return fmt.Errorf("%sDelivered: %v reached the subject %d times and the "+
					"reference %d; at-least-once permits more and never fewer",
					d.name, msg, got, want)
			}
		case AtMostOnce:
			if got > want {
				return fmt.Errorf("%sDelivered: %v reached the subject %d times and the "+
					"reference %d; at-most-once permits fewer and never more",
					d.name, msg, got, want)
			}
		case ExactlyOnce:
			if got != want {
				return fmt.Errorf("%sDelivered: %v reached the subject %d times and the "+
					"reference %d; exactly-once permits neither",
					d.name, msg, got, want)
			}
		}
	}
	return nil
}

// drain empties a channel without blocking, so an empty one reads as
// nothing further delivered rather than as a step that never returns.
func drain[Msg any](ch <-chan Msg) []Msg {
	var out []Msg
	for {
		select {
		case m, open := <-ch:
			if !open {
				return out
			}
			out = append(out, m)
		default:
			return out
		}
	}
}

// counts tallies a multiset.
func counts[Msg comparable](msgs []Msg) map[Msg]int {
	out := make(map[Msg]int, len(msgs))
	for _, m := range msgs {
		out[m]++
	}
	return out
}

// keysOf is every message either side saw, ordered so a failure names the
// same one on every replay.
//
// Both sides, because a subject inventing a message the reference never
// delivered is as much a divergence as one dropping it — and a loop over
// the reference alone would never look at it.
func keysOf[Msg comparable](a, b map[Msg]int) []Msg {
	seen := make(map[Msg]struct{}, len(a)+len(b))
	out := make([]Msg, 0, len(a)+len(b))
	for _, m := range []map[Msg]int{a, b} {
		for k := range m {
			if _, dup := seen[k]; dup {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, k)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return fmt.Sprint(out[i]) < fmt.Sprint(out[j])
	})
	return out
}
