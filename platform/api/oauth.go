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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/apex/log"
	"github.com/pkg/browser"
	"go.pinniped.dev/pkg/oidcclient/pkce"
	"golang.org/x/oauth2"

	"github.com/cisco-open/fsoc/config"
)

const (
	oauth2ClientId       = "default"
	oauth2AuthUriSuffix  = "oauth2/authorize" // API for obtaining authorization codes
	oauth2TokenUriSuffix = "oauth2/token"     // API for exchanging the auth code for a token
	oauthRedirectUri     = "http://127.0.0.1:3101/callback"
)

// appTokens is what the AppD backend returns when it hands back the tokens (in exchange for the authorization code)
type appTokens struct {
	AccessToken  string `json:"access_token"` // aka JWT token to make requests
	ExpiresIn    int    `json:"expires_in"`
	IdToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"` // this is what we use to get a fresh JWT token
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"` // e.g., bearer
}

// authCodes is the authorization that the oauth2 method provides
type authCodes struct {
	Code  string
	Scope string
	State string
}

// oauthErrorPayload is the structure returned by the auth/token endpoints on 4xx errors
type oauthErrorPayload struct {
	Error      string `json:"error"` // short error id, e.g., `invalid_client`
	ErrorDesc  string `json:"error_description"`
	ErrorHing  string `json:"error_hint"`
	StatusCode int    `json:"status_code"`
}

// tenantPayload is the structure returned by the tenant ID resolver endpoint
type tenantPayload struct {
	TenantId string `json:"tenantId"`
}

// refreshTokenStaleError is a custom error to indicate that the refresh token
// is rejected, differentiating it from any other error getting the refresh token
type refreshTokenStaleError struct{}

func (s refreshTokenStaleError) Error() string {
	return "Refresh token rejected (likely stale)"
}

// oauthLogin performs a login into the platform API and updates the token(s) in the provided context
func oauthLogin(ctx *callContext) error {
	log.Infof("Starting OAuth authentication flow")

	// get tenant if one is not provided, update into ctx
	if ctx.cfg.Tenant == "" {
		tenantId, err := resolveTenant(ctx)
		if err != nil {
			return fmt.Errorf("could not resolve tenant ID for %q: %v", ctx.cfg.URL, err.Error())
		}
		ctx.cfg.Tenant = tenantId
		log.Infof("Successfully resolved tenant ID to %v", ctx.cfg.Tenant)
		// tenant is now updated in ctx, will be saved if we successfully log in
	}

	// try refresh token if present
	if ctx.cfg.RefreshToken != "" {
		// refresh and return if successful
		err := oauthRefreshToken(ctx)
		if err == nil {
			log.Infof("Access token refreshed successfully")
			return nil
		}
		rtse, ok := err.(refreshTokenStaleError)
		if ok {
			log.Infof("%v; going for a new login", rtse) // display token refresh error
		} else {
			log.Infof("Refresh token rejected: %v; going for a new login", err)
		}
	}

	// generate PKCE codes
	code, err := pkce.Generate()
	if err != nil {
		return err // should never really fail
	}

	// generate a nonce to match the callback uniquely to our request (aka "state")
	stateCode, err := pkce.Generate()
	if err != nil {
		return err // should never really fail
	}
	state := string(stateCode)

	// prepare OAuth2 config
	conf := &oauth2.Config{
		ClientID:    oauth2ClientId,
		RedirectURL: oauthRedirectUri,
		Endpoint: oauth2.Endpoint{
			AuthURL:   oauthUriWithSuffix(ctx.cfg, oauth2AuthUriSuffix),
			TokenURL:  oauthUriWithSuffix(ctx.cfg, oauth2TokenUriSuffix),
			AuthStyle: oauth2.AuthStyleInParams,
		},
		Scopes: []string{"openid", "introspect_tokens", "offline_access"},
	}
	url := conf.AuthCodeURL(state,
		code.Method(),
		code.Challenge(),
	)

	// open browser to perform login, collect auth with a localhost http server
	authCode, err := getAuthorizationCodes(ctx, url)
	if err != nil {
		return fmt.Errorf("login failed to obtain the authorization code: %v", err)
	}

	// verify nonce, must match
	if state != authCode.State {
		return fmt.Errorf("login failed: received auth state doesn't match (a session replay or similar attack is likely in progress; please log out of all sessions!)")
	}

	// TODO: make the exchange work with the auth2 package (fails, likely due to us needing urlencoded data)
	// token, err := conf.Exchange(context.Background(), string(code),
	// 	code.Verifier(),
	// )
	// if err != nil {
	// 	return nil, fmt.Errorf("login failed: failed to obtain token: %v", err.Error())
	// }

	// exchange auth code for token (using a hand-crafted exchange request)
	token, err := exchangeCodeForToken(ctx, conf, code, authCode)
	if err != nil {
		return fmt.Errorf("failed to exchange auth code for a token: %v", err.Error())
	}

	userID, err := extractUser(token.AccessToken)
	if err != nil {
		log.Warnf("Could not extract user identity from the bearer token: %v. Continuing without user ID", err)
		userID = ""
		// fall through and continue without a user ID
	} else {
		log.WithFields(log.Fields{"userId": userID}).Info("Extracted user ID")
	}

	// update profile
	ctx.cfg.Token = token.AccessToken
	ctx.cfg.RefreshToken = token.RefreshToken
	if userID != "" {
		ctx.cfg.User = userID
	}

	return nil
}

func getAuthorizationCodes(ctx *callContext, url string) (*authCodes, error) {
	// start http server to receive the auth callback
	callbackServer, respChan, err := startCallbackServer()
	if err != nil {
		return nil, fmt.Errorf("could not start a local http server for auth: %v", err.Error())
	}
	defer func() {
		_ = stopCallbackServer(callbackServer) // no check needed, error should be logged
	}()

	// open browser
	log.Infof("Starting a browser to perform authentication")
	//fmt.Printf("If a browser window does not open shortly, please visit the following URL to login\n%v\n", url)
	if err = openBrowser(url); err != nil {
		log.Errorf("Failed to automatically launch browser auth window: %v", err)
		log.Errorf("Please visit the following URL to login\n%v\n", url)
		// fall through
	}

	// wait for authorization codes (TODO: add timeout, e.g., a few minutes)
	ctx.startSpinner("OAuth interactive authentication")
	authCode := <-respChan // nb: blocks until a callback is received on localhost with the correct path
	ctx.stopSpinner(true)  // TODO: figure out whether this can indicate fail/in what condition
	log.Infof("PKCE authorization codes received")

	return &authCode, nil
}

func exchangeCodeForToken(ctx *callContext, conf *oauth2.Config, pkce pkce.Code, auth *authCodes) (*appTokens, error) {
	log.Infof("Exchanging authorization codes for access token")

	// create http client for the request
	client := &http.Client{}

	// prepare urlencoded data body
	values := url.Values{}
	values.Add("grant_type", "authorization_code")
	values.Add("client_id", "default")
	values.Add("code_verifier", extractPrivateField(pkce.Verifier(), "v"))
	values.Add("code", auth.Code)
	values.Add("redirect_uri", oauthRedirectUri)
	bodyReader := bytes.NewReader([]byte(values.Encode()))

	// create a POST HTTP request
	req, err := http.NewRequest("POST", conf.Endpoint.TokenURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create a request %q: %v", conf.Endpoint.TokenURL, err.Error())
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// execute request
	ctx.startSpinner("OAuth auth codes exchange for token")
	resp, err := client.Do(req)
	ctx.stopSpinner(err == nil && resp.StatusCode/100 == 2)
	if err != nil {
		return nil, fmt.Errorf("POST request to %q failed: %v", req.RequestURI, err.Error())
	}

	// log error if it occurred
	if resp.StatusCode/100 != 2 {
		// log error before trying to parse body, more processing later
		log.Errorf("Request failed, status %q; more info to follow", resp.Status)
		// fall through
	}

	// collect response body (whether success or error)
	var respBytes []byte
	defer resp.Body.Close()
	respBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response to POST to %q: %v", req.RequestURI, err.Error())
	}

	// parse response body in case of error (special parsing logic, tolerate non-JSON responses)
	if resp.StatusCode/100 != 2 {
		var errobj oauthErrorPayload

		// try to unmarshal JSON
		err := json.Unmarshal(respBytes, &errobj)
		if err != nil {
			// process as a string instead, ignore parsing error
			return nil, fmt.Errorf("error response: `%v`", bytes.NewBuffer(respBytes).String())
		}
		return nil, fmt.Errorf("error response: %+v", errobj)
	}

	// parse tokens
	var tokenObject appTokens
	if err := json.Unmarshal(respBytes, &tokenObject); err != nil {
		return nil, fmt.Errorf("failed to JSON parse the response as a token object: %v", err.Error())
	}

	return &tokenObject, nil
}

// oauthRefreshToken tries to use the refresh token to get a new access token. If successful,
// it updates the access token in the context. It should be called only with a valid config
// that has a refresh token. Note that the refresh token also changes, so it will be updated
// as well.
func oauthRefreshToken(ctx *callContext) error {
	log.Infof("Trying to get a new access token using the refresh token")

	// create http client for the request
	client := &http.Client{}

	// prepare urlencoded data body
	values := url.Values{}
	values.Add("client_id", oauth2ClientId)
	values.Add("redirect_uri", oauthRedirectUri)
	values.Add("grant_type", "refresh_token")
	values.Add("refresh_token", ctx.cfg.RefreshToken)
	bodyReader := bytes.NewReader([]byte(values.Encode()))

	// create a POST HTTP request
	tokenUri := oauthUriWithSuffix(ctx.cfg, oauth2TokenUriSuffix)
	req, err := http.NewRequest("POST", tokenUri, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create a token refresh request %q: %v", tokenUri, err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	// execute request
	ctx.startSpinner("OAuth token refresh")
	resp, err := client.Do(req)
	ctx.stopSpinner(err == nil && resp.StatusCode/100 == 2)
	if err != nil {
		return fmt.Errorf("POST request to %q failed: %v", req.RequestURI, err.Error())
	}

	// log error if it occurred
	if resp.StatusCode/100 != 2 {
		// log error before trying to parse body, more processing later
		log.Errorf("Request failed, status %q; more info to follow", resp.Status)
		// fall through
	}

	// collect response body (whether success or error)
	var respBytes []byte
	defer resp.Body.Close()
	respBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed reading response to POST to %q: %v", req.RequestURI, err.Error())
	}

	// parse response body in case of error (special parsing logic, tolerate non-JSON responses)
	if resp.StatusCode/100 != 2 {
		return parseIntoError(resp, respBytes)
	}

	// parse tokens
	var tokenObject appTokens
	if err := json.Unmarshal(respBytes, &tokenObject); err != nil {
		return fmt.Errorf("failed to JSON parse the response as a token object: %w", err)
	}

	// update tokens in context
	ctx.cfg.Token = tokenObject.AccessToken
	ctx.cfg.RefreshToken = tokenObject.RefreshToken

	return nil
}

func oauthUriWithSuffix(ctx *config.Context, suffix string) string {
	uri, err := url.JoinPath(ctx.URL, "auth", ctx.Tenant, oauth2ClientId, suffix)
	if err != nil {
		log.Fatalf("unexpected failure constructing oauth2 endpoint URI: %v; terminating (likely a bug)", err)
	}
	return uri
	// return strings.Join([]string{ctx.URL, "auth", ctx.Tenant, oauth2ClientId, suffix}, "/")
}

func startCallbackServer() (*http.Server, chan authCodes, error) {
	// construct a channel for the response
	respChan := make(chan authCodes)

	// start server at oauthRedirectUri
	urlStruct, err := url.Parse(oauthRedirectUri)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse callback URL %q: %v", oauthRedirectUri, err)
	}
	server := &http.Server{
		Addr: urlStruct.Host, // host:port
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callbackHandler(respChan, w, r)
		}),
		//ErrorLog: log, // TODO: set apex/log as a logger for http
	}
	go func() {
		log.Infof("Starting the auth http server on %v", server.Addr)
		err := server.ListenAndServe()
		if err != nil && err.Error() != "http: Server closed" { // TODO: find a better way to ignore the close error specifically
			log.Errorf("Failed to start auth http server on %v: %v", server.Addr, err)
		}
	}()
	return server, respChan, nil
}

func stopCallbackServer(server *http.Server) error {
	if err := server.Close(); err != nil {
		err = fmt.Errorf("error stopping the auth http server on %v: %v", server.Addr, err)
		log.Errorf("%v", err)
		return err
	}

	log.Infof("Stopped the auth http server on %v", server.Addr)
	return nil
}

func callbackHandler(respChan chan authCodes, w http.ResponseWriter, r *http.Request) {
	// compute expected response path
	respUri, err := url.Parse(oauthRedirectUri)
	if err != nil {
		log.Errorf("Unexpected failure to obtain expected callback path (likely a bug): %v", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	callbackPath := respUri.Path

	//log.Infof("Request %+v", r)
	uri, err := url.Parse(r.RequestURI)
	if err != nil {
		log.Errorf("Unexpected failure to parse callback path received (malformed request?): %v", err.Error())
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	// reject all requests except the callback
	if uri.Path != callbackPath {
		log.Infof("Failing unexpected request for %q", uri.Path)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	// extract & send response codes (minimal validation, leave this to main)
	values := uri.Query()
	// for k, v := range values {
	// 	log.Infof("query %q\t%q", k, v)
	// }
	codes := authCodes{
		Code:  safeExtractFirstValue(values, "code"),
		Scope: safeExtractFirstValue(values, "scope"),
		State: safeExtractFirstValue(values, "state"),
	}

	// provide a stub page to be displayed in the browser after login
	fmt.Fprint(w, "Login successful. You can close this browser window.")

	//log.Infof("response codes %+v", codes)
	respChan <- codes
}

func safeExtractFirstValue(queryValues url.Values, field string) string {
	// note: url.Values is simply a map[string][]string

	// extract values by field name
	qv := queryValues[field]
	if qv == nil {
		log.Errorf("expected a value for auth response %q, received none", field)
		return ""
	}

	// extract value
	l := len(qv)
	if l < 1 {
		log.Errorf("expected a value for auth response %q, received none", field)
		return ""
	}
	if l > 1 {
		// log name and count but not values (values are likely secret)
		log.Warnf("expected a single value for auth response %q, received %v", field, l)
		// fall through, get just the first value
	}
	return qv[0]
}

func extractPrivateField(theStruct oauth2.AuthCodeOption, field string) string {
	sValue := reflect.ValueOf(theStruct)
	fValue := sValue.FieldByName(field)
	if fValue.Kind() != reflect.String {
		return "" // otherwise reflect.String() may convert it to <T value>
	}
	return fValue.String()
}

// openBrowser opens a browser window at the provided url. It also captures stdout message displayed
// by the command (if any: xdg-open in Linux says things like "Opening in existing browser session.") so
// that our stdout is not polluted (as it may be being captured for yaml/json parsing)
func openBrowser(url string) error {
	// redirect browser's package stdout to a pipe, saving the original stdout
	orig := browser.Stdout
	r, w, _ := os.Pipe()
	browser.Stdout = w
	defer func() {
		browser.Stdout = orig
	}()

	// start browser
	browserErr := browser.OpenURL(url) // check error later
	w.Close()                          // no more writing

	// copy the output in a separate goroutine so printing can't block indefinitely
	outChan := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r)
		if err != nil {
			log.Warnf("Error capturing browser launch output: %v; ignoring it", err)
			// fall through
		}
		outChan <- buf.String()
	}()

	// collect and log any message displayed
	outMsg := strings.TrimSpace(<-outChan)
	if outMsg != "" {
		log.Infof("Browser launch: %v", outMsg)
	}

	return browserErr
}
