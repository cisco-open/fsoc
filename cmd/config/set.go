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
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/exp/slices"
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

func newCmdConfigSet() *cobra.Command {

	var cmd = &cobra.Command{
		Use:         "set [--profile CONTEXT] auth=AUTH [flags]",
		Short:       "Create or modify a context entry in an fsoc config file",
		Long:        setContextLong,
		Args:        cobra.MaximumNArgs(9),
		Example:     setContextExample,
		Annotations: map[string]string{AnnotationForConfigBypass: ""},
		Run:         configSetContext,
	}

	// real command flag(s)
	cmd.Flags().Bool("patch", false, "Bypass field clearing")

	// deprecated flags representing config settings
	cmd.Flags().String(AppdPid, "", "pid to use (local auth type only, provide raw value to be encoded)")
	_ = cmd.Flags().MarkDeprecated(AppdPid, "please use arguments supplied as "+AppdPid+"="+strings.ToUpper(AppdPid))
	cmd.Flags().String(AppdTid, "", "tid to use (local auth type only, provide raw value to be encoded)")
	_ = cmd.Flags().MarkDeprecated(AppdTid, "please use arguments supplied as "+AppdTid+"="+strings.ToUpper(AppdTid))
	cmd.Flags().String(AppdPty, "", "pty to use (local auth type only, provide raw value to be encoded)")
	_ = cmd.Flags().MarkDeprecated(AppdPty, "please use arguments supplied as "+AppdPty+"="+strings.ToUpper(AppdPty))
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

	return cmd
}

func validateUrl(providedUrl string) (string, error) {
	parsedUrl, err := url.ParseRequestURI(providedUrl)
	if err != nil {
		parsedUrl, err = url.ParseRequestURI("https://" + providedUrl)
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
	if parsedUrl.String() != providedUrl {
		log.Warnf("The provided url, %q, is cleaned and stored as %q.", providedUrl, parsedUrl.String())
	}
	return parsedUrl.String(), nil
}

func validateArgs(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()
	allowedArgs := []string{AppdPid, AppdTid, AppdPty, "auth", "server", "url", "tenant", "token", "secret-file"}
	for i := 0; i < len(args); i++ {
		// check arg format ∑+=∑+
		stringSegments := strings.Split(args[i], "=")
		name, value := stringSegments[0], stringSegments[1]
		if len(stringSegments) != 2 {
			return fmt.Errorf("parameter name and value cannot contain \"=\"")
		}
		// check arg name is valid (i.e. no disallowed flags)
		if !slices.Contains(allowedArgs, name) {
			return fmt.Errorf("argument name %s must be one of the following values %s", name, strings.Join(allowedArgs, ", "))
		}
		// make sure flag isn't already set
		if flags.Changed(name) {
			return fmt.Errorf("cannot have both flag and argument with same name")
		}
		// Set flag manually
		err := flags.Set(name, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func configSetContext(cmd *cobra.Command, args []string) {
	var contextName string

	// Check that either context name or current context is specified
	if err := validateArgs(cmd, args); err != nil {
		log.Fatalf("%v", err)
	}

	// Check that at least one value is specified (including empty)
	flags := cmd.Flags()
	valid := false
	flags.VisitAll(func(flag *pflag.Flag) {
		valid = valid || flag.Changed
	})
	if !valid {
		optionNames := make([]string, 0)
		flags.VisitAll(func(flag *pflag.Flag) {
			optionNames = append(optionNames, "--"+flag.Name)
		})
		log.Fatalf("at least one of %v must be specified", strings.Join(optionNames, ", "))
	}

	// Get context name (whether it exists or not)
	contextName = GetCurrentProfileName()

	// Try to locate the named context
	contextExists := false
	var ctxPtr *Context
	cfg := getConfig()
	for idx, c := range cfg.Contexts {
		if c.Name == contextName {
			ctxPtr = &cfg.Contexts[idx]
			contextExists = true
			break
		}
	}

	// If context not found, create a new one
	if !contextExists {
		log.Infof("context %q doesn't exist, creating it", contextName)

		ctx := Context{
			Name: contextName,
		}
		cfg.Contexts = append(cfg.Contexts, ctx)
		ctxPtr = &cfg.Contexts[len(cfg.Contexts)-1]
	}

	patch, _ := cmd.Flags().GetBool("patch")
	// update only the fields for which flags were specified explicitly
	if flags.Changed("auth") {
		val, _ := flags.GetString("auth")
		if val != "" && !slices.Contains(GetAuthMethodsStringList(), val) {
			log.Fatalf(`Invalid --auth method %q; must be one of {"%v"}`, val, strings.Join(GetAuthMethodsStringList(), `", "`))
		}
		ctxPtr.AuthMethod = val
		// Clear All fields before setting other fields
		clearFields([]string{"url", "server", "tenant", "user", "token", "refresh_token", "secret-file"}, ctxPtr)
	}
	if flags.Changed("server") {
		err := validateWriteReq(cmd, ctxPtr.AuthMethod, "url")
		if err != nil {
			log.Fatal(err.Error())
		}
		providedServer, _ := flags.GetString("server")
		constructedUrl := "https://" + providedServer
		cleanedUrl, err := validateUrl(constructedUrl)
		if err != nil {
			log.Fatal(err.Error())
		}
		log.Warnf("The --server option is now deprecated. In the future, please use --url instead. We will set the url to %q for you now", cleanedUrl)
		ctxPtr.URL = cleanedUrl
		if !patch {
			automatedFieldClearing(ctxPtr, "url")
		}
	}
	if flags.Changed("url") {
		err := validateWriteReq(cmd, ctxPtr.AuthMethod, "url")
		if err != nil {
			log.Fatal(err.Error())
		}
		providedUrl, _ := flags.GetString("url")
		cleanedUrl, err := validateUrl(providedUrl)
		if err != nil {
			log.Fatal(err.Error())
		}
		ctxPtr.URL = cleanedUrl
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
			ctxPtr.SecretFile = path
		}
		ctxPtr.CsvFile = "" // CSV file is a backward-compatibility value only
		if !patch {
			automatedFieldClearing(ctxPtr, "secret-file")
		}
	}

	if ctxPtr.AuthMethod == AuthMethodLocal {
		if flags.Changed(AppdPid) {
			err := validateWriteReq(cmd, ctxPtr.AuthMethod, AppdPid)
			if err != nil {
				log.Fatal(err.Error())
			}
			pid, _ := flags.GetString(AppdPid)
			ctxPtr.LocalAuthOptions.AppdPid = pid
			if !patch {
				automatedFieldClearing(ctxPtr, AppdPid)
			}
		}
		if flags.Changed(AppdPty) {
			err := validateWriteReq(cmd, ctxPtr.AuthMethod, AppdPty)
			if err != nil {
				log.Fatal(err.Error())
			}
			pty, _ := flags.GetString(AppdPty)
			ctxPtr.LocalAuthOptions.AppdPty = pty
			if !patch {
				automatedFieldClearing(ctxPtr, AppdPty)
			}
		}
		if flags.Changed(AppdTid) {
			err := validateWriteReq(cmd, ctxPtr.AuthMethod, AppdTid)
			if err != nil {
				log.Fatal(err.Error())
			}
			tid, _ := flags.GetString(AppdTid)
			ctxPtr.LocalAuthOptions.AppdTid = tid
			if !patch {
				automatedFieldClearing(ctxPtr, AppdTid)
			}
		}
	}

	// upgrade config format from CsvFile to SecretFile, opportunistically using the update
	if ctxPtr.SecretFile == "" && ctxPtr.CsvFile != "" {
		ctxPtr.SecretFile = ctxPtr.CsvFile
		ctxPtr.CsvFile = ""
	}

	// update config file
	update := map[string]interface{}{"contexts": cfg.Contexts}
	if !contextExists && len(cfg.Contexts) == 1 { // just created the first context, set it as current
		update["current_context"] = contextName
		log.WithField("profile", contextName).Info("Setting context as current")
	}
	updateConfigFile(update)

	if contextExists {
		log.WithField("profile", contextName).Info("Updated context")
	} else {
		log.WithField("profile", contextName).Info("Created context")
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
	if getAuthFieldConfigRow(authProvider)[field] == 0 {
		return fmt.Errorf("cannot write to field %s because it is not allowed for authentication method %s", field, authProvider)
	}
	return nil
}

func clearFields(fields []string, ctxPtr *Context) {
	if slices.Contains(fields, "auth") {
		ctxPtr.AuthMethod = ""
	}
	if slices.Contains(fields, "url") {
		ctxPtr.URL = ""
	}
	if slices.Contains(fields, "server") {
		ctxPtr.Server = ""
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
	if slices.Contains(fields, "refresh_token") {
		ctxPtr.RefreshToken = ""
	}
	if slices.Contains(fields, "secret-file") {
		ctxPtr.SecretFile = ""
	}
}

func automatedFieldClearing(ctxPtr *Context, field string) {
	table := getAuthFieldClearConfig()
	clearFields(table[ctxPtr.AuthMethod][field], ctxPtr)
}
