FROM alpine:3.20

RUN apk add --no-cache git

COPY glab-docs /usr/bin/

WORKDIR /glab-docs

ENTRYPOINT ["glab-docs"]
