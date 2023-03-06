package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateUrlWithInvalidURL(t *testing.T) {
	url, err := validateUrl("/web")
	assert.Equal(t, "", url)
	assert.NotNil(t, err)
	assert.Regexp(t, "no host is provided in the url.*", err)

	url, err = validateUrl("h://web")
	assert.Equal(t, "", url)
	assert.NotNil(t, err)
	assert.Regexp(t, "invalid schema.*", err)
}

func TestValidateUrlWithDataNeedClean(t *testing.T) {
	url, err := validateUrl("mytenant.saas.observe.com")
	assert.Nil(t, err)
	assert.Equal(t, url, "https://mytenant.saas.observe.com")

	url, err = validateUrl("mytenant.saas.observe.com/index/a?b=1")
	assert.Nil(t, err)
	assert.Equal(t, url, "https://mytenant.saas.observe.com/index/a?b=1")
}

func TestValidateUrlWithValidData(t *testing.T) {
	url, err := validateUrl("http://mytenant.saas.observe.com")
	assert.Nil(t, err)
	assert.Equal(t, url, "http://mytenant.saas.observe.com")
}
