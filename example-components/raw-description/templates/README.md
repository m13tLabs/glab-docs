# yaml-lint

Lints the repository's YAML files with yamllint.

### Prerequisites

- a `.yamllint` config at the repository root (optional, the yamllint defaults apply otherwise)
- a `lint` stage in the calling pipeline

### Example

```yaml
include:
  - component: $CI_SERVER_FQDN/my-group/ci-components/yaml-lint@1.0.0
    inputs:
      paths: [ci/, templates/]
      strict: true
```

> [!NOTE]
> Findings are published as a Code Quality report, so they show up in the merge request widget.

## Usage

```yaml
include:
  - component: gitlab.com/m13tlabs/glab-docs/yaml-lint@<version>
    inputs:
      format: colored
      paths: ["."]
      stage: lint
      strict: false
```

## Inputs

### Behaviour

| Input | Type | Default | Options | Description |
|-------|------|---------|---------|-------------|
| format | string | `colored` | `colored`, `parsable`, `standard` | Output format of the job log. |
| strict | boolean | `false` |  | Fail on warnings, not only on errors. |

### General

| Input | Type | Default | Options | Description |
|-------|------|---------|---------|-------------|
| paths | array | `["."]` |  | Directories or files to lint.<br>Relative to the repository root. |
| stage | string | `lint` |  | Pipeline stage the job runs in. |

## Variables

| Variable | Default | Options | Description |
|----------|---------|---------|-------------|
| YAMLLINT_CONFIG | `.yamllint` in the repository root, if present |  | yamllint config file passed via `--config-file`. |

## Jobs

| Job | Stage | When | Needs | Description |
|-----|-------|------|-------|-------------|
| `.yamllint` | `test` |  |  | Shared yamllint setup, reusable via `extends:`. |
| `yaml-lint` | `$[[ inputs.stage ]]` |  |  | Runs yamllint and publishes a Code Quality report.<br>Extends: `.yamllint` |

