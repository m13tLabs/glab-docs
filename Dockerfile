FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 go build \
    -trimpath -buildvcs=false \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/glab-docs ./cmd/glab-docs

FROM alpine:3.24

COPY --from=build /out/glab-docs /usr/bin/glab-docs

WORKDIR /glab-docs

ENTRYPOINT ["glab-docs"]
