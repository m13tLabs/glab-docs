FROM alpine:3.20

COPY glab-docs /usr/bin/

WORKDIR /glab-docs

ENTRYPOINT ["glab-docs"]
