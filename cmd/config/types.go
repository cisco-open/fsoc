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
	b64 "encoding/base64"
	"net/http"
)

const (
	DefaultConfigFile = "~/.fsoc"
	DefaultContext    = "default"
	AppdPid           = "appd-pid"
	AppdTid           = "appd-tid"
	AppdPty           = "appd-pty"
)

// Supported authentication methods
const (
	// No authentication (used in local/dev environments)
	AuthMethodNone = "none"
	// OAuth using the same user credentials as in a browser
	AuthMethodOAuth = "oauth"
	// Use JWT token directly
	AuthMethodJWT = "jwt"
	// Use a service principal
	AuthMethodServicePrincipal = "service-principal"
	// Use an agent principal
	AuthMethodAgentPrincipal = "agent-principal"
	// Use Session Manager (experimental)
	AuthMethodSessionManager = "session-manager"
	// Use for local setup
	AuthMethodLocal = "local"
)

const (
	AnnotationForConfigBypass = "config/bypass-check"
)

// Struct Context defines a full configuration context (aka access profile). The Name
// field contains the name of the context (which is unique within the config file);
// the remaining fields define the access profile.
type Context struct {
	Name             string           `json:"name" yaml:"name"`
	AuthMethod       string           `json:"auth_method,omitempty" yaml:"auth_method,omitempty" mapstructure:"auth_method"`
	Server           string           `json:"server,omitempty" yaml:"server,omitempty"` // deprecated
	URL              string           `json:"url,omitempty" yaml:"url,omitempty"`
	Tenant           string           `json:"tenant,omitempty" yaml:"tenant,omitempty"`
	User             string           `json:"user,omitempty" yaml:"user,omitempty"`
	Token            string           `json:"token,omitempty" yaml:"token,omitempty"` // access token
	RefreshToken     string           `json:"refresh_token,omitempty" yaml:"refresh_token,omitempty" mapstructure:"refresh_token"`
	CsvFile          string           `json:"csv_file,omitempty" yaml:"csv_file,omitempty"`
	SecretFile       string           `json:"secret_file,omitempty" yaml:"secret_file,omitempty" mapstructure:"secret_file"`
	LocalAuthOptions LocalAuthOptions `json:"auth-options,omitempty" yaml:"auth-options,omitempty" mapstructure:"auth-options"`
	OpenAIApiKey     string           `json:"openai-api-key,omitempty" yaml:"openai-api-key,omitempty" mapstructure:"openai-api-key"`
}

type LocalAuthOptions struct {
	AppdPty string `json:"appd-pty" yaml:"appd-pty" mapstructure:"appd-pty"`
	AppdTid string `json:"appd-tid" yaml:"appd-tid" mapstructure:"appd-tid"`
	AppdPid string `json:"appd-pid" yaml:"appd-pid" mapstructure:"appd-pid"`
}

func (opt *LocalAuthOptions) AddHeaders(req *http.Request) {
	req.Header.Add(AppdPid, b64.StdEncoding.EncodeToString([]byte(opt.AppdPid)))
	req.Header.Add(AppdPty, b64.StdEncoding.EncodeToString([]byte(opt.AppdPty)))
	req.Header.Add(AppdTid, b64.StdEncoding.EncodeToString([]byte(opt.AppdTid)))
}

// internal, to be renamed to lower case
type configFileContents struct {
	Contexts       []Context
	CurrentContext string `mapstructure:"current_context" yaml:"current_context,omitempty" json:"current_context,omitempty"`
}

// GetAuthMethodsStringList returns the list of authentication methods as strings (for join, etc.)
func GetAuthMethodsStringList() []string {
	return []string{
		AuthMethodNone,
		AuthMethodOAuth,
		AuthMethodServicePrincipal,
		AuthMethodAgentPrincipal,
		AuthMethodJWT,
		AuthMethodLocal,
	}
}
