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

// Package cmd defines all CLI commands and their flags
package cmd

import (
	"context"
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/apex/log"
	"github.com/apex/log/handlers/json"
	"github.com/apex/log/handlers/multi"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/cmd/version"
	"github.com/cisco-open/fsoc/logfilter"
	"github.com/cisco-open/fsoc/platform/api"
)

var cfgFile string
var cfgProfile string
var outputFormat string

const (
	FSOC_CONFIG_ENVVAR    = "FSOC_CONFIG"
	FSOC_NO_VERSION_CHECK = "FSOC_NO_VERSION_CHECK"
)

const (
	secondsInDay      = 24 * 60 * 60
	timestampFileName = "fsoc.timestamp"
)

var updateChannel chan *semver.Version

// rootCmd represents the base command when called without any subcommands
// TODO: replace github link "for more info" with Cisco DevNet link for fsoc once published
var rootCmd = &cobra.Command{
	Use:   "fsoc",
	Short: "fsoc - Cisco FSO Platform Control Tool",
	Long: `fsoc is an open source utility that serves as an entry point for developers on the Cisco
Full Stack Observability (FSO) Platform (https://developer.cisco.com/docs/fso/).

It allows developers to interact with the product environments--developer, test and production--in a
uniform way and to perform common tasks. fsoc primarily targets developers building solutions on the platform.

You can use --config and --profile to select authentication credentials to use. You can also use
environment variables FSOC_CONFIG and FSOC_PROFILE, respectively. The command line flags take precedence.
If a profile is not specified otherwise, the current profile from the config file is used.

fsoc checks once a day if a newer version is available on github and warns if not running the latest stable version.
You can use --no-version-check or the FSOC_NO_VERSION_CHECK=1 environment variable to suppress the check.

Examples:
  fsoc config set auth=oauth url=https://MYTENANT.observe.appdynamics.com
  fsoc login
  fsoc uql "FETCH id, type, attributes FROM entities(k8s:workload)"
  fsoc solution list
  fsoc solution list -o json
  FSOC_CONFIG=tenant5-config.yaml fsoc solution subscribe spacefleet --profile admin

For more information, see https://github.com/cisco-open/fsoc

NOTE: fsoc is in alpha; breaking changes may occur`,
	PersistentPreRun:  preExecHook,
	PersistentPostRun: postExecHook,
	TraverseChildren:  true,
	DisableAutoGenTag: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default is %s). May be .yaml or .json", config.DefaultConfigFile))
	rootCmd.PersistentFlags().StringVar(&cfgProfile, "profile", "", "access profile (default is current or \"default\")")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "auto", "output format (auto, table, detail, json, yaml)")
	rootCmd.PersistentFlags().String("fields", "", "perform specified fields transform/extract JQ expression")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable detailed output")
	rootCmd.PersistentFlags().Bool("curl", false, "Log curl equivalent for platform API calls (implies --verbose)")
	rootCmd.PersistentFlags().String("log", path.Join(os.TempDir(), "fsoc.log"), "determines the location of the fsoc log file")
	rootCmd.PersistentFlags().Bool("no-version-check", false, "Skip the daily check for new versions of fsoc")
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	rootCmd.SetIn(os.Stdin)

	err := rootCmd.RegisterFlagCompletionFunc("profile",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return config.ListContexts(toComplete), cobra.ShellCompDirectiveDefault
		})
	if err != nil {
		log.Warnf("(likely bug) Failed to register completion function for --profile: %v", err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// use config file from env var
	if cfgFile == "" { // only if not set from command line (command line has priority)
		cfgFile = os.Getenv(FSOC_CONFIG_ENVVAR) // remains empty if not found
	}

	// finalize config file
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".fsoc" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".fsoc")
	}
	viper.SetConfigType("yaml")

	viper.AutomaticEnv() // read in environment variables that match
}

func registerSubsystem(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}

func helperFlagFormatter(fs *pflag.FlagSet) string {
	s := ""
	if fs != nil {
		fs.Visit(func(f *pflag.Flag) {
			if s != "" {
				s += " "
			}
			s += fmt.Sprintf("%v=%q", f.Name, f.Value)
		})
	}
	return "[" + s + "]"
}

// preExecHook is executed after the command line is parsed but
// before the command's handler is executed
func preExecHook(cmd *cobra.Command, args []string) {
	logLocation, _ := cmd.Flags().GetString("log")
	var file *os.File
	var cliHandler log.Handler

	// process logging level flags (verbose and curl)
	verbose, _ := cmd.Flags().GetBool("verbose")
	if curlify, _ := cmd.Flags().GetBool("curl"); curlify {
		api.FlagCurlifyRequests = true
		verbose = true // force verbose
	}
	if verbose {
		cliHandler = logfilter.New(os.Stderr, log.InfoLevel)
	} else {
		cliHandler = logfilter.New(os.Stderr, log.WarnLevel)
	}
	log.SetLevel(log.InfoLevel)

	_ = os.Truncate(logLocation, 0)
	file, err := os.Create(logLocation)
	if err != nil {
		log.Warnf("failed to create log at %s", logLocation)
		log.SetHandler(cliHandler)
	} else {
		jsonHandler := json.New(file)
		log.SetHandler(multi.New(cliHandler, jsonHandler))
	}

	log.WithFields(version.GetVersion()).Info("fsoc version")

	log.WithFields(log.Fields{
		"command":   cmd.Name(),
		"arguments": fmt.Sprintf("%q", args),
		"flags":     helperFlagFormatter(cmd.Flags())}).
		Info("fsoc command line")

	// Determine if a configured profile is required for this command
	// (bypassed only for commands that must work or can safely work without it)
	bypass := bypassConfig(cmd) || cmd.Name() == "help" || isCompletionCommand(cmd)

	// try to read the config file.and profile
	err = viper.ReadInConfig()
	if err != nil && !bypass {
		log.Fatalf("fsoc is not configured, please use \"fsoc config set\" to configure an initial context")
	}

	// override the config file's current profile from cmd line or env var
	config.SetCurrentProfile(cmd, args, bypass)
	if err != nil { // bypass == true
		log.Infof("Unable to read config file (%v), proceeding without a config", err)
	} else { // err == nil
		profile := config.GetCurrentProfileName()
		exists := config.GetCurrentContext() != nil
		if !exists && !bypass {
			log.Fatalf("fsoc is not fully configured: missing profile %q; please use \"fsoc config set\" to configure it", profile)
		}
		log.WithFields(log.Fields{
			"config_file": viper.ConfigFileUsed(),
			"profile":     profile,
			"existing":    exists,
		}).
			Info("fsoc context")
	}

	// Do version checking
	noVerCheck, _ := cmd.Flags().GetBool("no-version-check")
	envNoVerCheck, err := strconv.ParseBool(os.Getenv(FSOC_NO_VERSION_CHECK))
	if err != nil {
		envNoVerCheck = false
	}
	noVerCheck = noVerCheck || envNoVerCheck
	updateCheckNeeded := !noVerCheck && int(time.Now().Unix())-getLastVersionCheckTime() > int(secondsInDay)
	if updateCheckNeeded {
		updateChannel = make(chan *semver.Version)
		go version.CheckForUpdate(updateChannel)
	}
}

func getTimestampFilePath() string {
	return os.TempDir() + "/" + timestampFileName
}

func getLastVersionCheckTime() int {
	fInfo, err := os.Stat(getTimestampFilePath())
	if err != nil {
		return 0 // makes it a really old file
	}
	return int(fInfo.ModTime().Unix())
}

func postExecHook(cmd *cobra.Command, args []string) {
	if updateChannel != nil {
		// wait for the latest version and print warning if not running the latest
		var updateSemVar = <-updateChannel
		version.CompareAndLogVersions(updateSemVar)

		// Create new timestamp file (only if version was checked)
		_ = os.Remove(getTimestampFilePath())
		_, err := os.Create(getTimestampFilePath())
		if err != nil {
			log.Errorf("failed to create version check timestamp file: %v", err)
		}
	}

}

func bypassConfig(cmd *cobra.Command) bool {
	_, bypassConfig := cmd.Annotations[config.AnnotationForConfigBypass]
	return bypassConfig
}

func isCompletionCommand(cmd *cobra.Command) bool {
	p := cmd.Parent()
	return (p != nil && p.Name() == "completion")
}
