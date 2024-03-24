// Copyright 2024 Cisco Systems, Inc.
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
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/apex/log"
)

var gblShutdown bool // global flag to indicate server shutdown is pending

func RunProxyServer(port int, command string, statusPrinter func(string), exitCode *int) error {
	if statusPrinter == nil {
		statusPrinter = func(s string) {
			log.Info(s)
		}
	}

	// Create a new call context
	callCtx := newCallContext()
	cfg := callCtx.cfg // quick access to config

	// force login if no token
	if cfg.Token == "" {
		log.Info("No auth token available, trying to log in")
		if err := login(callCtx); err != nil {
			return err
		}
		cfg = callCtx.cfg // may have changed across login
	}

	// The remote URL to which the proxy server will forward requests
	listenAddr := fmt.Sprintf("localhost:%d", port)

	// Set up the reverse proxy handler
	url, err := url.Parse(cfg.URL)
	if err != nil {
		log.Fatalf("Invalid URL %q in profile %q: %v", cfg.URL, cfg.Name, err)
	}
	proxy := &httputil.ReverseProxy{
		Transport: newApiRetriableTransport(callCtx, statusPrinter),
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(url)
		},
	}

	// Set up the server with timeout configurations (timeouts are infinite by default)
	server := &http.Server{
		Addr:         listenAddr,
		Handler:      proxy,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	// Start the server in a goroutine
	go func() {
		statusPrinter(fmt.Sprintf("Running proxy server %v -> %v", listenAddr, cfg.URL))
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server failed to ListenAndServe: %v", err)
		}
	}()

	// If a command is provided, execute it and then shut down the server
	if command != "" {
		statusPrinter(fmt.Sprintf("Executing command: %v", command))
		runExitCode, err := runCommand(command)
		if err != nil {
			return err
		}
		gblShutdown = true
		if err := server.Shutdown(context.Background()); err != nil {
			log.Errorf("Error shutting down proxy server: %v\n", err)
		}

		// pass exit code back to caller
		if exitCode != nil {
			*exitCode = runExitCode
		} else if runExitCode != 0 {
			log.Warnf("Non-zero exit code (%v) ignored", runExitCode)
		}
		return nil
	}

	// Otherwise, run server in foreground until terminated

	// Set up a channel to listen for the termination signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Listen for signals and gracefully shut down the server when received
	go func() {
		sig := <-signalChan
		if sig != os.Interrupt && sig != syscall.SIGTERM {
			return // ignore all other OS signals
		}
		if gblShutdown {
			return // already shutting down
		}
		gblShutdown = true
		statusPrinter(fmt.Sprintf("Received signal %v. Shutting down...\n", sig))
		if err := server.Shutdown(context.Background()); err != nil {
			log.Errorf("Error shutting down proxy server: %v\n", err)
		}
	}()

	// Wait for server shutdown
	<-signalChan
	statusPrinter("Proxy server has successfully shut down")

	return nil
}

func runCommand(command string) (int, error) {
	// Separate the command and its arguments
	args := strings.Fields(command)
	if len(args) == 0 {
		return 1, fmt.Errorf("command cannot be empty")
	}

	// Execute the command and wait for it to complete
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

type apiRetriableTransport struct {
	transport     http.RoundTripper
	statusPrinter func(string)
	callContext   *callContext // contains the auth token
}

func newApiRetriableTransport(callContext *callContext, statusPrinter func(string)) *apiRetriableTransport {
	return &apiRetriableTransport{
		transport: &http.Transport{
			ResponseHeaderTimeout: 30 * time.Second,
			// the rest of the timeout fields have satisfactory default values
		},
		callContext:   callContext,
		statusPrinter: statusPrinter,
	}
}

func (t *apiRetriableTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// add auth header
	req.Header.Add("Authorization", "Bearer "+t.callContext.cfg.Token)

	// make the request and return, unless it's a 403 (likely expired token)
	t.statusPrinter(fmt.Sprintf("Proxying request %q", req.URL))
	resp, err := t.transport.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusForbidden {
		return resp, err // return the response as is
	}

	// refresh the API token
	log.Warn("Current token is no longer valid; trying to refresh")
	if err := login(t.callContext); err != nil {
		log.Errorf("Login failed: %w", err)
		return resp, nil // return the original 403 response
	}
	// note: callContext.cfg.Token has been updated by login()

	// retry the request with the new token
	t.statusPrinter(fmt.Sprintf("Retrying request %q with refreshed token", req.URL))
	req.Header.Del("Authorization")
	req.Header.Add("Authorization", "Bearer "+t.callContext.cfg.Token)
	return t.transport.RoundTrip(req)
}
