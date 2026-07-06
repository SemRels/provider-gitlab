// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The provider-gitlab Authors

package plugin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestConfigFromLookupEnvUsesDefaultsAndFallbacks(t *testing.T) {
	t.Parallel()

	cfg := ConfigFromLookupEnv(func(key string) (string, bool) {
		values := map[string]string{
			"SEMREL_PLUGIN_TOKEN": " secret-token ",
			"CI_PROJECT_ID":       "12345",
			"SEMREL_TAG_NAME":     "v1.2.3",
			"SEMREL_CHANGELOG":    "## Notes\n- shipped",
			"SEMREL_BRANCH":       "main",
		}
		value, ok := values[key]
		return value, ok
	})

	if cfg.BaseURL != defaultBaseURL {
		t.Fatalf("expected default base URL %q, got %q", defaultBaseURL, cfg.BaseURL)
	}
	if cfg.AuthHeader != "PRIVATE-TOKEN" {
		t.Fatalf("expected PRIVATE-TOKEN auth header, got %q", cfg.AuthHeader)
	}
	if cfg.Token != "secret-token" {
		t.Fatalf("expected trimmed token, got %q", cfg.Token)
	}
	if cfg.ProjectID != "12345" {
		t.Fatalf("expected CI project fallback, got %q", cfg.ProjectID)
	}
	if cfg.TagName != "v1.2.3" || cfg.Name != "v1.2.3" {
		t.Fatalf("expected tag-based release naming, got tag=%q name=%q", cfg.TagName, cfg.Name)
	}
	if cfg.Description != "## Notes\n- shipped" {
		t.Fatalf("expected changelog description, got %q", cfg.Description)
	}
	if cfg.Branch != "main" {
		t.Fatalf("expected branch main, got %q", cfg.Branch)
	}
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "missing token",
			cfg:  Config{BaseURL: defaultBaseURL, ProjectID: "42", TagName: "v1.2.3", Name: "v1.2.3"},
			want: "SEMREL_PLUGIN_TOKEN, SEMREL_PLUGIN_JOB_TOKEN, or CI_JOB_TOKEN is required",
		},
		{
			name: "missing project",
			cfg:  Config{BaseURL: defaultBaseURL, Token: "token", TagName: "v1.2.3", Name: "v1.2.3"},
			want: "SEMREL_PLUGIN_PROJECT_ID or CI_PROJECT_ID is required",
		},
		{
			name: "missing tag",
			cfg:  Config{BaseURL: defaultBaseURL, Token: "token", ProjectID: "42"},
			want: "SEMREL_TAG_NAME is required",
		},
		{
			name: "missing name",
			cfg:  Config{BaseURL: defaultBaseURL, Token: "token", ProjectID: "42", TagName: "v1.2.3"},
			want: "release name is required",
		},
		{
			name: "invalid base url",
			cfg:  Config{BaseURL: "://bad", Token: "token", ProjectID: "42", TagName: "v1.2.3", Name: "v1.2.3"},
			want: "SEMREL_PLUGIN_BASE_URL must be a valid absolute URL",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.cfg.Validate()
			if err == nil || err.Error() != tc.want {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestNewUsesDefaults(t *testing.T) {
	t.Parallel()

	creator := New(Config{Token: "token", ProjectID: "group/project", TagName: "v1.2.3", Name: "v1.2.3"})

	if creator.baseURL != defaultBaseURL {
		t.Fatalf("expected default base URL, got %q", creator.baseURL)
	}
	if creator.authHeader != "PRIVATE-TOKEN" {
		t.Fatalf("expected default PRIVATE-TOKEN auth header, got %q", creator.authHeader)
	}
	if creator.projectID != "group%2Fproject" {
		t.Fatalf("expected escaped project ID, got %q", creator.projectID)
	}
	if creator.httpClient == nil || creator.httpClient.Timeout != defaultTimeout {
		t.Fatalf("expected default timeout %s, got %#v", defaultTimeout, creator.httpClient)
	}
}

func TestCreateReleaseSuccess(t *testing.T) {
	t.Parallel()

	type releaseRequest struct {
		TagName     string `json:"tag_name"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Ref         string `json:"ref,omitempty"`
	}

	requests := make(chan releaseRequest, 1)
	headerNames := make(chan string, 1)
	headerValues := make(chan string, 1)
	paths := make(chan string, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var req releaseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		requests <- req
		if value := r.Header.Get("PRIVATE-TOKEN"); value != "" {
			headerNames <- "PRIVATE-TOKEN"
			headerValues <- value
		} else {
			headerNames <- "JOB-TOKEN"
			headerValues <- r.Header.Get("JOB-TOKEN")
		}
		paths <- r.URL.EscapedPath()

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Release{TagName: req.TagName, Name: req.Name, Description: req.Description})
	}))
	defer srv.Close()

	creator := New(Config{
		BaseURL:     srv.URL + "/",
		AuthHeader:  "PRIVATE-TOKEN",
		Token:       "test-token",
		ProjectID:   "group/project",
		TagName:     "v1.2.3",
		Name:        "v1.2.3",
		Description: "## Changelog\n- added",
		Branch:      "release/main",
		HTTPClient:  srv.Client(),
	})

	release, err := creator.CreateRelease(context.Background())
	if err != nil {
		t.Fatalf("CreateRelease returned error: %v", err)
	}

	gotRequest := <-requests
	if gotRequest.TagName != "v1.2.3" || gotRequest.Name != "v1.2.3" || gotRequest.Description != "## Changelog\n- added" || gotRequest.Ref != "release/main" {
		t.Fatalf("unexpected request payload: %+v", gotRequest)
	}
	if gotHeaderName := <-headerNames; gotHeaderName != "PRIVATE-TOKEN" {
		t.Fatalf("expected PRIVATE-TOKEN header name, got %q", gotHeaderName)
	}
	if gotHeaderValue := <-headerValues; gotHeaderValue != "test-token" {
		t.Fatalf("expected PRIVATE-TOKEN header value, got %q", gotHeaderValue)
	}
	if gotPath := <-paths; gotPath != "/api/v4/projects/group%2Fproject/releases" {
		t.Fatalf("unexpected request path %q", gotPath)
	}
	if release.TagName != "v1.2.3" {
		t.Fatalf("expected release tag v1.2.3, got %q", release.TagName)
	}
}

func TestCreateReleaseOmitsRefWhenBranchUnset(t *testing.T) {
	t.Parallel()

	bodyContainsRef := make(chan bool, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		bodyContainsRef <- strings.Contains(string(payload), "\"ref\"")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Release{TagName: "v1.2.3", Name: "v1.2.3"})
	}))
	defer srv.Close()

	creator := New(Config{
		BaseURL:    srv.URL,
		AuthHeader: "PRIVATE-TOKEN",
		Token:      "test-token",
		ProjectID:  "42",
		TagName:    "v1.2.3",
		Name:       "v1.2.3",
		HTTPClient: srv.Client(),
	})

	if _, err := creator.CreateRelease(context.Background()); err != nil {
		t.Fatalf("CreateRelease returned error: %v", err)
	}
	if <-bodyContainsRef {
		t.Fatal("expected ref to be omitted when branch is unset")
	}
}

func TestCreateReleaseReturnsStatusError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"bad request"}`))
	}))
	defer srv.Close()

	creator := New(Config{
		BaseURL:    srv.URL,
		AuthHeader: "PRIVATE-TOKEN",
		Token:      "test-token",
		ProjectID:  "42",
		TagName:    "v1.2.3",
		Name:       "v1.2.3",
		HTTPClient: srv.Client(),
	})

	_, err := creator.CreateRelease(context.Background())
	if err == nil || !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestCreateReleaseHonorsContextCancellation(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Release{TagName: "v1.2.3", Name: "v1.2.3"})
	}))
	defer srv.Close()

	creator := New(Config{
		BaseURL:    srv.URL,
		AuthHeader: "PRIVATE-TOKEN",
		Token:      "test-token",
		ProjectID:  "42",
		TagName:    "v1.2.3",
		Name:       "v1.2.3",
		HTTPClient: srv.Client(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := creator.CreateRelease(ctx)
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}

func TestAuthHeaderPrecedence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		env      map[string]string
		wantName string
		wantVal  string
	}{
		{
			name: "private token wins",
			env: map[string]string{
				"SEMREL_PLUGIN_TOKEN":         "private-token",
				"SEMREL_PLUGIN_JOB_TOKEN":     "job-token",
				"SEMREL_PLUGIN_USE_JOB_TOKEN": "true",
				"CI_JOB_TOKEN":                "ci-job-token",
			},
			wantName: "PRIVATE-TOKEN",
			wantVal:  "private-token",
		},
		{
			name: "explicit job token works",
			env: map[string]string{
				"SEMREL_PLUGIN_JOB_TOKEN":     "job-token",
				"SEMREL_PLUGIN_USE_JOB_TOKEN": "true",
			},
			wantName: "JOB-TOKEN",
			wantVal:  "job-token",
		},
		{
			name: "ci job token fallback works",
			env: map[string]string{
				"CI_JOB_TOKEN": "ci-job-token",
			},
			wantName: "JOB-TOKEN",
			wantVal:  "ci-job-token",
		},
		{
			name: "job token opt in uses ci job token",
			env: map[string]string{
				"SEMREL_PLUGIN_USE_JOB_TOKEN": "true",
				"CI_JOB_TOKEN":                "ci-job-token",
			},
			wantName: "JOB-TOKEN",
			wantVal:  "ci-job-token",
		},
		{
			name: "missing tokens",
			env:  map[string]string{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotName, gotVal := authHeader(func(key string) (string, bool) {
				value, ok := tc.env[key]
				return value, ok
			})

			if gotName != tc.wantName || gotVal != tc.wantVal {
				t.Fatalf("expected (%q, %q), got (%q, %q)", tc.wantName, tc.wantVal, gotName, gotVal)
			}
		})
	}
}

func TestCreateReleaseUsesJobTokenHeader(t *testing.T) {
	t.Parallel()

	headerNames := make(chan string, 1)
	headerValues := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if value := r.Header.Get("PRIVATE-TOKEN"); value != "" {
			headerNames <- "PRIVATE-TOKEN"
			headerValues <- value
		} else {
			headerNames <- "JOB-TOKEN"
			headerValues <- r.Header.Get("JOB-TOKEN")
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Release{TagName: "v1.2.3", Name: "v1.2.3"})
	}))
	defer srv.Close()

	creator := New(Config{
		BaseURL:    srv.URL,
		AuthHeader: "JOB-TOKEN",
		Token:      "job-token",
		ProjectID:  "42",
		TagName:    "v1.2.3",
		Name:       "v1.2.3",
		HTTPClient: srv.Client(),
	})

	if _, err := creator.CreateRelease(context.Background()); err != nil {
		t.Fatalf("CreateRelease returned error: %v", err)
	}
	if got := <-headerNames; got != "JOB-TOKEN" {
		t.Fatalf("expected JOB-TOKEN header name, got %q", got)
	}
	if got := <-headerValues; got != "job-token" {
		t.Fatalf("expected JOB-TOKEN header value, got %q", got)
	}
}
