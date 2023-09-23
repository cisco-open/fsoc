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

package config

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
)

var ErrProfileNotFound = errors.New("profile not found")

type ErrSubsystemConfig struct {
	Errors []error
}

func (e *ErrSubsystemConfig) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}

	texts := make([]string, len(e.Errors))
	for i, err := range e.Errors {
		texts[i] = fmt.Sprintf("\t- %v", err)
	}
	slices.Sort(texts)
	return fmt.Sprintf("%d error(s) in subsystem configuration:\n%v", len(e.Errors), strings.Join(texts, "\n"))
}

func (e *ErrSubsystemConfig) WrappedErrors() []error {
	return e.Errors
}

type ErrSubsystemParsingError struct {
	SubsystemName string
	ParsingError  error
}

func (e *ErrSubsystemParsingError) Error() string {
	// convert potentially multiline error output to a single line, replacing '\n' with '|' for better logging
	errText := "(unknown)"
	if me, ok := e.ParsingError.(*mapstructure.Error); ok {
		errlist := me.WrappedErrors()
		errTexts := []string{}
		if len(errlist) > 0 {
			for _, err := range errlist {
				errTexts = append(errTexts, err.Error())
			}
			errText = strings.Join(errTexts, " | ")
		}
	}

	return fmt.Sprintf("failed to parse configuration for subsystem %q: %v", e.SubsystemName, errText)
}

func (e *ErrSubsystemParsingError) Unwrap() error {
	return e.ParsingError
}

type ErrSubsystemNotFound struct {
	SubsystemName string
}

func (e *ErrSubsystemNotFound) Error() string {
	return fmt.Sprintf("unknown subsystem %q", e.SubsystemName)
}

type ErrSubsystemSettingNotFound struct {
	SubsystemName string
	SettingName   string
}

func (e *ErrSubsystemSettingNotFound) Error() string {
	return fmt.Sprintf("unknown setting %q for subsystem %q", e.SettingName, e.SubsystemName)
}
