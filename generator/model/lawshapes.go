// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import "go.thesmos.sh/testkit/core/lawid"

// lawShape names the closure a law field renders — the template dispatch.
type lawShape string

//nolint:gochecknoglobals // a lookup table, read-only after init.
var lawRoleShapes = map[string]map[string]lawShape{
	lawid.Cacheable:             {fRead: shapeKeyedRead},
	lawid.DefaultOnError:        {fRead: shapeKeyedRead},
	lawid.DeleteReturnsNotFound: {fRead: shapeKeyedRead},
	lawid.PointInTime:           {fRead: shapeKeyedRead},
	lawid.ReadAfterWrite:        {fRead: shapeKeyedRead},
	lawid.Sticky:                {fRead: shapeKeyedRead},
	lawid.WriteObservable:       {fWrite: shapeValueOp, fRead: shapeKeyedRead},

	lawid.StreamCompletion:   {fDrain: shapeDrainSlice},
	lawid.StreamNoDuplicates: {fDrain: shapeDrainSlice},
	lawid.StreamOverMatch:    {fDrain: shapeDrainSlice},
	lawid.StreamPermutation:  {fDrain: shapeDrainSlice},
	lawid.StreamReentrant:    {"Collect": shapeDrainSlice},
	lawid.StreamStableOrder:  {fDrain: shapeDrainSlice},
	lawid.StreamReflectsMutations: {
		"Put":    shapeValueOp,
		"Delete": shapeValueOp,
		fDrain:   shapeDrainSlice,
	},

	lawid.AggregatorBounded:        {fRead: shapeScalar},
	lawid.CountEqualsReference:     {fCount: shapeScalar},
	lawid.LifecycleRespectsContext: {"Op": shapeCtxOp},
	lawid.MonotonicNonDecreasing:   {fRead: shapeScalar},
	lawid.PoisonIdempotentRead:     {fProbe: shapeErrOp},
	lawid.PoisonNilOnFresh:         {fProbe: shapeErrOp},
	lawid.PredicateConsistent:      {fCall: shapeBoolCall},
	lawid.PureDeterministic:        {fCall: shapeResultCall},
	lawid.TotalOver:                {fCall: shapeInputCall},

	lawid.Associative:      {"Apply": shapeValueOp},
	lawid.AtomicWrite:      {fWrite: shapeValueOp},
	lawid.CommutativeWrite: {fWrite: shapeValueOp},
	lawid.Conservative:     {fWrite: shapeValueOp, "Sum": shapeSum},
	lawid.CRDTMerge:        {fWrite: shapeValueOp, fMerge: shapeMerge},
	lawid.IdempotentWrite:  {fWrite: shapeValueOp},
	lawid.InjectionSafe:    {fStore: shapeKVOp, "Load": shapeKeyedRead},
	lawid.XSSSafe:          {"Render": shapeInputCall},

	lawid.AppenderMonotonicOffsets: {"Append": shapeAppendOff},
	lawid.CASAtomicOneWinner:       {fCAS: shapeValueOp, fRead: shapeScalar},
	lawid.LeakFree:                 {"Open": shapeErrOp, "Close": shapeErrOp, "Outstanding": shapeScalar},
	lawid.LeaseDoubleAcquireBlocks: {"Acquire": shapeKeyedOp, "Release": shapeKeyedOp},
	lawid.PersisterRetrievable:     {"Save": shapeSave, fRead: shapeKeyedRead},
	lawid.Roundtrip:                {"Forward": shapeInputCall, "Inverse": shapeInputCall},
	lawid.LossyRoundtrip:           {"Forward": shapeInputCall, "Inverse": shapeInputCall},
	lawid.UpdaterReplaces:          {"Update": shapeValueOp, fRead: shapeKeyedRead},
	lawid.UpserterIdempotent:       {"Upsert": shapeValueOp, fRead: shapeKeyedRead},
	lawid.ValidTransition:          {fWrite: shapeValueOp},

	lawid.AppendOnlyGrows:          {fReplay: shapeReplay},
	lawid.AppendOnlyNoDrops:        {fReplay: shapeReplay},
	lawid.ReplayCausalOrdering:     {fReplay: shapeReplay},
	lawid.HashChainIntegrityVerify: {"Verify": shapeErrOp},
	lawid.ReplayDeterministic:      {fReplay: shapeReplay},

	lawid.TamperEvident:         {fWrite: shapeValueOp, "Tamper": shapeOkOp, "Verify": shapeErrOp},
	lawid.CursorCloseIdempotent: {fClose: shapeErrOp},
	lawid.CursorNextAfterClose:  {fClose: shapeErrOp, fNext: shapeNextOp},

	lawid.LeaseReleasedOnCancel: {"Acquire": shapeCtxKeyedOp},

	lawid.PublisherDelivers:    {fSubscribe: shapeSubscribe, fPublish: shapeValueOp},
	lawid.PublisherAtLeastOnce: {fSubscribe: shapeSubscribe, fPublish: shapeValueOp, fRedeliver: shapeValueOp},
	lawid.PublisherAtMostOnce:  {fSubscribe: shapeSubscribe, fPublish: shapeValueOp, fRedeliver: shapeValueOp},
	lawid.PublisherExactlyOnce: {fSubscribe: shapeSubscribe, fPublish: shapeValueOp, fRedeliver: shapeValueOp},

	lawid.TwoPhaseMutex: {
		fBegin: shapeHandleCall, fCommit: shapeHandleOp, fRollback: shapeHandleOp,
	},
	lawid.TwoPhaseRollbackAfterCommit: {
		fBegin: shapeHandleCall, fCommit: shapeHandleOp, fRollback: shapeHandleOp,
	},
	lawid.SagaFullCompensation:  {fRun: shapeSagaRun},
	lawid.SingleflightCoalesces: {fCall: shapeComputeCall},
	lawid.TransactionRollback: {
		fRun: shapeBodyRun, fRead: shapeKeyedRead, fWrite: shapeKeyedWrite,
	},
	lawid.PaginatorNoDuplicates: {fPage: shapePageRead},
	lawid.PaginatorResumable:    {fPage: shapePageRead},

	lawid.WatcherReturnsOnChange: {fWatch: shapeKeyedHandle, "Mutate": shapeKeyedWrite},

	lawid.TransactionNoMidTxVisibility: {
		fBegin: shapeHandleCall, fTxPut: shapeHandleWrite,
		"TxRollback": shapeHandleOp, fRead: shapeKeyedRead,
	},

	lawid.EventualConvergence: {
		fWrite: shapeValueOp, "Sync": shapePeerSync, "Settle": shapeEachSettle,
	},
	lawid.IdempotentLifecycle: {fCall: shapeErrOp},
	lawid.LifecycleAfterClose: {fClose: shapeErrOp, "Op": shapeErrOp},
	lawid.PoisonConsistent:    {"Poison": shapeDoOp, fProbe: shapeErrOp},

	lawid.TTLExpiry:                  {"Put": shapePinnedWrite, fRead: shapeKeyedRead},
	lawid.DeadlineRespecting:         {"Op": shapeCtxOpFixed},
	lawid.ScheduledFiresAfterAdvance: {"Schedule": shapeScheduleAt, "FiredCount": shapeCountObs},
	lawid.TimeawareMoves:             {fRead: shapeKeyedRead},
	lawid.Windowed:                   {"Incr": shapeKeyedOp, fCount: shapeKeyedRead},
}
