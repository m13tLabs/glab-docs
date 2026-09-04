FROM alpine:3.24

COPY glab-docs /usr/bin/

WORKDIR /glab-docs

ENTRYPOINT ["glab-docs"]
