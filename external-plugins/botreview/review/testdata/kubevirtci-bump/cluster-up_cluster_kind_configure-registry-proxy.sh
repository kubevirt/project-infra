diff --git a/cluster-up/cluster/kind/configure-registry-proxy.sh b/cluster-up/cluster/kind/configure-registry-proxy.sh
index f68cbfe0f..5b1a5abd2 100755
--- a/cluster-up/cluster/kind/configure-registry-proxy.sh
+++ b/cluster-up/cluster/kind/configure-registry-proxy.sh
@@ -20,7 +20,7 @@
 
 set -ex
 
-CRI=${CRI:-docker}
+CRI_BIN=${CRI_BIN:-docker}
 
 KIND_BIN="${KIND_BIN:-./kind}"
 PROXY_HOSTNAME="${PROXY_HOSTNAME:-docker-registry-proxy}"
@@ -29,7 +29,7 @@ CLUSTER_NAME="${CLUSTER_NAME:-sriov}"
 SETUP_URL="http://${PROXY_HOSTNAME}:3128/setup/systemd"
 pids=""
 for node in $($KIND_BIN get nodes --name "$CLUSTER_NAME"); do
-   $CRI exec "$node" sh -c "\
+   $CRI_BIN exec "$node" sh -c "\
       curl $SETUP_URL | \
       sed s/docker\.service/containerd\.service/g | \
       sed '/Environment/ s/$/ \"NO_PROXY=127.0.0.0\/8,10.0.0.0\/8,172.16.0.0\/12,192.168.0.0\/16\"/' | \
