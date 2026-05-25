# provider-gitlab

`provider-gitlab` is the SemRels subprocess plugin for creating GitLab releases.

## Behavior

The plugin reads SemRel release context from environment variables, validates its configuration, respects `SEMREL_DRY_RUN`, and creates a release with the GitLab REST API.

Required plugin configuration:

- `SEMREL_PLUGIN_TOKEN`
- `SEMREL_PLUGIN_PROJECT_ID` (or `CI_PROJECT_ID`)
- `SEMREL_TAG_NAME`

Optional configuration:

- `SEMREL_PLUGIN_BASE_URL` (defaults to `https://gitlab.com`)
- `SEMREL_BRANCH` (sent as `ref` when present)
- `SEMREL_CHANGELOG` (used as the release description)

## Repository Layout

```
cmd/plugin/              Subprocess plugin entrypoint
internal/plugin/         GitLab release creation logic
.github/workflows/       CI, release, and security automation
```

## Development

```
go build ./cmd/plugin
go test ./...
```
