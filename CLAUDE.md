# glab-docs

`glab-docs` auto-generates Markdown docs for **GitLab CI/CD components and pipelines** — the
GitLab analogue of [helm-docs](https://github.com/norwoodj/helm-docs) for Helm charts. It is a
hard fork of `norwoodj/helm-docs` (GPL-3.0): the gotemplate engine, the sprig funcmap and the
`# --` comment parser are inherited; the ignore-file handling was reimplemented to drop the
helm dependency; everything chart-specific was replaced. See [README.md](README.md) for
user-facing docs.

## What it reads

For every YAML file matched by `--search-pattern` under `--search-root` it parses:

- `spec:inputs:` (the component's parameter interface — analogue of `values.yaml`). Each input's
  `description` / `default` / `type` / `options` / `regex` are **native fields**; an input with
  no `default:` is `_required_`.
- the pipeline's top-level `variables:` block (scalar, or the `{value, description, options}`
  extended form).
- the top-level `include:` list (component / local / project / remote / template).
- every top-level key that isn't a reserved keyword (`reservedTopLevelKeys`) → a **job**, with
  `stage` / `when` (or `rules`/`only`/`except`) / `needs` / `extends` / `image` / a `# --`
  description. Ordered by `stages:` (or GitLab's implicit order); hidden `.jobs` are parsed but
  excluded from the rendered table (`getJobRows`).
- a leading `# --` file comment → the component description.

`# --` / `# <name> --` / `# @section` / `# @default` / `# (type)` / `# @ignore` comment
annotations still work and override the native fields.

## Repo layout

- `cmd/glab-docs/` — CLI. `command_line.go` = cobra flags + `GLAB_DOCS_*` env binding;
  `main.go` = walk → parse (parallel) → render.
- `pkg/gitlab/` — parsing (was `pkg/helm`).
  - `component_finder.go` — `FindComponentFiles`: walk + glob match. Patterns are matched
    against **every trailing path segment** of a file (and the search-root's own base name is
    prepended first), so `templates/*.yml` matches both `any/dir/templates/foo.yml` and
    `foo.yml` when `--search-root` points straight at a `templates/` dir. `DefaultSearchPatterns`
    lives here.
  - `component_info.go` — `ParseComponentInformation`: multi-doc YAML decode, locate the `spec:`
    doc and the body doc (`docLooksLikeBody`), pull `spec.inputs` / `variables` / `include` /
    `stages` nodes, `parseJobs` over the body's top-level keys, scan the raw file for old-style
    `# key --` comments, strict-mode lint (inputs, then variables, then jobs — stops at the
    first failing category).
  - `comment.go` — the inherited `ParseComment` (`# --` blocks). **Panics on an empty slice**
    (`commentLines[docStartIdx+1:]`), so every caller guards `len(commentLines) > 0`.
- `pkg/document/` — model + render.
  - `model.go` — `inputRow`, `variableRow`, `jobRow`, `componentTemplateData`, sorting,
    `@section` grouping (`groupInputSections`), `getComponentTemplateData` / `getJobRows`.
  - `inputs.go` / `variables.go` — build rows from the YAML nodes; `renderNodeValue`
    backtick-wraps scalars and JSON-encodes seq/map; `commentOverride` layers `# --` on top.
  - `template.go` — built-in `pipeline.*` templates (see README table) + `defaultDocumentationTemplate`.
  - `generate.go` — `PrintDocumentation`: render + `applyMarkDownFormat`, write `README.md`
    into the **source file's own directory**.
  - `files.go` / `util.go` — inherited `.Files` helper + sort constants.
- `pkg/util/` — `FuncMap` (sprig + toYaml/fromYaml), `git.go`, `ignore.go` +
  `gitignore.go` (a ~130-line dependency-free port of `helm.sh/helm/v3/pkg/ignore`, so the
  whole helm dependency tree is gone; `filepath.Match` semantics, `**` rejected).
- `example-components/` — fixtures with committed generated `README.md`s. `plain-pipeline/`
  exercises `variables:` + `include:` + a couple of simple jobs; `build-image/templates/`
  exercises `spec:inputs`. Richer job cases (`needs` as maps, `extends`, `when: manual`, hidden
  `.jobs`) live in `pkg/gitlab/test-fixtures/jobs/`. **Keep the fixtures minimal — no fictional
  deploy/dind jobs.**
- `templates/glab-docs.yml` — the published GitLab CI/CD component that runs the
  `m13tlabs/glab-docs` image (`check` = `git diff --exit-code` the regenerated docs; `generate`
  = just write). `templates/README.md` is its own generated doc. `.gitlab-ci.yml` is the
  integration pipeline whose **only job is to run the glab-docs component** against
  `example-components/` and `templates/` — do not add real build/deploy stages to it. The
  repo's own `.gitlab-ci.yml` is in `.glabdocsignore` so glab-docs never documents it.

## Design notes / gotchas

- The usage snippet (`pipeline.usageSection`) renders only when the file has `spec:inputs` AND
  `--component-prefix` (or the `$CI_SERVER_FQDN/...` placeholder) is set.
- Component name: file stem, or the parent dir name for `template.yml` / `.gitlab-ci.yml`.
- Strict mode (`-x`/`-y`/`-z`) lints inputs **and** variables by name; a native `description:`
  counts as documented (`documentedNames`).
- Dropped from helm-docs: `Chart.yaml`/`requirements.yaml`, umbrella dependency values,
  `--values-file`, `--document-dependency-values`, `--badge-style`, the HTML value renderer,
  the benchmark suite.
- The **`Dockerfile` is multi-stage** (`golang:1.26-alpine` builds the binary → `alpine:3.24`),
  so `docker build .` works standalone — GoReleaser does **not** build images. The build stage
  runs `go build -buildvcs=false` (no `.git` in the context) and takes a `VERSION` build-arg
  (`-X main.version`). `.dockerignore` allowlists only `cmd/ pkg/ go.mod go.sum`. Neither stage
  runs `apk add` — keep it that way (hadolint DL3018 is a *warning* and `.hadolint.yaml` sets
  `failure-threshold: warning`). The runtime image has no `git`; `pkg/util/ignore.go` falls
  back to a working-dir-relative `.glabdocsignore` lookup.

## CI

- `.github/workflows/build.yml` — `actionlint` (reviewdog, with `-ignore` flags for the stale
  `actions/create-github-app-token` input snapshot), `hadolint`, then vet + test +
  `goreleaser build --snapshot` + `docker/build-push-action` (load only) + `docker run --version`
  smoke test. No release here.
- `.github/workflows/release.yml` — tag `v*` / manual dispatch. `release` job: GoReleaser does
  archives / nfpm / checksums / GitHub release (`--skip=sign` when `SIGNING_KEY` is unset).
  `image` job: `docker/metadata-action` + multi-arch `docker/build-push-action`,
  `push: ${{ env.DOCKER_HUB_USER != '' }}` so it builds-only until Docker Hub creds exist.
- `.goreleaser.yml` — `project_name: glab-docs`, `./cmd/glab-docs`; no `dockers:` block.
  `snapshot.version_template` (not the removed `name_template`). `goreleaser check` is clean.

## Validating changes

```sh
go vet ./... && go test ./...

# end-to-end; regenerate every committed README under example-components/ and templates/
go run ./cmd/glab-docs --search-root . --component-prefix gitlab.com/m13tlabs/glab-docs
git diff example-components templates

goreleaser check
goreleaser release --snapshot --clean --skip=sign   # archives / nfpm / checksums
docker build -t glab-docs:test --build-arg VERSION=test . && docker run --rm glab-docs:test --version
actionlint .github/workflows/*.yml
hadolint --config .hadolint.yaml Dockerfile
```

## History

PR #4 merged the engine pivot + brand rebrand into `develop`. Small chore/lint/doc commits
land directly on `develop`; feature work goes on a branch.
