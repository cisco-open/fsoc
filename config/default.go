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

import "fmt"

// GetDefaultContextName gets the default context name for the config file
// Note that the default context may be different from the active (current) context
func GetDefaultContextName() string {
	cfg := getConfig()
	return cfg.CurrentContext
}

// SetDefaultContextName sets the default context name in the config file and updates the file
func SetDefaultContextName(name string) error {
	// look up selected context
	contextExists := false
	cfg := getConfig()
	for _, c := range cfg.Contexts {
		if c.Name == name {
			contextExists = true
			break
		}
	}
	if !contextExists {
		return fmt.Errorf("%q: %w", name, ErrProfileNotFound)
	}

	// update config file
	updateConfigFile(map[string]interface{}{"current_context": name})

	return nil
}
