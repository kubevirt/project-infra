diff --git a/cluster-up/cluster/kind/common.sh b/cluster-up/cluster/kind/common.sh
index d7e6ecc67..1bbedb63d 100755
--- a/cluster-up/cluster/kind/common.sh
+++ b/cluster-up/cluster/kind/common.sh
@@ -2,6 +2,12 @@
 
 set -e
 
+function detect_cri() {
+    if podman ps >/dev/null 2>&1; then echo podman; elif docker ps >/dev/null 2>&1; then echo docker; fi
+}
+
+export CRI_BIN=${CRI_BIN:-$(detect_cri)}
+
 # check CPU arch
 PLATFORM=$(uname -m)
 case ${PLATFORM} in
@@ -20,9 +26,9 @@ aarch64* | arm64*)
     ;;
 esac
 
-NODE_CMD="docker exec -it -d "
+NODE_CMD="${CRI_BIN} exec -it -d "
 export KIND_MANIFESTS_DIR="${KUBEVIRTCI_PATH}/cluster/kind/manifests"
-export KIND_NODE_CLI="docker exec -it "
+export KIND_NODE_CLI="${CRI_BIN} exec -it "
 export KUBEVIRTCI_PATH
 export KUBEVIRTCI_CONFIG_PATH
 KIND_DEFAULT_NETWORK="kind"
@@ -44,7 +50,7 @@ function _wait_kind_up {
     else
         selector="control-plane"
     fi
-    while [ -z "$(docker exec --privileged ${CLUSTER_NAME}-control-plane kubectl --kubeconfig=/etc/kubernetes/admin.conf get nodes --selector=node-role.kubernetes.io/${selector} -o=jsonpath='{.items..status.conditions[-1:].status}' | grep True)" ]; do
+    while [ -z "$(${CRI_BIN} exec --privileged ${CLUSTER_NAME}-control-plane kubectl --kubeconfig=/etc/kubernetes/admin.conf get nodes --selector=node-role.kubernetes.io/${selector} -o=jsonpath='{.items..status.conditions[-1:].status}' | grep True)" ]; do
         echo "Waiting for kind to be ready ..."
         sleep 10
     done
@@ -83,18 +89,18 @@ function _insecure-registry-config-cmd() {
 
 # this works since the nodes use the same names as containers
 function _ssh_into_node() {
-    docker exec -it "$1" bash
+    ${CRI_BIN} exec -it "$1" bash
 }
 
 function _run_registry() {
     local -r network=${1}
 
-    until [ -z "$(docker ps -a | grep $REGISTRY_NAME)" ]; do
-        docker stop $REGISTRY_NAME || true
-        docker rm $REGISTRY_NAME || true
+    until [ -z "$($CRI_BIN ps -a | grep $REGISTRY_NAME)" ]; do
+        ${CRI_BIN} stop $REGISTRY_NAME || true
+        ${CRI_BIN} rm $REGISTRY_NAME || true
         sleep 5
     done
-    docker run -d --network=${network} -p $HOST_PORT:5000  --restart=always --name $REGISTRY_NAME quay.io/kubevirtci/library-registry:2.7.1
+    ${CRI_BIN} run -d --network=${network} -p $HOST_PORT:5000  --restart=always --name $REGISTRY_NAME quay.io/kubevirtci/library-registry:2.7.1
 
 }
 
@@ -103,7 +109,7 @@ function _configure_registry_on_node() {
     local -r network=${2}
 
     _configure-insecure-registry-and-reload "${NODE_CMD} ${node} bash -c"
-    ${NODE_CMD} ${node} sh -c "echo $(docker inspect --format "{{.NetworkSettings.Networks.${network}.IPAddress }}" $REGISTRY_NAME)'\t'registry >> /etc/hosts"
+    ${NODE_CMD} ${node} sh -c "echo $(${CRI_BIN} inspect --format "{{.NetworkSettings.Networks.${network}.IPAddress }}" $REGISTRY_NAME)'\t'registry >> /etc/hosts"
 }
 
 function _install_cnis {
@@ -120,8 +126,8 @@ function _install_cni_plugins {
     fi
 
     for node in $(_get_nodes | awk '{print $1}'); do
-        docker cp "${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/$CNI_ARCHIVE" $node:/
-        docker exec $node /bin/sh -c "tar xf $CNI_ARCHIVE -C /opt/cni/bin"
+        ${CRI_BIN} cp "${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/$CNI_ARCHIVE" $node:/
+        ${CRI_BIN} exec $node /bin/sh -c "tar xf $CNI_ARCHIVE -C /opt/cni/bin"
     done
 }
 
@@ -173,24 +179,20 @@ function _fix_node_labels() {
     done
 }
 
-function _get_cri_bridge_mtu() {
-  docker network inspect -f '{{index .Options "com.docker.network.driver.mtu"}}' bridge
-}
-
 function setup_kind() {
     $KIND --loglevel debug create cluster --retain --name=${CLUSTER_NAME} --config=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml --image=$KIND_NODE_IMAGE
     $KIND get kubeconfig --name=${CLUSTER_NAME} > ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
 
-    docker cp ${CLUSTER_NAME}-control-plane:$KUBECTL_PATH ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
+    ${CRI_BIN} cp ${CLUSTER_NAME}-control-plane:$KUBECTL_PATH ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
     chmod u+x ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
 
     if [ $KUBEVIRT_WITH_KIND_ETCD_IN_MEMORY == "true" ]; then
         for node in $(_get_nodes | awk '{print $1}' | grep control-plane); do
             echo "[$node] Checking KIND cluster etcd data is mounted to RAM: $ETCD_IN_MEMORY_DATA_DIR"
-            docker exec $node df -h $(dirname $ETCD_IN_MEMORY_DATA_DIR) | grep -P '(tmpfs|ramfs)'
+            ${CRI_BIN} exec $node df -h $(dirname $ETCD_IN_MEMORY_DATA_DIR) | grep -P '(tmpfs|ramfs)'
             [ $(echo $?) != 0 ] && echo "[$node] etcd data directory is not mounted to RAM" && return 1
 
-            docker exec $node du -h $ETCD_IN_MEMORY_DATA_DIR
+            ${CRI_BIN} exec $node du -h $ETCD_IN_MEMORY_DATA_DIR
             [ $(echo $?) != 0 ] && echo "[$node] Failed to check etcd data directory" && return 1
         done
     fi
@@ -306,6 +308,6 @@ function down() {
     fi
     # On CI, avoid failing an entire test run just because of a deletion error
     $KIND delete cluster --name=${CLUSTER_NAME} || [ "$CI" = "true" ]
-    docker rm -f $REGISTRY_NAME >> /dev/null
+    ${CRI_BIN} rm -f $REGISTRY_NAME >> /dev/null
     rm -f ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
 }
