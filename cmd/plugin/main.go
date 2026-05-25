// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The provider-gitlab Authors

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	plugin "github.com/SemRels/provider-gitlab/internal/plugin"
)

const commandTimeout = 30 * time.Second

func main() {
	os.Exit(run(context.Background(), os.Stdout, os.Stderr, os.LookupEnv))
}

func run(parent context.Context, stdout, stderr io.Writer, lookupEnv func(string) (string, bool)) int {
	cfg := plugin.ConfigFromLookupEnv(lookupEnv)
	if err := cfg.Validate(); err != nil {
		_, _ = fmt.Fprintf(stderr, "provider-gitlab: %v\n", err)
		return 1
	}

	if isDryRun(lookupEnv) {
		_, _ = fmt.Fprintf(stdout, "[dry-run] would create GitLab release %s for project %s\n", cfg.TagName, cfg.ProjectID)
		return 0
	}

	ctx, cancel := context.WithTimeout(parent, commandTimeout)
	defer cancel()

	release, err := plugin.New(cfg).CreateRelease(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "provider-gitlab: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "created GitLab release %s\n", release.TagName)
	return 0
}

func isDryRun(lookupEnv func(string) (string, bool)) bool {
	return strings.EqualFold(strings.TrimSpace(valueFromEnv(lookupEnv, "SEMREL_DRY_RUN", "")), "true")
}

func valueFromEnv(lookupEnv func(string) (string, bool), key, fallback string) string {
	value, ok := lookupEnv(key)
	if !ok {
		return fallback
	}
	return value
}
