# update-docs

Generates or checks GitLab CI component/pipeline documentation with glab-docs.

## Usage

```yaml
include:
  - component: gitlab.com/m13tlabs/glab-docs/update-docs@<version>
    inputs:
      allow-failure: false
      before-script: ["set -eu\nif [ \"$[[ inputs.mode ]]\" = \"check\" ]; then\n  apk add --no-cache git\nfi\n"]
      component-prefix: ""
      extra-args: ""
      gitlab-server-url: $CI_SERVER_URL
      image: m13t/glab-docs
      job-name: glab-docs
      mode: check
      output-file: README.md
      search-pattern: ""
      search-root: .
      stage: test
      strict: false
      template-files: README.md.gotmpl
      version: 0.6.0
```

## Inputs

| Input | Type | Default | Options | Description |
|-------|------|---------|---------|-------------|
| allow-failure | boolean | `false` |  | Mark the job as allowed to fail. |
| before-script | array | `["set -eu\nif [ \"$[[ inputs.mode ]]\" = \"check\" ]; then\n  apk add --no-cache git\nfi\n"]` |  | Run before steps, default to install git |
| component-prefix | string | _none_ |  | Address prefix override for the generated include snippet (`--component-prefix`). Empty resolves the project path automatically (from `$CI_PROJECT_PATH`) behind a literal `$CI_SERVER_FQDN` placeholder. |
| extra-args | string | _none_ |  | Extra raw arguments appended to the glab-docs command. |
| gitlab-server-url | string | `$CI_SERVER_URL` |  | GitLab server base URL used to resolve `$CI_SERVER_FQDN` in documented `component:` includes and link them to their source project (`--gitlab-server-url`). |
| image | string | `m13t/glab-docs` |  | glab-docs container image, without the tag. |
| job-name | string | `glab-docs` |  | Name of the generated job. |
| mode | string | `check` | `check`, `generate` | `check` fails the job when the committed docs are stale; `generate` just writes them. |
| output-file | string | `README.md` |  | Generated file name passed to `--output-file`. |
| search-pattern | string | _none_ |  | Comma-separated `--search-pattern` globs. Empty keeps the built-in defaults. |
| search-root | string | `.` |  | Directory glab-docs searches for CI YAML files (`--search-root`). |
| stage | string | `test` |  | Pipeline stage the generated job runs in. |
| strict | boolean | `false` |  | Fail on undocumented inputs / variables (adds `--documentation-strict-mode`). |
| template-files | string | `README.md.gotmpl` |  | Template file name passed to `--template-files`. |
| version | string | `0.6.0` |  | glab-docs container image tag. |

## Jobs

| Job | Stage | When | Needs | Description |
|-----|-------|------|-------|-------------|
| `$[[ inputs.job-name ]]` | `$[[ inputs.stage ]]` |  |  |  |

