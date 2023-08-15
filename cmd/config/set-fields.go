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

type AuthFieldConfig int8

const (
	ClearField AuthFieldConfig = 0
	AllowField AuthFieldConfig = 1
)

type AuthFieldConfigRow map[string]AuthFieldConfig

func getAuthFieldWritePermissions() map[string]AuthFieldConfigRow {
	return map[string]AuthFieldConfigRow{
		AuthMethodNone: {
			"client-ID":     ClearField,
			"secret-file":   ClearField,
			"token":         ClearField,
			"tenant":        ClearField,
			"url":           AllowField,
			"refresh-token": ClearField,
			"user":          ClearField,
			AppdTid:         ClearField,
			AppdPty:         ClearField,
			AppdPid:         ClearField,
		},
		AuthMethodOAuth: {
			"client-ID":     ClearField,
			"secret-file":   ClearField,
			"token":         ClearField,
			"tenant":        ClearField,
			"url":           AllowField,
			"refresh-token": ClearField,
			"user":          ClearField,
			AppdTid:         ClearField,
			AppdPty:         ClearField,
			AppdPid:         ClearField,
		},
		AuthMethodJWT: {
			"client-ID":     ClearField,
			"secret-file":   ClearField,
			"token":         AllowField,
			"tenant":        AllowField,
			"url":           AllowField,
			"refresh-token": ClearField,
			"user":          AllowField,
			AppdTid:         ClearField,
			AppdPty:         ClearField,
			AppdPid:         ClearField,
		},
		AuthMethodServicePrincipal: {
			"client-ID":     ClearField,
			"secret-file":   AllowField,
			"token":         ClearField,
			"tenant":        ClearField,
			"url":           AllowField,
			"refresh-token": ClearField,
			"user":          ClearField,
			AppdTid:         ClearField,
			AppdPty:         ClearField,
			AppdPid:         ClearField,
		},
		AuthMethodAgentPrincipal: {
			"client-ID":     ClearField,
			"secret-file":   AllowField,
			"token":         ClearField,
			"tenant":        ClearField,
			"url":           AllowField,
			"refresh-token": ClearField,
			"user":          ClearField,
			AppdTid:         ClearField,
			AppdPty:         ClearField,
			AppdPid:         ClearField,
		},
		AuthMethodLocal: {
			"client-ID":     ClearField,
			"secret-file":   ClearField,
			"token":         ClearField,
			"tenant":        ClearField,
			"url":           AllowField,
			"refresh-token": ClearField,
			"user":          ClearField,
			AppdTid:         AllowField,
			AppdPty:         AllowField,
			AppdPid:         AllowField,
		},
	}
}

type authClearFields map[string][]string

// returns a map which dictates which fields to remove when changing config settings
func getAuthFieldClearConfig() map[string]authClearFields {
	return map[string]authClearFields{
		AuthMethodNone: {
			"client-ID":     {},
			"secret-file":   {},
			"token":         {},
			"tenant":        {},
			"url":           {},
			"refresh-token": {},
			"user":          {},
			AppdTid:         {},
			AppdPty:         {},
			AppdPid:         {},
		},
		AuthMethodOAuth: {
			"client-ID":     {}, //NA
			"secret-file":   {}, //NA
			"token":         {}, //NA
			"tenant":        {}, //NA
			"url":           {"tenant", "user", "token", "refresh_token", "secret-file"},
			"refresh-token": {}, //NA
			"user":          {}, //NA
			AppdTid:         {}, //NA
			AppdPty:         {}, //NA
			AppdPid:         {}, //NA
		},
		AuthMethodJWT: {
			"client-ID":     {},
			"secret-file":   {},
			"token":         {},
			"tenant":        {"token", "user"},
			"url":           {"token", "tenant", "user"},
			"refresh-token": {},
			"user":          {},
			AppdTid:         {},
			AppdPty:         {},
			AppdPid:         {},
		},
		AuthMethodServicePrincipal: {
			"client-ID":     {},
			"secret-file":   {"url", "server", "tenant", "user", "token", "refresh_token"},
			"token":         {},
			"tenant":        {},
			"url":           {"tenant", "user", "token", "refresh_token"},
			"refresh-token": {},
			"user":          {},
			AppdTid:         {},
			AppdPty:         {},
			AppdPid:         {},
		},
		AuthMethodAgentPrincipal: {
			"client-ID":     {},
			"secret-file":   {"url", "server", "tenant", "user", "token", "refresh_token"},
			"token":         {},
			"tenant":        {},
			"url":           {"tenant", "user", "token", "refresh_token"},
			"refresh-token": {},
			"user":          {},
			AppdTid:         {},
			AppdPty:         {},
			AppdPid:         {},
		},
		AuthMethodLocal: {
			"client-ID":     {},
			"secret-file":   {},
			"token":         {},
			"tenant":        {},
			"url":           {},
			"refresh-token": {},
			"user":          {},
			AppdTid:         {},
			AppdPty:         {},
			AppdPid:         {},
		},
	}
}
