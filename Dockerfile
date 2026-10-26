FROM alpine:3.24

ARG TARGETARCH

ENV ASSETS_DIR=/usr/local/bin \
    HOME=/home/codebase-operator \
    OPERATOR=/usr/local/bin/codebase-operator \
    SSH_KNOWN_HOSTS=/etc/codebase-operator/ssh/ssh_known_hosts \
    USER_NAME=codebase-operator \
    USER_UID=1001

# pkg/git/cli.go runs git and ssh. The ~= pins match the minors the FROM image ships: bump them with the FROM tag.
RUN apk upgrade --no-cache && \
    apk add --no-cache git~=2.54 openssh-client~=10.3

RUN adduser -h ${HOME} -s /bin/ash -D -u ${USER_UID} codebase-operator

COPY build/bin ${ASSETS_DIR}
COPY --chmod=0775 build/templates ${ASSETS_DIR}/templates
COPY build/configs ${ASSETS_DIR}/configs

RUN  chmod u+x ${ASSETS_DIR}/user_setup && \
     chmod ugo+x ${ASSETS_DIR}/entrypoint && \
     ${ASSETS_DIR}/user_setup

# install operator binary
COPY ./dist/manager-${TARGETARCH} ${OPERATOR}

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
