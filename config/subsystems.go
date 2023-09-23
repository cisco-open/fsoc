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
	"reflect"

	"github.com/apex/log"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/exp/maps"
)

var subsystemConfigs = map[string]any{}

// RegisterSubsystemConfigStorage registers storage (a pointer to a struct) for a subsystem's configuration. In addition
// to using the storage itself, this function uses the structure to introspect it for setting names, types and even
// help strings.
func RegisterSubsystemConfigStorage(subsystemName string, store any) error {
	if _, found := subsystemConfigs[subsystemName]; found {
		return fmt.Errorf("(bug) subsystem config already registered")
	}
	if store == nil {
		return fmt.Errorf("(bug) subsystem config may not be nil; must be a pointer to an allocated structure")
	}

	// validate that the provided store is a pointer to a structure
	val := reflect.ValueOf(store)
	if val.Kind() != reflect.Pointer && val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("(bug) subsystem config must be a structure; found %T instead", store)
	}

	// add to registry
	subsystemConfigs[subsystemName] = store

	return nil
}

// GetRegisteredSubsystems returns the names of subsystems that have registered a config template
func GetRegisteredSubsystems() []string {
	return maps.Keys(subsystemConfigs)
}

// GetSubsytemConfig returns a pointer to config storage for a given subsystem
func GetSubsytemConfigTemplate(subsystemName string) (any, error) {
	tmpl, ok := subsystemConfigs[subsystemName]
	if !ok {
		return nil, &ErrSubsystemNotFound{subsystemName}
	}
	return tmpl, nil
}

// @@
// SetSubsystemSetting updates a single value into the subsystem-specific settings of the context.
// It does not update the config file (if needed, call UpsertContext when all settings are in ready)
func SetSubsystemSetting(ctx *Context, subsystemName string, settingName string, value any) error {
	// fail if the subsystem doesn't exist or has not registered a config template
	_, ok := subsystemConfigs[subsystemName]
	if !ok {
		return &ErrSubsystemNotFound{subsystemName}
	}

	// add value to the context (without parsing or validation, as the structure may not be final)
	ssmap, ok := ctx.SubsystemConfigs[subsystemName]
	if !ok {
		ssmap = map[string]any{settingName: value}
		ctx.SubsystemConfigs[subsystemName] = ssmap
	} else {
		ssmap[settingName] = value
	}

	return nil
}

// UpdateSubsystemConfigs updates the subsystem-specific configurations from
// the config context into subsystem-provided structured store. If update fails
// for a subsystem, an error for it will be recorded and updates to other subsystem
// configurations continue. This allows callers to ignore subsystems with failed
// configuration while still getting configs for correctly configured systems.
// Returns nil or a slice of errors (the slice, if not nil, will never be empty)
func UpdateSubsystemConfigs(ctx *Context) error {

	// parse all provided configs (TODO: zero all others)
	errlist := []error{}
	for name, config := range ctx.SubsystemConfigs {
		configStruct, ok := subsystemConfigs[name]
		if !ok {
			err := fmt.Errorf("found configuration for %w", &ErrSubsystemNotFound{name})
			//log.Error(err.Error())
			errlist = append(errlist, err)
			continue
		}

		// create a decoder with the desired options & decode
		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			ErrorUnused: true,         // no extra settings that are not recognized by the subsystem; this is mostly to avoid typos
			ZeroFields:  true,         // on re-parsing/re-loading, ensure that any maps start from empty (although we currently support only atomic types)
			Result:      configStruct, // target which will be used for introspection and result storage
		})
		if err != nil {
			log.Fatalf("(bug) failed to create mapstrucure decoder: %v", err) // nb: likely not subsystem-specific, so no need to print name
		}
		parseErr := decoder.Decode(config)
		if parseErr != nil {
			err := &ErrSubsystemParsingError{name, parseErr}
			//log.Error(err.Error())
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
		return &ErrSubsystemConfig{errlist}
	}
	return nil
}
