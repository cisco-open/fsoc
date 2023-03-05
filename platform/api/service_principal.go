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
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/apex/log"

	"github.com/cisco-open/fsoc/cmd/config"
)

type tokenStruct struct {
	AccessToken      string `json:"access_token"`
	ExpiresInSeconds int    `json:"expires_in"`
	Scope            string `json:"scope"` //TODO: parse as a list
	TokenType        string `json:"token_type"`
}

type credentialsStruct struct {
	TenantID string `json:"Tenant ID"`
	TokenURL string `json:"Token URL"`
	ClientID string `json:"Client ID"`
	Secret   string `json:"Secret"`
}

// servicePrincipalLogin performs a login into the platform API and updates the token(s) in the provided context
func servicePrincipalLogin(cfg *config.Context) error {
	log.Infof("Starting login flow using a service principal")

	// read service principal credentials file
	file := cfg.SecretFile
	if file == "" {
		file = cfg.CsvFile // implicitly update config schema
	}
	credentials, err := readCredentials(file)
	if err != nil {
		log.Errorf("Failed to read credentials file %q: %v", file, err.Error())
		return err
	}

	// check/backfill missing fields (the new, JSON, format of the credentials has tenant and server URL)
	if cfg.Tenant == "" {
		// bail if using the old CSV format which does not provide tenant ID
		// (we can resolve the tenant ID from the server URL but no need to do this for legacy CSV service principal format)
		if credentials.TenantID == "" {
			return fmt.Errorf("Missing tenant ID, please specify using `fsoc config set --tenant=TENANTID`")
		}
		cfg.Tenant = credentials.TenantID // the new JSON credentials format has the tenant
		log.Infof("Extracted tenant ID %q from the credentials file", cfg.Tenant)
	}
	if cfg.URL == "" {
		if credentials.TokenURL == "" {
			return fmt.Errorf("Missing Server URL, please specify using `fsoc config set --url=SERVERURL`")
		}
		urlStruct, err := url.Parse(credentials.TokenURL)
		if err != nil {
			return fmt.Errorf("Failed to parse server URL from the credentials token URL, %q: %v", credentials.TokenURL, err)
		}
		cfg.URL = urlStruct.Scheme + "://" + urlStruct.Host
		log.Infof("Extracted server URL %q from the credentials file", cfg.URL)
	}

	// create a HTTP request

	url, err := url.Parse(cfg.URL)
	if err != nil {
		log.Fatalf("Failed to parse the url provided in context. URL: %s, err: %s", cfg.URL, err)
	}
	url.Path = "auth/" + cfg.Tenant + "/default/oauth2/token"

	client := &http.Client{}
	req, err := http.NewRequest("POST", url.String(), strings.NewReader("grant_type=client_credentials")) //TODO: urlencode data!
	if err != nil {
		log.Errorf("Failed to create a request %q: %v", url.String(), err.Error())
		return err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	req.SetBasicAuth(credentials.ClientID, credentials.Secret)

	// execute request
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("GET request to %q failed: %v", url.String(), err.Error())
		//TODO: provide more details from the response object for enhanced error info
		return err
	}
	if resp.StatusCode != 200 {
		// log error here before trying to parse body, more processing later
		log.Errorf("Login failed, status %q", resp.Status)
		// fall through to reading the payload body for more error info
	}

	// read body (success or error)
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Failed reading login response from %q: %v", url.String(), err.Error())
		return err
	}

	// report error if failed & return
	if resp.StatusCode != 200 { // exactly 200 expected, none of the other 2xx is good
		var errobj any

		// try to unmarshal JSON
		err := json.Unmarshal(respBytes, &errobj)
		if err != nil {
			// process as a string instead, ignore parsing error
			errobj = bytes.NewBuffer(respBytes).String()
		}
		return fmt.Errorf("Error response: %+v", errobj)
	}

	// update context with token
	var token tokenStruct
	err = json.Unmarshal(respBytes, &token)
	if err != nil {
		log.Errorf("Failed to parse token: %v", err.Error())
		return err
	}
	log.Info("Login returned a valid token")
	cfg.Token = token.AccessToken

	return nil
}

func readCredentials(file string) (*credentialsStruct, error) {
	file = expandPath(file)
	if strings.ToLower(path.Ext(file)) == ".csv" {
		// handle legacy format (only if .csv extension)
		return readCsvCredentials(file)
	} else {
		// assume new, json format
		return readJsonCredentials(file)
	}
}

func readJsonCredentials(file string) (*credentialsStruct, error) {
	f, err := os.Open(file)
	if err != nil {
		log.Errorf("Failed to open the credentials JSON file %q: %v", file, err.Error())
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		log.Errorf("Failed to read the credentials JSON file %q: %v", file, err.Error())
		return nil, err
	}

	var credentials credentialsStruct
	if err = json.Unmarshal(data, &credentials); err != nil {
		log.Errorf("Failed to parse credentials JSON file %q: %v", file, err.Error())
		return nil, err
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

// Expand ~ in the path
func expandPath(file string) string {
	if strings.HasPrefix(file, "~/") {
		dirname, _ := os.UserHomeDir()
		file = filepath.Join(dirname, file[2:])
	}
	return file
}
