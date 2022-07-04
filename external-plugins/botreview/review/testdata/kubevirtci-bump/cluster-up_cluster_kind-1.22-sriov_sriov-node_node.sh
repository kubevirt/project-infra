diff --git a/cluster-up/cluster/kind-1.22-sriov/sriov-node/node.sh b/cluster-up/cluster/kind-1.22-sriov/sriov-node/node.sh
index 61eb40244..8d1a997c2 100644
--- a/cluster-up/cluster/kind-1.22-sriov/sriov-node/node.sh
+++ b/cluster-up/cluster/kind-1.22-sriov/sriov-node/node.sh
@@ -56,7 +56,7 @@ function node::configure_sriov_pfs() {
     pfs_in_use+=( $pf_name )
 
     # KIND mounts sysfs as read-only by default, remount as R/W"
-    node_exec="docker exec $node"
+    node_exec="${CRI_BIN} exec $node"
     $node_exec mount -o remount,rw /sys
 
     ls_node_dev_vfio="${node_exec} ls -la -Z /dev/vfio"
@@ -81,15 +81,15 @@ function node::configure_sriov_vfs() {
     local -r config_vf_script=$(basename "$CONFIGURE_VFS_SCRIPT_PATH")
 
     for node in "${nodes_array[@]}"; do
-      docker cp "$CONFIGURE_VFS_SCRIPT_PATH" "$node:/"
-      docker exec "$node" bash -c "DRIVER=$driver DRIVER_KMODULE=$driver_kmodule VFS_COUNT=$vfs_count ./$config_vf_script"
-      docker exec "$node" ls -la -Z /dev/vfio
+      ${CRI_BIN} cp "$CONFIGURE_VFS_SCRIPT_PATH" "$node:/"
+      ${CRI_BIN} exec "$node" bash -c "DRIVER=$driver DRIVER_KMODULE=$driver_kmodule VFS_COUNT=$vfs_count ./$config_vf_script"
+      ${CRI_BIN} exec "$node" ls -la -Z /dev/vfio
     done
 }
 
 function prepare_node_netns() {
   local -r node_name=$1
-  local -r node_pid=$(docker inspect -f '{{.State.Pid}}' "$node_name")
+  local -r node_pid=$($CRI_BIN inspect -f '{{.State.Pid}}' "$node_name")
 
   # Docker does not create the required symlink for a container netns
   # it perverts iplink from learning that container netns.
@@ -112,7 +112,7 @@ function move_pf_to_node_netns() {
 
 function node::total_vfs_count() {
   local -r node_name=$1
-  local -r node_pid=$(docker inspect -f '{{.State.Pid}}' "$node_name")
+  local -r node_pid=$($CRI_BIN inspect -f '{{.State.Pid}}' "$node_name")
   local -r pfs_sriov_numvfs=( $(cat /proc/$node_pid/root/sys/class/net/*/device/sriov_numvfs) )
   local total_vfs_on_node=0
 
