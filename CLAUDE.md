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
    doc and the body doc, pull `spec.inputs` / `variables` / `include` nodes, scan the raw file
    for old-style `# key --` comments, strict-mode lint.
  - `comment.go` — the inherited `ParseComment` (`# --` blocks). **Panics on an empty slice**
    (`commentLines[docStartIdx+1:]`), so every caller guards `len(commentLines) > 0`.
- `pkg/document/` — model + render.
  - `model.go` — `inputRow`, `variableRow`, `componentTemplateData`, sorting, `@section`
    grouping (`groupInputSections`), `getComponentTemplateData`.
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
  exercises `variables:` + `include:`; `build-image/templates/` exercises `spec:inputs`.
- `templates/glab-docs.yml` — the published GitLab CI/CD component that runs the
  `m13tlabs/glab-docs` image (`check` = `git diff --exit-code` the regenerated docs; `generate`
  = just write). `templates/README.md` is its own generated doc. `.gitlab-ci.yml` is the
  integration pipeline that includes the component `@$CI_COMMIT_SHA` against
  `example-components/` and `templates/`. The repo's own `.gitlab-ci.yml` is in
  `.glabdocsignore` so glab-docs never documents it.

## Design notes / gotchas

- The usage snippet (`pipeline.usageSection`) renders only when the file has `spec:inputs` AND
  `--component-prefix` (or the `$CI_SERVER_FQDN/...` placeholder) is set.
- Component name: file stem, or the parent dir name for `template.yml` / `.gitlab-ci.yml`.
- Strict mode (`-x`/`-y`/`-z`) lints inputs **and** variables by name; a native `description:`
  counts as documented (`documentedNames`).
- Dropped from helm-docs: `Chart.yaml`/`requirements.yaml`, umbrella dependency values,
  `--values-file`, `--document-dependency-values`, `--badge-style`, the HTML value renderer,
  the benchmark suite.
- The Docker image has **no `git`** — `pkg/util/ignore.go` falls back to a working-dir-relative
  `.glabdocsignore` lookup when `git rev-parse` fails, which is the normal container case. Do
  not re-add an unpinned `apk add` (hadolint DL3018).

## CI

- `.github/workflows/build.yml` — `actionlint` (reviewdog, with `-ignore` flags for the stale
  `actions/create-github-app-token` input snapshot), `hadolint`, then vet + test +
  `goreleaser build --snapshot`. No release here.
- `.github/workflows/release.yml` — tag `v*` / manual dispatch only. Docker Hub login and GPG
  import are optional: a step assembles `--skip=sign,publish` when `DOCKER_HUB_USER` /
  `SIGNING_KEY` secrets are unset, so a tag build passes before credentials are wired up.
- `.goreleaser.yml` — `project_name: glab-docs`, `./cmd/glab-docs`, `m13tlabs/glab-docs`
  images. Uses v1 `dockers` + `docker_manifests` (deprecation *warning* only; `dockers_v2`
  multi-platform context did not resolve the binary). `snapshot.version_template` (not the
  removed `name_template`).
- `.hadolint.yaml` — just `failure-threshold: warning`.

## Validating changes

```sh
go vet ./... && go test ./...

# end-to-end; regenerate every committed README under example-components/ and templates/
go run ./cmd/glab-docs --search-root . --component-prefix gitlab.com/m13tlabs/glab-docs
git diff example-components templates

goreleaser check                       # config sanity (warns on deprecated dockers — ok)
goreleaser release --snapshot --clean --skip=publish,sign   # full build incl. docker
actionlint .github/workflows/*.yml
hadolint --config .hadolint.yaml Dockerfile
```

## History

PR #4 merged the engine pivot + brand rebrand into `develop`. Small chore/lint/doc commits
land directly on `develop`; feature work goes on a branch.
