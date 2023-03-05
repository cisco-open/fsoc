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

	"github.com/cisco-open/fsoc/cmd/config"
)

// requiredSettings defines what config.Context fields are required for each authentication method
var requiredSettings = map[string][]string{
	config.AuthMethodNone:             {},
	config.AuthMethodOAuth:            {"Server"},
	config.AuthMethodServicePrincipal: {"SecretFile"},      // tenant and server can usually be obtained from the file (new, JSON format)
	config.AuthMethodJWT:              {"Server", "Token"}, // tenant is desired but may not be mandatory for all requests
}

// fieldToFlag maps a config.Context field to CLI flag name, so that we can display better
// help/error message for missing fields
var fieldToFlag = map[string]string{
	"Server":     "server",
	"Token":      "token",
	"SecretFile": "secret-filer",
	"AuthMethod": "auth",
	"Tenant":     "tenant",
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
	case config.AuthMethodNone:
		authErr = nil // nothing to do
	case config.AuthMethodJWT:
		authErr = nil // nothing to do (TODO: we may check its validity by executing a no-op request)
	case config.AuthMethodServicePrincipal:
		authErr = servicePrincipalLogin(callCtx)
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
	nonZeroFields := []string{}
	for _, field := range reflect.VisibleFields(structValue.Type()) {
		if !structValue.FieldByIndex(field.Index).IsZero() {
			nonZeroFields = append(nonZeroFields, field.Name)
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
		return fmt.Errorf(`Authentication method %q is not supported yet, please use one of {"%v"} 
		Example:
		fsoc config set --secret-file=~/secret.json --auth=service-principal`, cfg.AuthMethod, strings.Join(methods, `", "`))
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
		usage := "Use `fsoc config set [--profile=PROFILE] --server=myhost.mydomain.com --tenant=TENANT --secret-file=CREDENTIALS`"
		return fmt.Errorf("The current context is missing required configuration to perform a login: %v\n%v", strings.Join(missList, ","), usage)
	}

	return nil
}
