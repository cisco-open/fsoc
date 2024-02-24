// Copyright 2024 Cisco Systems, Inc.
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
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"

	cfg "github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

var (
	//	currentContext bool
	createContextLong = `Create a new context entry (profile) in an fsoc config file.

Specify the desired name for the new profile, followed by desired settings for the profile as a list of SETTING=VALUE pairs. 
The settings are specific to the authentication method used by the context, so specifying the auth method first is a good practice. 

To see the list of configuration settings supported by fsoc, use the "fsoc config show-fields" command.	
Note that each authentication method requires slightly different set of settings, so see the examples below for your use case.

When creating the initial context for fsoc (after installing it), it is customary not to specify the context name, which will
use the default name, "` + cfg.DefaultContext + `". This also becomes the current context that will be used by fsoc until changed.

After creating the new context, fsoc will attempt to log in to it in order to verify that it is operable. Use the --no-login flag to
skip this step.

Once created, the new context can be modified with the "fsoc config set" command. The "fsoc config use" command can be used to
make the new context the default one for the config file. See "fsoc help config" for more information.

`

	createContextExample = `
  # Create an OAuth-based profile (recommended for interactive use)
  fsoc config create auth=oauth url=https://mytenant.observe.appdynamics.com
  fsoc config create tenant2 auth=oauth url=https://tenant2.observe.appdynamics.com 

  # Create profiles with service or agent principal credentials (secret file must remain accessible)
  fsoc config create service auth=service-principal secret-file=my-service-principal.json
  fsoc config create agent1 auth=agent-principal secret-file=collectors-values.yaml
  fsoc config create agent2 auth=agent-principal secret-file=client-values.json tenant=123456 url=https://mytenant.observe.appdynamics.com

  # Set local access
  fsoc config create dev auth=local url=http://localhost appd-pid=PID appd-tid=TID appd-pty=PTY
 
  # Create profiles for different purposes (e.g., continuous integration, ingestion)
  fsoc config create auth=service-principal secret-file=my-service-principal.json --no-login --config=ci-config.yaml
  fsoc config create ingest auth=agent-principal secret-file=agent-helm-values.yaml
`
)

func newCmdConfigCreate() *cobra.Command {

	var cmd = &cobra.Command{
		Use:         "create [CONTEXT_NAME] [--config CONFIG_FILE] [SETTING=VALUE]+",
		Short:       "Create a new context in an fsoc config file",
		Long:        createContextLong,
		Example:     createContextExample,
		Annotations: map[string]string{cfg.AnnotationForConfigBypass: ""},
		Args:        cobra.MinimumNArgs(1),
		Run:         configCreateContext,
	}

	cmd.Flags().Bool("no-login", false, "Do not attempt to log in to the new context after creating it")

	return cmd
}

func configCreateContext(cmd *cobra.Command, args []string) {
	var contextName string

	// -- Perform command-line parsing checks first (syntax)

	// warn if the --profile flag is used (it does not apply to this command)
	if cmd.Flags().Changed("profile") {
		log.Warn("Ignored the --profile flag, as it is not used by this command. Please specify the profile name to create as a first positional argument instead")
	}

	// get the context name, if specified (it will be the first argument and not contain an '=')
	if !strings.Contains(args[0], "=") {
		contextName = args[0]
		args = args[1:]
	}
	if contextName == "" {
		contextName = cfg.DefaultContext
	}

	// parse settings and separate the core fsoc settings from any subsystem-specific settings
	// note that, unlike on `set`, this command does not support the legacy --flag-based settings (since it's a new command)
	coreArgs, subsystemSettingArgs, err := parseCoreArgs(cmd, args)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// Check that at least one config value is specified (including empty); either core or subsytem-specific setting satisfies this check
	if len(coreArgs) == 0 && len(subsystemSettingArgs) == 0 {
		log.Fatal("At least one setting must be specified when creating a new context")
	}

	// -- Perform other non-parsing validations (e.g., whether the context already exists)

	// fail if the context name already exists
	if _, err := cfg.GetContext(contextName); err == nil {
		log.Fatalf("Context %q already exists; create with a different name or use 'fsoc config set' to update this one", contextName)
	}

	// set the profile name (as if it was provided by --profile)
	// this is necessary to ensure that the correct profile is used when loggin in and, in general, avoid clobbering an existing profile
	cfg.ForceSetActiveProfileName(contextName)

	// -- Fill in the new context

	// Create a new context
	ctxPtr := &cfg.Context{Name: contextName}

	// Process core settings
	if err = updateCoreSettings(cmd, ctxPtr, coreArgs, false); err != nil {
		log.Fatalf("Failed to set core settings: %v", err)
	}

	// process subsystem-specific settings
	if err := processSubsystemSettings(ctxPtr, subsystemSettingArgs); err != nil {
		log.Fatalf("Failed to set subsystem-specific settings: %v", err)
	}

	// -- Apply the new context

	// update config file
	// TODO: avoid modifying the file if login fails
	if err := cfg.UpsertContext(ctxPtr); err != nil {
		log.Fatalf("%v", err)
	}

	// init a generic output message (to be updated later with more specific status)
	var message string

	// try to log in to the new context (don't check auth methods, api.Login() will handle degenerate cases)
	noLogin, _ := cmd.Flags().GetBool("no-login")
	if !noLogin {
		if err := api.Login(); err != nil {
			log.Warnf(`Failed to log in to the new context: %v; use "config set [SETTING=VALUE]+ --login --profile %v" to modify and try again`, err, contextName)
			message = fmt.Sprintf("Context %q created but is not operable.", contextName)
		} else {
			message = fmt.Sprintf("Context %q created successfully.", contextName)
		}
	} else {
		message = fmt.Sprintf("Context %q created ok but not verified by logging in.", contextName)
	}

	output.PrintCmdOutput(cmd, message)
}

// parseCoreArgs separates the core fsoc settings from any subsystem-specific settings. It returns the two groups of settings and an error
func parseCoreArgs(cmd *cobra.Command, args []string) (map[string]string, []string, error) {
	settings := map[string]string{}
	remainder := []string{}
	for i := 0; i < len(args); i++ {
		// split argument into name=value
		stringSegments := strings.SplitN(args[i], "=", 2)
		if len(stringSegments) < 2 {
			return settings, args, fmt.Errorf("setting %q at position %d must be in the form KEY=VALUE", args[i], i+1)
		}
		name, value := stringSegments[0], stringSegments[1]

		// if the argument is a subsystem-specific setting, leave it in the remainder
		if strings.Contains(name, ".") {
			remainder = append(remainder, args[i])
			continue
		}

		// check arg name is valid (i.e. no disallowed flags)
		if !slices.Contains(configArgs, name) {
			// TODO expand the message to accommodate subsystem-specific settings
			return settings, args, fmt.Errorf("setting name %s must be one of the following values %s", name, strings.Join(configArgs, ", "))
		}

		// add setting
		settings[name] = value
		// make sure flag isn't already set
	}
	return settings, remainder, nil
}

// updateCoreSettings sets the core fsoc settings on the context. It returns an error if any setting fails to be set.
func updateCoreSettings(cmd *cobra.Command, ctxPtr *cfg.Context, settings map[string]string, patch bool) error {
	var val string
	var ok bool

	// process settings in the order they will be applied, allowing for non-patch to clear dependent fields before they are set (if set)

	val, ok = settings["auth"]
	if ok {
		if !slices.Contains(GetAuthMethodsStringList(), val) {
			log.Fatalf(`Invalid auth method %q; must be one of {"%v"}`, val, strings.Join(GetAuthMethodsStringList(), `", "`))
		}
		ctxPtr.AuthMethod = val
		delete(settings, "auth")

		// Clear All fields before setting other fields
		if !patch {
			clearFields([]string{"url", "server", "tenant", "user", "token", "refresh_token", "secret-file"}, ctxPtr)
		}
	}

	val, ok = settings["envtype"]
	if ok {
		potentialEnvTypes := []string{"prod", "dev"}
		if !slices.Contains(potentialEnvTypes, val) {
			log.Fatalf("envtype can only take on one of the following values: %s", strings.Join(potentialEnvTypes, ", "))
		}
		ctxPtr.EnvType = val
		delete(settings, "envtype")
	}

	// handle url and, for backward compatibility, server, which is deprecated
	val, ok = settings["url"]
	if !ok {
		val, ok = settings["server"]
	}
	if ok {
		cleanedUrl, err := validateUrl(val)
		if err != nil {
			log.Fatal(err.Error())
		}
		if settings["server"] != "" {
			log.Warnf(`The "server" setting is now deprecated. In the future, please use the "url" setting instead. We will set the url to %q for you now`, cleanedUrl)
		} else if cleanedUrl != val {
			log.Warnf("The specified url, %q, has been automatically updated to %q", val, cleanedUrl)
		}
		ctxPtr.URL = cleanedUrl

		// Automate setting EnvType from url if EnvType is not already set (above)
		if ctxPtr.EnvType == "" {
			parsedUrl, err := url.Parse(cleanedUrl)
			if err != nil {
				log.Fatalf("Failed to parse url: %v", err)
			}
			host := parsedUrl.Host
			if !strings.HasSuffix(host, ".observe.appdynamics.com") {
				ctxPtr.EnvType = "dev"
				log.Warnf("Automatically setting envtype to %q", ctxPtr.EnvType)
			}
		}

		if !patch {
			automatedFieldClearing(ctxPtr, "url")
		}
	}

	val, ok = settings["tenant"]
	if ok {
		// reject if tenant is not an allowed setting for this authentication type
		err := validateWriteReq(cmd, ctxPtr.AuthMethod, "tenant")
		if err != nil {
			log.Fatal(err.Error())
		}

		// update tenant
		ctxPtr.Tenant = val
		delete(settings, "tenant")

		// clear dependent fields if not patching
		if !patch {
			automatedFieldClearing(ctxPtr, "tenant")
		}
	}

	val, ok = settings["token"]
	if ok {
		// reject if token is not an allowed setting for this authentication type
		err := validateWriteReq(cmd, ctxPtr.AuthMethod, "token")
		if err != nil {
			log.Fatal(err.Error())
		}

		// update token
		if val == "-" { // token to come from stdin
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			ctxPtr.Token = scanner.Text()
		} else {
			ctxPtr.Token = val
		}
		delete(settings, "token")

		// clear dependent fields if not patching
		if !patch {
			automatedFieldClearing(ctxPtr, "token")
		}
	}

	val, ok = settings["secret-file"]
	if ok {
		// reject if secret-file is not an allowed setting for this authentication type
		err := validateWriteReq(cmd, ctxPtr.AuthMethod, "secret-file")
		if err != nil {
			log.Fatal(err.Error())
		}

		// canonicalize the path and update it into the context
		path := expandHomePath(val)
		ctxPtr.SecretFile, err = filepath.Abs(path)
		if err != nil {
			log.WithFields(log.Fields{"path": path, "error": err}).Warn("Failed to convert secret file's path to an absolute path; using it as is")
			ctxPtr.SecretFile = path
		}
		ctxPtr.CsvFile = "" // CSV file is a backward-compatibility value only
		delete(settings, "secret-file")

		if !patch {
			automatedFieldClearing(ctxPtr, "secret-file")
		}
	}

	// populate fields for local auth
	if ctxPtr.AuthMethod == cfg.AuthMethodLocal {
		val, ok = settings[cfg.AppdPid]
		if ok {
			// reject if appd-pid is not an allowed setting for this authentication type
			err := validateWriteReq(cmd, ctxPtr.AuthMethod, cfg.AppdPid)
			if err != nil {
				log.Fatal(err.Error())
			}

			// update value
			ctxPtr.LocalAuthOptions.AppdPid = val
			delete(settings, cfg.AppdPid)
			if !patch {
				automatedFieldClearing(ctxPtr, cfg.AppdPid)
			}
		}
		val, ok = settings[cfg.AppdPty]
		if ok {
			// reject if appd-pty is not an allowed setting for this authentication type
			err := validateWriteReq(cmd, ctxPtr.AuthMethod, cfg.AppdPty)
			if err != nil {
				log.Fatal(err.Error())
			}

			// update value
			ctxPtr.LocalAuthOptions.AppdPty = val
			delete(settings, cfg.AppdPty)
			if !patch {
				automatedFieldClearing(ctxPtr, cfg.AppdPty)
			}
		}
		val, ok = settings[cfg.AppdTid]
		if ok {
			// reject if appd-tid is not an allowed setting for this authentication type
			err := validateWriteReq(cmd, ctxPtr.AuthMethod, cfg.AppdTid)
			if err != nil {
				log.Fatal(err.Error())
			}

			// update value
			ctxPtr.LocalAuthOptions.AppdTid = val
			delete(settings, cfg.AppdTid)
			if !patch {
				automatedFieldClearing(ctxPtr, cfg.AppdTid)
			}
		}
	}

	// upgrade config format from CsvFile to SecretFile, opportunistically using the update
	if ctxPtr.SecretFile == "" && ctxPtr.CsvFile != "" {
		ctxPtr.SecretFile = ctxPtr.CsvFile
		ctxPtr.CsvFile = ""
	}

	return nil
}
