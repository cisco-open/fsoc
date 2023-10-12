package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cisco-open/fsoc/config"
)

func TestPrepareHTTPRequest(t *testing.T) {
	client := &http.Client{}
	cfg := &config.Context{
		URL: "http://localhost:8080",
	}
	req, err := prepareHTTPRequest(cfg, client, "POST", "/test/path/1", nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, "http://localhost:8080/test/path/1", req.URL.String())
}

func TestPrepareJSONRequest(t *testing.T) {
	client := &http.Client{}
	cfg := &config.Context{
		URL: "http://localhost:8080",
	}
	req, err := prepareHTTPRequest(cfg, client, "POST", "/test/path/1", nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, "http://localhost:8080/test/path/1", req.URL.String())
}
