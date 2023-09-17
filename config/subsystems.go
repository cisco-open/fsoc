// Copyright 2023 Cisco Systems, Inc.
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
	"fmt"

	"github.com/apex/log"
	"github.com/mitchellh/mapstructure"
)

type ErrSubsystemParsingError struct {
	SubsystemName string
	ParsingError  error
}

func (e *ErrSubsystemParsingError) Error() string {
	return fmt.Sprintf("failed to parse configuration for subsystem %q: %v", e.SubsystemName, e.ParsingError)
}

func (e *ErrSubsystemParsingError) Unwrap() error {
	return e.ParsingError
}

type ErrSubsystemNotFound struct {
	SubsystemName string
}

func (e *ErrSubsystemNotFound) Error() string {
	return fmt.Sprintf("found configuration for unknown subsystem %q", e.SubsystemName)
}

// UpdateSubsystemConfigs updates the subsystem-specific configurations from
// the config file into target structures. If update fails for a subsystem, an
// error for it will be recorded and updates to other subsystem configurations
// continue. This allows callers to ignore subsystems with failed configuration
// while still getting configs for correctly configured systems.
// Returns nil or a slice of errors (the slice, if not nil, will never be empty)
func UpdateSubsystemConfigs(ctx *Context, subsystemConfigs map[string]any) []error {
	errlist := []error{}
	for name, config := range subsystemConfigs {
		configStruct, ok := subsystemConfigs[name]
		if !ok {
			err := &ErrSubsystemNotFound{name}
			log.Error(err.Error())
			errlist = append(errlist, err)
			continue
		}

		parseErr := mapstructure.Decode(config, &configStruct)
		if parseErr != nil {
			err := &ErrSubsystemParsingError{name, parseErr}
			log.Error(err.Error())
			errlist = append(errlist, err)
		}

		// log successful configuration
		fields := log.Fields{
			"subsystem": name,
			"config":    config,
		}
		log.WithFields(fields).Info("Updated subsystem configuration")
	}

	if len(errlist) > 0 {
		return errlist
	}
	return nil
}
