// Copyright 2023 Cisco Systems, Inc.
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
	cfg "github.com/cisco-open/fsoc/config"
)

type AuthFieldConfig int8

const (
	ClearField AuthFieldConfig = 0
	AllowField AuthFieldConfig = 1
)

type AuthFieldConfigRow map[string]AuthFieldConfig

func getAuthFieldWritePermissions() map[string]AuthFieldConfigRow {
	return map[string]AuthFieldConfigRow{
		cfg.AuthMethodNone: {
			"client-ID":     ClearField,
			"secret-file":   ClearField,
			"token":         ClearField,
			"tenant":        ClearField,
			"url":           AllowField,
			"refresh-token": ClearField,
			"user":          ClearField,
			cfg.AppdTid:     ClearField,
			cfg.AppdPty:     ClearField,
			cfg.AppdPid:     ClearField,
		},
		cfg.AuthMethodOAuth: {
			"client-ID":     ClearField,
			"secret-file":   ClearField,
			"token":         ClearField,
			"tenant":        ClearField,
			"url":           AllowField,
			"refresh-token": ClearField,
			"user":          ClearField,
			cfg.AppdTid:     ClearField,
			cfg.AppdPty:     ClearField,
			cfg.AppdPid:     ClearField,
		},
		cfg.AuthMethodJWT: {
			"client-ID":     ClearField,
			"secret-file":   ClearField,
			"token":         AllowField,
			"tenant":        AllowField,
			"url":           AllowField,
			"refresh-token": ClearField,
			"user":          AllowField,
			cfg.AppdTid:     ClearField,
			cfg.AppdPty:     ClearField,
			cfg.AppdPid:     ClearField,
		},
		cfg.AuthMethodServicePrincipal: {
			"client-ID":     ClearField,
			"secret-file":   AllowField,
			"token":         ClearField,
			"tenant":        AllowField,
			"url":           AllowField,
			"refresh-token": ClearField,
			"user":          ClearField,
			cfg.AppdTid:     ClearField,
			cfg.AppdPty:     ClearField,
			cfg.AppdPid:     ClearField,
		},
		cfg.AuthMethodAgentPrincipal: {
			"client-ID":     ClearField,
			"secret-file":   AllowField,
			"token":         ClearField,
			"tenant":        AllowField,
			"url":           AllowField,
			"refresh-token": ClearField,
			"user":          ClearField,
			cfg.AppdTid:     ClearField,
			cfg.AppdPty:     ClearField,
			cfg.AppdPid:     ClearField,
		},
		cfg.AuthMethodLocal: {
			"client-ID":     ClearField,
			"secret-file":   ClearField,
			"token":         ClearField,
			"tenant":        ClearField,
			"url":           AllowField,
			"refresh-token": ClearField,
			"user":          ClearField,
			cfg.AppdTid:     AllowField,
			cfg.AppdPty:     AllowField,
			cfg.AppdPid:     AllowField,
		},
	}
}

type authClearFields map[string][]string

// returns a map which dictates which fields to remove when changing config settings
func getAuthFieldClearConfig() map[string]authClearFields {
	return map[string]authClearFields{
		cfg.AuthMethodNone: {
			"client-ID":     {},
			"secret-file":   {},
			"token":         {},
			"tenant":        {},
			"url":           {},
			"refresh-token": {},
			"user":          {},
			cfg.AppdTid:     {},
			cfg.AppdPty:     {},
			cfg.AppdPid:     {},
		},
		cfg.AuthMethodOAuth: {
			"client-ID":     {}, //NA
			"secret-file":   {}, //NA
			"token":         {}, //NA
			"tenant":        {}, //NA
			"url":           {"tenant", "user", "token", "refresh-token", "secret-file"},
			"refresh-token": {}, //NA
			"user":          {}, //NA
			cfg.AppdTid:     {}, //NA
			cfg.AppdPty:     {}, //NA
			cfg.AppdPid:     {}, //NA
		},
		cfg.AuthMethodJWT: {
			"client-ID":     {},
			"secret-file":   {},
			"token":         {},
			"tenant":        {"token", "user"},
			"url":           {"token", "tenant", "user"},
			"refresh-token": {},
			"user":          {},
			cfg.AppdTid:     {},
			cfg.AppdPty:     {},
			cfg.AppdPid:     {},
		},
		cfg.AuthMethodServicePrincipal: {
			"client-ID":     {},
			"secret-file":   {"url", "tenant", "user", "token", "refresh-token"},
			"token":         {},
			"tenant":        {},
			"url":           {"tenant", "user", "token", "refresh-token"},
			"refresh-token": {},
			"user":          {},
			cfg.AppdTid:     {},
			cfg.AppdPty:     {},
			cfg.AppdPid:     {},
		},
		cfg.AuthMethodAgentPrincipal: {
			"client-ID":     {},
			"secret-file":   {"url", "tenant", "user", "token", "refresh-token"},
			"token":         {},
			"tenant":        {},
			"url":           {"tenant", "user", "token", "refresh-token"},
			"refresh-token": {},
			"user":          {},
			cfg.AppdTid:     {},
			cfg.AppdPty:     {},
			cfg.AppdPid:     {},
		},
		cfg.AuthMethodLocal: {
			"client-ID":     {},
			"secret-file":   {},
			"token":         {},
			"tenant":        {},
			"url":           {},
			"refresh-token": {},
			"user":          {},
			cfg.AppdTid:     {},
			cfg.AppdPty:     {},
			cfg.AppdPid:     {},
		},
	}
}
