package api

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cisco-open/fsoc/cmd/config"
)

func TestRecursiveWalkThroughStruct(t *testing.T) {
	ctx := config.Context{
		Name:       "some-name",
		AuthMethod: "local",
		LocalAuthOptions: config.LocalAuthOptions{
			AppdTid: "ttt",
		},
	}
	fields := nonZeroStructFields(&ctx)
	assert.ElementsMatch(t, fields, []string{"Name", "AuthMethod", "LocalAuthOptions", "LocalAuthOptions.AppdTid"})
}
