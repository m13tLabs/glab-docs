glab-docs:
	go build github.com/m13tLabs/glab-docs/cmd/glab-docs

install:
	go install github.com/m13tLabs/glab-docs/cmd/glab-docs

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: test
test:
	go test -v ./...

.PHONY: clean
clean:
	rm -f glab-docs

.PHONY: dist
dist:
	goreleaser release --rm-dist --snapshot --skip=sign
