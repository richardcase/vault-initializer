FROM scratch
ADD vault-initializer /vault-initializer
ENTRYPOINT ["/vault-initializer"]

# Build-time metadata as defined at http://label-schema.org
ARG BUILD_DATE
ARG VCS_REF
ARG VERSION
LABEL org.label-schema.build-date=$BUILD_DATE \
    org.label-schema.name="Vault Kubernetes Initializer" \
    org.label-schema.description="A Kubernetes initializer that injects secrets from Vault" \
    org.label-schema.url="https://github.com/richardcase/k8sinit" \
    org.label-schema.vcs-ref=$VCS_REF \
    org.label-schema.vcs-url="https://github.com/richardcase/k8sinit" \
    org.label-schema.vendor="Richard Case" \
    org.label-schema.version=$VERSION \
    org.label-schema.schema-version="1.0"
