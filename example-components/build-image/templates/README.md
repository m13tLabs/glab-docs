# build-image

Builds a container image with Kaniko and pushes it to the project registry.

## Usage

```yaml
include:
  - component: gitlab.com/m13tlabs/glab-docs/build-image@<version>
    inputs:
      build_args: []
      context: .
      destination: "" # required
      dockerfile: Dockerfile
      log_level: info
      push: true
      stage: build
      tag: latest
```

## Inputs

| Input | Type | Default | Options | Description |
|-------|------|---------|---------|-------------|
| build_args | array | `[]` |  | Additional `--build-arg KEY=VALUE` entries. |
| context | string | `.` |  | Build context passed to the builder. |
| destination | string | _required_ |  | Fully-qualified image reference to push (registry/image:tag). |
| dockerfile | string | `Dockerfile` |  | Path to the Dockerfile, relative to the context. |
| log_level | string | `info` | `debug`, `info`, `warn`, `error` | Builder log verbosity. |
| push | boolean | `true` |  | Push the image after a successful build. |
| stage | string | `build` |  | Pipeline stage the job runs in. |
| tag | string | `latest` |  | Tag applied to the built image. Must be a valid Docker tag.<br>Pattern: `^[\w][\w.-]{0,127}$` |

## Jobs

| Job | Stage | When | Needs | Description |
|-----|-------|------|-------|-------------|
| `build-image` | `$[[ inputs.stage ]]` |  |  |  |

