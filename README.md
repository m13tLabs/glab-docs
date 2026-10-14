<p align="center">
  <img src=".github/assets/logo.svg" alt="glab-docs" width="440">
</p>

# glab-docs

[![CI](https://github.com/m13tLabs/glab-docs/actions/workflows/build.yml/badge.svg)](https://github.com/m13tLabs/glab-docs/actions/workflows/build.yml)
[![Docker Pulls](https://img.shields.io/docker/pulls/m13t/glab-docs)](https://hub.docker.com/r/m13t/glab-docs)


`glab-docs` auto-generates Markdown documentation for **GitLab CI/CD components and pipelines**,
the way [helm-docs](https://github.com/norwoodj/helm-docs) does for Helm charts. It reads a CI
YAML file's `spec:inputs:` header, its top-level `variables:` block, its `include:` list and its
jobs, and renders a `README.md` with a table for each.

This project is a fork of `norwoodj/helm-docs` (GPL-3.0); the Go templating engine and comment
parser are inherited from it.

## What it produces

Given `templates/build-image.yml`:

```yaml
# -- Builds a container image with Kaniko and pushes it to the project registry.
spec:
  inputs:
    stage:
      default: build
      description: Pipeline stage the job runs in.
    destination:
      description: Fully-qualified image reference to push.
    log_level:
      default: info
      description: Builder log verbosity.
      options: [debug, info, warn, error]
    tag:
      default: latest
      regex: '^[\w][\w.-]{0,127}$'
---
"build-image":
  stage: $[[ inputs.stage ]]
  script: [ /kaniko/executor --destination "$[[ inputs.destination ]]" ]
```

`glab-docs` writes a `README.md` next to it with a usage snippet plus:

| Input | Type | Default | Options | Description |
|-------|------|---------|---------|-------------|
| destination | string | _required_ | | Fully-qualified image reference to push. |
| log_level | string | `info` | `debug`, `info`, `warn`, `error` | Builder log verbosity. |
| stage | string | `build` | | Pipeline stage the job runs in. |
| tag | string | `latest` | | Pattern: `^[\w][\w.-]{0,127}$` |

## Installation

```bash
go install github.com/m13tLabs/glab-docs/cmd/glab-docs@latest
```

Or build from source:

```bash
make glab-docs      # produces ./glab-docs
```

Container image: `m13tlabs/glab-docs:latest`.

You can also use directly the [CI/CD component](https://gitlab.com/explore/catalog/m13tLabs/glab-docs).

## Usage

```bash
glab-docs                     # walk . and (re)write README.md for every match
glab-docs --dry-run           # print to stdout instead of writing files
glab-docs -c path/to/repo --component-prefix gitlab.com/my-group/my-project
```

### What gets discovered

`--search-pattern` is a repeatable glob list, matched against every path suffix of each file
under `--search-root`. Defaults:

```
templates/*.yml   templates/*.yaml
templates/*/template.yml   templates/*/template.yaml
*.gitlab-ci.yml   *.gitlab-ci.yaml   .gitlab-ci.yml   .gitlab-ci.yaml
```

A `.glabdocsignore` file (same syntax as `.gitignore`) excludes paths.

### Component name and README location

The component name is the file stem, or the parent directory name for `template.yml` /
`.gitlab-ci.yml`. The `README.md` is written into the file's own directory; use
`-o ../README.md` to hoist it.

When several component files resolve to the _same_ output path - most commonly several flat
`templates/*.yml` siblings, which all share the `templates` directory - they're combined into one
README instead of clobbering each other: an `--combined-title` H1 (default `Glab Pipeline Docs`,
empty to omit it) sits at the top, and each template gets its own `##` section (one heading level
deeper than a standalone README), with a single version footer for the whole file.
`--component-to-generate` still works against individual files, but if it selects only some
members of a combined group the whole group is (re)written together, so the shared file is never
left with only a partial set of sections. See
[example-components/combined](example-components/combined/templates/README.md).

A component's `include: - local: ...` entries that resolve to another documented component are
turned into links in its Includes table - an in-page `#anchor` when both share a combined README,
otherwise a relative link to that component's own README (with `#anchor` if it's part of a
combined README there too). `local:` paths are taken relative to the project root (as GitLab
does: `$CI_PROJECT_DIR`, else the git toplevel), so this also works with a `--search-root` below
it, e.g. `templates`. A `local:` include of a file that isn't a documented component (a helper
file outside the search root, say) links to the file itself, if it exists. `project:` and `component:` includes are also linked when
`--gitlab-server-url` is set (see below); `project:` links straight to the included `file:`'s blob
at its `ref:` when one is given, otherwise to the project root; `component:` links to
`templates/<component-name>.yml` at its `@<ref>` (or `HEAD` when unpinned) - the component
address's project path isn't itself a browsable file, and glab-docs' own flat `templates/*.yml`
layout is assumed since the alternative `templates/<name>/template.yml` form can't be told apart
from the address alone. `remote:`/`template:` includes are listed as-is, since they can't be
resolved to a project on the same GitLab instance.

The Includes table's **Description** column summarizes what each include brings into the
pipeline: the included file's own leading `# --` description, its `variables:` (each with its
value and description) and its jobs (each with its `# --` description; hidden `.templates` included, since helper includes usually export those for `extends:` /
`!reference`). Nested includes are followed, so a `project:` include of an entry file that only
`include:`s other files still lists everything those files define. Files are read from:

- `local:` - the checkout (`$CI_PROJECT_DIR`, else the git toplevel, else `--search-root`);
- `project:` of the documented repository itself (same project matching as for components
  below) - local git at the include's `ref:`: the working tree (uncommitted edits included) when
  the ref is the checked-out one (`HEAD`, its branch or SHA, `$CI_COMMIT_REF_NAME`,
  `$CI_COMMIT_SHA`) or no `ref:` is given, otherwise `git show <ref>:<file>` (then
  `origin/<ref>`). Nested `local:` includes are read at the same ref. If git doesn't have the
  ref (a shallow CI clone, no git in the image), the API below is tried instead;
- any other `project:` - the GitLab repository files API of `--gitlab-server-url`, at the
  include's `ref:` (or `HEAD`);
- `component:` of one of the repository's own components (the address's project matches
  `$CI_PROJECT_PATH`, the `--component-prefix` project or the git remote, case-insensitively, or
  is literally `$CI_PROJECT_PATH`) - the discovered `templates/` file in the checkout, whatever
  the ref;
- any other `component:` - the same API, trying `templates/<name>.yml` then `templates/<name>/template.yml`
  at the address's `@<ref>`; a `$CI_SERVER_FQDN` address needs `--gitlab-server-url`, a literal
  host is fetched from `https://<host>` (anonymously, unless it's the `--gitlab-server-url` host).
  `~latest`-style refs aren't resolved.

Requests to `--gitlab-server-url` authenticate with `--gitlab-token` / `GLAB_DOCS_GITLAB_TOKEN`
(`PRIVATE-TOKEN`, needs `read_api`), falling back to `$CI_JOB_TOKEN` (`JOB-TOKEN`). In the
`update-docs` component, set `GLAB_DOCS_GITLAB_TOKEN` as a masked CI/CD variable rather than an
input. Anything that can't be resolved just leaves its cell empty (with a warning); pass
`--include-details=false` to skip resolving entirely, e.g. offline. Note that an include pinned
to a moving ref (a branch like `main`) makes the generated docs change whenever that branch does,
which `mode: check` will then report as stale docs.

### The usage snippet

`--component-prefix <host>/<group>/<project>` produces a real
`include: - component: <prefix>/<name>@<version>` block. Without it, the project path is resolved
automatically - from `$CI_PROJECT_PATH` when running as a GitLab CI job, otherwise from the local
git remote - behind a literal `$CI_SERVER_FQDN` placeholder:
`$CI_SERVER_FQDN/<project-path>/<name>@<version>` (falling back to a literal `<path-to-project>`
placeholder when neither is available). The snippet is only emitted for files that declare
`spec:inputs:`.

## Documenting inputs, variables and jobs

`spec:inputs:` entries are documented from their native fields — `description`, `default`,
`type` (`string` / `number` / `boolean` / `array`, otherwise inferred from the default),
`options`, `regex`. An input with no `default:` is shown as `_required_`.

`variables:` entries are documented from their value, or from the extended
`{ value, description, options }` form.

Every top-level key that isn't a reserved GitLab keyword is treated as a **job** and listed in
a Jobs table with its `stage`, `when` (or `rules` / `only`/`except`), `needs`, `extends` and a
`# --` description comment. Jobs are ordered by the pipeline's `stages:` (or GitLab's implicit
order). Hidden jobs (`.name`) are skipped unless they carry a `# --` description comment, in
which case they're listed like any other job (still ranked first, ahead of the `stages:`
order, since they're templates/anchors rather than pipeline stages).

```yaml
stages: [test, deploy]

# -- Runs the unit test suite with coverage.
unit-tests:
  stage: test
  script: [go test ./...]
```

### Comment annotations

Inherited from helm-docs and layered on top of the native fields:

```yaml
spec:
  inputs:
    # -- Overrides the native description for this input.
    # @section -- Networking
    proxy_url:
      default: ""
```

- `# -- <text>` on the line(s) immediately above a key overrides its description. Continuation
  lines without the `--` are appended.
- `# <name> -- <text>` anywhere in the file does the same, addressed by name.
- `# (type) -- <text>` sets the rendered type.
- `# @section -- <name>` groups the input under a sub-heading.
- `# @default -- <text>` overrides the rendered default.
- `# @ignore` on a key drops it from the tables.
- `# @raw` keeps the line breaks of the following comment lines instead of joining them with
  spaces (see below).

### Adding your own content with `@raw`

Use `# @raw` in the file's leading `# --` description to put your own Markdown into the README:
headings, lists, code blocks, notes. Every comment line after `@raw` is copied as-is, with only the
leading `#` and one space removed, and a bare `#` becomes an empty line:

```yaml
# -- Deploys the application to Kubernetes.
# @raw
#
# ## Prerequisites
#
# - a `KUBECONFIG` CI/CD variable (type _File_)
# - a `deploy` stage in the calling pipeline
#
# ```yaml
# stages: [build, deploy]
# ```
spec:
  inputs:
    ...
```

renders right below the `# <name>` heading:

````markdown
# deploy

Deploys the application to Kubernetes.

## Prerequisites

- a `KUBECONFIG` CI/CD variable (type _File_)
- a `deploy` stage in the calling pipeline

```yaml
stages: [build, deploy]
```

## Usage
...
````

Notes:

- `@raw` goes after the `# --` line. Lines between the two are still joined with spaces.
- Any further indentation after the `#` and its space is kept, so nested lists and indented code work.
- `@default`, `@section` and `@notationType` lines are still read as annotations, not copied.
- Only use `@raw` in the file description. On an input, variable or job it puts line breaks into
  a table cell, which breaks the table. Use an inline `<br>` there instead.
- For content that doesn't fit into a comment (extra sections, badges, a different order), use a
  `README.md.gotmpl` (see [Templates](#templates)).

For a complete example, see
[example-components/raw-description](example-components/raw-description/templates/yaml-lint.yml)
and [the README it produces](example-components/raw-description/templates/README.md).

## Templates

Rendering is [gotemplate](https://pkg.go.dev/text/template) driven, with the
[sprig](https://github.com/Masterminds/sprig) function library. Drop a `README.md.gotmpl` next
to a component to customise it; otherwise the built-in default template is used.

Built-in sub-templates:

| Name | Description |
|------|-------------|
| `pipeline.header` | `# <name>` heading |
| `pipeline.description` | the leading `# --` file description |
| `pipeline.usageSection` | the `include:` usage snippet (components only) |
| `pipeline.inputsSection` / `pipeline.inputsTable` | the inputs table (grouped by `@section` when used) |
| `pipeline.variablesSection` / `pipeline.variablesTable` | the `variables:` table |
| `pipeline.jobsSection` / `pipeline.jobsTable` | the Jobs table |
| `pipeline.includesSection` / `pipeline.includesTable` | the `include:` table |
| `glab-docs.versionFooter` | autogeneration footer (suppress with `--skip-version-footer`) |

A custom template can mix these sub-templates with your own Markdown, for example:

```gotemplate
{{ template "pipeline.header" . }}

{{ template "pipeline.description" . }}

{{ template "pipeline.usageSection" . }}

## Configuration

Your own text, placed between the generated sections.

{{ template "pipeline.inputsSection" . }}

{{ template "glab-docs.versionFooter" . }}
```

See [example-components/custom-template](example-components/custom-template/templates/README.md.gotmpl)
for a full template and [its output](example-components/custom-template/templates/README.md).

## Examples

[`example-components/`](example-components) contains sample components together with the
`README.md` that `glab-docs` generates for each of them:

| Example | Shows |
|---------|-------|
| [build-image](example-components/build-image/templates/) | `spec:inputs:` with native `description` / `type` / `options` / `regex`, a `# --` override, `variables:` and a job |
| [raw-description](example-components/raw-description/templates/) | `@raw` own Markdown in the file description, `@section`, `@default`, `@ignore`, `<br>` in a table cell, a hidden `.job` used by `extends:` |
| [custom-template](example-components/custom-template/templates/) | a `README.md.gotmpl` that adds own sections around the built-in ones |
| [combined](example-components/combined/templates/) | two flat `templates/*.yml` combined into one README, with an in-page link for a `local:` include |
| [plain-pipeline](example-components/plain-pipeline/) | a plain `.gitlab-ci.yml` (no `spec:inputs:`) with `component:`, `project:`, `local:` and `template:` includes |

To regenerate them after a change:

```bash
go run ./cmd/glab-docs --search-root . --component-prefix gitlab.com/m13tlabs/glab-docs
```

## Strict mode

`-x` fails generation when an input, variable or job has no description (native or comment).
`-y` takes allowed-undocumented paths (`inputs.foo`, `variables.BAR`, `jobs.deploy`, or a bare
name); `-z` takes
regexps. Both are repeatable.

## GitLab CI/CD component

To run `glab-docs` inside a GitLab pipeline, include the bundled component instead of writing
a job by hand:

```yaml
include:
  - component: $CI_SERVER_FQDN/m13tlabs/glab-docs/glab-docs@<version>
    inputs:
      search-root: templates
      # mode: check   # default - fails the job when committed docs are stale
      # mode: generate
```

See [templates/README.md](templates/README.md) for every input. The component runs the
`m13tlabs/glab-docs` image; `check` mode `git diff --exit-code`s the regenerated output.

## Pre-commit

```yaml
repos:
  - repo: https://github.com/m13tLabs/glab-docs
    rev: ""
    hooks:
      - id: glab-docs            # needs glab-docs on PATH
      # - id: glab-docs-built    # builds from source
      # - id: glab-docs-container # runs m13tlabs/glab-docs:latest
        args: [--search-root=templates]
```

## Configuration reference

Every flag is also settable via a `GLAB_DOCS_`-prefixed env var (dashes → underscores).

| Flag | Default | Purpose |
|------|---------|---------|
| `-c, --search-root` | `.` | root to walk |
| `-p, --search-pattern` | see above | globs identifying CI YAML files |
| `-o, --output-file` | `README.md` | output path relative to each file's directory |
| `-t, --template-files` | `README.md.gotmpl` | extra templates |
| `--component-prefix` | _(empty)_ | include-snippet address prefix |
| `--gitlab-server-url` | _(empty)_ | resolves a literal `$CI_SERVER_FQDN` in documented `component:` includes and links them to their source project |
| `--include-details` | `true` | fill the Includes table's Description column (variables / jobs each include adds) |
| `--gitlab-token` | _(empty)_ | token for fetching `project:`/`component:` includes; falls back to `$CI_JOB_TOKEN` |
| `-s, --sort-values-order` | `alphanum` | `alphanum` or `file` |
| `-i, --ignore-file` | `.glabdocsignore` | ignore file name |
| `-d, --dry-run` | `false` | print instead of write |
| `-x / -y / -z` | off | strict mode + allowlists |
| `--skip-version-footer` | `false` | drop the footer |
| `--combined-title` | `Glab Pipeline Docs` | H1 above a combined multi-template README (empty to omit) |
| `-g, --component-to-generate` | _(all)_ | limit to specific files |
