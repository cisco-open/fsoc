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

// Package config provides access to fsoc configuration, both to obtain the current
// configuration and to incrementally or fully modify the configuration.
// The fsoc configuration has two dimension: a config file and a context within the config file.
// Each config file contains one or more contexts plus a setting indicating which of them is the current one.
package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	FSOC_PROFILE_ENVVAR = "FSOC_PROFILE"
	FSOC_CONFIG_ENVVAR  = "FSOC_CONFIG"
)

var activeProfile string

func getContext(name string) *Context {
	// read config file
	cfg := getConfig()
	if len(cfg.Contexts) == 0 {
		return nil
	}

	// locate & return the named context
	for _, c := range cfg.Contexts {
		if c.Name == name {
			// ensure a subsystem config map exists, even if empty
			if c.SubsystemConfigs == nil {
				c.SubsystemConfigs = map[string]map[string]any{}
			}
			return &c
		}
	}

	return nil
}

func checkUpgradeScheme(c *configFileContents) {
	needReWrite := false
	newContexts := make([]Context, len(c.Contexts))
	for i, context := range c.Contexts {
		if context.Server != "" {
			context.URL = "https://" + context.Server
			log.WithFields(log.Fields{
				"context": context.Name,
				"server":  context.Server,
				"url":     context.URL,
			}).Warn("The \"server\" config attribute is deprecated; replacing it with \"url\" now.")
			context.Server = ""
			needReWrite = true
		}
		newContexts[i] = context
	}
	if needReWrite {
		updateConfigFile(map[string]interface{}{
			"contexts": newContexts,
		})
		c.Contexts = newContexts
		log.Warnf("Config file updated to upgrade settings schema.")
	}
}

func getConfig() configFileContents {
	// read config file with all contexts
	var c configFileContents
	err := viper.Unmarshal(&c)
	if err != nil {
		log.Fatalf("unable to read config: %v", err)
	}

	// check if scheme needs to be upgraded; do it and update file if so
	checkUpgradeScheme(&c)

	return c
}

func updateConfigFile(keyValues map[string]interface{}) {
	// update values
	for key, value := range keyValues {
		viper.Set(key, value)
	}

	// set up config file in viper
	viper.SetConfigType("yaml")
	if viper.ConfigFileUsed() == "" {
		home, _ := os.UserHomeDir()
		configFileLocation := strings.Replace(DefaultConfigFile, "~", home, 1)
		viper.SetConfigFile(configFileLocation)
	}
	viper.SetConfigPermissions(0600) // o=rw

	// ensure file exists (viper fails to create it, likely a bug in viper)
	ensureConfigFile()

	// update file contents
	err := viper.WriteConfig()
	if err != nil {
		log.Fatalf("failed to write config file %q: %v", viper.ConfigFileUsed(), err)
	}
}

func ensureConfigFile() {
	appFs := afero.NewOsFs()

	// finalize the path to use
	var fileLoc = viper.ConfigFileUsed()
	if strings.Contains(fileLoc[:2], "~/") {
		homeDir, _ := os.UserHomeDir()
		fileLoc = strings.Replace(fileLoc, "~", homeDir, 1)
	}
	configPath, _ := filepath.Abs(fileLoc)

	// try to open the file, create it if it doesn't exist
	_, err := appFs.Open(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			_, err = appFs.Create(configPath)
			if err != nil {
				log.Fatalf("failed to create config file %q: %v", configPath, err)
			}
			viper.SetConfigFile(configPath)
		} else {
			log.Fatalf("failed to open config file %q: %v", configPath, err)
		}
	}
}

func updateContext(ctx *Context) {
	contextExists := false
	var ctxPtr *Context

	if ctx.Name == "" {
		log.Fatalf("bug: context name cannot be empty when updating context")
	}

	cfg := getConfig()
	for idx, c := range cfg.Contexts {
		if c.Name == ctx.Name {
			ctxPtr = &cfg.Contexts[idx]
			contextExists = true
			break
		}
	}

	// If context not found, create a new one
	if !contextExists {
		ctx := Context{
			Name: ctx.Name,
		}
		cfg.Contexts = append(cfg.Contexts, ctx)
		ctxPtr = &cfg.Contexts[len(cfg.Contexts)-1]
	}

	// copy context if needed
	if ctx != ctxPtr {
		*ctxPtr = *ctx // copy, in case ctx is not what GetCurrentContext() had returned
	}

	update := map[string]interface{}{"contexts": cfg.Contexts}
	if !contextExists && len(cfg.Contexts) == 1 { // just created the first context, set it as current
		update["current_context"] = ctx.Name
		log.Infof("Setting context %s as current", ctx.Name)
	}
	updateConfigFile(update)

	if contextExists {
		log.WithField("profile", ctx.Name).Info("Updated context")
	} else {
		log.WithField("profile", ctx.Name).Info("Created context")
	}
}

// ReplaceCurrentContext updates the all values within the current context.
// It accepts a Context structure, which may or may not be returned by GetCurrentContext().
// Note that the Context.Name must match the current context.
func ReplaceCurrentContext(ctx *Context) {
	// enforce that the *current* context is being replaced
	curCtx := GetCurrentContext()
	if curCtx == nil {
		log.Errorf("Attempt to update current context as %q when there is no current context; update ignored", ctx.Name)
		return
	}
	if ctx.Name != curCtx.Name {
		log.Errorf("Attempt to update current context %q using non-matching context name %q; update ignored", curCtx.Name, ctx.Name)
		return
	}

	// update context
	updateContext(ctx)
}

// SetActiveProfile sets the name of the profile that should be used instead of the
// config file's current profile value.
func SetActiveProfile(cmd *cobra.Command, args []string, emptyOK bool) {
	var profile string // used only in this block

	if cmd.Flags().Changed("profile") {
		profile, _ = cmd.Flags().GetString("profile")
	} else {
		profile = os.Getenv(FSOC_PROFILE_ENVVAR) // remains empty if not defined
	}
	if profile == "" {
		return // no change
	}
	// Check if profile exists
	if !emptyOK && getContext(profile) == nil {
		log.Fatalf("Could not find profile %q", profile)
	}
	if activeProfile != "" && activeProfile != profile {
		log.Warnf("The selected profile is being overridden: old=%q, new=%q", activeProfile, profile)
	}
	activeProfile = profile
}

// ForceSetActiveProfileName sets the name of the profile to the specified value. This is used
// primarily when managing profiles, for commands where the profile name is given as an argument
// (which takes precedence over any name set in env var or config file's default). Note that this
// function does not validate the profile name or even its existence.
func ForceSetActiveProfileName(profile string) {
	activeProfile = profile
}

// GetCurrentProfileName returns the profile name that is used to select the context.
// This is mostly the same as returned by GetCurrentContext().Name, except for the
// case when a new profile is being created.
func GetCurrentProfileName() string {
	// start with default
	profile := DefaultContext

	// use the profile from command line or the config file's current
	if activeProfile != "" {
		profile = activeProfile
	} else {
		// get profile that is current for the config file
		cfg := getConfig()
		if cfg.CurrentContext == "" {
			cfg.CurrentContext = DefaultContext // dealing with old "current-context" keys (temporary)
		}
		if cfg.CurrentContext != "" {
			profile = cfg.CurrentContext
		}
		// note: the profile may not exist, that's OK
	}

	return profile
}
