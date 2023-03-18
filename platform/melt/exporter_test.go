package melt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	common "go.opentelemetry.io/proto/otlp/common/v1"
	metrics "go.opentelemetry.io/proto/otlp/metrics/v1"
)

// TestBuildMetricsPayload - Test metrics payload
func TestBuildMetricsPayload(t *testing.T) {
	e := newTestEntity()
	el := []*Entity{e}

	tests := []struct {
		name                       string
		contentType                string
		aggregationTemporality     AggregationTemporality
		otlpType                   string
		otlpAggregationTemporality metrics.AggregationTemporality
	}{
		{
			name:                       "gauge_metric",
			contentType:                "gauge",
			otlpType:                   "gauge",
			otlpAggregationTemporality: metrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
		},
		{
			name:                       "sum_metric",
			contentType:                "sum",
			otlpType:                   "sum",
			aggregationTemporality:     AggregationTemporalityCumulative,
			otlpAggregationTemporality: metrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
		},
		{
			name:                       "sum_metric",
			contentType:                "sum",
			otlpType:                   "sum",
			aggregationTemporality:     AggregationTemporalityDelta,
			otlpAggregationTemporality: metrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
		},
	}
	expReq := newExportRequest()
	exp := &Exporter{}
	for _, test := range tests {
		e.ClearMetrics()
		m := NewMetric(test.name, "s", test.contentType, "long")
		if test.aggregationTemporality != AggregationTemporalityUnspecified {
			m.AggregationTemporality = test.aggregationTemporality
		}
		e.AddMetric(m)

		emsr := exp.buildMetricsPayload(el, expReq)
		require.NotNil(t, emsr)
		for _, rm := range emsr.ResourceMetrics {
			r := rm.Resource

			// assert resouce attributes against entity attributes
			assertAttributes(t, e.Attributes, r.Attributes)

			// assert metrics
			for _, sm := range rm.ScopeMetrics {
				for _, m := range sm.Metrics {
					require.Equal(t, test.name, m.Name)
					switch test.otlpType {
					case "gauge":
						require.NotNil(t, m.GetGauge())
					case "sum":
						require.NotNil(t, m.GetSum())
					}
					if test.aggregationTemporality != AggregationTemporalityUnspecified {
						require.Equal(t, test.otlpAggregationTemporality, m.GetSum().AggregationTemporality)
					}
				}
			}
		}
	}
}

func TestBuildLogsPayload(t *testing.T) {
	e := newTestEntity()
	el := []*Entity{e}

	tests := []struct {
		body       string
		severity   string
		timestamp  int64
		attributes map[string]string
	}{
		{"debug log", "debug", time.Now().UnixNano(), map[string]string{"key1": "value1"}},
		{"info log", "debug", time.Now().UnixNano(), map[string]string{"key2": "value2"}},
		{"warn log", "warn", time.Now().UnixNano(), map[string]string{"key3": "value3"}},
		{"error log", "error", time.Now().UnixNano(), map[string]string{"key4": "value4"}},
	}

	expReq := newExportRequest()
	exp := &Exporter{}
	for _, test := range tests {
		e.ClearLogs()

		l := NewLog()
		l.Severity = test.severity
		l.Body = test.body
		l.Timestamp = test.timestamp
		l.Attributes = test.attributes
		elsr := exp.buildLogsPayload(el, expReq)
		require.NotNil(t, elsr)

		for _, rl := range elsr.ResourceLogs {
			r := rl.Resource

			// assert resouce attributes against entity attributes
			assertAttributes(t, e.Attributes, r.Attributes)

			for _, sl := range rl.ScopeLogs {
				lr := sl.LogRecords[0]
				require.Equal(t, test.body, lr.Body)
				require.Equal(t, test.severity, lr.SeverityText)
				require.Equal(t, test.timestamp, lr.TimeUnixNano)
				require.Equal(t, test.body, lr.Body)
				// assert log attributes
				assertAttributes(t, test.attributes, lr.Attributes)
			}
		}
	}
}

func TestBuildEventsPayload(t *testing.T) {
	e := newTestEntity()
	el := []*Entity{e}

	tests := []struct {
		typeName   string
		timestamp  int64
		attributes map[string]string
	}{
		{"mynamespace:event1", time.Now().UnixNano(), map[string]string{"key1": "value1", "key2": "value2"}},
	}

	expReq := newExportRequest()
	exp := &Exporter{}
	for _, test := range tests {
		e.ClearLogs()

		l := NewEvent(test.typeName)
		l.Timestamp = test.timestamp
		l.Attributes = test.attributes
		elsr := exp.buildLogsPayload(el, expReq)
		require.NotNil(t, elsr)

		for _, rl := range elsr.ResourceLogs {
			r := rl.Resource

			// assert resouce attributes against entity attributes
			assertAttributes(t, e.Attributes, r.Attributes)

			for _, sl := range rl.ScopeLogs {
				lr := sl.LogRecords[0]
				require.Equal(t, test.typeName, lookupByKey(keyAppdEventType, lr.Attributes))
				require.Equal(t, "true", lookupByKey(keyAppdIsEvent, lr.Attributes))
				// assert log attributes
				assertAttributes(t, test.attributes, lr.Attributes)
			}
		}
	}
}

func assertAttributes(t *testing.T, expected map[string]string, actual []*common.KeyValue) {
	for k, v := range expected {
		require.Equal(t, v, lookupByKey(k, actual))
	}
}

func newTestEntity() *Entity {
	e := NewEntity("geometry:shape")
	e.SetAttribute("geometry.shape.type", "square")
	e.SetAttribute("geometry.shape.name", "My Square")
	e.SetAttribute("geometry.square.side", "10")
	return e
}

func lookupByKey(key string, kvl []*common.KeyValue) string {
	for _, kv := range kvl {
		if kv.Key == key {
			return kv.GetValue().GetStringValue()
		}
	}
	return ""
}

func newExportRequest() ExportRequest {
	return ExportRequest{
		EndPoint: "https://localhost:8080",
		Credentials: Credentials{
			Token: "dummy",
		},
	}
}
