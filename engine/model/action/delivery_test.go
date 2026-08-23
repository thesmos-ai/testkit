// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action_test

import (
	"context"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/action"
)

// bus is the subject and the reference both: a broker whose deviation from
// correct is a field, so one type covers every case these tests need.
type bus struct {
	subs   []chan string
	copies int // how many times a publish reaches each subscriber
}

func newBus(copies int) *bus { return &bus{copies: copies} }

func (b *bus) Subscribe(context.Context) (<-chan string, error) {
	ch := make(chan string, 64)
	b.subs = append(b.subs, ch)
	return ch, nil
}

func (b *bus) Publish(_ context.Context, msg string) error {
	for _, ch := range b.subs {
		for range b.copies {
			ch <- msg
		}
	}
	return nil
}

// deliveryOf builds the action set these tests drive, at the given mode.
func deliveryOf(mode action.DeliveryMode) *action.Delivery[*bus, string] {
	return action.NewDelivery[*bus, string](
		"Feed",
		func(ctx context.Context, b *bus) (<-chan string, error) { return b.Subscribe(ctx) },
		func(ctx context.Context, b *bus, m string) error { return b.Publish(ctx, m) },
		rapid.SampledFrom([]string{"a", "b"}),
		mode,
	)
}

// runFeed drives subscribe, then n publishes, then the comparison, and
// reports what the comparison said.
//
// A fixed order rather than a drawn one, because these tests are about the
// verdict rather than about the interleaving: the run supplies the
// interleaving in earnest, and a test that drew one would be asserting
// against whichever it happened to get.
func runFeed(t *testing.T, d *action.Delivery[*bus, string], sut, ref *bus, n int) error {
	t.Helper()
	acts := d.Actions()
	var got error
	rapid.Check(t, func(rt *rapid.T) {
		for _, a := range acts {
			if a.Reset != nil {
				a.Reset()
			}
		}
		sut.subs, ref.subs = nil, nil
		acts[0].Run(rt, sut, ref)
		for range n {
			acts[1].Run(rt, sut, ref)
		}
		if res := acts[2].Run(rt, sut, ref); res.Err != nil {
			got = res.Err
		}
	})
	return got
}

// TestDeliveryAcceptsASubjectMatchingItsReference is the green case in every
// mode: one copy each side, which no mode forbids.
func TestDeliveryAcceptsASubjectMatchingItsReference(t *testing.T) {
	t.Parallel()

	for name, mode := range map[string]action.DeliveryMode{
		"at-least-once": action.AtLeastOnce,
		"at-most-once":  action.AtMostOnce,
		"exactly-once":  action.ExactlyOnce,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := runFeed(t, deliveryOf(mode), newBus(1), newBus(1), 4)
			testkit.NoError(t, err, "a subject delivering what the reference did is correct in every mode")
		})
	}
}

// TestDeliveryCatchesLossWhereTheModeForbidsIt is the direction at-least-once
// and exactly-once own, and the one at-most-once permits.
func TestDeliveryCatchesLossWhereTheModeForbidsIt(t *testing.T) {
	t.Parallel()

	t.Run("at-least-once refuses", func(t *testing.T) {
		t.Parallel()
		err := runFeed(t, deliveryOf(action.AtLeastOnce), newBus(0), newBus(1), 4)
		testkit.Error(t, err, "a subject that delivered nothing broke at-least-once")
		testkit.Contains(t, err.Error(), "permits more and never fewer",
			"and the message says which direction the mode owns")
	})

	t.Run("at-most-once permits", func(t *testing.T) {
		t.Parallel()
		err := runFeed(t, deliveryOf(action.AtMostOnce), newBus(0), newBus(1), 4)
		testkit.NoError(t, err, "loss is what at-most-once buys")
	})
}

// TestDeliveryCatchesDuplicationWhereTheModeForbidsIt is the mirror: the
// direction at-most-once and exactly-once own, and at-least-once permits.
func TestDeliveryCatchesDuplicationWhereTheModeForbidsIt(t *testing.T) {
	t.Parallel()

	t.Run("at-most-once refuses", func(t *testing.T) {
		t.Parallel()
		err := runFeed(t, deliveryOf(action.AtMostOnce), newBus(2), newBus(1), 4)
		testkit.Error(t, err, "a subject that delivered twice broke at-most-once")
		testkit.Contains(t, err.Error(), "permits fewer and never more",
			"and the message says which direction the mode owns")
	})

	t.Run("at-least-once permits", func(t *testing.T) {
		t.Parallel()
		err := runFeed(t, deliveryOf(action.AtLeastOnce), newBus(2), newBus(1), 4)
		testkit.NoError(t, err, "duplication is what at-least-once costs")
	})
}

// TestExactlyOnceRefusesBothDirections is the mode that permits neither, and
// the one a subject cannot satisfy by being generous.
func TestExactlyOnceRefusesBothDirections(t *testing.T) {
	t.Parallel()

	for name, subject := range map[string]*bus{
		"a subject that dropped":    newBus(0),
		"a subject that duplicated": newBus(2),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := runFeed(t, deliveryOf(action.ExactlyOnce), subject, newBus(1), 4)
			testkit.Error(t, err, "exactly-once permits neither direction")
			testkit.Contains(t, err.Error(), "permits neither", "and says so")
		})
	}
}

// TestDeliveryCountsAcrossTheWholeSequence is what this action set exists
// for, and what no single subscribe-publish-drain cycle can ask.
//
// A subject that delivers correctly for a while and then stops has a right
// count after one publish and a wrong one after four. The reference is what
// carries the difference: it accumulated the same publishes, so its drain
// IS the expectation, and nothing about the fourth publish alone says what
// the subscriber was owed by then.
func TestDeliveryCountsAcrossTheWholeSequence(t *testing.T) {
	t.Parallel()

	sut, ref := newBus(1), newBus(1)
	d := deliveryOf(action.ExactlyOnce)
	acts := d.Actions()

	var got error
	rapid.Check(t, func(rt *rapid.T) {
		for _, a := range acts {
			if a.Reset != nil {
				a.Reset()
			}
		}
		sut.subs, ref.subs = nil, nil
		acts[0].Run(rt, sut, ref)
		acts[1].Run(rt, sut, ref)
		if res := acts[2].Run(rt, sut, ref); res.Err != nil {
			got = res.Err
		}
		// It stops delivering, and only a later comparison can tell.
		sut.copies = 0
		for range 3 {
			acts[1].Run(rt, sut, ref)
		}
		if res := acts[2].Run(rt, sut, ref); res.Err != nil {
			got = res.Err
		}
	})
	testkit.Error(t, got, "the subject stopped delivering part-way and the totals diverged")
	testkit.True(t, strings.Contains(got.Error(), "reached the subject"),
		"the message names what each side delivered")
}
