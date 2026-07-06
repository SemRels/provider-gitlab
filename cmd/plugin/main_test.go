// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The provider-gitlab Authors

package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunReturnsValidationError(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), &stdout, &stderr, func(key string) (string, bool) {
		values := map[string]string{
			"SEMREL_TAG_NAME": "v1.2.3",
		}
		value, ok := values[key]
		return value, ok
	})

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "SEMREL_PLUGIN_TOKEN, SEMREL_PLUGIN_JOB_TOKEN, or CI_JOB_TOKEN is required") {
		t.Fatalf("expected validation error, got %q", stderr.String())
	}
}

func TestRunDryRun(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), &stdout, &stderr, func(key string) (string, bool) {
		values := map[string]string{
			"SEMREL_PLUGIN_TOKEN":      "token",
			"SEMREL_PLUGIN_PROJECT_ID": "42",
			"SEMREL_TAG_NAME":          "v1.2.3",
			"SEMREL_DRY_RUN":           "true",
		}
		value, ok := values[key]
		return value, ok
	})

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "[dry-run] would create GitLab release v1.2.3") {
		t.Fatalf("expected dry-run message, got %q", stdout.String())
	}
	if stderr.String() != "plugin_schema_version=1\n" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunCreateReleaseSuccess(t *testing.T) {
	t.Parallel()

	requested := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested <- r.URL.Path
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3","name":"v1.2.3"}`))
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), &stdout, &stderr, func(key string) (string, bool) {
		values := map[string]string{
			"SEMREL_PLUGIN_BASE_URL":   srv.URL,
			"SEMREL_PLUGIN_TOKEN":      "token",
			"SEMREL_PLUGIN_PROJECT_ID": "42",
			"SEMREL_TAG_NAME":          "v1.2.3",
			"SEMREL_CHANGELOG":         "notes",
			"SEMREL_BRANCH":            "main",
		}
		value, ok := values[key]
		return value, ok
	})

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", exitCode, stderr.String())
	}
	if gotPath := <-requested; gotPath != "/api/v4/projects/42/releases" {
		t.Fatalf("unexpected request path %q", gotPath)
	}
	if !strings.Contains(stdout.String(), "created GitLab release v1.2.3") {
		t.Fatalf("expected success message, got %q", stdout.String())
	}
	if stderr.String() != "plugin_schema_version=1\n" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunUsesCIProjectIDFallback(t *testing.T) {
	t.Parallel()

	requested := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested <- r.URL.Path
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"tag_name":"v2.0.0","name":"v2.0.0"}`))
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), &stdout, &stderr, func(key string) (string, bool) {
		values := map[string]string{
			"SEMREL_PLUGIN_BASE_URL": srv.URL,
			"SEMREL_PLUGIN_TOKEN":    "token",
			"CI_PROJECT_ID":          "77",
			"SEMREL_TAG_NAME":        "v2.0.0",
		}
		value, ok := values[key]
		return value, ok
	})

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", exitCode, stderr.String())
	}
	if gotPath := <-requested; gotPath != "/api/v4/projects/77/releases" {
		t.Fatalf("unexpected request path %q", gotPath)
	}
}

func TestRunCreateReleaseFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"unauthorized"}`))
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), &stdout, &stderr, func(key string) (string, bool) {
		values := map[string]string{
			"SEMREL_PLUGIN_BASE_URL":   srv.URL,
			"SEMREL_PLUGIN_TOKEN":      "token",
			"SEMREL_PLUGIN_PROJECT_ID": "42",
			"SEMREL_TAG_NAME":          "v1.2.3",
		}
		value, ok := values[key]
		return value, ok
	})

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "status 401") {
		t.Fatalf("expected API error, got %q", stderr.String())
	}
}

func TestHelpers(t *testing.T) {
	t.Parallel()

	lookup := func(key string) (string, bool) {
		values := map[string]string{
			"SEMREL_DRY_RUN": "TrUe",
			"VALUE":          "x",
		}
		value, ok := values[key]
		return value, ok
	}

	if !isDryRun(lookup) {
		t.Fatal("expected dry run to be true")
	}
	if got := valueFromEnv(lookup, "VALUE", "fallback"); got != "x" {
		t.Fatalf("expected value x, got %q", got)
	}
	if got := valueFromEnv(lookup, "MISSING", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
}
