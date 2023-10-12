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

package api

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/apex/log"

	"github.com/cisco-open/fsoc/config"
)

// requiredSettings defines what config.Context fields are required for each authentication method
var requiredSettings = map[string][]string{
	config.AuthMethodNone:             {},
	config.AuthMethodLocal:            {"LocalAuthOptions.AppdPty", "LocalAuthOptions.AppdPid", "LocalAuthOptions.AppdTid"},
	config.AuthMethodOAuth:            {"URL"},
	config.AuthMethodServicePrincipal: {"SecretFile"},   // tenant and server can usually be obtained from the file
	config.AuthMethodAgentPrincipal:   {"SecretFile"},   // tenant and server can usually be obtained from the file
	config.AuthMethodJWT:              {"URL", "Token"}, // tenant is desired but may not be mandatory for all requests
}

// fieldToFlag maps a config.Context field to CLI flag name, so that we can display better
// help/error message for missing fields
var fieldToFlag = map[string]string{
	"Url":                      "url",
	"Token":                    "token",
	"SecretFile":               "secret-file",
	"AuthMethod":               "auth",
	"Tenant":                   "tenant",
	"LocalAuthOptions.AppdPty": "appd-pty",
	"LocalAuthOptions.AppdPid": "appd-pid",
	"LocalAuthOptions.AppdTid": "appd-tid",
}

// Login performs a login into the platform API and saves the provided access token.
// Login respects different access profile types (when supported) to provide the correct
// login mechanism for each.
func Login() error {
	callCtx := newCallContext()
	defer callCtx.stopSpinner(false) // ensure not running when returning

	return login(callCtx)
}

func login(callCtx *callContext) error {
	log.Infof("Login is forced in order to get a valid access token")

	// check current context for required fields
	cfg := callCtx.cfg
	if err := checkConfigForAuth(cfg); err != nil {
		return err
	}

	var authErr error
	switch cfg.AuthMethod {
	case config.AuthMethodLocal:
		authErr = nil
	case config.AuthMethodNone:
		authErr = nil // nothing to do
	case config.AuthMethodJWT:
		authErr = nil // nothing to do (TODO: we may check its validity by executing a no-op request)
	case config.AuthMethodServicePrincipal:
		authErr = servicePrincipalLogin(callCtx)
	case config.AuthMethodAgentPrincipal:
		authErr = agentPrincipalLogin(callCtx)
	case config.AuthMethodOAuth:
		authErr = oauthLogin(callCtx)
	default:
		panic(fmt.Sprintf("bug: unhandled authentication method %q", cfg.AuthMethod))
	}
	if authErr != nil {
		return authErr
	}

	// update current context with logged in credentials (token(s)) to use
	config.ReplaceCurrentContext(cfg)

	// reload context
	callCtx.cfg = config.GetCurrentContext()

	return nil
}

func nonZeroStructFields(theStruct *config.Context) []string {
	if theStruct == nil {
		return []string{}
	}
	structValue := reflect.ValueOf(*theStruct) // must be a struct, bug otherwise
	return recursiveWalkThrough("", structValue)
}

func recursiveWalkThrough(prefix string, value reflect.Value) []string {
	nonZeroFields := []string{}
	for _, field := range reflect.VisibleFields(value.Type()) {
		currentField := value.FieldByIndex(field.Index)
		if !currentField.IsZero() {
			nonZeroFields = append(nonZeroFields, prefix+field.Name)
			if currentField.Kind() == reflect.Struct {
				nestedFields := recursiveWalkThrough(field.Name+".", value.FieldByIndex(field.Index))
				nonZeroFields = append(nonZeroFields, nestedFields...)
			}
		}
	}
	return nonZeroFields
}

// checkConfigForAuth checks that all required configuration is available for the selected
// authentication method, with detailed error messages and suggested remedies
func checkConfigForAuth(cfg *config.Context) error {
	// fail if not configured
	if cfg == nil {
		return fmt.Errorf("fsoc is not configured, please run 'fsoc config set' first")
	}

	// collect list of config fields that are defined
	fieldsPresent := nonZeroStructFields(cfg)
	if cfg.AuthMethod == "" {
		cfg.AuthMethod = config.AuthMethodServicePrincipal // backward compatibility
	}

	// get method's required settings
	required, methodFound := requiredSettings[cfg.AuthMethod]

	// fail if method is not supported
	if !methodFound {
		methods := []string{}
		for k := range requiredSettings { // look at keys only to determine which methods are supported
			methods = append(methods, k)
		}
		return fmt.Errorf(`authentication method %q is not supported yet, please use one of {"%v"} 
		Example:
		fsoc config set auth=oauth url=https://MYTENANT.observe.appdynamics.com`, cfg.AuthMethod, strings.Join(methods, `", "`))
	}

	// fail if any of the required settings for this method are not set
	missing := []string{}
	for _, requiredField := range required {
		found := false
		for _, presentField := range fieldsPresent {
			if requiredField == presentField {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, requiredField)
		}
	}
	if len(missing) > 0 {
		missList := []string{}
		for _, field := range missing {
			missList = append(missList, fieldToFlag[field])
		}
		usage := `Use "fsoc config set [--config CONFIG_FILE] [--profile=PROFILE] auth=AUTH_METHOD ..."`
		return fmt.Errorf("the current context is missing required configuration to perform a login: %v\n%v", strings.Join(missList, ","), usage)
	}

	return nil
}
