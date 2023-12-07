diff --git a/cluster-up/hack/common.sh b/cluster-up/hack/common.sh
index 3b2e31000..4802c83a8 100644
--- a/cluster-up/hack/common.sh
+++ b/cluster-up/hack/common.sh
@@ -43,4 +43,4 @@ provider_prefix=${JOB_NAME:-${KUBEVIRT_PROVIDER}}${EXECUTOR_NUMBER}
 job_prefix=${JOB_NAME:-kubevirt}${EXECUTOR_NUMBER}
 
 mkdir -p $KUBEVIRTCI_CONFIG_PATH/$KUBEVIRT_PROVIDER
-KUBEVIRTCI_TAG=2206291207-35b9c64
+KUBEVIRTCI_TAG=2207050817-da6af04
