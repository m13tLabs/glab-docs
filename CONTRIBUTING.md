# Contributing to glab-docs

## Build

```bash
make glab-docs        # ./glab-docs
# or
go build ./cmd/glab-docs
```

Install from source:

```bash
go install github.com/m13tLabs/glab-docs/cmd/glab-docs@latest
```

## Testing

```bash
go vet ./...
go test ./...
```

The `example-components/` fixtures carry committed `README.md` files. If your change affects
rendering, regenerate them and commit the result:

```bash
go run ./cmd/glab-docs --search-root . --component-prefix gitlab.com/m13tlabs/glab-docs
git diff example-components templates
```

### GitHub Actions

You may use [act](https://github.com/nektos/act) to run the workflows locally, e.g.
`act -j build`.
