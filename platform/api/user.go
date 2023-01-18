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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type user struct {
	ID string `json:"sub"`
}

func extractUser(accessToken string) (string, error) {
	var userData user
	metaDataStringArray := strings.Split(accessToken, ".")
	if len(metaDataStringArray) < 3 {
		return "", fmt.Errorf("Invalid bearer token detected")
	}

	// try to decode metadata token
	metaDataString := metaDataStringArray[1]
	decodedMetaDataBytes, err := base64.RawStdEncoding.DecodeString(metaDataString)
	if err != nil {
		return "", fmt.Errorf("Failed to decode base64 string: %v", err.Error())
	}
	if err := json.Unmarshal(decodedMetaDataBytes, &userData); err != nil {
		return "", fmt.Errorf("Failed to JSON parse the `sub` from the decoded bearer token with error %v", err.Error())
	}

	return userData.ID, nil
}
