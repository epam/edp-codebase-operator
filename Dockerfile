FROM alpine:3.18.9

ENV ASSETS_DIR=/usr/local/bin \
    HOME=/home/codebase-operator \
    OPERATOR=/usr/local/bin/codebase-operator \
    SSH_KNOWN_HOSTS=/home/codebase-operator/.ssh/known_hosts \
    USER_NAME=codebase-operator \
    USER_UID=1001

RUN apk add --no-cache ca-certificates=20241121-r1 \
                       openssh-client==9.3_p2-r3  \
                       git==2.40.4-r0

RUN adduser -h ${HOME} -s /bin/ash -D -u ${USER_UID} codebase-operator

COPY build/bin ${ASSETS_DIR}
COPY --chmod=0775 build/templates ${ASSETS_DIR}/templates
COPY build/configs ${ASSETS_DIR}/configs

RUN  chmod u+x ${ASSETS_DIR}/user_setup && \
     chmod ugo+x ${ASSETS_DIR}/entrypoint && \
     ${ASSETS_DIR}/user_setup

# install operator binary
COPY ./dist/manager ${OPERATOR}

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
