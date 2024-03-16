package melt

import (
	"testing"

	yaml "gopkg.in/yaml.v2"
)

func TestAggregationTemporalityParsing(t *testing.T) {

	validYamlStrings := map[string]AggregationTemporality{
		"aggregationtemporality: UNSPECIFIED":                         AggregationTemporalityUnspecified,
		"aggregationtemporality: delta":                               AggregationTemporalityDelta,
		"aggregationtemporality: Cumulative":                          AggregationTemporalityCumulative,
		"aggregationtemporality: AGGREGATION_TEMPORALITY_UNSPECIFIED": AggregationTemporalityUnspecified,
		"aggregationtemporality: AGGREGATION_TEMPORALITY_DELTA":       AggregationTemporalityDelta,
		"aggregationtemporality: AGGREGATION_TEMPORALITY_CUMULATIVE":  AggregationTemporalityCumulative,
	}
	validYamlNumbers := map[string]AggregationTemporality{
		"aggregationtemporality: 0": AggregationTemporalityUnspecified,
		"aggregationtemporality: 1": AggregationTemporalityDelta,
		"aggregationtemporality: 2": AggregationTemporalityCumulative,
	}
	invalidYaml := []string{
		"aggregationtemporality: INVALID",
		"aggregationtemporality: -1",
		"aggregationtemporality: 3",
		"aggregationtemporality: 1.5",
	}

	var temp Metric

	for text, temporality := range validYamlStrings {
		err := yaml.Unmarshal([]byte(text), &temp)
		if err != nil {
			t.Errorf("Failed to parse valid YAML %q: %v", text, err)
		}
		if temp.AggregationTemporality != temporality {
			t.Errorf("Expected %v, got %v for YAML %q", temporality, temp.AggregationTemporality, text)
		}
	}

	for text, temporality := range validYamlNumbers {
		err := yaml.Unmarshal([]byte(text), &temp)
		if err != nil {
			t.Errorf("Failed to parse valid YAML %q: %v", text, err)
		}
		if temp.AggregationTemporality != temporality {
			t.Errorf("Expected %v, got %v for YAML %q", temporality, temp.AggregationTemporality, text)
		}
	}

	for _, text := range invalidYaml {
		err := yaml.Unmarshal([]byte(text), &temp)
		if err == nil {
			t.Errorf("Expected error when parsing invalid YAML %q, got nil", text)
		}
	}
}
