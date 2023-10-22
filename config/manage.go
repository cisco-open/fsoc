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
	"strings"

	"github.com/apex/log"
)

// ListAllContexts returns a list of all context names
func ListAllContexts() []string {
	return ListContexts("")
}

// ListContexts returns a list of context names which begin with `prefix`,
// used for the command line autocompletion
func ListContexts(prefix string) []string {
	config := getConfig()
	var ret []string
	for _, c := range config.Contexts {
		name := c.Name
		if strings.HasPrefix(name, prefix) {
			ret = append(ret, name)
		}
	}
	return ret
}

// GetCurrentContext returns the context (access profile) selected by the user
// for the particular invocation of the fsoc utility. Returns nil if no current context is defined (and the
// only command allowed in this state is `config set`, which will create the context).
// Note that GetCurrentContext returns a pointer into the config file's overall configuration; it can be
// modified and then updated using ReplaceCurrentContext().
func GetCurrentContext() *Context {
	profileName := GetCurrentProfileName()
	c := getContext(profileName)
	return c
}

// GetContext
func GetContext(name string) (*Context, error) {
	ctx := getContext(name)
	if ctx == nil {
		return nil, fmt.Errorf("%q: %w", name, ErrProfileNotFound)
	}

	return ctx, nil
}

// UpsertContext updates or adds a context and updates the file
// The context pointer may or may not have been returned by GetContext()/GetCurrentContext()
func UpsertContext(ctx *Context) error {
	updateContext(ctx)
	return nil
}

// DeleteContext deletes specified profile and updates the config file
// If the deleted context is the default one, xxx
func DeleteContext(name string) error {
	// find profile
	cfg := getConfig()
	profileIdx := -1
	for idx, c := range cfg.Contexts {
		if name == c.Name {
			profileIdx = idx
			break
		}
	}
	if profileIdx == -1 {
		return fmt.Errorf("%q: %w", name, ErrProfileNotFound)
	}

	// Delete context from config
	newContexts := append(cfg.Contexts[:profileIdx], cfg.Contexts[profileIdx+1:]...)
	update := map[string]interface{}{"contexts": newContexts}
	log.Infof("Deleted profile %q", name)

	// Reassign the current profile setting to an existing (or the default) profile
	if cfg.CurrentContext == name {
		var newCurrentContext string
		if len(newContexts) > 0 {
			newCurrentContext = newContexts[0].Name
		} else {
			newCurrentContext = DefaultContext
		}
		update["current_context"] = newCurrentContext
		log.Infof("Setting current profile to %q", newCurrentContext)
	}

	// Update config file
	updateConfigFile(update)

	return nil
}
