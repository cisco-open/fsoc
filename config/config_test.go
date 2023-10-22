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
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestUpdateConfigWhenGetConfig(t *testing.T) {
	viper.SetConfigFile(".tmp-config")
	viper.SetConfigType("yaml")
	fileName := viper.ConfigFileUsed()
	fo, err := os.Create(fileName)
	assert.Nil(t, err, "Failed to create temp config file")
	oldConfigs := `
contexts:
    - name: default
      auth_method: none
      server: mytenant.saas.observer.com
current_context: default
`
	_, err = fo.Write([]byte(oldConfigs))
	fo.Close()
	assert.Nil(t, err, "Failed to write temp config file")
	defer os.Remove(fileName)
	err = viper.ReadInConfig()
	assert.Nil(t, err, "Failed to read config file")

	config := getConfig()
	assert.Equal(t, "", config.Contexts[0].Server)
	assert.Equal(t, "https://mytenant.saas.observer.com", config.Contexts[0].URL)

	err = viper.ReadInConfig()
	assert.Nil(t, err, "Failed to read config file after update")
	newContexts := viper.Get("contexts").([]Context)
	assert.Equal(t, 1, len(newContexts))
	assert.Equal(t, "", newContexts[0].Server)
	assert.Equal(t, "https://mytenant.saas.observer.com", newContexts[0].URL)
}
