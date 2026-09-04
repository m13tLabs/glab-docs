<p align="center">
  <img src=".github/assets/logo.svg" alt="glab-docs" width="440">
</p>

glab-docs
=========

`glab-docs` auto-generates Markdown documentation for **GitLab CI/CD components and pipelines**,
the way [helm-docs](https://github.com/norwoodj/helm-docs) does for Helm charts. It reads a CI
YAML file's `spec:inputs:` header, its top-level `variables:` block and its `include:` list and
renders a `README.md` with tables for each.

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

### The usage snippet

`--component-prefix <host>/<group>/<project>` produces a real
`include: - component: <prefix>/<name>@<version>` block. Without it a
`$CI_SERVER_FQDN/<path-to-project>/<name>@<version>` placeholder is used. The snippet is only
emitted for files that declare `spec:inputs:`.

## Documenting inputs and variables

`spec:inputs:` entries are documented from their native fields — `description`, `default`,
`type` (`string` / `number` / `boolean` / `array`, otherwise inferred from the default),
`options`, `regex`. An input with no `default:` is shown as `_required_`.

`variables:` entries are documented from their value, or from the extended
`{ value, description, options }` form.

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
| `pipeline.includesSection` / `pipeline.includesTable` | the `include:` table |
| `glab-docs.versionFooter` | autogeneration footer (suppress with `--skip-version-footer`) |

## Strict mode

`-x` fails generation when an input or variable has no description (native or comment). `-y`
takes allowed-undocumented paths (`inputs.foo`, `variables.BAR`, or a bare name); `-z` takes
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
| `-s, --sort-values-order` | `alphanum` | `alphanum` or `file` |
| `-i, --ignore-file` | `.glabdocsignore` | ignore file name |
| `-d, --dry-run` | `false` | print instead of write |
| `-x / -y / -z` | off | strict mode + allowlists |
| `--skip-version-footer` | `false` | drop the footer |
| `-g, --component-to-generate` | _(all)_ | limit to specific files |
