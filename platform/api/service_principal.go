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
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/apex/log"
	"gopkg.in/yaml.v3"
)

type tokenStruct struct {
	AccessToken      string `json:"access_token"`
	ExpiresInSeconds int    `json:"expires_in"`
	Scope            string `json:"scope"` //TODO: parse as a list
	TokenType        string `json:"token_type"`
}

// json-style solution principal credentials file format
type credentialsStruct struct {
	TenantID string `json:"Tenant ID"`
	TokenURL string `json:"Token URL"`
	ClientID string `json:"Client ID"`
	Secret   string `json:"Secret"`
}

// json response to "/administration/v1beta/clients/agents" request. Only the
// "id" and "clientSecret" fields are needed; the others can be ignored
type agentCredentialsStruct struct {
	ClientID string `json:"id"`
	Secret   string `json:"clientSecret"`
	// "displayName": "IAM Collector 1"
	// "description": "This is collector test agent."
	// "authType": "client_secret_basic"
	// "hasRotatedSecrets": false
	// "createdAt": "2023-03-22T02:04:51.000Z"
	// "updatedAt": "2023-03-22T02:04:51.757Z"
}

// agent principal credentials returned as values.yaml for k8s agent helm chart
type helmSettingsStruct struct {
	Global      any                   `yaml:"global,omitempty"`
	Credentials helmCredentialsStruct `yaml:"appdynamics-otel-collector,omitempty"`
}

type helmCredentialsStruct struct {
	ClientID     string `yaml:"clientId"`
	ClientSecret string `yaml:"clientSecret"`
	DataEndpoint string `yaml:"endpoint"`
	TokenURL     string `yaml:"tokenUrl"`
}

// servicePrincipalLogin performs a login into the platform API and updates the token(s) in the provided context
func servicePrincipalLogin(ctx *callContext) error {
	// read credentials file
	file := ctx.cfg.SecretFile
	if file == "" {
		file = ctx.cfg.CsvFile // implicitly update config schema (backward compatibility)
	}
	credentials, err := readServiceCredentials(file)
	if err != nil {
		return fmt.Errorf("Failed to read credentials file %q: %v", file, err)
	}

	return agentOrServicePrincipalLogin(ctx, "service principal", credentials)
}

// agentPrincipalLogin performs a login into the platform API and updates the token(s) in the provided context
func agentPrincipalLogin(ctx *callContext) error {
	// read credentials file
	file := ctx.cfg.SecretFile
	credentials, err := readAgentCredentials(file)
	if err != nil {
		return fmt.Errorf("Failed to read credentials file %q: %v", file, err)
	}

	return agentOrServicePrincipalLogin(ctx, "agent principal", credentials)
}

// agentOrServicePrincipalLogin performs a login into the platform API and updates the token(s) in the provided context
func agentOrServicePrincipalLogin(ctx *callContext, principalType string, credentials *credentialsStruct) error {
	log.Infof("Starting login flow using %v", principalType)

	// check/backfill missing fields (the new, JSON, format of the credentials has tenant and server URL)
	if ctx.cfg.Tenant == "" {
		// some credentials formats include the tenant, use it if it's provided
		if credentials.TenantID == "" {
			return fmt.Errorf("Missing tenant ID, please specify using `fsoc config set --tenant=TENANTID`")
		}
		ctx.cfg.Tenant = credentials.TenantID
		log.WithField("tenantID", ctx.cfg.Tenant).Info("Extracted tenant ID from the credentials file")
	}
	if ctx.cfg.URL == "" {
		// some credentials formats provide the tokenURL from which we can get the server URL
		if credentials.TokenURL == "" {
			return fmt.Errorf("Missing server URL, please specify using `fsoc config set --url=SERVERURL`")
		}
		urlStruct, err := url.Parse(credentials.TokenURL)
		if err != nil {
			return fmt.Errorf("Failed to parse server URL from the credentials token URL, %q: %v", credentials.TokenURL, err)
		}
		ctx.cfg.URL = urlStruct.Scheme + "://" + urlStruct.Host
		log.WithField("url", ctx.cfg.URL).Info("Extracted server URL from the credentials file")
	}

	// create a HTTP request
	url, err := url.Parse(ctx.cfg.URL)
	if err != nil {
		log.Fatalf("Failed to parse the url provided in context. URL: %s, err: %s", ctx.cfg.URL, err)
	}
	url.Path = "auth/" + ctx.cfg.Tenant + "/default/oauth2/token"

	client := &http.Client{}
	req, err := http.NewRequest("POST", url.String(), strings.NewReader("grant_type=client_credentials")) //TODO: urlencode data!
	if err != nil {
		return fmt.Errorf("Failed to create a request for %q: %v", url.String(), err)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	req.SetBasicAuth(credentials.ClientID, credentials.Secret)

	// execute request
	ctx.startSpinner(fmt.Sprintf("Exchange %v for auth token", principalType))
	resp, err := client.Do(req)
	ctx.stopSpinner(err == nil && resp.StatusCode == 200)
	if err != nil {
		return fmt.Errorf("Failed to request auth (%q): %w", url.String(), err)
	}
	if resp.StatusCode != 200 {
		// log error here before trying to parse body, more processing later
		log.Errorf("Login failed, status %q; details to follow", resp.Status)
		// fall through to reading the payload body for more error info
	}

	// read body (success or error)
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Failed reading login response from %q: %w", url.String(), err)
	}

	// report error if failed & return
	if resp.StatusCode != 200 { // exactly 200 expected, none of the other 2xx is good
		return parseIntoError(resp, respBytes)
	}

	// update context with token
	var token tokenStruct
	err = json.Unmarshal(respBytes, &token)
	if err != nil {
		log.Errorf("Failed to parse token: %v", err.Error())
		return err
	}
	log.Info("Login returned a valid token")
	ctx.cfg.Token = token.AccessToken

	return nil
}

func readServiceCredentials(file string) (*credentialsStruct, error) {
	ext := strings.ToLower(path.Ext(file))
	if ext == ".csv" {
		// handle legacy format (only if .csv extension)
		return readCsvCredentials(file)
	} else {
		// assume new, json format, service or agent principal
		return readJsonCredentials(file)
	}
}

func readAgentCredentials(file string) (*credentialsStruct, error) {
	ext := strings.ToLower(path.Ext(file))
	if ext == ".yaml" { // agent principal from a helm chart
		return readAgentHelmCredentials(file)
	} else {
		// assume json format
		return readAgentJsonCredentials(file)
	}
}

func readJsonCredentials(file string) (*credentialsStruct, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("Failed to open the credentials file %q: %w", file, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("Failed to read the credentials file %q: %w", file, err)
	}

	var credentials credentialsStruct
	if err = json.Unmarshal(data, &credentials); err != nil {
		return nil, fmt.Errorf("Failed to parse credentials file %q: %w", file, err)
	}

	return &credentials, nil
}

func readCsvCredentials(file string) (*credentialsStruct, error) {
	// open and read the CSV file
	f, err := os.Open(file)
	if err != nil {
		log.Errorf("Failed to open the credentials CSV file %q: %v", file, err.Error())
		return nil, err
	}
	defer f.Close()
	csvReader := csv.NewReader(f) // nb: f is os.File which implements io.Reader, good for small files like this one
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Errorf("Failed to read and parse the credentials CSV file %q: %v", file, err.Error())
		return nil, err
	}

	// check CSV file contents (assuming it parsed OK)
	if len(records) != 2 {
		err := fmt.Errorf("credentials CSV file %q has %v records, expected 2", file, len(records))
		log.Errorf("Unexpected credentials format: %v", err)
		return nil, err
	}
	headings := records[0]
	if len(headings) != 2 || headings[0] != "Client ID" || headings[1] != "Secret" {
		err := fmt.Errorf("credentials CSV file %q has unexpected columns %+v, expected [\"Client ID\" \"Secret\"]", file, headings)
		log.Errorf("Unexpected credentials format: %v", err)
		return nil, err
	}
	values := records[1]
	if len(records) != 2 {
		err := fmt.Errorf("credentials CSV file %q has %v values, expected 2", file, len(values))
		log.Errorf("Unexpected credentials format: %v", err)
		return nil, err
	}

	// fill in and return (tenant ID and tokenURL are not provided in CSV)
	return &credentialsStruct{
		ClientID: values[0],
		Secret:   values[1],
		TenantID: "",
		TokenURL: "",
	}, nil
}

func readAgentHelmCredentials(file string) (*credentialsStruct, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("Failed to open the credentials file %q: %w", file, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("Failed to read the credentials file %q: %w", file, err)
	}

	// read YAML file with helm credentials for an agent
	var helmVars helmSettingsStruct
	err = yaml.Unmarshal(data, &helmVars)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse credentials file %q: %w", file, err)
	}

	// extract tenant ID from tokenURL (best effort)
	var tenantID string
	tokenURL, err := url.Parse(helmVars.Credentials.TokenURL)
	if err == nil {
		elements := strings.Split(tokenURL.Path, "/")
		if len(elements) >= 3 {
			tenantID = elements[2]
		}
	}

	return &credentialsStruct{
		ClientID: helmVars.Credentials.ClientID,
		Secret:   helmVars.Credentials.ClientSecret,
		TokenURL: helmVars.Credentials.TokenURL,
		TenantID: tenantID,
	}, nil
}

func readAgentJsonCredentials(file string) (*credentialsStruct, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("Failed to open the credentials file %q: %w", file, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("Failed to read the credentials file %q: %w", file, err)
	}

	var agentCredentials agentCredentialsStruct
	if err = json.Unmarshal(data, &agentCredentials); err != nil {
		return nil, fmt.Errorf("Failed to parse credentials file %q: %w", file, err)
	}

	return &credentialsStruct{
		ClientID: agentCredentials.ClientID,
		Secret:   agentCredentials.Secret,
	}, nil
}
