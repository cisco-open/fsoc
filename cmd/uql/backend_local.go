//go:build uql_direct

package uql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/apex/log"
	"github.com/pkg/errors"
)

var localUqlUrl string
var localUqlTenantId string

// replaces the default backend with a version that calls a UQL instance directly (e.g. a locally deployed one) instead
// this is useful when a feature developed for fsoc requires changes to UQL as well
//
// to connect to a locally deployed UQL build the fsoc client with the 'uql_direct' build tag and provide values for localUqlUrl and localUqlTenantId
// e.g.
//
//	go build -tags uql_direct -ldflags="-X 'github.com/cisco-open/fsoc/cmd/uql.localUqlUrl=http://localhost:8042' -X 'github.com/cisco-open/fsoc/cmd/uql.localUqlTenantId=00000000-0000-0000-0000-00000000'"
func init() {
	client.Backend = &localBackend{baseUrl: localUqlUrl, tenantId: localUqlTenantId, client: &http.Client{}}
}

type localBackend struct {
	baseUrl  string
	tenantId string
	client   *http.Client
}

func (b *localBackend) Execute(query *Query, apiVersion ApiVersion) (parsedResponse, error) {
	request, err := http.NewRequest("POST", b.baseUrl+"/monitoring/"+string(apiVersion)+"/query/execute", bytes.NewBufferString(query.Str))
	if err != nil {
		return parsedResponse{}, errors.Wrap(err, "failed to create http request")
	}
	request.Header.Add("content-type", "text/plain")
	request.Header.Add("accept", "application/json")
	request.Header.Add("appd-tid", b.tenantId)
	return b.sendRequest(request)
}

func (b *localBackend) Continue(link *Link) (parsedResponse, error) {
	request, err := http.NewRequest("GET", b.baseUrl+link.Href, nil)
	if err != nil {
		return parsedResponse{}, errors.Wrap(err, "failed to create http request")
	}
	request.Header.Add("accept", "application/json")
	request.Header.Add("appd-tid", b.tenantId)
	return b.sendRequest(request)
}

func (b *localBackend) sendRequest(req *http.Request) (parsedResponse, error) {
	response, err := b.client.Do(req)
	if err != nil {
		return parsedResponse{}, errors.Wrap(err, "http request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		log.Errorf("request failed, status %q", response.Status)
	}

	var responseBody []byte
	responseBody, err = io.ReadAll(response.Body)
	if err != nil {
		return parsedResponse{}, errors.Wrap(err, "failed to read response body")
	}
	if response.StatusCode != 200 {
		return parsedResponse{}, errors.Wrap(err, fmt.Sprintf("error response body: %q", string(responseBody)))
	}
	var chunks []parsedChunk
	err = json.Unmarshal(responseBody, &chunks)
	if err != nil {
		return parsedResponse{}, errors.Wrap(err, "failed to unmarshal response")
	}
	var rawJson = json.RawMessage(responseBody)
	return parsedResponse{
		chunks:  chunks,
		rawJson: &rawJson,
	}, nil
}
