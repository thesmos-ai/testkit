# Sim

> **Status: planned.** This generator is not implemented. No directive is registered for it, and `testkit run` does not produce the output described below. This page records the intended design so adoption can be planned against a stable target — the directive, the output paths and the generated surface may all differ once it ships.
>
> **Not to be confused with the sim CHECK FAMILY, which does ship.** A generated suite already carries `sim/<iface>/recovery` and `sim/<iface>/fault` rows: a drawn schedule of writes, reads and crash points, judged against what the writes acknowledged. Those are per-interface checks rendered by the model tier and reached through `RunX`, and they are the reason a generated harness has a `Recover` field. This page is about something else — a harness a consumer drives, wrapping a whole subsystem. The ID family and the generator that renders it deliberately do not have to match; see `projection.RowKinds`.

Tier 5 substrate. Subsystem-shaped deterministic simulation harness wrapping the full production stack: stubs auto-wrapped with recording-stamped `OnRecord` hooks emitting into the engine trace; `Clock` and `RandSource` plumbed from engine seeds; completion-event sinks; capture-on-failure with minimal-reproducer seed extraction; `Workload[T]` and `Invariant[T]` registration verbs; cooperative-quiescence `AssertAll`. Per-subsystem composition — one `Sim` per top-level interface — replaces hand-rolled per-package sim packages.

`sim` is the harness `chaos` and `replay` build on top of.

## Planned directive

```go
//go:generate testkit sim -o storetest/store_sim.gen.go Store
```

## Planned shape

```go
sim := storetest.NewStoreSim(t, seed, cfg)
sim.RegisterWorkload(workload)
sim.RegisterInvariant(invariant)
sim.AssertAll()
```

The harness composes the generated stub with the engine's clock, rand source, and trace, so a single seed reproduces an entire run.

## See also

- [Primitives / Clock](../primitives/clock.md), [RandSource](../primitives/rand.md)
- [Generators / chaos](chaos.md), [replay](replay.md), [differential-rollout](differential-rollout.md)
