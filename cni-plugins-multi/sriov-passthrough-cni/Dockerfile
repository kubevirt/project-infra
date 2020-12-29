FROM centos:7.6.1810

ENV SRC_DIR=/plugin-src
ARG PLUGIN_BIN=plugin/sriov-passthrough-cni
ARG PLUGIN_CONF=plugin/90-sriov-passthrough-cni.conf
ARG INSTALL_SCRIPT=install-plugin

RUN mkdir -p $SRC_DIR
COPY $PLUGIN_BIN $SRC_DIR
COPY $PLUGIN_CONF $SRC_DIR
COPY $INSTALL_SCRIPT /bin

ENTRYPOINT ["install-plugin"]
