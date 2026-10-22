# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /

ARG TARGETARCH

ENV ASSETS_DIR=/usr/local/bin \
    SSH_KNOWN_HOSTS=/etc/codebase-operator/ssh/ssh_known_hosts

COPY --chmod=0755 build/templates ${ASSETS_DIR}/templates
COPY --chmod=0755 build/configs ${ASSETS_DIR}/configs
COPY ./dist/manager-${TARGETARCH} /manager

USER 65532:65532

ENTRYPOINT ["/manager"]
