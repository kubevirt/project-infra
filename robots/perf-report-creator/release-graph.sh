OUTPUT_DIR=${1}

for dir in ${OUTPUT_DIR}/results/*; do
    if [ -d "$dir" ] && [[ ! "$dir" =~ "cluster-100-density-test" && ! "$dir" =~ "kwok" ]]; then
        dir_name=$(basename $dir)

        perf-report-creator weekly-report --output-dir=${OUTPUT_DIR}/weekly \
          --results-dir=${OUTPUT_DIR}/results/${dir_name} \
          --vmi-metrics-list=vmiCreationToRunningSecondsP50,vmiCreationToRunningSecondsP95,LIST-virtualmachineinstances-count,LIST-pods-count,LIST-nodes-count,LIST-virtualmachineinstancemigrations-count,LIST-endpoints-count,GET-virtualmachineinstances-count,GET-pods-count,GET-nodes-count,GET-virtualmachineinstancemigrations-count,GET-endpoints-count,CREATE-virtualmachineinstances-count,CREATE-pods-count,CREATE-nodes-count,CREATE-virtualmachineinstancemigrations-count,CREATE-endpoints-count,PATCH-virtualmachineinstances-count,PATCH-pods-count,PATCH-nodes-count,PATCH-virtualmachineinstancemigrations-count,PATCH-endpoints-count,UPDATE-virtualmachineinstances-count,UPDATE-pods-count,UPDATE-nodes-count,UPDATE-virtualmachineinstancemigrations-count,UPDATE-endpoints-count,avgVirtAPIMemoryUsageInMB,minVirtAPIMemoryUsageInMB,maxVirtAPIMemoryUsageInMB,avgVirtAPICPUUsage,avgVirtControllerMemoryUsageInMB,minVirtControllerMemoryUsageInMB,maxVirtControllerMemoryUsageInMB,avgVirtControllerCPUUsage,avgVirtHandlerMemoryUsageInMB,minVirtHandlerMemoryUsageInMB,maxVirtHandlerMemoryUsageInMB,avgVirtHandlerCPUUsage \
          --vm-metrics-list vmiCreationToRunningSecondsP50,vmiCreationToRunningSecondsP95,LIST-virtualmachineinstances-count,WATCH-virtualmachineinstances-count,LIST-pods-count,WATCH-pods-count,LIST-nodes-count,LIST-virtualmachineinstancemigrations-count,WATCH-virtualmachineinstancemigrations-count,LIST-endpoints-count,GET-virtualmachineinstances-count,GET-pods-count,GET-nodes-count,GET-virtualmachineinstancemigrations-count,GET-endpoints-count,CREATE-virtualmachineinstances-count,CREATE-pods-count,CREATE-nodes-count,CREATE-virtualmachineinstancemigrations-count,CREATE-endpoints-count,PATCH-virtualmachineinstances-count,PATCH-pods-count,PATCH-nodes-count,PATCH-virtualmachineinstancemigrations-count,PATCH-endpoints-count,UPDATE-virtualmachineinstances-count,UPDATE-pods-count,UPDATE-nodes-count,UPDATE-virtualmachineinstancemigrations-count,UPDATE-endpoints-count,avgVirtAPIMemoryUsageInMB,minVirtAPIMemoryUsageInMB,maxVirtAPIMemoryUsageInMB,avgVirtAPICPUUsage,avgVirtControllerMemoryUsageInMB,minVirtControllerMemoryUsageInMB,maxVirtControllerMemoryUsageInMB,avgVirtControllerCPUUsage,avgVirtHandlerMemoryUsageInMB,minVirtHandlerMemoryUsageInMB,maxVirtHandlerMemoryUsageInMB,avgVirtHandlerCPUUsage
    fi
done

# plot sig-performance 1.31 weekly vmi
perf-report-creator weekly-graph \
  --resource=vmi \
  --weekly-reports-dir=${OUTPUT_DIR}/weekly \
  --plotly-html=true \
  --is-during-release=true \
  --metrics-list=vmiCreationToRunningSecondsP50,vmiCreationToRunningSecondsP95,LIST-virtualmachineinstances-count,LIST-pods-count,LIST-nodes-count,LIST-virtualmachineinstancemigrations-count,LIST-endpoints-count,GET-virtualmachineinstances-count,GET-pods-count,GET-nodes-count,GET-virtualmachineinstancemigrations-count,GET-endpoints-count,CREATE-virtualmachineinstances-count,CREATE-pods-count,CREATE-nodes-count,CREATE-virtualmachineinstancemigrations-count,CREATE-endpoints-count,PATCH-virtualmachineinstances-count,PATCH-pods-count,PATCH-nodes-count,PATCH-virtualmachineinstancemigrations-count,PATCH-endpoints-count,UPDATE-virtualmachineinstances-count,UPDATE-pods-count,UPDATE-nodes-count,UPDATE-virtualmachineinstancemigrations-count,UPDATE-endpoints-count,avgVirtAPIMemoryUsageInMB,minVirtAPIMemoryUsageInMB,maxVirtAPIMemoryUsageInMB,avgVirtAPICPUUsage,avgVirtControllerMemoryUsageInMB,minVirtControllerMemoryUsageInMB,maxVirtControllerMemoryUsageInMB,avgVirtControllerCPUUsage,avgVirtHandlerMemoryUsageInMB,minVirtHandlerMemoryUsageInMB,maxVirtHandlerMemoryUsageInMB,avgVirtHandlerCPUUsage

## plot sig-performance 1.31 weekly vm
perf-report-creator weekly-graph \
  --resource=vm \
  --weekly-reports-dir=${OUTPUT_DIR}/weekly \
  --plotly-html=true \
  --is-during-release=true \
  --metrics-list=vmiCreationToRunningSecondsP50,vmiCreationToRunningSecondsP95,LIST-virtualmachineinstances-count,WATCH-virtualmachineinstances-count,LIST-pods-count,WATCH-pods-count,LIST-nodes-count,LIST-virtualmachineinstancemigrations-count,WATCH-virtualmachineinstancemigrations-count,LIST-endpoints-count,GET-virtualmachineinstances-count,GET-pods-count,GET-nodes-count,GET-virtualmachineinstancemigrations-count,GET-endpoints-count,CREATE-virtualmachineinstances-count,CREATE-pods-count,CREATE-nodes-count,CREATE-virtualmachineinstancemigrations-count,CREATE-endpoints-count,PATCH-virtualmachineinstances-count,PATCH-pods-count,PATCH-nodes-count,PATCH-virtualmachineinstancemigrations-count,PATCH-endpoints-count,UPDATE-virtualmachineinstances-count,UPDATE-pods-count,UPDATE-nodes-count,UPDATE-virtualmachineinstancemigrations-count,UPDATE-endpoints-count,avgVirtAPIMemoryUsageInMB,minVirtAPIMemoryUsageInMB,maxVirtAPIMemoryUsageInMB,avgVirtAPICPUUsage,avgVirtControllerMemoryUsageInMB,minVirtControllerMemoryUsageInMB,maxVirtControllerMemoryUsageInMB,avgVirtControllerCPUUsage,avgVirtHandlerMemoryUsageInMB,minVirtHandlerMemoryUsageInMB,maxVirtHandlerMemoryUsageInMB,avgVirtHandlerCPUUsage
