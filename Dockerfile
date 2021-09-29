FROM alpine:3.13.6

ENV OPERATOR=/usr/local/bin/codebase-operator \
    USER_UID=1001 \
    USER_NAME=codebase-operator \
    HOME=/home/codebase-operator \
    SSH_KNOWN_HOSTS=/home/codebase-operator/.ssh/known_hosts

RUN apk add --no-cache ca-certificates==20191127-r5 \
                       openssh-client==8.4_p1-r4 \
                       git==2.30.2-r0

COPY build/bin /usr/local/bin
COPY build/templates /usr/local/bin/templates
COPY build/pipelines /usr/local/bin/pipelines
COPY build/configs /usr/local/bin/configs
COPY build/img /usr/local/bin/img

RUN chgrp -R 0 /usr/local/bin/templates /usr/local/bin/pipelines && \
    chmod -R g=u /usr/local/bin/templates /usr/local/bin/pipelines

RUN  chmod u+x /usr/local/bin/user_setup && \
     chmod ugo+x /usr/local/bin/entrypoint && \
     /usr/local/bin/user_setup

# install operator binary
COPY go-binary ${OPERATOR}

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
