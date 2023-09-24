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
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/exp/slices"

	cfg "github.com/cisco-open/fsoc/config"
)

var (
	//	currentContext bool
	setContextLong = `Create or modify a context entry in an fsoc config file.

Specifying a name that already exists will merge new fields on top of existing values for those fields.
if on context name is specified, the current context is created/updated.`

	setContextExample = `
  # Set oauth credentials (recommended for interactive use)
  fsoc config set auth=oauth url=https://mytenant.observe.appdynamics.com

  # Set service or agent principal credentials (secret file must remain accessible)
  fsoc config set auth=service-principal secret-file=my-service-principal.json
  fsoc config set auth=agent-principal secret-file=agent-helm-values.yaml
  fsoc config set auth=agent-principal secret-file=client-values.json tenant=123456 url=https://mytenant.observe.appdynamics.com

  # Set local access
  fsoc config set auth=local url=http://localhost appd-pid=PID appd-tid=TID appd-pty=PTY

  # Set the token field on the "prod" context entry without touching other values
  fsoc config set profile prod token=top-secret`
)

// configArgs are the positional arguments of form <name>=<value> that can be set.
// They also correspond to the --flags for the same, for backward compatibility (deprecated)
// The order here is how the fields are displayed in `config show-help` topic
var configArgs = []string{"auth", "url", "tenant", "secret-file", "envtype", "token", cfg.AppdTid, cfg.AppdPty, cfg.AppdPid, "server"}

func newCmdConfigSet() *cobra.Command {

	var cmd = &cobra.Command{
		Use:         "set [--config CONFIG_FILE] [--profile CONTEXT] [KEY=VALUE]+",
		Short:       "Create or modify a context entry in an fsoc config file",
		Long:        setContextLong,
		Example:     setContextExample,
		Annotations: map[string]string{cfg.AnnotationForConfigBypass: ""},
		Run:         configSetContext,
	}

	// real command flag(s)
	cmd.Flags().Bool("patch", false, "Bypass field clearing")

	// deprecated flags representing core config settings
	cmd.Flags().String(cfg.AppdPid, "", "pid to use (local auth type only, provide raw value to be encoded)")
	_ = cmd.Flags().MarkDeprecated(cfg.AppdPid, "please use arguments supplied as "+cfg.AppdPid+"="+strings.ToUpper(cfg.AppdPid))
	cmd.Flags().String(cfg.AppdTid, "", "tid to use (local auth type only, provide raw value to be encoded)")
	_ = cmd.Flags().MarkDeprecated(cfg.AppdTid, "please use arguments supplied as "+cfg.AppdTid+"="+strings.ToUpper(cfg.AppdTid))
	cmd.Flags().String(cfg.AppdPty, "", "pty to use (local auth type only, provide raw value to be encoded)")
	_ = cmd.Flags().MarkDeprecated(cfg.AppdPty, "please use arguments supplied as "+cfg.AppdPty+"="+strings.ToUpper(cfg.AppdPty))
	cmd.Flags().String("auth", "", fmt.Sprintf(`Select authentication method, one of {"%v"}`, strings.Join(GetAuthMethodsStringList(), `", "`)))
	_ = cmd.Flags().MarkDeprecated("auth", `please use non-flag argument in the form "auth=AUTH"`)
	cmd.Flags().String("server", "", "Set server host name")
	_ = cmd.Flags().MarkDeprecated("server", `please use the url argument instead, in the form "url=URL"`)
	cmd.Flags().String("url", "", "Set server URL (with http or https schema)")
	_ = cmd.Flags().MarkDeprecated("url", `please use non-flag argument in the form "url=URL"`)
	cmd.Flags().String("tenant", "", "Set tenant ID")
	_ = cmd.Flags().MarkDeprecated("tenant", `please use non-flag argument in the form "tenant=TENANT"`)
	cmd.Flags().String("token", "", "Set token value (use --token=- to get from stdin)")
	_ = cmd.Flags().MarkDeprecated("token", `please use non-flag argument in the form "token=TOKEN"`)
	cmd.Flags().String("secret-file", "", "Set a credentials file to use for service principal (.json or .csv) or agent principal (.yaml)")
	_ = cmd.Flags().MarkDeprecated("secret-file", `please use non-flag argument in the form "secret-file=SECRET-TOKEN"`)
	cmd.Flags().String("envtype", "", "envtype can be \"dev\", \"prod\", or \"\". When it is \"dev\", solution tags will always be set to stable")
	_ = cmd.Flags().MarkDeprecated("envtype", `please use non-flag argument in the form "envtype=ENVTYPE"`)

	return cmd
}

func configSetContext(cmd *cobra.Command, args []string) {
	var contextName string
	var subsystemSettingArgs []string
	var err error

	// transfer core fsoc settings from args to legacy flags, and extract the remainder
	// as subsystem-specific settings
	subsystemSettingArgs, err = transferCoreArgs(cmd, args)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// Check that at least one config value is specified (including empty); either core or subsytem-specific setting satisfies this check
	flags := cmd.Flags()
	valid := len(subsystemSettingArgs) > 0 // at least one non-core setting is defined
	flags.VisitAll(func(flag *pflag.Flag) {
		if slices.Contains(configArgs, flag.Name) {
			valid = valid || flag.Changed
		}
	})
	if !valid {
		log.Fatalf("at least one of %v must be specified", strings.Join(configArgs, ", ")) // TODO expand the message to accommodate subsystem-specific settings
	}

	// Try to locate the named context, whether it exists or not
	contextName = cfg.GetCurrentProfileName() // it may not exist
	ctxPtr, err := cfg.GetContext(contextName)
	if errors.Is(err, cfg.ErrProfileNotFound) {
		log.Infof("Context %q doesn't exist, creating it", contextName)

		ctxPtr = &cfg.Context{
			Name: contextName,
		}
	}

	patch, _ := cmd.Flags().GetBool("patch")

	// update only the fields for which flags were specified explicitly
	// (and, force-clear dependent/auto-derived fields unless --patch)

	if flags.Changed("auth") {
		val, _ := flags.GetString("auth")
		if val != "" && !slices.Contains(GetAuthMethodsStringList(), val) {
			log.Fatalf(`Invalid --auth method %q; must be one of {"%v"}`, val, strings.Join(GetAuthMethodsStringList(), `", "`))
		}
		ctxPtr.AuthMethod = val

		// Clear All fields before setting other fields
		if !patch {
			clearFields([]string{"url", "server", "tenant", "user", "token", "refresh_token", "secret-file"}, ctxPtr)
		}
	}

	if flags.Changed("envtype") {
		val, _ := flags.GetString("envtype")
		potentialEnvTypes := []string{"prod", "dev"}
		if !slices.Contains(potentialEnvTypes, val) {
			log.Fatalf("envtype can only take on one of the following values: %s", strings.Join(potentialEnvTypes, ", "))
		}
		ctxPtr.EnvType = val
	}

	// handle url (and, server, for backward compatibility)
	if flags.Changed("server") || flags.Changed("url") {
		providedUrl, _ := flags.GetString("url")
		if flags.Changed("server") {
			providedUrl, _ = flags.GetString("server")
		}
		cleanedUrl, err := validateUrl(providedUrl)
		if err != nil {
			log.Fatal(err.Error())
		}
		if flags.Changed("server") {
			log.Warnf("The --server option is now deprecated. In the future, please use --url instead. We will set the url to %q for you now", cleanedUrl)
		}
		ctxPtr.URL = cleanedUrl

		// Automate setting EnvType from url
		// (note that ctxPtr.EnvType is already set if specified on the command line)
		parsedUrl, err := url.Parse(cleanedUrl)
		if err != nil {
			log.Fatalf("Failed to parse url: %v", err)
		}
		host := parsedUrl.Host
		if !strings.HasSuffix(host, ".observe.appdynamics.com") && (ctxPtr.EnvType != "") {
			ctxPtr.EnvType = "dev"
			log.Warnf("Automatically setting envtype to %q", ctxPtr.EnvType)
		}

		if !patch {
			automatedFieldClearing(ctxPtr, "url")
		}
	}

	if flags.Changed("tenant") {
		err := validateWriteReq(cmd, ctxPtr.AuthMethod, "tenant")
		if err != nil {
			log.Fatal(err.Error())
		}
		ctxPtr.Tenant, _ = flags.GetString("tenant")
		if !patch {
			automatedFieldClearing(ctxPtr, "tenant")
		}
	}

	if flags.Changed("token") {
		err := validateWriteReq(cmd, ctxPtr.AuthMethod, "token")
		if err != nil {
			log.Fatal(err.Error())
		}
		value, _ := flags.GetString("token")
		if value == "-" { // token to come from stdin
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			ctxPtr.Token = scanner.Text()
		} else {
			ctxPtr.Token = value
		}
		if !patch {
			automatedFieldClearing(ctxPtr, "token")
		}
	}

	if flags.Changed("secret-file") {
		err := validateWriteReq(cmd, ctxPtr.AuthMethod, "secret-file")
		if err != nil {
			log.Fatal(err.Error())
		}
		path, _ := flags.GetString("secret-file")
		path = expandHomePath(path)
		ctxPtr.SecretFile, err = filepath.Abs(path)
		if err != nil {
			log.WithFields(log.Fields{"path": path, "error": err}).Warn("Failed to convert secret file's path to absolute path; using it as is")
			ctxPtr.SecretFile = path
		}
		ctxPtr.CsvFile = "" // CSV file is a backward-compatibility value only
		if !patch {
			automatedFieldClearing(ctxPtr, "secret-file")
		}
	}

	// populate fields for local auth
	if ctxPtr.AuthMethod == cfg.AuthMethodLocal {
		if flags.Changed(cfg.AppdPid) {
			err := validateWriteReq(cmd, ctxPtr.AuthMethod, cfg.AppdPid)
			if err != nil {
				log.Fatal(err.Error())
			}
			pid, _ := flags.GetString(cfg.AppdPid)
			ctxPtr.LocalAuthOptions.AppdPid = pid
			if !patch {
				automatedFieldClearing(ctxPtr, cfg.AppdPid)
			}
		}
		if flags.Changed(cfg.AppdPty) {
			err := validateWriteReq(cmd, ctxPtr.AuthMethod, cfg.AppdPty)
			if err != nil {
				log.Fatal(err.Error())
			}
			pty, _ := flags.GetString(cfg.AppdPty)
			ctxPtr.LocalAuthOptions.AppdPty = pty
			if !patch {
				automatedFieldClearing(ctxPtr, cfg.AppdPty)
			}
		}
		if flags.Changed(cfg.AppdTid) {
			err := validateWriteReq(cmd, ctxPtr.AuthMethod, cfg.AppdTid)
			if err != nil {
				log.Fatal(err.Error())
			}
			tid, _ := flags.GetString(cfg.AppdTid)
			ctxPtr.LocalAuthOptions.AppdTid = tid
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

	// process subsystem-specific settings
	if err := processSubsystemSettings(ctxPtr, subsystemSettingArgs); err != nil {
		log.Fatalf("Failed to set subsystem-specific settings: %v", err)
	}

	// update config file
	if err := cfg.UpsertContext(ctxPtr); err != nil {
		log.Fatalf("%v", err)
	}
}

// expandHomePath replaces ~ in the path with the absolute home directory
func expandHomePath(file string) string {
	if strings.HasPrefix(file, "~/") {
		dirname, _ := os.UserHomeDir()
		file = filepath.Join(dirname, file[2:])
	}
	return file
}

func getAuthFieldConfigRow(authService string) AuthFieldConfigRow {
	return getAuthFieldWritePermissions()[authService]
}

func validateWriteReq(cmd *cobra.Command, authService string, field string) error {
	flags := cmd.Flags()
	authProvider := authService
	if flags.Changed("auth") {
		authProvider, _ = flags.GetString("auth")
	}
	if authProvider == "" {
		return fmt.Errorf("must provide an authentication type before or while writing to other context fields")
	}
	if getAuthFieldConfigRow(authProvider)[field] == ClearField {
		return fmt.Errorf("cannot write to field %s because it is not allowed for authentication method %s", field, authProvider)
	}
	return nil
}

func clearFields(fields []string, ctxPtr *cfg.Context) {
	if slices.Contains(fields, "auth") {
		ctxPtr.AuthMethod = ""
	}
	if slices.Contains(fields, "url") {
		ctxPtr.URL = ""
		ctxPtr.Server = "" // server is just the old name of url
	}
	if slices.Contains(fields, "tenant") {
		ctxPtr.Tenant = ""
	}
	if slices.Contains(fields, "user") {
		ctxPtr.User = ""
	}
	if slices.Contains(fields, "token") {
		ctxPtr.Token = ""
	}
	if slices.Contains(fields, "refresh-token") {
		ctxPtr.RefreshToken = ""
	}
	if slices.Contains(fields, "secret-file") {
		ctxPtr.SecretFile = ""
	}
}

func automatedFieldClearing(ctxPtr *cfg.Context, field string) {
	table := getAuthFieldClearConfig()
	clearFields(table[ctxPtr.AuthMethod][field], ctxPtr)
}

func validateUrl(providedUrl string) (string, error) {
	parsedUrl, err := url.Parse(providedUrl)
	if err == nil && parsedUrl.Scheme == "" {
		parsedUrl, err = url.Parse("https://" + providedUrl)
	}
	if err != nil {
		return "", fmt.Errorf("the provided url, %q, is not valid: %w", providedUrl, err)
	}
	if parsedUrl.Host == "" {
		return "", fmt.Errorf("no host is provided in the url %q", providedUrl)
	}
	if parsedUrl.Scheme != "https" && parsedUrl.Scheme != "http" {
		return "", fmt.Errorf("the provided scheme, %q, is not recognized; use %q or %q", parsedUrl.Scheme, "http", "https")
	}

	retUrl := parsedUrl.String()
	if retUrl != providedUrl {
		log.Warnf("The provided url, %q, was adjusted and stored as %q.", providedUrl, parsedUrl.String())
	}
	return retUrl, nil
}

// transferCoreArgs extracts arguments from the commend line that are used in the core context (i.e., not subystem-related).
// It translates these into flags (supporting legacy/deprecated flag-based) and removes them from the argument list. It returns
// a list with the remaining arguments. If there is an error parsing, it returns the unmodified arguments and the error
func transferCoreArgs(cmd *cobra.Command, args []string) ([]string, error) {
	flags := cmd.Flags()
	remainder := []string{}
	for i := 0; i < len(args); i++ {
		// split argument into name=value
		stringSegments := strings.SplitN(args[i], "=", 2)
		if len(stringSegments) < 2 {
			return args, fmt.Errorf("argument %q at position %d must be in the form KEY=VALUE", args[i], i+1)
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
			return args, fmt.Errorf("argument name %s must be one of the following values %s", name, strings.Join(configArgs, ", "))
		}
		// make sure flag isn't already set
		if flags.Changed(name) {
			return args, fmt.Errorf("cannot have both flag and argument with same name")
		}
		// Set flag manually
		err := flags.Set(name, value)
		if err != nil {
			return args, err
		}
	}
	return remainder, nil
}

func processSubsystemSettings(ctx *cfg.Context, args []string) error {
	for _, arg := range args {
		// split argument into name=value
		stringSegments := strings.SplitN(arg, "=", 2)
		if len(stringSegments) < 2 {
			return fmt.Errorf("argument %q must be in the form KEY=VALUE", arg)
		}
		name, value := stringSegments[0], stringSegments[1]

		// parse subsystem name and setting name, enforcing simple, flat settings
		nameSegments := strings.Split(name, ".")
		if l := len(nameSegments); l < 2 {
			// this should never happen, since args coming here are already vetted to include at least one '.' in the name
			return fmt.Errorf("(bug) missing the subsystem name in argument %q", arg)
		} else if l > 2 {
			return fmt.Errorf("the setting name %q in argument %q must be in the form SUBSYSTEM.SETTING", nameSegments[0], arg)
		}
		subsystemName, settingName := nameSegments[0], nameSegments[1]

		// update (or delete) the setting in the context
		var err error
		if value != "" {
			err = cfg.SetSubsystemSetting(ctx, subsystemName, settingName, value)
		} else {
			err = cfg.DeleteSubsystemSetting(ctx, subsystemName, settingName)
		}
		if err != nil {
			return fmt.Errorf("error processing argument %q: %v", arg, err)
		}
	}

	// update subsystem-specific settings, both for consistency and to cause the settings
	// to be parsed according to the provided templates
	if err := cfg.UpdateSubsystemConfigs(ctx); err != nil {
		return err
	}

	return nil
}
