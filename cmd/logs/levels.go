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

package logs

import (
	"math"
	"strings"
)

type level struct {
	name  string
	value int
}

var (
	fatalLevel   = level{"FATAL", 100}
	errorLevel   = level{"ERROR", 200}
	warnLevel    = level{"WARN", 300}
	infoLevel    = level{"INFO", 400}
	debugLevel   = level{"DEBUG", 500}
	traceLevel   = level{"TRACE", 600}
	unknownLevel = level{"UNKNOWN", math.MaxInt32}
	allLevels    = []level{fatalLevel, errorLevel, warnLevel, infoLevel, debugLevel, traceLevel, unknownLevel}
)

func validLevel(level string) bool {
	return findLevel(level) != nil
}

func findLevel(level string) *level {
	lowerCaseLevel := strings.ToUpper(level)
	for _, l := range allLevels {
		if l.name == lowerCaseLevel {
			return &l
		}
	}
	return nil
}

func findLowerOrEqualLevels(level string) []string {
	foundLevel := findLevel(level)
	if foundLevel == nil {
		return nil
	}
	var lowerOrEqualLevels []string
	for _, l := range allLevels {
		if l.value <= foundLevel.value {
			lowerOrEqualLevels = append(lowerOrEqualLevels, l.name)
		}
	}
	return lowerOrEqualLevels
}

func allLevelsNames() []string {
	var levelNames []string
	for _, l := range allLevels {
		levelNames = append(levelNames, l.name)
	}
	return levelNames
}
