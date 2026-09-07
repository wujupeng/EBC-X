package graph

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrShadowRebuildFailed      = errors.New("shadow rebuild failed")
	ErrReconciliationFailed     = errors.New("reconciliation failed — cutover aborted")
	ErrShadowRebuildRTOExceeded = errors.New("shadow rebuild RTO exceeded 30min")
)

const MaxRebuildRTOMinutes = 30

type ShadowRebuildResult struct {
	RebuildID            string
	StartedAt            time.Time
	CompletedAt          time.Time
	SourceInstance       string
	TargetInstance       string
	EventsReplayed       int64
	NodesBuilt           int64
	EdgesBuilt           int64
	ReconciliationResult ReconciliationResult
	CutoverAt            time.Time
	RTOMs                int64
	AvailabilityWindowMs int64
	Err                  error
}

type ShadowRebuilder struct {
	dual    *DualGraph
	rules   *ProjectionRuleRegistry
	metrics *MetricsCollector
	mu      sync.Mutex
}

func NewShadowRebuilder(dual *DualGraph, rules *ProjectionRuleRegistry, metrics *MetricsCollector) *ShadowRebuilder {
	return &ShadowRebuilder{
		dual:    dual,
		rules:   rules,
		metrics: metrics,
	}
}

func (s *ShadowRebuilder) Rebuild(ctx context.Context, events []OutboxEvent, tenantID string) ShadowRebuildResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	started := time.Now()

	if s.metrics != nil {
		s.metrics.SetMode(ModeRebuild)
	}

	shadow := s.dual.Shadow
	shadow.Clear()
	shadow.SetAvailable(true)

	consumer := NewProjectionConsumer(shadow, s.rules, s.metrics)

	for _, evt := range events {
		result := consumer.Consume(ctx, evt)
		if result.Err != nil {
			return ShadowRebuildResult{
				StartedAt:            started,
				CompletedAt:          time.Now(),
				Err:                  result.Err,
				AvailabilityWindowMs: 0,
			}
		}
	}

	recon := Reconcile(s.dual.Primary, s.dual.Shadow, tenantID)

	if !recon.OverallPass {
		return ShadowRebuildResult{
			StartedAt:            started,
			CompletedAt:          time.Now(),
			SourceInstance:       s.dual.PrimaryLabel(),
			TargetInstance:       "B",
			EventsReplayed:       int64(len(events)),
			NodesBuilt:           int64(shadow.NodeCount()),
			EdgesBuilt:           int64(shadow.EdgeCount()),
			ReconciliationResult: recon,
			AvailabilityWindowMs: 0,
			Err:                  ErrReconciliationFailed,
		}
	}

	cutoverStart := time.Now()
	s.dual.Cutover()
	cutoverAt := time.Now()
	cutoverMs := cutoverAt.Sub(cutoverStart).Milliseconds()

	completed := time.Now()
	rtoMs := completed.Sub(started).Milliseconds()

	if rtoMs > MaxRebuildRTOMinutes*60*1000 {
		return ShadowRebuildResult{
			StartedAt:            started,
			CompletedAt:          completed,
			SourceInstance:       s.dual.PrimaryLabel(),
			TargetInstance:       "B",
			EventsReplayed:       int64(len(events)),
			NodesBuilt:           int64(shadow.NodeCount()),
			EdgesBuilt:           int64(shadow.EdgeCount()),
			ReconciliationResult: recon,
			CutoverAt:            cutoverAt,
			RTOMs:                rtoMs,
			AvailabilityWindowMs: cutoverMs,
			Err:                  ErrShadowRebuildRTOExceeded,
		}
	}

	if s.metrics != nil {
		s.metrics.RecordRebuildDuration(rtoMs)
		s.metrics.SetMode(ModeNormal)
	}

	return ShadowRebuildResult{
		StartedAt:            started,
		CompletedAt:          completed,
		SourceInstance:       s.dual.PrimaryLabel(),
		TargetInstance:       "B",
		EventsReplayed:       int64(len(events)),
		NodesBuilt:           int64(shadow.NodeCount()),
		EdgesBuilt:           int64(shadow.EdgeCount()),
		ReconciliationResult: recon,
		CutoverAt:            cutoverAt,
		RTOMs:                rtoMs,
		AvailabilityWindowMs: cutoverMs,
	}
}

func (s *ShadowRebuilder) ActiveGraph() *GraphInstance {
	return s.dual.Active()
}
