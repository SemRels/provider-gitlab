# provider-gitlab

GitLab provider plugin for Semantic Release.

Provides GitLab repository, release, and metadata integration for Semantic Release.

## Documentation

- Docs (coming soon): <https://github.com/SemRels/semrel/tree/main/docs/plugins/provider-gitlab>
- Template source: <https://github.com/SemRels/plugin-template>

## Repository Layout

`	ext
cmd/plugin/              Plugin entry point
internal/plugin/         Business logic scaffold
internal/grpc/           gRPC transport scaffold
proto/v1                 Symlink to the SemRel protobuf contract
.github/workflows/       CI, release, and security automation
`

## Development

`ash
go build ./cmd/plugin
go test ./...
`

## Configuration Example

`yaml
plugins:
  - name: provider-gitlab
    type: provider
    config:
      api_url: https://gitlab.com/api/v4
      project_id: group/example-repo
      token: ${GITLAB_TOKEN}
`

## Status

This repository is bootstrapped from SemRels/plugin-template and is ready for implementation.