OUTPUT_DIR=${1}


# plot sig-performance 1.25 weekly vmi
perf-report-creator weekly-graph \
  --resource=vmi \
  --weekly-reports-dir=${OUTPUT_DIR}/weekly/periodic-kubevirt-e2e-k8s-1.27-sig-performance \
  --plotly-html=true \
  --since=2022-11-29 \
  --metrics-list=vmiCreationToRunningSecondsP50,vmiCreationToRunningSecondsP95,LIST-virtualmachineinstances-count,LIST-pods-count,LIST-nodes-count,LIST-virtualmachineinstancemigrations-count,LIST-endpoints-count,GET-virtualmachineinstances-count,GET-pods-count,GET-nodes-count,GET-virtualmachineinstancemigrations-count,GET-endpoints-count,CREATE-virtualmachineinstances-count,CREATE-pods-count,CREATE-nodes-count,CREATE-virtualmachineinstancemigrations-count,CREATE-endpoints-count,PATCH-virtualmachineinstances-count,PATCH-pods-count,PATCH-nodes-count,PATCH-virtualmachineinstancemigrations-count,PATCH-endpoints-count,UPDATE-virtualmachineinstances-count,UPDATE-pods-count,UPDATE-nodes-count,UPDATE-virtualmachineinstancemigrations-count,UPDATE-endpoints-count

## plot 100-density-test results
perf-report-creator weekly-graph \
  --resource=vm \
  --weekly-reports-dir=${OUTPUT_DIR}/weekly/periodic-kubevirt-e2e-k8s-1.27-sig-performance \
  --plotly-html=true \
  --since=2022-11-29 \
  --metrics-list=vmiCreationToRunningSecondsP50,vmiCreationToRunningSecondsP95,LIST-virtualmachineinstances-count,LIST-pods-count,LIST-nodes-count,LIST-virtualmachineinstancemigrations-count,LIST-endpoints-count,GET-virtualmachineinstances-count,GET-pods-count,GET-nodes-count,GET-virtualmachineinstancemigrations-count,GET-endpoints-count,CREATE-virtualmachineinstances-count,CREATE-pods-count,CREATE-nodes-count,CREATE-virtualmachineinstancemigrations-count,CREATE-endpoints-count,PATCH-virtualmachineinstances-count,PATCH-pods-count,PATCH-nodes-count,PATCH-virtualmachineinstancemigrations-count,PATCH-endpoints-count,UPDATE-virtualmachineinstances-count,UPDATE-pods-count,UPDATE-nodes-count,UPDATE-virtualmachineinstancemigrations-count,UPDATE-endpoints-count
#
## plot sig-performance 1.25 weekly vm
perf-report-creator weekly-graph \
  --resource=vmi \
  --weekly-reports-dir=${OUTPUT_DIR}/weekly/periodic-kubevirt-performance-cluster-100-density-test \
  --plotly-html=true \
  --since=2022-11-29 \
  --metrics-list=vmiCreationToRunningSecondsP50,vmiCreationToRunningSecondsP95,LIST-virtualmachineinstances-count,LIST-pods-count,LIST-nodes-count,LIST-virtualmachineinstancemigrations-count,LIST-endpoints-count,GET-virtualmachineinstances-count,GET-pods-count,GET-nodes-count,GET-virtualmachineinstancemigrations-count,GET-endpoints-count,CREATE-virtualmachineinstances-count,CREATE-pods-count,CREATE-nodes-count,CREATE-virtualmachineinstancemigrations-count,CREATE-endpoints-count,PATCH-virtualmachineinstances-count,PATCH-pods-count,PATCH-nodes-count,PATCH-virtualmachineinstancemigrations-count,PATCH-endpoints-count,UPDATE-virtualmachineinstances-count,UPDATE-pods-count,UPDATE-nodes-count,UPDATE-virtualmachineinstancemigrations-count,UPDATE-endpoints-count


# for any one who's is looking for the output of the above commands, run the following loop:
#for file in $(find ${OUTPUT_DIR}/weekly -name index.html)
#do
#    echo "commit: $file"
#done