OUTPUT_DIR=${1}
KUBEVIRT_PROVIDER=${2}

# Validate KUBEVIRT_PROVIDER
if [ -z "${KUBEVIRT_PROVIDER}" ]; then
    echo "Error: KUBEVIRT_PROVIDER is required, but is empty"
    exit 1
fi

# plot sig-performance ${KUBEVIRT_PROVIDER} weekly vmi
perf-report-creator weekly-graph \
  --resource=vmi \
  --weekly-reports-dir=${OUTPUT_DIR}/weekly/periodic-kubevirt-e2e-${KUBEVIRT_PROVIDER}-sig-performance \
  --plotly-html=true \
  --since=2022-11-29 \
  --metrics-list=vmiCreationToRunningSecondsP50,vmiCreationToRunningSecondsP95,LIST-virtualmachineinstances-count,LIST-pods-count,LIST-nodes-count,LIST-virtualmachineinstancemigrations-count,LIST-endpoints-count,GET-virtualmachineinstances-count,GET-pods-count,GET-nodes-count,GET-virtualmachineinstancemigrations-count,GET-endpoints-count,CREATE-virtualmachineinstances-count,CREATE-pods-count,CREATE-nodes-count,CREATE-virtualmachineinstancemigrations-count,CREATE-endpoints-count,PATCH-virtualmachineinstances-count,PATCH-pods-count,PATCH-nodes-count,PATCH-virtualmachineinstancemigrations-count,PATCH-endpoints-count,UPDATE-virtualmachineinstances-count,UPDATE-pods-count,UPDATE-nodes-count,UPDATE-virtualmachineinstancemigrations-count,UPDATE-endpoints-count,avgVirtAPIMemoryUsageInMB,minVirtAPIMemoryUsageInMB,maxVirtAPIMemoryUsageInMB,avgVirtAPICPUUsage,avgVirtControllerMemoryUsageInMB,minVirtControllerMemoryUsageInMB,maxVirtControllerMemoryUsageInMB,avgVirtControllerCPUUsage,avgVirtHandlerMemoryUsageInMB,minVirtHandlerMemoryUsageInMB,maxVirtHandlerMemoryUsageInMB,avgVirtHandlerCPUUsage

## plot sig-performance ${KUBEVIRT_PROVIDER} weekly vm
perf-report-creator weekly-graph \
  --resource=vm \
  --weekly-reports-dir=${OUTPUT_DIR}/weekly/periodic-kubevirt-e2e-${KUBEVIRT_PROVIDER}-sig-performance \
  --plotly-html=true \
  --since=2022-11-29 \
  --metrics-list=vmiCreationToRunningSecondsP50,vmiCreationToRunningSecondsP95,LIST-virtualmachineinstances-count,WATCH-virtualmachineinstances-count,LIST-pods-count,WATCH-pods-count,LIST-nodes-count,LIST-virtualmachineinstancemigrations-count,WATCH-virtualmachineinstancemigrations-count,LIST-endpoints-count,GET-virtualmachineinstances-count,GET-pods-count,GET-nodes-count,GET-virtualmachineinstancemigrations-count,GET-endpoints-count,CREATE-virtualmachineinstances-count,CREATE-pods-count,CREATE-nodes-count,CREATE-virtualmachineinstancemigrations-count,CREATE-endpoints-count,PATCH-virtualmachineinstances-count,PATCH-pods-count,PATCH-nodes-count,PATCH-virtualmachineinstancemigrations-count,PATCH-endpoints-count,UPDATE-virtualmachineinstances-count,UPDATE-pods-count,UPDATE-nodes-count,UPDATE-virtualmachineinstancemigrations-count,UPDATE-endpoints-count,avgVirtAPIMemoryUsageInMB,minVirtAPIMemoryUsageInMB,maxVirtAPIMemoryUsageInMB,avgVirtAPICPUUsage,avgVirtControllerMemoryUsageInMB,minVirtControllerMemoryUsageInMB,maxVirtControllerMemoryUsageInMB,avgVirtControllerCPUUsage,avgVirtHandlerMemoryUsageInMB,minVirtHandlerMemoryUsageInMB,maxVirtHandlerMemoryUsageInMB,avgVirtHandlerCPUUsage
#
## plot 100-density-test results
perf-report-creator weekly-graph \
  --resource=vmi \
  --weekly-reports-dir=${OUTPUT_DIR}/weekly/periodic-kubevirt-performance-cluster-100-density-test \
  --plotly-html=true \
  --since=2022-11-29 \
  --metrics-list=vmiCreationToRunningSecondsP50,vmiCreationToRunningSecondsP95,LIST-virtualmachineinstances-count,WATCH-virtualmachineinstances-count,LIST-pods-count,WATCH-pods-count,LIST-nodes-count,LIST-virtualmachineinstancemigrations-count,WATCH-virtualmachineinstancemigrations-count,LIST-endpoints-count,GET-virtualmachineinstances-count,GET-pods-count,GET-nodes-count,GET-virtualmachineinstancemigrations-count,GET-endpoints-count,CREATE-virtualmachineinstances-count,CREATE-pods-count,CREATE-nodes-count,CREATE-virtualmachineinstancemigrations-count,CREATE-endpoints-count,PATCH-virtualmachineinstances-count,PATCH-pods-count,PATCH-nodes-count,PATCH-virtualmachineinstancemigrations-count,PATCH-endpoints-count,UPDATE-virtualmachineinstances-count,UPDATE-pods-count,UPDATE-nodes-count,UPDATE-virtualmachineinstancemigrations-count,UPDATE-endpoints-count,avgVirtAPIMemoryUsageInMB,minVirtAPIMemoryUsageInMB,maxVirtAPIMemoryUsageInMB,avgVirtAPICPUUsage,avgVirtControllerMemoryUsageInMB,minVirtControllerMemoryUsageInMB,maxVirtControllerMemoryUsageInMB,avgVirtControllerCPUUsage,avgVirtHandlerMemoryUsageInMB,minVirtHandlerMemoryUsageInMB,maxVirtHandlerMemoryUsageInMB,avgVirtHandlerCPUUsage


# for any one who's is looking for the output of the above commands, run the following loop:
#for file in $(find ${OUTPUT_DIR}/weekly -name index.html)
#do
#    echo "commit: $file"
#done
