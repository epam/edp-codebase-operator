FROM alpine:3.16.2

ENV ASSETS_DIR=/usr/local/bin \
    HOME=/home/codebase-operator \
    OPERATOR=/usr/local/bin/codebase-operator \
    SSH_KNOWN_HOSTS=/home/codebase-operator/.ssh/known_hosts \
    USER_NAME=codebase-operator \
    USER_UID=1001

RUN apk add --no-cache ca-certificates=20220614-r0 \
                       openssh-client==9.0_p1-r2 \
                       git==2.36.3-r0

RUN adduser -h ${HOME} -s /bin/ash -D -u ${USER_UID} codebase-operator

COPY build/bin ${ASSETS_DIR}
COPY --chmod=0775 build/templates ${ASSETS_DIR}/templates
COPY build/configs ${ASSETS_DIR}/configs
COPY build/img ${ASSETS_DIR}/img

RUN  chmod u+x ${ASSETS_DIR}/user_setup && \
     chmod ugo+x ${ASSETS_DIR}/entrypoint && \
     ${ASSETS_DIR}/user_setup

# install operator binary
COPY ./dist/go-binary ${OPERATOR}

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
