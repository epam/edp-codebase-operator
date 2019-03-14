FROM golang:1.10.3-alpine3.8

ENV OPERATOR=/usr/local/bin/business-app-handler-controller \
    USER_UID=1001 \
    USER_NAME=business-app-handler-controller \
    HOME=/home/business-app-handler-controller

# install operator binary
COPY business-app-handler-controller ${OPERATOR}

RUN apk add --no-cache ca-certificates

COPY bin /usr/local/bin
COPY templates /usr/local/bin/templates
COPY pipelines /usr/local/bin/pipelines

RUN chgrp -R 0 /usr/local/bin/templates /usr/local/bin/pipelines && \
    chmod -R g=u /usr/local/bin/templates /usr/local/bin/pipelines

RUN  chmod u+x /usr/local/bin/user_setup && \
     chmod ugo+x /usr/local/bin/entrypoint && \
     /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_NAME}
