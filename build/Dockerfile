FROM alpine:3.8

ENV OPERATOR=/usr/local/bin/business-app-handler-controller \
    USER_UID=1001 \
    USER_NAME=business-app-handler-controller

# install operator binary
COPY business-app-handler-controller ${OPERATOR}

COPY bin /usr/local/bin
RUN  chmod u+x /usr/local/bin/user_setup && /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
