// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The provider-gitlab Authors

// Package plugin provides a subprocess SemRel plugin that creates GitLab releases.
package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://gitlab.com"
	defaultTimeout = 30 * time.Second
)

// Config contains the GitLab release settings sourced from the SemRel environment.
type Config struct {
	BaseURL     string
	AuthHeader  string
	Token       string
	ProjectID   string
	TagName     string
	Name        string
	Description string
	Branch      string
	HTTPClient  *http.Client
}

// Release represents the subset of the GitLab release response used by the plugin.
type Release struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Creator creates GitLab releases.
type Creator struct {
	baseURL     string
	authHeader  string
	token       string
	projectID   string
	tagName     string
	name        string
	description string
	branch      string
	httpClient  *http.Client
}

type createReleaseRequest struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Ref         string `json:"ref,omitempty"`
}

// ConfigFromEnv loads plugin configuration from the SemRel subprocess environment.
func ConfigFromEnv() Config {
	return ConfigFromLookupEnv(os.LookupEnv)
}

// ConfigFromLookupEnv loads plugin configuration from the provided environment lookup.
func ConfigFromLookupEnv(lookupEnv func(string) (string, bool)) Config {
	baseURL := strings.TrimSpace(envValue(lookupEnv, "SEMREL_PLUGIN_BASE_URL"))
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	projectID := strings.TrimSpace(envValue(lookupEnv, "SEMREL_PLUGIN_PROJECT_ID"))
	if projectID == "" {
		projectID = strings.TrimSpace(envValue(lookupEnv, "CI_PROJECT_ID"))
	}

	tagName := strings.TrimSpace(envValue(lookupEnv, "SEMREL_TAG_NAME"))
	authHeaderName, authHeaderValue := authHeader(lookupEnv)

	return Config{
		BaseURL:     baseURL,
		AuthHeader:  authHeaderName,
		Token:       authHeaderValue,
		ProjectID:   projectID,
		TagName:     tagName,
		Name:        tagName,
		Description: envValue(lookupEnv, "SEMREL_CHANGELOG"),
		Branch:      strings.TrimSpace(envValue(lookupEnv, "SEMREL_BRANCH")),
	}
}

// Validate reports missing or malformed configuration.
func (c Config) Validate() error {
	if strings.TrimSpace(c.Token) == "" {
		return errors.New("SEMREL_PLUGIN_TOKEN, SEMREL_PLUGIN_JOB_TOKEN, or CI_JOB_TOKEN is required")
	}
	if strings.TrimSpace(c.ProjectID) == "" {
		return errors.New("SEMREL_PLUGIN_PROJECT_ID or CI_PROJECT_ID is required")
	}
	if strings.TrimSpace(c.TagName) == "" {
		return errors.New("SEMREL_TAG_NAME is required")
	}
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("release name is required")
	}

	parsedURL, err := url.Parse(strings.TrimSpace(c.BaseURL))
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("SEMREL_PLUGIN_BASE_URL must be a valid absolute URL")
	}

	return nil
}

// New returns a GitLab release creator.
func New(cfg Config) *Creator {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	authHeaderName := strings.TrimSpace(cfg.AuthHeader)
	if authHeaderName == "" && strings.TrimSpace(cfg.Token) != "" {
		authHeaderName = "PRIVATE-TOKEN"
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	return &Creator{
		baseURL:     baseURL,
		authHeader:  authHeaderName,
		token:       strings.TrimSpace(cfg.Token),
		projectID:   url.PathEscape(strings.TrimSpace(cfg.ProjectID)),
		tagName:     strings.TrimSpace(cfg.TagName),
		name:        strings.TrimSpace(cfg.Name),
		description: cfg.Description,
		branch:      strings.TrimSpace(cfg.Branch),
		httpClient:  httpClient,
	}
}

// CreateRelease creates a GitLab release from the current SemRel context.
func (c *Creator) CreateRelease(ctx context.Context) (*Release, error) {
	payload := createReleaseRequest{
		TagName:     c.tagName,
		Name:        c.name,
		Description: c.description,
		Ref:         c.branch,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("gitlab: marshal release request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/v4/projects/%s/releases", c.baseURL, c.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gitlab: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(c.authHeader, c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitlab: create release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("gitlab: create release: status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("gitlab: decode release response: %w", err)
	}

	return &release, nil
}

func envValue(lookupEnv func(string) (string, bool), key string) string {
	value, ok := lookupEnv(key)
	if !ok {
		return ""
	}
	return value
}

func authHeader(lookupEnv func(string) (string, bool)) (name, value string) {
	if token := strings.TrimSpace(envValue(lookupEnv, "SEMREL_PLUGIN_TOKEN")); token != "" {
		return "PRIVATE-TOKEN", token
	}
	if token := strings.TrimSpace(envValue(lookupEnv, "SEMREL_PLUGIN_JOB_TOKEN")); token != "" {
		return "JOB-TOKEN", token
	}
	if envBool(lookupEnv, "SEMREL_PLUGIN_USE_JOB_TOKEN") {
		if token := strings.TrimSpace(envValue(lookupEnv, "CI_JOB_TOKEN")); token != "" {
			return "JOB-TOKEN", token
		}
		return "", ""
	}
	if token := strings.TrimSpace(envValue(lookupEnv, "CI_JOB_TOKEN")); token != "" {
		return "JOB-TOKEN", token
	}
	return "", ""
}

func envBool(lookupEnv func(string) (string, bool), key string) bool {
	return strings.EqualFold(strings.TrimSpace(envValue(lookupEnv, key)), "true")
}
