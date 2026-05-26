# provider-gitlab

Publishes the semrel release to GitLab.

This plugin is distributed as the standalone Go binary `semrel-plugin-provider-gitlab`. Semrel executes the binary as a subprocess, provides plugin configuration through `SEMREL_PLUGIN_*` environment variables, provides release context through `SEMREL_*` environment variables, reads standard output, and treats exit code `0` as success and any non-zero exit code as failure. Install the binary in `~/.semrel/plugins/` or anywhere on your `$PATH`.

## Installation

```bash
go install github.com/SemRels/provider-gitlab/cmd/plugin@latest
```

## Configuration

```yaml
plugins:
  - name: provider-gitlab
    path: ~/.semrel/plugins/semrel-plugin-provider-gitlab
    env:
      SEMREL_PLUGIN_TOKEN: "${GITLAB_TOKEN}"
      SEMREL_PLUGIN_BASE_URL: "https://gitlab.com"
      SEMREL_PLUGIN_PROJECT_ID: "12345"
      SEMREL_PLUGIN_MILESTONE: "v{{ .Version }}"
```

## `SEMREL_PLUGIN_*` variables

| Name | Required | Description | Default |
| --- | --- | --- | --- |
| `SEMREL_PLUGIN_TOKEN` | Required | GitLab API token. | None |
| `SEMREL_PLUGIN_BASE_URL` | Optional | Base URL of the GitLab instance. | https://gitlab.com |
| `SEMREL_PLUGIN_PROJECT_ID` | Optional | GitLab project ID. Defaults from the git remote when available. | Derived from git remote |
| `SEMREL_PLUGIN_MILESTONE` | Optional | Milestone name to associate with the release. | None |

## `SEMREL_*` release context used

| Variable | Description |
| --- | --- |
| `SEMREL_TAG_NAME` | Git tag name semrel will create or publish. |
| `SEMREL_CHANGELOG` | Generated changelog text for the release. |
| `SEMREL_BRANCH` | Git branch associated with the current release run. |
| `SEMREL_DRY_RUN` | Whether semrel is running in dry-run mode. |

## Example behavior

The plugin creates a GitLab release entry for the current tag and can attach the generated changelog and milestone metadata.

## License

Apache-2.0
