# plain-pipeline

Reference pipeline wiring the shared templates together.

## Variables

| Variable | Default | Options | Description |
|----------|---------|---------|-------------|
| DEPLOY_ENV | `staging` | `staging`, `production` | Environment the reference pipeline deploys to. |
| FF_USE_FASTZIP | `true` |  |  |
| LINT_IMAGE | `registry.gitlab.com/m13tlabs/ci-images/lint:latest` |  | Container image used for lint jobs. |

## Jobs

| Job | Stage | When | Needs | Description |
|-----|-------|------|-------|-------------|
| `lint` | `lint` |  |  | Runs golangci-lint over the module. |
| `unit-tests` | `test` |  | `lint` | Unit tests with coverage; publishes a JUnit report. |

## Includes

| Type | Location | Ref | Description |
|------|----------|-----|-------------|
| component | `$CI_SERVER_FQDN/m13tlabs/glab-docs/build-image` | `main` | Builds a container image with Kaniko and pushes it to the project registry.<br>**Variables:**<ul><li>`CI_DEBUG` = `true` - Enable this to enable debug logging in Gitlab</li></ul>**Jobs:**<ul><li>`build-image` - Build Dockerfile with Kaniko</li></ul> |
| project | `m13tLabs/glab-docs (file: example-components/include.yml)` | `develop` |  |
| local | `/ci/lint.yml` |  |  |
| template | `Security/SAST.gitlab-ci.yml` |  |  |

