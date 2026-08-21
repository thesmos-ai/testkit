// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection

import "strings"

// Bind names one assertion body a check delegates to: an engine law by
// its lawid constant, with the probe set the binding arms. It renders
// into the lock's fourth column, where narrowing a probe set diffs.
type Bind struct {
	// Law is the engine law's identifier — always a core/lawid
	// constant at the construction site, never a literal.
	Law string
	// Probes names the methods the law probes, for laws whose claim
	// spans several; empty for single-probe laws.
	Probes []string
}

// Render spells the lock-column form: "LAW" or "LAW(P1 P2)". The
// format has this one home; the runtime's LockLines validates the
// result against the manifest grammar.
func (b Bind) Render() string {
	if len(b.Probes) == 0 {
		return b.Law
	}
	return b.Law + "(" + strings.Join(b.Probes, " ") + ")"
}

// RenderBinds projects a plan's binds into the runtime's lock-row
// shape.
func RenderBinds(binds []Bind) []string {
	if len(binds) == 0 {
		return nil
	}
	out := make([]string, len(binds))
	for i, b := range binds {
		out[i] = b.Render()
	}
	return out
}
