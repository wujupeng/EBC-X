package graph

import (
	"math"
	"sync"
	"time"
)

type Mode string

const (
	ModeNormal   Mode = "normal"
	ModeDegraded Mode = "degraded"
	ModeRecovery Mode = "recovery"
	ModeRebuild  Mode = "rebuild"
)

const (
	MetricEventLag              = "event_lag"
	MetricProjectionLag         = "projection_lag"
	MetricConsumerLag           = "consumer_lag"
	MetricRebuildDuration       = "rebuild_duration"
	MetricFailedProjectionCount = "failed_projection_count"
	MetricDLQCount              = "dlq_count"
)

const (
	NormalModeProjectionLagThresholdMs int64 = 3000
	DegradedModeAlertWindowSeconds     int64 = 60
	NormalModeEventLagThresholdMs      int64 = 1000
	ConsumerLagThreshold                     = 1000
	RebuildRTOThresholdMinutes         int   = 30
)

type MetricsCollector struct {
	mu                    sync.Mutex
	mode                  Mode
	projectionLagSamples  []int64
	eventLagSamples       []int64
	consumerLagSamples    []int64
	rebuildDurationMs     int64
	failedProjectionCount int64
	dlqCount              int64
	alerts                []Alert
}

type Alert struct {
	Timestamp time.Time
	Mode      Mode
	Metric    string
	Message   string
	Severity  string
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{mode: ModeNormal}
}

func (m *MetricsCollector) RecordProjectionLag(lagMs int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.projectionLagSamples = append(m.projectionLagSamples, lagMs)
	if m.mode == ModeNormal && m.p95ProjectionLag() > NormalModeProjectionLagThresholdMs {
		m.alerts = append(m.alerts, Alert{
			Timestamp: time.Now(),
			Mode:      ModeNormal,
			Metric:    MetricProjectionLag,
			Message:   "projection_lag P95 > 3s in Normal Mode",
			Severity:  "warning",
		})
		m.mode = ModeDegraded
	}
}

func (m *MetricsCollector) RecordEventLag(lagMs int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventLagSamples = append(m.eventLagSamples, lagMs)
}

func (m *MetricsCollector) RecordConsumerLag(lag int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.consumerLagSamples = append(m.consumerLagSamples, lag)
	if lag > ConsumerLagThreshold {
		m.alerts = append(m.alerts, Alert{
			Timestamp: time.Now(),
			Mode:      m.mode,
			Metric:    MetricConsumerLag,
			Message:   "consumer_lag > 1000",
			Severity:  "warning",
		})
	}
}

func (m *MetricsCollector) RecordRebuildDuration(durationMs int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rebuildDurationMs = durationMs
	if durationMs > int64(RebuildRTOThresholdMinutes*60*1000) {
		m.alerts = append(m.alerts, Alert{
			Timestamp: time.Now(),
			Mode:      ModeRebuild,
			Metric:    MetricRebuildDuration,
			Message:   "rebuild_duration > 30min RTO threshold",
			Severity:  "critical",
		})
	}
}

func (m *MetricsCollector) IncrementFailedProjection() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failedProjectionCount++
	m.alerts = append(m.alerts, Alert{
		Timestamp: time.Now(),
		Mode:      m.mode,
		Metric:    MetricFailedProjectionCount,
		Message:   "projection failed",
		Severity:  "warning",
	})
}

func (m *MetricsCollector) IncrementDLQ() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dlqCount++
	m.alerts = append(m.alerts, Alert{
		Timestamp: time.Now(),
		Mode:      m.mode,
		Metric:    MetricDLQCount,
		Message:   "event moved to DLQ",
		Severity:  "critical",
	})
}

func (m *MetricsCollector) SetMode(mode Mode) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mode = mode
}

func (m *MetricsCollector) Mode() Mode {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.mode
}

func (m *MetricsCollector) p95ProjectionLag() int64 {
	if len(m.projectionLagSamples) == 0 {
		return 0
	}
	sorted := make([]int64, len(m.projectionLagSamples))
	copy(sorted, m.projectionLagSamples)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	idx := int(math.Ceil(0.95*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	return sorted[idx]
}

func (m *MetricsCollector) P95ProjectionLag() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.p95ProjectionLag()
}

func (m *MetricsCollector) Snapshot() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	return map[string]any{
		"mode":                      string(m.mode),
		MetricProjectionLag:         m.p95ProjectionLag(),
		MetricEventLag:              m.eventLagSamples,
		MetricConsumerLag:           m.consumerLagSamples,
		MetricRebuildDuration:       m.rebuildDurationMs,
		MetricFailedProjectionCount: m.failedProjectionCount,
		MetricDLQCount:              m.dlqCount,
		"alerts":                    m.alerts,
	}
}

func (m *MetricsCollector) Alerts() []Alert {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.alerts
}

func (m *MetricsCollector) DLQCount() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dlqCount
}

func (m *MetricsCollector) FailedProjectionCount() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.failedProjectionCount
}
