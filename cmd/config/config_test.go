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
