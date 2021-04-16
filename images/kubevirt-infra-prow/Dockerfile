ARG BASEIMAGE
FROM ${BASEIMAGE}
ARG POD_UTILITY
ARG ARCH

ENV CMD=$POD_UTILITY
RUN apk add --no-cache git openssh ca-certificates && update-ca-certificates

COPY prow/${POD_UTILITY}-${ARCH} /${POD_UTILITY}

ENTRYPOINT /${CMD}
