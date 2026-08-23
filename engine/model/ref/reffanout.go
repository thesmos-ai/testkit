// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref

import (
	"context"
	"errors"
	"sync"
)

// subCap sizes a subscription's buffer.
//
// Larger than any drawn sequence can fill, which is the whole point: this
// reference must never drop. A reference sharing the subject's back-pressure
// could not disagree with a subject that lost a message, and disagreeing is
// the only reason it is here.
const subCap = 4096

// ErrFanOutClosed is what a closed broker reports.
//
// The oracle's own, because a publisher declaring a lifecycle names its
// sentinel through the lifecycle mixin and the laws reading it are that
// mixin's. Nothing here compares the two: a fan-out that was never closed
// never reports it.
var ErrFanOutClosed = errors.New("ref: the fan-out is closed")

// FanOut is the delivery model a topic-free publisher implies: every
// subscriber open at the time of a publish receives that message, once.
//
// Shaped to the interface rather than to a queue, and that is the
// difference between this and [AtLeastOnce] beside it. Those three model a
// broker a consumer DRAINS — Subscribe answers an index, Drain takes what
// is waiting — which is right for a law driving the broker directly and
// wrong for an adapter, because no Go interface is written that way. A
// subject's Subscribe answers a channel, so this one does too, and the
// generated adapter forwards instead of translating.
//
// It delivers exactly once to each subscriber and never drops. Both are
// deliberate: a subject claiming at-least-once may deliver more and one
// claiming at-most-once may deliver less, and a reference sitting at
// neither extreme is what lets a comparison say which way a subject
// deviated. See [action.Delivery], which is the comparison.
//
// Thread-safe via mutex, like every reference. An oracle, not production
// code.
type FanOut[Msg any] struct {
	mu     sync.Mutex
	subs   []chan Msg
	closed bool
}

// NewFanOut returns a broker with no subscribers.
func NewFanOut[Msg any]() *FanOut[Msg] { return &FanOut[Msg]{} }

// Subscribe opens a subscription and answers the channel it delivers on.
//
// A cancelled context refuses, the shape every context-taking operation on
// a reference is held to: the oracle stands opposite subjects whose
// lifecycle-shaped methods must respect cancellation, and one that ignored
// it would fail the law its adapter is driven through.
func (f *FanOut[Msg]) Subscribe(ctx context.Context) (<-chan Msg, error) {
	if err := refCtxErr(ctx); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil, ErrFanOutClosed
	}
	ch := make(chan Msg, subCap)
	f.subs = append(f.subs, ch)
	return ch, nil
}

// Publish delivers the message to every subscription open at this moment.
//
// Open at this moment, not at any moment: a subscriber that arrives after a
// publish has missed it, and one that never subscribed was never owed
// anything. That is the ordering a real broker keeps and the one a
// comparison over a sequence depends on.
func (f *FanOut[Msg]) Publish(ctx context.Context, msg Msg) error {
	if err := refCtxErr(ctx); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return ErrFanOutClosed
	}
	for _, ch := range f.subs {
		select {
		case ch <- msg:
		default:
			// Unreachable at subCap for any sequence this drives, and a
			// drop here would make the reference lossy in exactly the way
			// its whole purpose forbids. Left as a select rather than a
			// bare send so a run that somehow reached it stalls nothing.
		}
	}
	return nil
}

// Close stops further publishes and subscriptions. Idempotent: a second
// Close changes nothing, which is the reading every lifecycle law takes.
func (f *FanOut[Msg]) Close(ctx context.Context) error {
	if err := refCtxErr(ctx); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

// refCtxErr reports a cancelled context and tolerates a nil one, which a
// generated check hands over on purpose.
func refCtxErr(ctx context.Context) error {
	if ctx == nil {
		return context.Canceled
	}
	return ctx.Err()
}
