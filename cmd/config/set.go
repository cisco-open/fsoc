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

	setContextExample = `# Set the token field on the "prod" context entry without touching other values
fsoc config set --profile prod --token=top-secret`
)

func newCmdConfigSet() *cobra.Command {

	var cmd = &cobra.Command{
		Use:         "set --profile [CONTEXT] [token=VALUE][tenant=TENANT_ID][secret-file=PATH]",
		Short:       "Create or modify a context entry in an fsoc config file",
		Long:        setContextLong,
		Args:        cobra.MaximumNArgs(1),
		Example:     setContextExample,
		Annotations: map[string]string{AnnotationForConfigBypass: ""},
		Run:         configSetContext,
	}
	cmd.Flags().String(AppdPid, "", "[Local auth type only]. The pid to send with HTTP request. Please provide raw value, and it will be encoded automatically.")
	cmd.Flags().String(AppdTid, "", "[Local auth type only]. The tid to send with HTTP request. Please provide raw value, and it will be encoded automatically.")
	cmd.Flags().String(AppdPty, "", "[Local auth type only]. The pty to send with HTTP request. Please provide raw value, and it will be encoded automatically.")
	cmd.Flags().String("auth", "", fmt.Sprintf(`Select authentication method, one of {"%v"}`, strings.Join(GetAuthMethodsStringList(), `", "`)))
	cmd.Flags().String("server", "", "Set server host name")
	_ = cmd.Flags().MarkDeprecated("server", "The --server flag is deprecated, please use --url instead.")
	cmd.Flags().String("url", "", "Set server URL (with http or https schema)")
	cmd.Flags().String("tenant", "", "Set tenant ID")
	cmd.Flags().String("token", "", "Set token value (use --token=- to get from stdin)")
	cmd.Flags().String("secret-file", "", "Set a credentials file to use for service principal (.json or .csv)")
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

func configSetContext(cmd *cobra.Command, args []string) {
	var contextName string

	// Check that either context name or current context is specified
	if len(args) > 0 {
		_ = cmd.Help()
		log.Fatalf("Unexpected args: %v", args)
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

	// update only the fields for which flags were specified explicitly
	if flags.Changed("server") {
		providedServer, _ := flags.GetString("server")
		constructedUrl := "https://" + providedServer
		cleanedUrl, err := validateUrl(constructedUrl)
		if err != nil {
			log.Fatal(err.Error())
		}
		log.Warnf("The --server option is now deprecated. In the future, please use --url instead. We will set the url to %q for you now", cleanedUrl)
		ctxPtr.URL = cleanedUrl
	}
	if flags.Changed("url") {
		providedUrl, _ := flags.GetString("url")
		cleanedUrl, err := validateUrl(providedUrl)
		if err != nil {
			log.Fatal(err.Error())
		}
		ctxPtr.URL = cleanedUrl
	}
	if flags.Changed("tenant") {
		ctxPtr.Tenant, _ = flags.GetString("tenant")
	}
	if flags.Changed("token") {
		value, _ := flags.GetString("token")
		if value == "-" { // token to come from stdin
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			ctxPtr.Token = scanner.Text()
		} else {
			ctxPtr.Token = value
		}
	}
	if flags.Changed("secret-file") {

		path, _ := flags.GetString("secret-file")
		var err error
		ctxPtr.SecretFile, err = filepath.Abs(path)
		if err != nil {
			ctxPtr.SecretFile = path
		}
		ctxPtr.CsvFile = "" // CSV file is a backward-compatibility value only
	}
	if flags.Changed("auth") {
		val, _ := flags.GetString("auth")
		if val != "" && !slices.Contains(GetAuthMethodsStringList(), val) {
			log.Fatalf(`Invalid --auth method %q; must be one of {"%v"}`, val, strings.Join(GetAuthMethodsStringList(), `", "`))
		}
		ctxPtr.AuthMethod = val
	}

	if ctxPtr.AuthMethod == AuthMethodLocal {
		if flags.Changed(AppdPid) {
			pid, _ := flags.GetString(AppdPid)
			ctxPtr.LocalAuthOptions.AppdPid = pid
		}
		if flags.Changed(AppdPty) {
			pty, _ := flags.GetString(AppdPty)
			ctxPtr.LocalAuthOptions.AppdPty = pty
		}
		if flags.Changed(AppdTid) {
			tid, _ := flags.GetString(AppdTid)
			ctxPtr.LocalAuthOptions.AppdTid = tid
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
		log.Infof("Setting context %q as current", contextName)
	}
	updateConfigFile(update)

	if contextExists {
		log.Infof("Updated context %q", contextName)
	} else {
		log.Infof("Created context %q", contextName)
	}
}
