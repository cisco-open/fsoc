package melt

// AggregationTemporality - aggretation temporality
type AggregationTemporality int8

// SpanKind - spand kinds
type SpanKind int8

// SpanStatusCode - spand status code
type SpanStatusCode int8

const (
	// AggregationTemporalityUnspecified -  temporality unspecified
	AggregationTemporalityUnspecified AggregationTemporality = 0

	// AggregationTemporalityDelta -  temporality delta
	AggregationTemporalityDelta AggregationTemporality = 1

	// AggregationTemporalityCumulative -  temporality comulative
	AggregationTemporalityCumulative AggregationTemporality = 2

	// SpanKindUnspecified - unspecified
	SpanKindUnspecified SpanKind = 0
	// SpanKindInternal - intermal
	SpanKindInternal SpanKind = 1
	// SpanKindServer - server
	SpanKindServer SpanKind = 2
	// SpanKindClient - client
	SpanKindClient SpanKind = 3
	// SpanKindProducer - producer
	SpanKindProducer SpanKind = 4
	// SpanKindConsumer - consumer
	SpanKindConsumer SpanKind = 5

	// SpanStatusCodeUnset - unset
	SpanStatusCodeUnset SpanStatusCode = 0
	// SpanStatusCodeOK - unset
	SpanStatusCodeOK SpanStatusCode = 1
	// SpanStatusCodeError - unset
	SpanStatusCodeError SpanStatusCode = 2
)

type FsocData struct {
	Melt []*Entity
}

// Entity - type for holding entity inforrmation
type Entity struct {
	TypeName      string
	ID            string `yaml:"id,omitempty"`
	Attributes    map[string]string
	Metrics       []*Metric
	Logs          []*Log
	Relationships []*Relationship
	Spans         []*Span
}

// Metric - structs for metrics
type Metric struct {
	TypeName               string
	ContentType            string
	Unit                   string
	Type                   string
	Resource               Resource `yaml:"resource,omitempty"`
	Attributes             map[string]string
	DataPoints             []*DataPoint           `yaml:"datapoints,omitempty"`
	Min                    string                 `yaml:"min,omitempty"`
	Max                    string                 `yaml:"max,omitempty"`
	IsMonotonic            bool                   `yaml:"ismonotonic,omitempty"`
	AggregationTemporality AggregationTemporality `yaml:"aggregationtemporality,omitempty"`
}

// Log - structs for logs“
type Log struct {
	Resource   Resource `yaml:"resource,omitempty"`
	Body       string
	Severity   string // use for log
	Timestamp  int64  `yaml:"timestamp,omitempty"`
	Attributes map[string]string
	IsEvent    bool   `yaml:"isevent,omitempty"`
	TypeName   string `yaml:"typename,omitempty"`
}

// Resource - structs for data point
type Resource struct {
	TypeName   string
	Attributes map[string]string
}

// Span - structs for a span
type Span struct {
	TraceID      string
	SpanID       string
	TraceState   string
	ParentSpanID string
	Name         string
	Kind         SpanKind
	StartTime    int64
	EndTime      int64
	Attributes   map[string]string
	Events       []*SpanEvent
	Links        []*SpanLink
	Status       *SpanStatus
}

// SpanEvent - event for span
type SpanEvent struct {
	Name       string
	Timestamp  int64
	Attributes map[string]string
}

// SpanLink - link for span
type SpanLink struct {
	TraceID    string
	SpanID     string
	TraceState string
	Attributes map[string]string
}

// SpanStatus - status for span
type SpanStatus struct {
	Message string
	Code    SpanStatusCode
}

// DataPoint - structs for data point
type DataPoint struct {
	StartTime int64
	EndTime   int64
	Value     float64
}

// Relationship - structs for holding relationship info
type Relationship struct {
	Attributes map[string]string
}

// NewEntity - Returns a new entity
func NewEntity(typeName string) *Entity {
	return &Entity{
		TypeName:      typeName,
		Metrics:       []*Metric{},
		Attributes:    map[string]string{},
		Relationships: []*Relationship{},
		Spans:         []*Span{},
	}
}

// AddMetric - add a metric
func (e *Entity) AddMetric(m *Metric) *Entity {
	e.Metrics = append(e.Metrics, m)
	return e
}

// ClearMetrics - clear the metrics
func (e *Entity) ClearMetrics() *Entity {
	e.Metrics = []*Metric{}
	return e
}

// AddLog - add a log/event
func (e *Entity) AddLog(l *Log) *Entity {
	e.Logs = append(e.Logs, l)
	return e
}

// ClearLogs - clear the logs
func (e *Entity) ClearLogs() *Entity {
	e.Logs = []*Log{}
	return e
}

// AddSpan - add a span
func (e *Entity) AddSpan(l *Span) *Entity {
	e.Spans = append(e.Spans, l)
	return e
}

// AddRelationship - add a Relationship
func (e *Entity) AddRelationship(r *Relationship) *Entity {
	e.Relationships = append(e.Relationships, r)
	return e
}

// NewMetric - Returns a new metric
func NewMetric(typeName string, unit string, contentType string, metricType string) *Metric {
	return &Metric{
		TypeName:               typeName,
		Unit:                   unit,
		ContentType:            contentType,
		Type:                   metricType,
		Attributes:             map[string]string{},
		DataPoints:             []*DataPoint{},
		AggregationTemporality: AggregationTemporalityUnspecified,
	}
}

// NewLog - Returns a new log
func NewLog() *Log {
	return &Log{
		Attributes: map[string]string{},
		IsEvent:    false,
	}
}

// NewEvent - Returns a new event
func NewEvent(typeName string) *Log {
	return &Log{
		Attributes: map[string]string{},
		TypeName:   typeName,
		IsEvent:    true,
	}
}

// NewRelationship - Returns a new Relationship
func NewRelationship() *Relationship {
	return &Relationship{
		Attributes: map[string]string{},
	}
}

// NewSpan - Returns a new span
func NewSpan(traceID, spanID, name string) *Span {
	return &Span{
		SpanID:     spanID,
		TraceID:    traceID,
		Attributes: map[string]string{},
		Name:       name,
	}
}

// SetAttribute - Set an attribute
func (e *Entity) SetAttribute(key, value string) *Entity {
	e.Attributes[key] = value
	return e
}

// SetAttribute - Set an attribute on metric
func (m *Metric) SetAttribute(key, value string) *Metric {
	m.Attributes[key] = value
	return m
}

// SetAttribute - Set an attribute on log
func (l *Log) SetAttribute(key, value string) *Log {
	l.Attributes[key] = value
	return l
}

// SetAttribute - Set an attribute on log
func (r *Relationship) SetAttribute(key, value string) *Relationship {
	r.Attributes[key] = value
	return r
}

// SetAttribute - Set an attribute on log
func (s *Span) SetAttribute(key, value string) *Span {
	s.Attributes[key] = value
	return s
}

// NewEvent - add a new event to span
func (s *Span) NewEvent(name string, timeStamp int64) *SpanEvent {
	e := &SpanEvent{
		Name:       name,
		Timestamp:  timeStamp,
		Attributes: map[string]string{},
	}
	s.Events = append(s.Events, e)
	return e
}

// SetAttribute - Set an attribute on span event
func (s *SpanEvent) SetAttribute(key, value string) *SpanEvent {
	s.Attributes[key] = value
	return s
}

// NewLink - add a new link on span
func (s *Span) NewLink(traceID, spanID, traceState string) *SpanLink {
	l := &SpanLink{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceState: traceState,
		Attributes: map[string]string{},
	}
	s.Links = append(s.Links, l)
	return l
}

// SetAttribute - Set an attribute on span link
func (s *SpanLink) SetAttribute(key, value string) *SpanLink {
	s.Attributes[key] = value
	return s
}

// SetStatus - set the span status
func (s *Span) SetStatus(message string, code SpanStatusCode) *Span {
	s.Status = &SpanStatus{
		Message: message,
		Code:    code,
	}
	return s
}

// AddDataPoint - Add a data point
func (m *Metric) AddDataPoint(startTime, endTime int64, value float64) *Metric {
	dp := &DataPoint{
		StartTime: startTime,
		EndTime:   endTime,
		Value:     value,
	}
	m.DataPoints = append(m.DataPoints, dp)
	return m
}

// ClearDataPoints - clears the data points
func (m *Metric) ClearDataPoints() {
	m.DataPoints = []*DataPoint{}
}
