// Copyright 2022 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package melt

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/apex/log"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/platform/melt"
)

// geometryTestCmd represents the report command
var geometryTestCmd = &cobra.Command{
	Use:              "geometry-test",
	Short:            "Send geometry test MELT data",
	Long:             `Send geometry test MELT data.`,
	Example:          `  fsoc melt geometry-test`,
	Args:             cobra.ExactArgs(0),
	Run:              test,
	TraverseChildren: true,
}

func init() {
	meltCmd.AddCommand(geometryTestCmd)
}

func test(cmd *cobra.Command, args []string) {
	cmd.Println("-----------------------------------------------------------------------------------------")
	cmd.Println("Exporting metrics...")
	err1 := exportMetrics()
	if err1 != nil {
		log.Errorf("Error exporting metrics: %s", err1)
	}

	cmd.Println("-----------------------------------------------------------------------------------------")
	cmd.Println("Exporting relationship...")
	err2 := exportRelationships()
	if err2 != nil {
		log.Errorf("Error exporting relatioships: %s", err2)
	}

	cmd.Println("-----------------------------------------------------------------------------------------")
	cmd.Println("Exporting logs...")
	err3 := exportLogs()
	if err2 != nil {
		log.Errorf("Error exporting logs: %s", err3)
	}

	cmd.Println("-----------------------------------------------------------------------------------------")
	cmd.Println("Exporting events...")
	err4 := exportEvents()
	if err4 != nil {
		log.Errorf("Error exporting events: %s", err4)
	}

	cmd.Println("-----------------------------------------------------------------------------------------")
	cmd.Println("Exporting spans...")
	err5 := exportSpans()
	if err5 != nil {
		log.Errorf("Error exporting spans: %s", err5)
	}

}

func exportMetrics() error {
	el := []*melt.Entity{}

	// create an entity
	e1 := melt.NewEntity("geometry:square")
	e1.SetAttribute("geometry.shape.name", "Square entity1")
	e1.SetAttribute("geometry.shape.type", "square")
	e1.SetAttribute("geometry.square.side", "10")
	e1.SetAttribute("telemetry.sdk.name", "appd-datagen")

	et := time.Now()
	st := et.Add(time.Minute * -1)

	// add a gauge metric to the entity
	m1 := melt.NewMetric("geometry:gauge", "count", "gauge", "double")
	m1.AddDataPoint(st.UnixNano(), et.UnixNano(), map[string]interface{}{}, rand.Float64()*5)
	e1.AddMetric(m1)

	// add a sum delta metric to the entity
	m2 := melt.NewMetric("geometry:sum_delta", "sum", "sum", "double")
	m2.AddDataPoint(st.UnixNano(), et.UnixNano(), map[string]interface{}{}, rand.Float64()*5)
	m2.AggregationTemporality = melt.AggregationTemporalityDelta
	e1.AddMetric(m2)

	// add a sum cumulative metric to the entity
	m3 := melt.NewMetric("geometry:sum_cumulative", "sum", "sum", "double")
	m3.AddDataPoint(st.UnixNano(), et.UnixNano(), map[string]interface{}{}, rand.Float64()*5)
	m3.IsMonotonic = true
	m3.AggregationTemporality = melt.AggregationTemporalityCumulative
	e1.AddMetric(m3)

	el = append(el, e1)

	// invoke the exporter
	exp := &melt.Exporter{}
	return exp.ExportMetrics(el)
}

func exportLogs() error {
	el := []*melt.Entity{}

	// create an entity
	e1 := melt.NewEntity("geometry:square")
	e1.SetAttribute("geometry.shape.name", "Square 6")
	e1.SetAttribute("geometry.shape.type", "square")
	e1.SetAttribute("geometry.square.side", "10")
	e1.SetAttribute("telemetry.sdk.name", "appd-datagen")

	// add logs tp the entity
	for i := 1; i < 5; i++ {
		l := melt.NewLog()
		l.SetAttribute("level", "debug")
		l.Severity = "INFO"
		l.Body = fmt.Sprintf("hello world-%d", i)
		l.Timestamp = time.Now().UnixNano()
		e1.AddLog(l)
	}
	el = append(el, e1)

	// invoke the exporter
	exp := &melt.Exporter{}
	return exp.ExportLogs(el)
}

func exportEvents() error {
	el := []*melt.Entity{}

	// create an entity
	e1 := melt.NewEntity("geometry:square")
	e1.SetAttribute("geometry.shape.name", "Square 100")
	e1.SetAttribute("geometry.shape.type", "square")
	e1.SetAttribute("geometry.square.side", "10")
	e1.SetAttribute("telemetry.sdk.name", "appd-datagen")

	// add events to the entity
	for i := 1; i < 5; i++ {
		l := melt.NewEvent("geometry:operation")
		l.SetAttribute("type", "draw")
		l.Timestamp = time.Now().UnixNano()
		e1.AddLog(l)
	}
	el = append(el, e1)

	// invoke the exporter
	exp := &melt.Exporter{}
	return exp.ExportLogs(el)
}

func exportSpans() error {
	el := []*melt.Entity{}

	// create an entity
	e1 := melt.NewEntity("geometry:square")
	e1.SetAttribute("geometry.shape.name", "Square 6")
	e1.SetAttribute("geometry.shape.type", "square")
	e1.SetAttribute("geometry.square.side", "10")
	e1.SetAttribute("telemetry.sdk.name", "appd-datagen")

	// add spans
	et := time.Now()
	st := et.Add(time.Minute * -1)

	for i := 1; i < 5; i++ {
		traceID := uuid.New()
		spanID := uuid.New()
		s := melt.NewSpan(traceID.String(), spanID.String(), fmt.Sprintf("span-%d", i))
		s.StartTime = st.UnixNano()
		s.EndTime = et.UnixNano()
		s.Kind = melt.SpanKindClient
		s.SetAttribute("span-attribute-1", "value")
		// events
		s.Events = []*melt.SpanEvent{
			{
				Name:      fmt.Sprintf("span event %d", i),
				Timestamp: et.UnixNano(),
			},
		}
		// links
		traceID1 := uuid.New()
		spanID1 := uuid.New()

		s.Links = []*melt.SpanLink{
			{
				TraceID:    traceID1.String(),
				SpanID:     spanID1.String(),
				TraceState: "state",
				Attributes: map[string]interface{}{
					"span-link-attrbute1": "value",
				},
			},
		}
		e1.AddSpan(s)
	}
	el = append(el, e1)

	// invoke the exporter
	exp := &melt.Exporter{}
	return exp.ExportSpans(el)
}

func exportRelationships() error {
	el := []*melt.Entity{}

	shapeName := "Square VWXYZ"
	shapeType := "square"

	// create edges
	edges := []string{"AB", "BC", "CD", "AD"}
	for _, en := range edges {
		edgeEntity := melt.NewEntity("geometry:edge")
		edgeEntity.SetAttribute("geometry.edge.name", en)
		edgeEntity.SetAttribute("geometry.parent.name", shapeName)
		edgeEntity.SetAttribute("geometry.parent.type", shapeType)
		edgeEntity.SetAttribute("geometry.shape.type", "edge")
		edgeEntity.SetAttribute("telemetry.sdk.name", "appd-datagen")

		m := melt.NewMetric("geometry:dummy", "count", "gauge", "double")
		et := time.Now()
		st := et.Add(time.Minute * -1)
		m.AddDataPoint(st.UnixNano(), et.UnixNano(), map[string]interface{}{}, rand.Float64()*5)
		edgeEntity.AddMetric(m)

		el = append(el, edgeEntity)
	}

	// send square entity
	e1 := melt.NewEntity("geometry:square")
	e1.SetAttribute("geometry.shape.name", shapeName)
	e1.SetAttribute("geometry.shape.type", shapeType)
	e1.SetAttribute("geometry.square.side", "10")
	e1.SetAttribute("telemetry.sdk.name", "appd-datagen")

	// add metric
	m := melt.NewMetric("geometry:dummy", "count", "gauge", "double")
	et := time.Now()
	st := et.Add(time.Minute * -1)
	m.AddDataPoint(st.UnixNano(), et.UnixNano(), map[string]interface{}{}, rand.Float64()*5)
	e1.AddMetric(m)

	// add relationship to edges
	for _, en := range edges {
		rel := melt.NewRelationship()
		rel.SetAttribute("geometry.parent.name", shapeName)
		rel.SetAttribute("geometry.parent.type", shapeType)
		rel.SetAttribute("geometry.edge.name", en)
		rel.SetAttribute("geometry.shape.type", "edge")
		e1.AddRelationship(rel)
	}

	el = append(el, e1)

	// invoke the exporter
	exp := &melt.Exporter{}
	return exp.ExportMetrics(el)
}
