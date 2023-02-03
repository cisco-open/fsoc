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

var selectedProfile string

// Package registration function for the config root command
func NewSubCmd() *cobra.Command {
	// cmd represents the config sub command root
	var cmd = &cobra.Command{
		Use:   "config SUBCOMMAND [options]",
		Short: "Configure fsoc",
		Long:  `View and modify fsoc config files and contexts`,
		// Run: func(cmd *cobra.Command, args []string) {
		// 	fmt.Println("config called")
		// },
	}

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// configCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// configCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	cmd.AddCommand(newCmdConfigGet())
	cmd.AddCommand(newCmdConfigSet())
	cmd.AddCommand(newCmdConfigUse())
	cmd.AddCommand(newCmdConfigList())

	return cmd
}

// GetCurrentContext returns the context (access profile) selected by the user
// for the particular invocation of the fsoc utility. Returns nil if no current context is defined (and the
// only command allowed in this state is `config set`, which will create the context).
// Note that GetCurrentContext returns a pointer into the config file's overall configuration; it can be
// modified and then updated using ReplaceCurrentContext().
func GetCurrentContext() *Context {
	profile := GetCurrentProfileName()

	// read config file
	cfg := getConfig()
	if len(cfg.Contexts) == 0 {
		return nil
	}

	// locate & return the named context
	for _, c := range cfg.Contexts {
		if c.Name == profile {
			return &c
		}
	}

	return nil
}

func getConfig() configFileContents {
	var c configFileContents
	err := viper.Unmarshal(&c)
	if err != nil {
		log.Fatalf("unable to read config, %v", err)
	}
	return c
}

// listContexts returns a list of context names which begin with `toComplete`,
// used for the command line autocompletion
// func listContexts(toComplete string) []string {
// 	config := getConfig()
// 	var ret []string
// 	for _, c := range config.Contexts {
// 		name := c.Name
// 		if strings.HasPrefix(name, toComplete) {
// 			ret = append(ret, name)
// 		}
// 	}
// 	return ret
// }

func updateConfigFile(keyValues map[string]interface{}) {
	var err error

	// update values
	for key, value := range keyValues {
		viper.Set(key, value)
	}
	// set up config file in viper
	viper.SetConfigType("yaml")
	if viper.ConfigFileUsed() == "" {
		home, _ := os.UserHomeDir()
		configFileLocation := strings.Replace(defaultConfigFile, "~", home, 1)
		viper.SetConfigFile(configFileLocation)
	}
	viper.SetConfigPermissions(0600) // o=rw

	// ensure file exists (viper fails to create it, likely a bug in viper)
	ensureConfigFile()

	// update file contents
	err = viper.WriteConfig()
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
		log.Infof("Updated context %q", ctx.Name)
	} else {
		log.Infof("Created context %q", ctx.Name)
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

// SetSelectedProfile sets the name of the profile that should be used instead of the
// config file's current profile value. This function should not be used outside of the
// fsoc root pre-command.
func SetSelectedProfile(name string) {
	if name == "" {
		log.Fatalf("Profile name cannot be empty, please specify a non-empty profile name")
	}
	if selectedProfile != "" {
		log.Warnf("The selected profile is being overridden: old=%q, new=%q", selectedProfile, name)
	}
	selectedProfile = name
}

// GetCurrentProfileName returns the profile name that is used to select the context.
// This is mostly the same as returned by GetCurrentContext().Name, except for the
// case when a new profile is being created.
func GetCurrentProfileName() string {
	// start with default
	profile := defaultContext

	// use the profile from command line or the config file's current
	if selectedProfile != "" {
		profile = selectedProfile
	} else {
		// get profile that is current for the config file
		cfg := getConfig()
		if cfg.CurrentContext == "" {
			cfg.CurrentContext = defaultContext // dealing with old "current-context" keys (temporary)
		}
		if cfg.CurrentContext != "" {
			profile = cfg.CurrentContext
		}
		// note: the profile may not exist, that's OK
	}

	return profile
}
