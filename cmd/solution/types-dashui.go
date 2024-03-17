// Copyright 2024 Cisco Systems, Inc.
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

package solution

type DashuiTemplate struct {
	Kind    string      `json:"kind" yaml:"kind"`
	Name    string      `json:"name" yaml:"name"`
	Target  string      `json:"target" yaml:"target"`
	View    string      `json:"view" yaml:"view"`
	Element interface{} `json:"element" yaml:"element"`
}

type DashuiTemplatePropsExtension struct {
	Kind                string      `json:"kind" yaml:"kind"`
	Id                  string      `json:"id" yaml:"id"`
	Name                string      `json:"name" yaml:"name"`
	View                string      `json:"view" yaml:"view"`
	Target              string      `json:"target" yaml:"target"`
	RequiredEntityTypes []string    `json:"requiredEntityTypes" yaml:"requiredEntityTypes"`
	Props               interface{} `json:"props" yaml:"props"`
}

type DashuiWidget struct {
	InstanceOf interface{} `json:"instanceOf" yaml:"instanceOf"`
	Props      interface{} `json:"props,omitempty" yaml:"props,omitempty"`
	Elements   interface{} `json:"elements,omitempty" yaml:"elements,omitempty"`
	Element    interface{} `json:"element,omitempty" yaml:"element,omitempty"`
}

type DashuiFocusedEntity struct {
	*DashuiWidget
	Mode string `json:"mode" yaml:"mode"`
}

type DashuiString struct {
	Content    string `json:"content" yaml:"content"`
	InstanceOf string `json:"instanceOf" yaml:"instanceOf"`
}

type DashuiLabel struct {
	InstanceOf interface{} `json:"instanceOf" yaml:"instanceOf"`
	Path       interface{} `json:"path" yaml:"path"`
}

type DashuiProperties struct {
	InstanceOf interface{}       `json:"instanceOf" yaml:"instanceOf"`
	Elements   []*DashuiProperty `json:"elements" yaml:"elements"`
}
type DashuiProperty struct {
	Label *DashuiString `json:"label" yaml:"label"`
	Value *DashuiLabel  `json:"value" yaml:"value"`
}

type DashuiGrid struct {
	*DashuiWidget
	RowSets          interface{}         `json:"rowSets" yaml:"rowSets"`
	Style            interface{}         `json:"style,omitempty" yaml:"style,omitempty"`
	Mode             string              `json:"mode" yaml:"mode"`
	Columns          []*DashuiGridColumn `json:"columns" yaml:"columns"`
	OnRowSingleClick *DashuiEvent        `json:"onRowSingleClick,omitempty" yaml:"onRowSingleClick,omitempty"`
	OnRowDoubleClick *DashuiEvent        `json:"onRowDoubleClick,omitempty" yaml:"onRowDoubleClick,omitempty"`
}

type DashuiGridColumn struct {
	Label string          `json:"label" yaml:"label"`
	Flex  int             `json:"flex" yaml:"flex"`
	Width int             `json:"width" yaml:"width"`
	Cell  *DashuiGridCell `json:"cell" yaml:"cell"`
}

type DashuiGridCell struct {
	Default interface{} `json:"default,omitempty" yaml:"default,omitempty"`
}

type DashuiTooltip struct {
	*DashuiWidget
	Truncate bool        `json:"truncate,omitempty" yaml:"truncate,omitempty"`
	Trigger  interface{} `json:"trigger,omitempty" yaml:"trigger,omitempty"`
}

type DashuiClickable struct {
	*DashuiWidget
	OnClick *DashuiEvent `json:"onClick,omitempty" yaml:"onClick,omitempty"`
	Trigger *DashuiLabel `json:"trigger,omitempty" yaml:"trigger,omitempty"`
}

type DashuiEvent struct {
	Type       string   `json:"type" yaml:"type"`
	Paths      []string `json:"paths,omitempty" yaml:"paths,omitempty"`
	Expression string   `json:"expression" yaml:"expression"`
}

type EcpLeftBar struct {
	*DashuiWidget
	Label string `json:"label" yaml:"label"`
}

type EcpRelationshipMapEntry struct {
	Key             string `json:"key" yaml:"key"`
	Path            string `json:"path" yaml:"path"`
	EntityAttribute string `json:"entityAttribute" yaml:"entityAttribute"`
	IconName        string `json:"iconName" yaml:"iconName"`
}

type EcpInspectorWidget struct {
	*DashuiWidget
	Title string `json:"title" yaml:"title"`
}

type EcpHome struct {
	Sections []*DashuiEcpHomeSection `json:"sections" yaml:"sections"`
	Entities []*DashuiEcpHomeEntity  `json:"entities" yaml:"entities"`
}

type DashuiEcpHomeSection struct {
	Index int    `json:"index" yaml:"index"`
	Name  string `json:"name" yaml:"name"`
	Title string `json:"title" yaml:"title"`
}

type DashuiEcpHomeEntity struct {
	Index           int    `json:"index" yaml:"index"`
	Section         string `json:"section" yaml:"section"`
	EntityAttribute string `json:"entityAttribute" yaml:"entityAttribute"`
	TargetType      string `json:"targetType" yaml:"targetType"`
}

type DashuiOcpSingle struct {
	*DashuiWidget
	NameAttribute string `json:"nameAttribute" yaml:"nameAttribute"`
}

type DashuiCartesian struct {
	*DashuiWidget
	Children []*DashuiCartesianSeries `json:"children" yaml:"children"`
}

type DashuiCartesianSeries struct {
	Props  interface{}            `json:"props" yaml:"props"`
	Metric *DashuiCartesianMetric `json:"metric" yaml:"metric"`
	Type   string                 `json:"type" yaml:"type"`
}

type DashuiCartesianMetric struct {
	Name   string               `json:"name" yaml:"name"`
	Source string               `json:"source" yaml:"source"`
	Y      *DashuiCartesianAxis `json:"y" yaml:"y"`
}

type DashuiCartesianAxis struct {
	Field string `json:"type" yaml:"type"`
}

type DashuiLogsWidget struct {
	InstanceOf interface{} `json:"instanceOf" yaml:"instanceOf"`
	Source     string      `json:"source" yaml:"source"`
}

type DashuiHtmlWidget struct {
	*DashuiWidget
	Style interface{} `json:"style,omitempty" yaml:"style,omitempty"`
}
