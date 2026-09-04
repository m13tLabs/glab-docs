# plain-pipeline

Reference pipeline wiring the shared templates together.

## Variables

| Variable | Default | Options | Description |
|----------|---------|---------|-------------|
| DEPLOY_ENV | `staging` | `staging`, `production` | Environment the reference pipeline deploys to. |
| FF_USE_FASTZIP | `true` |  |  |
| LINT_IMAGE | `registry.gitlab.com/m13tlabs/ci-images/lint:latest` |  | Container image used for lint jobs. |

## Includes

| Type | Location | Ref |
|------|----------|-----|
| component | `$CI_SERVER_FQDN/m13tlabs/glab-docs/build-image@main` |  |
| local | `/ci/lint.yml` |  |
| project | `m13tlabs/ci-common :: /templates/scan.yml` | `v2.1.0` |
| template | `Security/SAST.gitlab-ci.yml` |  |

