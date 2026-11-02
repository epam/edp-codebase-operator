FROM alpine:3.22.7

ARG TARGETARCH

ENV ASSETS_DIR=/usr/local/bin \
    HOME=/home/codebase-operator \
    OPERATOR=/usr/local/bin/codebase-operator \
    SSH_KNOWN_HOSTS=/etc/codebase-operator/ssh/ssh_known_hosts \
    USER_NAME=codebase-operator \
    USER_UID=1001

# Alpine keeps only the latest build of a package per branch: pin minors (~=), not builds, and bump them with the FROM branch.
# apk upgrade also brings the base image's CA bundle up to date.
RUN apk upgrade --no-cache && \
    apk add --no-cache git~=2.49 openssh-client~=10.0

RUN adduser -h ${HOME} -s /bin/ash -D -u ${USER_UID} codebase-operator

COPY build/bin ${ASSETS_DIR}
COPY --chmod=0755 build/templates ${ASSETS_DIR}/templates
COPY --chmod=0755 build/configs ${ASSETS_DIR}/configs

RUN  chmod u+x ${ASSETS_DIR}/user_setup && \
     chmod ugo+x ${ASSETS_DIR}/entrypoint && \
     ${ASSETS_DIR}/user_setup

# install operator binary
COPY ./dist/manager-${TARGETARCH} ${OPERATOR}

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
