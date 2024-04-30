package test

import (
	"fmt"
	"os"

	"github.com/apex/log"
	"github.com/spf13/viper"

	"github.com/cisco-open/fsoc/config"
)

var TEST_CONTEXT_NAME string = `__test__`
var TEST_CONFIG_FILE_NAME string = `__test_fsoc__`

// SetActiveConfigProfileServer creates a test profile configured with the given URL. Note the original config will
// need to be restored which is most easily accomplished with the returned teardown method by defering it in the caller like such
//
//	server := httptest.NewServer(...)
//	defer config.SetActiveConfigProfileServer(server.URL)()
func SetActiveConfigProfileServer(serverUrl string) (teardown func()) {
	testContext := &config.Context{
		Name:       TEST_CONTEXT_NAME,
		AuthMethod: config.AuthMethodNone,
		URL:        serverUrl,
	}
	testConfigFile := fmt.Sprintf("%v/%v", os.TempDir(), TEST_CONFIG_FILE_NAME)

	filename := viper.ConfigFileUsed()
	teardown = func() { viper.SetConfigFile(filename) }
	if filename == "" {
		viper.SetConfigType("yaml")
	}
	viper.SetConfigFile(testConfigFile)
	if err := config.UpsertContext(testContext); err != nil {
		log.Errorf("failed to create test context: %w", err)
	}

	return
}
