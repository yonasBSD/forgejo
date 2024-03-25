FROM code.forgejo.org/oci/alpine:3.19@sha256:6457d53fb065d6f250e1504b9bc42d5b6c65941d57532c072d929dd0628977d0
ARG RELEASE_VERSION=unkown
RUN mkdir -p /app/gitea
RUN ( echo '#!/bin/sh' ; echo "echo forgejo v$RELEASE_VERSION" ) > /app/gitea/gitea ; chmod +x /app/gitea/gitea
