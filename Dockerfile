FROM --platform=$BUILDPLATFORM docker.io/tonistiigi/xx@sha256:0cd3f05c72d6c9b038eb135f91376ee1169ef3a330d34e418e65e2a5c2e9c0d4 AS xx

FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.21-alpine3.19@sha256:d7c6083c5400694f7a17b07c4fad8db9115c0e8e3cf62f781cb29cc890a64e8e as build-env

ARG GOPROXY
ENV GOPROXY ${GOPROXY:-direct}

ARG RELEASE_VERSION
ARG TAGS="sqlite sqlite_unlock_notify"
ENV TAGS "bindata timetzdata $TAGS"
ARG CGO_EXTRA_CFLAGS

#
# Transparently cross compile for the target platform
#
COPY --from=xx / /
ARG TARGETPLATFORM
RUN apk --no-cache add clang lld
RUN xx-apk --no-cache add gcc musl-dev
ENV CGO_ENABLED=1
RUN xx-go --wrap
#
# for go generate and binfmt to find
# without it the generate phase will fail with
# #19 25.04 modules/public/public_bindata.go:8: running "go": exit status 1
# #19 25.39 aarch64-binfmt-P: Could not open '/lib/ld-musl-aarch64.so.1': No such file or directory
# why exactly is it needed? where is binfmt involved?
#
RUN cp /*-alpine-linux-musl*/lib/ld-musl-*.so.1 /lib || true

RUN apk --no-cache add build-base git nodejs npm

COPY . ${GOPATH}/src/code.gitea.io/gitea
WORKDIR ${GOPATH}/src/code.gitea.io/gitea

RUN make clean
RUN make frontend
RUN go build contrib/environment-to-ini/environment-to-ini.go && xx-verify environment-to-ini
RUN make RELEASE_VERSION=$RELEASE_VERSION go-check generate-backend static-executable && xx-verify gitea

# Copy local files
COPY docker/root /tmp/local

# Set permissions
RUN chmod 755 /tmp/local/usr/bin/entrypoint \
              /tmp/local/usr/local/bin/gitea \
              /tmp/local/etc/s6/gitea/* \
              /tmp/local/etc/s6/openssh/* \
              /tmp/local/etc/s6/.s6-svscan/* \
              /go/src/code.gitea.io/gitea/gitea \
              /go/src/code.gitea.io/gitea/environment-to-ini
RUN chmod 644 /go/src/code.gitea.io/gitea/contrib/autocompletion/bash_autocomplete

FROM docker.io/library/alpine:3.19@sha256:c5b1261d6d3e43071626931fc004f70149baeba2c8ec672bd4f27761f8e1ad6b
LABEL maintainer="contact@forgejo.org"

EXPOSE 22 3000

RUN apk --no-cache add \
    bash \
    ca-certificates \
    curl \
    gettext \
    git \
    linux-pam \
    openssh \
    s6 \
    sqlite \
    su-exec \
    gnupg \
    && rm -rf /var/cache/apk/*

RUN addgroup \
    -S -g 1000 \
    git && \
  adduser \
    -S -H -D \
    -h /data/git \
    -s /bin/bash \
    -u 1000 \
    -G git \
    git && \
  echo "git:*" | chpasswd -e

ENV USER git
ENV GITEA_CUSTOM /data/gitea

VOLUME ["/data"]

ENTRYPOINT ["/usr/bin/entrypoint"]
CMD ["/bin/s6-svscan", "/etc/s6"]

COPY --from=build-env /tmp/local /
RUN cd /usr/local/bin ; ln -s gitea forgejo
COPY --from=build-env /go/src/code.gitea.io/gitea/gitea /app/gitea/gitea
COPY --from=build-env /go/src/code.gitea.io/gitea/environment-to-ini /usr/local/bin/environment-to-ini
COPY --from=build-env /go/src/code.gitea.io/gitea/contrib/autocompletion/bash_autocomplete /etc/profile.d/gitea_bash_autocomplete.sh
