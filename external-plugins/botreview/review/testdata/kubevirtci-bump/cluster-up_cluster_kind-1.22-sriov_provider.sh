diff --git a/cluster-up/cluster/kind-1.22-sriov/provider.sh b/cluster-up/cluster/kind-1.22-sriov/provider.sh
index dcaf1b041..9faf391c7 100755
--- a/cluster-up/cluster/kind-1.22-sriov/provider.sh
+++ b/cluster-up/cluster/kind-1.22-sriov/provider.sh
@@ -29,9 +29,9 @@ function print_sriov_data() {
         if [[ ! "$node" =~ .*"control-plane".* ]]; then
             echo "Node: $node"
             echo "VFs:"
-            docker exec $node bash -c "ls -l /sys/class/net/*/device/virtfn*"
+            ${CRI_BIN} exec $node bash -c "ls -l /sys/class/net/*/device/virtfn*"
             echo "PFs PCI Addresses:"
-            docker exec $node bash -c "grep PCI_SLOT_NAME /sys/class/net/*/device/uevent"
+            ${CRI_BIN} exec $node bash -c "grep PCI_SLOT_NAME /sys/class/net/*/device/uevent"
         fi
     done
 }
@@ -51,7 +51,7 @@ function configure_registry_proxy() {
 function up() {
     # print hardware info for easier debugging based on logs
     echo 'Available NICs'
-    docker run --rm --cap-add=SYS_RAWIO quay.io/phoracek/lspci@sha256:0f3cacf7098202ef284308c64e3fc0ba441871a846022bb87d65ff130c79adb1 sh -c "lspci | egrep -i 'network|ethernet'"
+    ${CRI_BIN} run --rm --cap-add=SYS_RAWIO quay.io/phoracek/lspci@sha256:0f3cacf7098202ef284308c64e3fc0ba441871a846022bb87d65ff130c79adb1 sh -c "lspci | egrep -i 'network|ethernet'"
     echo ""
 
     cp $KIND_MANIFESTS_DIR/kind.yaml ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
