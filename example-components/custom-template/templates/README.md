# markdown-lint

Lints the repository's Markdown files with markdownlint-cli2.

> This README is rendered from `README.md.gotmpl`. Edit the template, not this file.

## Usage

```yaml
include:
  - component: gitlab.com/m13tlabs/glab-docs/markdown-lint@<version>
    inputs:
      globs: ["**/*.md"]
      stage: lint
```

## Configuration

The rules come from a `.markdownlint-cli2.yaml` at the repository root. Without one, the
markdownlint defaults apply.

## Inputs

| Input | Type | Default | Options | Description |
|-------|------|---------|---------|-------------|
| globs | array | `["**/*.md"]` |  | Globs of the files to lint. |
| stage | string | `lint` |  | Pipeline stage the job runs in. |

## Jobs

| Job | Stage | When | Needs | Description |
|-----|-------|------|-------|-------------|
| `markdown-lint` | `$[[ inputs.stage ]]` |  |  | Runs markdownlint-cli2 over the configured globs. |

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `MD013/line-length` on long tables | disable `MD013` for tables (`tables: false`) |
| files in `node_modules/` are linted | add `"#node_modules"` to `globs` |

