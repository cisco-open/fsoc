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

package solution

type DashuiTemplate struct {
	Kind    string      `json:"kind"`
	Name    string      `json:"name"`
	Target  string      `json:"target"`
	View    string      `json:"view"`
	Element interface{} `json:"element"`
}

type DashuiTemplatePropsExtension struct {
	Kind                string      `json:"kind"`
	Id                  string      `json:"id"`
	Name                string      `json:"name"`
	View                string      `json:"view"`
	Target              string      `json:"target"`
	RequiredEntityTypes []string    `json:"requiredEntityTypes"`
	Props               interface{} `json:"props"`
}

type DashuiWidget struct {
	InstanceOf interface{} `json:"instanceOf"`
	Props      interface{} `json:"props,omitempty"`
	Elements   interface{} `json:"elements,omitempty"`
	Element    interface{} `json:"element,omitempty"`
}

type DashuiFocusedEntity struct {
	*DashuiWidget
	Mode string `json:"mode"`
}

type DashuiString struct {
	InstanceOf string `json:"instanceOf"`
	Content    string `json:"content"`
}

type DashuiLabel struct {
	InstanceOf interface{} `json:"instanceOf"`
	Path       interface{} `json:"path"`
}

type DashuiProperties struct {
	InstanceOf interface{}       `json:"instanceOf"`
	Elements   []*DashuiProperty `json:"elements"`
}
type DashuiProperty struct {
	Label *DashuiString `json:"label"`
	Value *DashuiLabel  `json:"value"`
}

type DashuiGrid struct {
	*DashuiWidget
	RowSets          interface{}         `json:"rowSets"`
	Style            interface{}         `json:"style,omitempty"`
	Mode             string              `json:"mode"`
	Columns          []*DashuiGridColumn `json:"columns"`
	OnRowSingleClick *DashuiEvent        `json:"onRowSingleClick,omitempty"`
	OnRowDoubleClick *DashuiEvent        `json:"onRowDoubleClick,omitempty"`
}

type DashuiGridColumn struct {
	Label string          `json:"label"`
	Flex  int             `json:"flex"`
	Width int             `json:"width"`
	Cell  *DashuiGridCell `json:"cell"`
}

type DashuiGridCell struct {
	Default interface{} `json:"default,omitempty"`
}

type DashuiTooltip struct {
	*DashuiWidget
	Truncate bool        `json:"truncate,omitempty"`
	Trigger  interface{} `json:"trigger,omitempty"`
}

type DashuiClickable struct {
	*DashuiWidget
	OnClick *DashuiEvent `json:"onClick,omitempty"`
	Trigger *DashuiLabel `json:"trigger,omitempty"`
}

type DashuiEvent struct {
	Type       string   `json:"type"`
	Paths      []string `json:"paths,omitempty"`
	Expression string   `json:"expression"`
}

type EcpLeftBar struct {
	*DashuiWidget
	Label string `json:"label"`
}

type EcpRelationshipMapEntry struct {
	Key             string `json:"key"`
	Path            string `json:"path"`
	EntityAttribute string `json:"entityAttribute"`
	IconName        string `json:"iconName"`
}

type EcpInspectorWidget struct {
	*DashuiWidget
	Title string `json:"title"`
}

type EcpHome struct {
	Sections []*DashuiEcpHomeSection `json:"sections"`
	Entities []*DashuiEcpHomeEntity  `json:"entities"`
}

type DashuiEcpHomeSection struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
	Title string `json:"title"`
}

type DashuiEcpHomeEntity struct {
	Index           int    `json:"index"`
	Section         string `json:"section"`
	EntityAttribute string `json:"entityAttribute"`
	TargetType      string `json:"targetType"`
}

type DashuiOcpSingle struct {
	*DashuiWidget
	NameAttribute string `json:"nameAttribute"`
}

type DashuiCartesian struct {
	*DashuiWidget
	Children []*DashuiCartesianSeries `json:"children"`
}

type DashuiCartesianSeries struct {
	Props  interface{}            `json:"props"`
	Metric *DashuiCartesianMetric `json:"metric"`
	Type   string                 `json:"type"`
}

type DashuiCartesianMetric struct {
	Name   string               `json:"name"`
	Source string               `json:"source"`
	Y      *DashuiCartesianAxis `json:"y"`
}

type DashuiCartesianAxis struct {
	Field string `json:"type"`
}

type DashuiLogsWidget struct {
	InstanceOf interface{} `json:"instanceOf"`
	Source     string      `json:"source"`
}

type DashuiHtmlWidget struct {
	*DashuiWidget
	Style interface{} `json:"style,omitempty"`
}
