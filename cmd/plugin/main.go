// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The semrel Authors

package main

import (
	"log"

	plugin "github.com/SemRels/provider-gitlab/internal/plugin"
)

func main() {
	client := plugin.NewClient(plugin.Config{})
	log.Printf("provider-gitlab plugin ready: creates GitLab releases (%T)", client)
}
