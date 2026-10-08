# update-docs

Generates or checks GitLab CI component/pipeline documentation with glab-docs.

## Usage

```yaml
include:
  - component: gitlab.com/m13tlabs/glab-docs/update-docs@<version>
    inputs:
      allow-failure: false
      before-script: ["set -eu\nif [ \"$GLAB_DOCS_MODE\" = \"check\" ] || [ \"$GLAB_DOCS_COMMIT\" = \"true\" ]; then\n  apk add --no-cache git\nfi\n"]
      commit: true
      commit-author-email: glab-docs@noreply.$CI_SERVER_HOST
      commit-author-name: glab-docs
      commit-message: docs: regenerate glab-docs output
      component-prefix: ""
      extra-args: ""
      gitlab-server-url: $CI_SERVER_URL
      image: m13t/glab-docs
      job-name: glab-docs
      mode: check
      output-file: README.md
      commit-push-token: $CI_JOB_TOKEN
      search-pattern: ""
      search-root: .
      stage: test
      strict: false
      template-files: README.md.gotmpl
      version: 1.0.0
```

## Inputs

| Input | Type | Default | Options | Description |
|-------|------|---------|---------|-------------|
| allow-failure | boolean | `false` |  | Mark the job as allowed to fail. |
| before-script | array | `["set -eu\nif [ \"$GLAB_DOCS_MODE\" = \"check\" ] || [ \"$GLAB_DOCS_COMMIT\" = \"true\" ]; then\n  apk add --no-cache git\nfi\n"]` |  | Steps run before the main script (`before_script`). The default installs git when `mode` is `check` or `commit` is enabled; these are available as `$GLAB_DOCS_MODE` and `$GLAB_DOCS_COMMIT`. |
| commit | boolean | `true` |  | In `generate` mode, commit changed docs and push them to the pipeline's branch. Skipped in tag and merge request pipelines. Needs `commit-push-token`. |
| commit-author-email | string | `glab-docs@noreply.$CI_SERVER_HOST` |  | Author and committer email of the docs commit. |
| commit-author-name | string | `glab-docs` |  | Author and committer name of the docs commit. |
| commit-message | string | `docs: regenerate glab-docs output` |  | Message of the commit created when `commit` is enabled. The push uses `-o ci.skip`, so it doesn't start a new pipeline. |
| component-prefix | string | _none_ |  | Address prefix override for the generated include snippet (`--component-prefix`). Empty resolves the project path automatically (from `$CI_PROJECT_PATH`) behind a literal `$CI_SERVER_FQDN` placeholder. |
| extra-args | string | _none_ |  | Extra raw arguments appended to the glab-docs command. |
| gitlab-server-url | string | `$CI_SERVER_URL` |  | GitLab server base URL used to resolve `$CI_SERVER_FQDN` in documented `component:` includes and link them to their source project (`--gitlab-server-url`). |
| image | string | `m13t/glab-docs` |  | glab-docs container image, without the tag. |
| job-name | string | `glab-docs` |  | Name of the generated job. |
| mode | string | `check` | `check`, `generate` | `check` fails the job when the committed docs are stale; `generate` writes them (and commits them when `commit` is enabled). |
| output-file | string | `README.md` |  | Generated file name passed to `--output-file`. |
| commit-push-token | string | `$CI_JOB_TOKEN` |  | Token used to push the docs commit, as a CI/CD variable reference. The default job token needs "Allow Git push requests to the repository" enabled under Settings > CI/CD > Job token permissions. Otherwise pass a masked variable holding a project access token with `write_repository`; don't pass the token itself. |
| search-pattern | string | _none_ |  | Comma-separated `--search-pattern` globs. Empty keeps the built-in defaults. |
| search-root | string | `.` |  | Directory glab-docs searches for CI YAML files (`--search-root`). |
| stage | string | `test` |  | Pipeline stage the generated job runs in. |
| strict | boolean | `false` |  | Fail on undocumented inputs / variables (adds `--documentation-strict-mode`). |
| template-files | string | `README.md.gotmpl` |  | Template file name passed to `--template-files`. |
| version | string | `1.0.0` |  | glab-docs container image tag. |

## Jobs

| Job | Stage | When | Needs | Description |
|-----|-------|------|-------|-------------|
| `$[[ inputs.job-name ]]` | `$[[ inputs.stage ]]` |  |  |  |

