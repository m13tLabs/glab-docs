# glab-docs

Generates or checks GitLab CI component/pipeline documentation with glab-docs.

## Usage

```yaml
include:
  - component: $CI_SERVER_FQDN/<path-to-project>/glab-docs@<version>
    inputs:
      allow-failure: false
      component-prefix: $CI_SERVER_FQDN/$CI_PROJECT_PATH
      extra-args: ""
      image: m13t/glab-docs
      job-name: glab-docs
      mode: check
      output-file: README.md
      search-pattern: ""
      search-root: .
      stage: test
      strict: false
      template-files: README.md.gotmpl
      version: 0.2.1
```

## Inputs

| Input | Type | Default | Options | Description |
|-------|------|---------|---------|-------------|
| allow-failure | boolean | `false` |  | Mark the job as allowed to fail. |
| component-prefix | string | `$CI_SERVER_FQDN/$CI_PROJECT_PATH` |  | Address prefix for the generated include snippet (`--component-prefix`). |
| extra-args | string | _none_ |  | Extra raw arguments appended to the glab-docs command. |
| image | string | `m13t/glab-docs` |  | glab-docs container image, without the tag. |
| job-name | string | `glab-docs` |  | Name of the generated job. |
| mode | string | `check` | `check`, `generate` | `check` fails the job when the committed docs are stale; `generate` just writes them. |
| output-file | string | `README.md` |  | Generated file name passed to `--output-file`. |
| search-pattern | string | _none_ |  | Comma-separated `--search-pattern` globs. Empty keeps the built-in defaults. |
| search-root | string | `.` |  | Directory glab-docs searches for CI YAML files (`--search-root`). |
| stage | string | `test` |  | Pipeline stage the generated job runs in. |
| strict | boolean | `false` |  | Fail on undocumented inputs / variables (adds `--documentation-strict-mode`). |
| template-files | string | `README.md.gotmpl` |  | Template file name passed to `--template-files`. |
| version | string | `0.2.1` |  | glab-docs container image tag. |

## Jobs

| Job | Stage | When | Needs | Description |
|-----|-------|------|-------|-------------|
| `$[[ inputs.job-name ]]` | `$[[ inputs.stage ]]` |  |  |  |

