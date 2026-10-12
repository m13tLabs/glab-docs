# Glab Pipeline Docs

## dockerfile-lint

Lints Dockerfiles with hadolint.

### Usage

```yaml
include:
  - component: gitlab.com/m13tlabs/glab-docs/dockerfile-lint@<version>
    inputs:
      dockerfile: Dockerfile
      stage: lint
```

### Inputs

| Input | Type | Default | Options | Description |
|-------|------|---------|---------|-------------|
| dockerfile | string | `Dockerfile` |  | Path to the Dockerfile to lint. |
| stage | string | `lint` |  | Pipeline stage the job runs in. |

### Jobs

| Job | Stage | When | Needs | Description |
|-----|-------|------|-------|-------------|
| `dockerfile-lint` | `$[[ inputs.stage ]]` |  |  | Runs hadolint on the configured Dockerfile. |

### Includes

| Type | Location | Ref | Description |
|------|----------|-----|-------------|
| local | [`/example-components/combined/templates/shell-lint.yml`](#shell-lint) |  | Lints shell scripts with ShellCheck.<br>**Jobs:**<ul><li>`shell-lint` - Runs ShellCheck over every `*.sh` file.</li></ul> |

## shell-lint

Lints shell scripts with ShellCheck.

### Usage

```yaml
include:
  - component: gitlab.com/m13tlabs/glab-docs/shell-lint@<version>
    inputs:
      severity: warning
      stage: lint
```

### Inputs

| Input | Type | Default | Options | Description |
|-------|------|---------|---------|-------------|
| severity | string | `warning` | `error`, `warning`, `info`, `style` | Minimum severity that fails the job. |
| stage | string | `lint` |  | Pipeline stage the job runs in. |

### Jobs

| Job | Stage | When | Needs | Description |
|-----|-------|------|-------|-------------|
| `shell-lint` | `$[[ inputs.stage ]]` |  |  | Runs ShellCheck over every `*.sh` file. |

