FROM alpine:3.13.7

ENV ASSETS_DIR=/usr/local/bin \
    HOME=/home/codebase-operator \
    OPERATOR=/usr/local/bin/codebase-operator \
    SSH_KNOWN_HOSTS=/home/codebase-operator/.ssh/known_hosts \
    USER_NAME=codebase-operator \
    USER_UID=1001

RUN apk add --no-cache ca-certificates==20211220-r0 \
                       openssh-client==8.4_p1-r4 \
                       git==2.30.2-r0

COPY build/bin ${ASSETS_DIR}
COPY build/templates ${ASSETS_DIR}/templates
COPY build/configs ${ASSETS_DIR}/configs
COPY build/img ${ASSETS_DIR}/img

RUN chgrp -R 0 ${ASSETS_DIR}/templates && \
    chmod -R g=u ${ASSETS_DIR}/templates

RUN  chmod u+x ${ASSETS_DIR}/user_setup && \
     chmod ugo+x ${ASSETS_DIR}/entrypoint && \
     ${ASSETS_DIR}/user_setup

RUN adduser -h /home/codebase-operator -s /bin/ash -D -u 1001 codebase-operator

# install operator binary
COPY ./dist/go-binary ${OPERATOR}

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
