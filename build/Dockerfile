FROM golang:1.10.3-alpine3.8

ENV OPERATOR=/usr/local/bin/codebase-operator \
    USER_UID=1001 \
    USER_NAME=codebase-operator \
    HOME=/home/codebase-operator \
    SSH_KNOWN_HOSTS=/home/codebase-operator/.ssh/known_hosts


# install operator binary
COPY codebase-operator ${OPERATOR}

RUN apk add --no-cache ca-certificates openssh-client git

COPY build/bin /usr/local/bin
COPY build/templates /usr/local/bin/templates
COPY build/pipelines /usr/local/bin/pipelines
COPY build/configs /usr/local/bin/configs

RUN chgrp -R 0 /usr/local/bin/templates /usr/local/bin/pipelines && \
    chmod -R g=u /usr/local/bin/templates /usr/local/bin/pipelines

RUN  chmod u+x /usr/local/bin/user_setup && \
     chmod ugo+x /usr/local/bin/entrypoint && \
     /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
