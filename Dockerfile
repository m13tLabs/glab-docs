FROM golang:1.27-alpine AS build

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

ARG BUILD_DATE
ARG APP_VERSION

LABEL org.opencontainers.image.authors='Martin Reinhardt (martin@m13t.de)' \
    org.opencontainers.image.created=$BUILD_DATE \
    org.opencontainers.image.version=$APP_VERSION \
    org.opencontainers.image.url='https://hub.docker.com/r/m13t/openjarvis' \
    org.opencontainers.image.documentation='https://github.com/m13tLabs/openjarvis' \
    org.opencontainers.image.source='https://github.com/m13tLabs/openjarvis.git' \
    org.opencontainers.image.licenses='MIT'

COPY --from=build /out/glab-docs /usr/bin/glab-docs

WORKDIR /glab-docs

ENTRYPOINT ["glab-docs"]
