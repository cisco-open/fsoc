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
	"errors"
	"fmt"
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
	Name             string                    `json:"name" yaml:"name" mapstructure:"name"`
	AuthMethod       string                    `json:"auth_method" yaml:"auth_method" mapstructure:"auth_method"`
	Server           string                    `json:"server,omitempty" yaml:"server,omitempty" mapstructure:"server,omitempty"` // deprecated
	URL              string                    `json:"url" yaml:"url" mapstructure:"url"`
	Tenant           string                    `json:"tenant,omitempty" yaml:"tenant,omitempty" mapstructure:"tenant,omitempty"`
	User             string                    `json:"user,omitempty" yaml:"user,omitempty" mapstructure:"user,omitempty"`
	Token            string                    `json:"token,omitempty" yaml:"token,omitempty" mapstructure:"token,omitempty"` // access token
	RefreshToken     string                    `json:"refresh_token,omitempty" yaml:"refresh_token,omitempty" mapstructure:"refresh_token,omitempty"`
	CsvFile          string                    `json:"csv_file,omitempty" yaml:"csv_file,omitempty" mapstructure:"csv_file,omitempty"`
	SecretFile       string                    `json:"secret_file,omitempty" yaml:"secret_file,omitempty" mapstructure:"secret_file,omitempty"`
	EnvType          string                    `json:"env_type,omitempty" yaml:"env_type,omitempty" mapstructure:"env_type,omitempty"`
	LocalAuthOptions LocalAuthOptions          `json:"auth-options,omitempty" yaml:"auth-options,omitempty" mapstructure:"auth-options,omitempty"`
	SubsystemConfigs map[string]map[string]any `json:"subsystems,omitempty" yaml:"subsystems,omitempty" mapstructure:"subsystems,omitempty"`
	// Note: when adding fields, remember to add display for them in get.go
}

type LocalAuthOptions struct {
	AppdPty string `json:"appd-pty" yaml:"appd-pty" mapstructure:"appd-pty"`
	AppdTid string `json:"appd-tid" yaml:"appd-tid" mapstructure:"appd-tid"`
	AppdPid string `json:"appd-pid" yaml:"appd-pid" mapstructure:"appd-pid"`
}

func (o *LocalAuthOptions) String() string {
	if o.AppdPid == "" && o.AppdTid == "" && o.AppdPty == "" {
		return ""
	}
	return fmt.Sprintf("appd-pty=%v appd-pid=%v appd-tid=%v", o.AppdPty, o.AppdPid, o.AppdTid)
}

type configFileContents struct {
	Contexts       []Context
	CurrentContext string `mapstructure:"current_context" yaml:"current_context,omitempty" json:"current_context,omitempty"`
}

// Errors

var ErrProfileNotFound = errors.New("profile not found")

type ErrSubsystemParsingError struct {
	SubsystemName string
	ParsingError  error
}

func (e *ErrSubsystemParsingError) Error() string {
	return fmt.Sprintf("failed to parse configuration for subsystem %q: %v", e.SubsystemName, e.ParsingError)
}

func (e *ErrSubsystemParsingError) Unwrap() error {
	return e.ParsingError
}

type ErrSubsystemNotFound struct {
	SubsystemName string
}

func (e *ErrSubsystemNotFound) Error() string {
	return fmt.Sprintf("unknown subsystem %q", e.SubsystemName)
}

type ErrSubsystemSettingNotFound struct {
	SubsystemName string
	SettingName   string
}

func (e *ErrSubsystemSettingNotFound) Error() string {
	return fmt.Sprintf("unknown setting %q for subsystem %q", e.SettingName, e.SubsystemName)
}
