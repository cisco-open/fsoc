package api

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cisco-open/fsoc/cmd/config"
)

func TestComputeResolverEndpointForLocalSetup(t *testing.T) {
	cfg := &config.Context{
		URL:    "http://localhost:8080",
		Tenant: "123-123",
	}
	endpoint, err := computeResolverEndpoint(cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "", endpoint)
}

func TestComputeResolverEndpointForProduction(t *testing.T) {
	cfg := &config.Context{
		URL:    "https://MYTENANT.saas.appd-test.com",
		Tenant: "123-123",
	}
	endpoint, err := computeResolverEndpoint(cfg)
	assert.Nil(t, err)
	assert.Equal(t, "https://observe-tenant-lookup-api.saas.appd-test.com/tenants/lookup/MYTENANT.saas.appd-test.com", endpoint)
}
