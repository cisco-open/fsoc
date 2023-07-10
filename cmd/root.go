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
	"github.com/apex/log"
	"github.com/apex/log/handlers/json"
	"github.com/apex/log/handlers/multi"
	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/cmd/version"
	"github.com/cisco-open/fsoc/logfilter"
	"github.com/cisco-open/fsoc/platform/api"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var cfgFile string
var cfgProfile string
var outputFormat string

const (
	FSOC_CONFIG_ENVVAR  = "FSOC_CONFIG"
	FSOC_PROFILE_ENVVAR = "FSOC_PROFILE"
)

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

Examples:
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

	// override the config file's current profile from cmd line or env var
	var profile string // used only in this block
	if cmd.Flags().Changed("profile") {
		profile, _ = cmd.Flags().GetString("profile")
	} else {
		profile = os.Getenv(FSOC_PROFILE_ENVVAR) // remains empty if not defined
	}
	if profile != "" { // allow empty string on cmd line to mean use current
		config.SetSelectedProfile(profile)
	}

	// Determine if a configured profile is required for this command
	// (bypassed only for commands that must work or can safely work without it)
	bypass := bypassConfig(cmd) || cmd.Name() == "help" || isCompletionCommand(cmd)

	// try to read the config file.and profile
	err = viper.ReadInConfig()
	if err == nil {
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
	} else {
		if bypass {
			log.Infof("Unable to read config file (%v), proceeding without a config", err)
		} else {
			log.Fatalf("fsoc is not configured, please use \"fsoc config set\" to configure an initial context")
		}
	}
	if time.Now().Unix()-getRecentTimestamp() > 86000 {
		checkForUpdate()
	}
}

func getRecentTimestamp() int64 {
	potentialFiles, err := findFiles(os.TempDir())
	if err != nil {
		log.Fatalf(err.Error())
	}
	var largestTimestamp int64 = 0
	for i := 0; i < len(potentialFiles); i++ {
		stringSections := strings.Split(potentialFiles[i].Name(), "/")
		desiredSections := strings.Split(stringSections[len(stringSections)-1], "fsoctimestamp")
		newTimestamp, _ := strconv.ParseInt(desiredSections[0], 10, 64)
		if largestTimestamp < newTimestamp {
			largestTimestamp = newTimestamp
		}
	}

	for i := 0; i < len(potentialFiles); i++ {
		err := os.Remove(potentialFiles[i].Name())
		if err != nil {
			log.Fatalf(err.Error())
		}
	}

	return largestTimestamp
}

func findFiles(root string) ([]*os.File, error) {
	var files []*os.File
	err := filepath.WalkDir(root, func(pathh string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			if ok := strings.Contains(pathh, "fsoctimestamp"); ok && err == nil {
				file, err := os.Open(pathh)
				if err != nil {
					log.Fatalf(err.Error())
				}
				files = append(files, file)
			}
		}
		return nil
	})
	return files, err
}

func checkForUpdate() {
	newestVersionChan := make(chan string)
	go func() {
		err := getVersion(newestVersionChan)
		if err != nil {
			log.Fatalf(err.Error())
		}
	}()
	var newestVersion string
	newestVersion = <-newestVersionChan
	if compareVersion(newestVersion) {
		log.Warnf("There is a newer version of FSOC available, please upgrade to version %s", newestVersion)
	}
}

func getVersion(ver chan string) error {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get("https://github.com/cisco-open/fsoc/releases/latest")
	if err != nil {
		return err
	}

	split := strings.Split(resp.Header.Get("Location"), "/")
	ver <- split[len(split)-1]
	return nil
}

func compareVersion(newestVersion string) bool { // Returns if the parameter is a newer version than the installed one
	currentVersion := version.GetVersion()
	newestVersionMajor, newestVersionMinor, newestVersionPatch := parseVersion(newestVersion)
	newerMajorVersion := uint(newestVersionMajor) > currentVersion.VersionMajor
	newerMinorVersion := uint(newestVersionMinor) > currentVersion.VersionMinor
	newerPatchVersion := uint(newestVersionPatch) > currentVersion.VersionPatch
	if newerMajorVersion {
		return true
	}
	if newerMinorVersion {
		return true
	}
	if newerPatchVersion {
		return true
	}
	return false
}

func parseVersion(version string) (uint64, uint64, uint64) {
	version = version[1:]
	versionSections := strings.Split(version, ".")
	major, _ := strconv.ParseUint(versionSections[0], 10, 32)
	minor, _ := strconv.ParseUint(versionSections[1], 10, 32)
	patch, _ := strconv.ParseUint(versionSections[2], 10, 32)
	return major, minor, patch
}

func postExecHook(cmd *cobra.Command, args []string) {
	// Create File
	_, err := os.CreateTemp(os.TempDir(), strconv.FormatInt(time.Now().Unix(), 10)+"fsoctimestamp")
	if err != nil {
		log.Fatalf(err.Error())
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
