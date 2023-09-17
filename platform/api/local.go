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

// Package api provides access to the platform API, in all forms supported
// by the config context (aka access profile)
package api

import (
	"encoding/base64"
	"net/http"

	"github.com/cisco-open/fsoc/config"
)

func AddLocalAuthReqHeaders(req *http.Request, opt *config.LocalAuthOptions) {
	req.Header.Add(config.AppdPid, base64.StdEncoding.EncodeToString([]byte(opt.AppdPid)))
	req.Header.Add(config.AppdPty, base64.StdEncoding.EncodeToString([]byte(opt.AppdPty)))
	req.Header.Add(config.AppdTid, base64.StdEncoding.EncodeToString([]byte(opt.AppdTid)))
}
