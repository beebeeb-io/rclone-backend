// Beebeeb rclone backend — configuration
// Copyright (C) 2026 Beebeeb
// SPDX-License-Identifier: AGPL-3.0-or-later

package beebeeb

import (
	"fmt"
	"os"
)

const (
	// DefaultAPIURL is the default Beebeeb API base URL used in development.
	DefaultAPIURL = "http://localhost:3001"
)

// Config holds the settings needed to connect to a Beebeeb API instance.
type Config struct {
	// APIURL is the base URL of the Beebeeb server (e.g. "https://api.beebeeb.io").
	APIURL string

	// Token is the Bearer session token used for authentication.
	Token string
}

// ConfigFromEnv builds a Config from environment variables:
//
//	BB_API_URL  — API base URL  (default http://localhost:3001)
//	BB_TOKEN    — session token (required)
func ConfigFromEnv() (*Config, error) {
	apiURL := os.Getenv("BB_API_URL")
	if apiURL == "" {
		apiURL = DefaultAPIURL
	}

	token := os.Getenv("BB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("BB_TOKEN environment variable is required")
	}

	return &Config{
		APIURL: apiURL,
		Token:  token,
	}, nil
}

// ConfigFromMap builds a Config from a key-value map, as rclone passes
// configuration options to backends. Recognised keys:
//
//	api_url — API base URL  (default http://localhost:3001)
//	token   — session token (required)
func ConfigFromMap(m map[string]string) (*Config, error) {
	apiURL := m["api_url"]
	if apiURL == "" {
		apiURL = os.Getenv("BB_API_URL")
	}
	if apiURL == "" {
		apiURL = DefaultAPIURL
	}

	token := m["token"]
	if token == "" {
		token = os.Getenv("BB_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf("beebeeb: token is required (set 'token' in config or BB_TOKEN env)")
	}

	return &Config{
		APIURL: apiURL,
		Token:  token,
	}, nil
}
