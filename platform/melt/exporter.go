package melt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/apex/log"
	colllogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	collmetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	collspans "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	common "go.opentelemetry.io/proto/otlp/common/v1"
	logs "go.opentelemetry.io/proto/otlp/logs/v1"
	metrics "go.opentelemetry.io/proto/otlp/metrics/v1"
	resource "go.opentelemetry.io/proto/otlp/resource/v1"
	spans "go.opentelemetry.io/proto/otlp/trace/v1"
)

const (
	pathMetrics                   = "metrics"
	pathLogs                      = "logs"
	pathSpans                     = "trace"
	agentName                     = "sample-datagen"
	agentVersion                  = "0.0.1"
	keyAppdFMMEntityRelationships = "appd.fmm.entity.relations"
	keyAppdIsEvent                = "appd.isevent"
	keyAppdEventType              = "appd.event.type"
)

// Exporter -  exporter for entities, metrics and logs
type Exporter struct{}

// ExportMetrics - export metrics
func (exp *Exporter) ExportMetrics(ctx context.Context, req ExportRequest, entities []*Entity) error {
	emsr := exp.buildMetricsPayload(entities, req)

	if emsr.ResourceMetrics == nil {
		log.Info("No metrics to send")
		return nil
	}

	b, _ := json.Marshal(emsr)
	log.Debugf("METRICS: %s", string(b))

	return exp.exportHTTP(ctx, pathMetrics, req, emsr)
}

// ExportLogs - export resource logs
func (exp *Exporter) ExportLogs(ctx context.Context, req ExportRequest, entities []*Entity) error {
	elsr := exp.buildLogsPayload(entities, req)

	if elsr.ResourceLogs == nil {
		log.Info("No logs to send")
		return nil
	}

	b, _ := json.Marshal(elsr)
	log.Debugf("LOGS: %s", string(b))

	return exp.exportHTTP(ctx, pathLogs, req, elsr)
}

// ExportEvents - export events as resource logs
// OTEL does not distibguish between events and logs
func (exp *Exporter) ExportEvents(ctx context.Context, req ExportRequest, entities []*Entity) error {
	return exp.ExportLogs(ctx, req, entities)
}

// ExportSpans - export resource spans
func (exp *Exporter) ExportSpans(ctx context.Context, req ExportRequest, entities []*Entity) error {
	essr := exp.buildSpansPayload(entities, req)

	if essr.ResourceSpans == nil {
		log.Info("No spans to send")
		return nil
	}

	b, _ := json.Marshal(essr)
	log.Debugf("SPANS: %s", string(b))

	return exp.exportHTTP(ctx, pathSpans, req, essr)
}

func (exp *Exporter) buildMetricsPayload(entities []*Entity, req ExportRequest) *collmetrics.ExportMetricsServiceRequest {
	emsr := &collmetrics.ExportMetricsServiceRequest{}

	for _, entity := range entities {

		rm := &metrics.ResourceMetrics{}
		rm.Resource = &resource.Resource{
			Attributes: toKeyValueList(entity.Attributes, req),
		}

		exp.addRelationships(entity.Relationships, rm, req)

		ilm := &metrics.ScopeMetrics{}

		ml := []*metrics.Metric{}
		for _, m := range entity.Metrics {
			otm := exp.createOtelMetric(req, m)

			ml = append(ml, otm)
		}
		ilm.Metrics = ml
		ilm.Scope = getInstrumentationScope()

		ilml := []*metrics.ScopeMetrics{ilm}

		rm.ScopeMetrics = ilml

		// append rresource metrics
		emsr.ResourceMetrics = append(emsr.ResourceMetrics, rm)
	}

	return emsr
}

func (exp *Exporter) addRelationships(rels []*Relationship, rm *metrics.ResourceMetrics, req ExportRequest) {
	// add relationships
	if len(rels) > 0 {
		attrib := &common.KeyValue{
			Key: keyAppdFMMEntityRelationships,
		}
		val := &common.AnyValue_ArrayValue{
			ArrayValue: &common.ArrayValue{
				Values: []*common.AnyValue{},
			},
		}
		for _, r := range rels {
			kvlv := &common.AnyValue{
				Value: &common.AnyValue_KvlistValue{
					KvlistValue: &common.KeyValueList{
						Values: toKeyValueList(r.Attributes, req),
					},
				},
			}
			val.ArrayValue.Values = append(val.ArrayValue.Values, kvlv)
		}
		attrib.Value = &common.AnyValue{
			Value: val,
		}
		rm.Resource.Attributes = append(rm.Resource.Attributes, attrib)
	}
}

func (exp *Exporter) buildLogsPayload(entities []*Entity, req ExportRequest) *colllogs.ExportLogsServiceRequest {
	elsr := &colllogs.ExportLogsServiceRequest{}

	for _, e := range entities {
		if len(e.Logs) == 0 {
			continue
		}

		rl := &logs.ResourceLogs{}

		rl.Resource = &resource.Resource{
			Attributes: toKeyValueList(e.Attributes, req),
		}

		ill := &logs.ScopeLogs{}

		lr := []*logs.LogRecord{}
		for _, l := range e.Logs {
			otl := exp.createOtelLog(l, req)

			lr = append(lr, otl)
		}
		ill.LogRecords = lr
		ill.Scope = getInstrumentationScope()

		illl := []*logs.ScopeLogs{ill}

		rl.ScopeLogs = illl

		// append resource logs
		elsr.ResourceLogs = append(elsr.ResourceLogs, rl)
	}

	return elsr
}

func (exp *Exporter) buildSpansPayload(entities []*Entity, req ExportRequest) *collspans.ExportTraceServiceRequest {
	etsr := &collspans.ExportTraceServiceRequest{}

	for _, e := range entities {
		if len(e.Spans) == 0 {
			continue
		}

		rs := &spans.ResourceSpans{}

		rs.Resource = &resource.Resource{
			Attributes: toKeyValueList(e.Attributes, req),
		}

		sl := []*spans.Span{}
		for _, s := range e.Spans {
			sl = append(sl, exp.createOtelSpan(s, req))
		}
		ss := &spans.ScopeSpans{
			Spans: sl,
			Scope: getInstrumentationScope(),
		}
		ssl := []*spans.ScopeSpans{ss}
		rs.ScopeSpans = ssl

		// append resource logs
		etsr.ResourceSpans = append(etsr.ResourceSpans, rs)
	}

	return etsr
}

func (exp *Exporter) createOtelMetric(req ExportRequest, m *Metric) *metrics.Metric {
	otm := &metrics.Metric{
		Name: m.TypeName,
	}

	switch m.ContentType {
	case "sum":
		mAttribs := toKeyValueList(m.Attributes, req)

		s := &metrics.Sum{
			AggregationTemporality: metrics.AggregationTemporality_AGGREGATION_TEMPORALITY_UNSPECIFIED,
			IsMonotonic:            m.IsMonotonic,
		}
		switch m.AggregationTemporality {
		case AggregationTemporalityDelta:
			s.AggregationTemporality = metrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA
		case AggregationTemporalityCumulative:
			s.AggregationTemporality = metrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE
		}
		for _, dp := range m.DataPoints {
			sdp := &metrics.NumberDataPoint{
				StartTimeUnixNano: uint64(dp.StartTime),
				TimeUnixNano:      uint64(dp.EndTime),
				Attributes:        mAttribs,
			}
			switch m.Type {
			case "long":
				sdp.Value = &metrics.NumberDataPoint_AsInt{AsInt: int64(dp.Value)}
			case "double":
				sdp.Value = &metrics.NumberDataPoint_AsDouble{AsDouble: dp.Value}
			}
			s.DataPoints = append(s.DataPoints, sdp)
		}
		otm.Data = &metrics.Metric_Sum{Sum: s}

		return otm

	case "gauge":
		mAttribs := toKeyValueList(m.Attributes, req)
		s := &metrics.Gauge{}

		for _, dp := range m.DataPoints {
			otdp := &metrics.NumberDataPoint{
				StartTimeUnixNano: uint64(dp.StartTime),
				TimeUnixNano:      uint64(dp.EndTime),
				Attributes:        mAttribs,
			}

			switch m.Type {
			case "long":
				otdp.Value = &metrics.NumberDataPoint_AsInt{AsInt: int64(dp.Value)}
			case "double":
				otdp.Value = &metrics.NumberDataPoint_AsDouble{AsDouble: dp.Value}
			}

			s.DataPoints = append(s.DataPoints, otdp)
		}

		otm.Data = &metrics.Metric_Gauge{Gauge: s}

		return otm
	}

	log.Errorf("unsuported metrics type: %s", m.ContentType)

	return nil
}

func (exp *Exporter) createOtelLog(l *Log, req ExportRequest) *logs.LogRecord {
	// indicators for events
	if l.IsEvent {
		l.Attributes[keyAppdIsEvent] = "true"
		l.Attributes[keyAppdEventType] = l.TypeName
	}

	lAttribs := toKeyValueList(l.Attributes, req)

	otl := &logs.LogRecord{
		Body: &common.AnyValue{
			Value: &common.AnyValue_StringValue{
				StringValue: l.Body,
			},
		},
		TimeUnixNano: uint64(l.Timestamp),
		Attributes:   lAttribs,
	}
	if l.Severity != "" {
		otl.SeverityText = l.Severity
	}

	return otl
}

func (exp *Exporter) createOtelSpan(t *Span, req ExportRequest) *spans.Span {
	ots := &spans.Span{
		Name:              t.Name,
		TraceId:           []byte(t.TraceID),
		SpanId:            []byte(t.SpanID),
		TraceState:        t.TraceState,
		ParentSpanId:      []byte(t.ParentSpanID),
		Kind:              spans.Span_SpanKind(t.Kind),
		StartTimeUnixNano: uint64(t.StartTime),
		EndTimeUnixNano:   uint64(t.EndTime),
		Attributes:        toKeyValueList(t.Attributes, req),
	}

	// events
	for _, e := range t.Events {
		ots.Events = append(ots.Events, &spans.Span_Event{
			TimeUnixNano: uint64(e.Timestamp),
			Name:         e.Name,
			Attributes:   toKeyValueList(e.Attributes, req),
		})
	}

	// links
	for _, l := range t.Links {
		ots.Links = append(ots.Links, &spans.Span_Link{
			TraceId:    []byte(l.TraceID),
			SpanId:     []byte(l.SpanID),
			TraceState: l.TraceState,
			Attributes: toKeyValueList(l.Attributes, req),
		})
	}

	// status
	if t.Status != nil {
		ots.Status = &spans.Status{
			Message: t.Status.Message,
			Code:    spans.Status_StatusCode(t.Status.Code),
		}
	}
	return ots
}

func (exp *Exporter) exportHTTP(ctx context.Context, path string, exportReq ExportRequest, m protoreflect.ProtoMessage) error {
	url := fmt.Sprintf("%s/%s", exportReq.EndPoint, path)
	log.Debugf("Connecting to ingestion endpoint %s\n", url)
	data, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", exportReq.Credentials.Token))
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Accept", "application/x-protobuf")
	req.Header.Set("User-Agent", "spacefleet-datagen")
	client := &http.Client{}
	resp, err := client.Do(req)

	log.Debug("Sending request...")

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Received non-ok response code from CIS %d", resp.StatusCode)
	}

	log.Debugf("Response received code[%d], headers[%s]", resp.StatusCode, resp.Header)

	return nil
}

func toKeyValueList(a map[string]string, req ExportRequest) []*common.KeyValue {
	attribs := []*common.KeyValue{}
	for k, v := range a {
		key := k
		attribs = append(attribs, &common.KeyValue{
			Key: key,
			Value: &common.AnyValue{
				Value: &common.AnyValue_StringValue{
					StringValue: v,
				},
			},
		})
	}
	return attribs
}

func getInstrumentationScope() *common.InstrumentationScope {
	return &common.InstrumentationScope{
		Name:    agentName,
		Version: agentVersion,
	}
}
