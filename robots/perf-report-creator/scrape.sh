set -euo pipefail

CREDENTIALS_FILE=${1}
BENCHMARKS_DIR=${2}
OUTPUT_DIR=${3:-output}
KUBEVIRT_PROVIDER=${4}

# Validate KUBEVIRT_PROVIDER
if [ -z "${KUBEVIRT_PROVIDER}" ]; then
    echo "Error: KUBEVIRT_PROVIDER is required, but is empty"
    exit 1
fi

# The scrape window overlaps the previous ISO week so that late artifact
# uploads and job start time drift are still picked up. Overlapping runs are
# deduplicated when weekly-report merges into the published data.
SINCE=${SINCE:-192h0s}

# scrape sig-performance ${KUBEVIRT_PROVIDER} results
perf-report-creator results --credentials-file=${CREDENTIALS_FILE} --output-dir ${OUTPUT_DIR}/results --since ${SINCE} --performance-job-name periodic-kubevirt-e2e-${KUBEVIRT_PROVIDER}-sig-performance
# scrape 100 density test results
perf-report-creator results --credentials-file=${CREDENTIALS_FILE} --output-dir ${OUTPUT_DIR}/results --since ${SINCE} --performance-job-name periodic-kubevirt-performance-cluster-100-density-test
# scrape kwok density test results
perf-report-creator results --credentials-file=${CREDENTIALS_FILE} --output-dir ${OUTPUT_DIR}/results --since ${SINCE} --performance-job-name periodic-kubevirt-e2e-${KUBEVIRT_PROVIDER}-sig-performance-kwok

# Seed weekly output with already published data so weekly-report can merge
# instead of overwriting earlier days in the same ISO week.
if [ -d "${BENCHMARKS_DIR}/weekly" ]; then
  mkdir -p "${OUTPUT_DIR}/weekly"
  cp -a "${BENCHMARKS_DIR}/weekly/." "${OUTPUT_DIR}/weekly/"
fi

# aggregate sig-performance ${KUBEVIRT_PROVIDER} results in weekly directory
perf-report-creator weekly-report --output-dir=${OUTPUT_DIR}/weekly/periodic-kubevirt-e2e-${KUBEVIRT_PROVIDER}-sig-performance \
  --results-dir=${OUTPUT_DIR}/results/periodic-kubevirt-e2e-${KUBEVIRT_PROVIDER}-sig-performance \
  --vmi-metrics-list=vmiCreationToRunningSecondsP50,vmiCreationToRunningSecondsP95,LIST-virtualmachineinstances-count,LIST-pods-count,LIST-nodes-count,LIST-virtualmachineinstancemigrations-count,LIST-endpoints-count,GET-virtualmachineinstances-count,GET-pods-count,GET-nodes-count,GET-virtualmachineinstancemigrations-count,GET-endpoints-count,CREATE-virtualmachineinstances-count,CREATE-pods-count,CREATE-nodes-count,CREATE-virtualmachineinstancemigrations-count,CREATE-endpoints-count,PATCH-virtualmachineinstances-count,PATCH-pods-count,PATCH-nodes-count,PATCH-virtualmachineinstancemigrations-count,PATCH-endpoints-count,UPDATE-virtualmachineinstances-count,UPDATE-pods-count,UPDATE-nodes-count,UPDATE-virtualmachineinstancemigrations-count,UPDATE-endpoints-count,avgVirtAPIMemoryUsageInMB,minVirtAPIMemoryUsageInMB,maxVirtAPIMemoryUsageInMB,avgVirtAPICPUUsage,avgVirtControllerMemoryUsageInMB,minVirtControllerMemoryUsageInMB,maxVirtControllerMemoryUsageInMB,avgVirtControllerCPUUsage,avgVirtHandlerMemoryUsageInMB,minVirtHandlerMemoryUsageInMB,maxVirtHandlerMemoryUsageInMB,avgVirtHandlerCPUUsage,virtControllerWorkqueueAddRate,virtControllerWorkqueueDepth,virtControllerWorkqueueP99Latency \
  --vm-metrics-list vmiCreationToRunningSecondsP50,vmiCreationToRunningSecondsP95,LIST-virtualmachineinstances-count,WATCH-virtualmachineinstances-count,LIST-pods-count,WATCH-pods-count,LIST-nodes-count,LIST-virtualmachineinstancemigrations-count,WATCH-virtualmachineinstancemigrations-count,LIST-endpoints-count,GET-virtualmachineinstances-count,GET-pods-count,GET-nodes-count,GET-virtualmachineinstancemigrations-count,GET-endpoints-count,CREATE-virtualmachineinstances-count,CREATE-pods-count,CREATE-nodes-count,CREATE-virtualmachineinstancemigrations-count,CREATE-endpoints-count,PATCH-virtualmachineinstances-count,PATCH-pods-count,PATCH-nodes-count,PATCH-virtualmachineinstancemigrations-count,PATCH-endpoints-count,UPDATE-virtualmachineinstances-count,UPDATE-pods-count,UPDATE-nodes-count,UPDATE-virtualmachineinstancemigrations-count,UPDATE-endpoints-count,avgVirtAPIMemoryUsageInMB,minVirtAPIMemoryUsageInMB,maxVirtAPIMemoryUsageInMB,avgVirtAPICPUUsage,avgVirtControllerMemoryUsageInMB,minVirtControllerMemoryUsageInMB,maxVirtControllerMemoryUsageInMB,avgVirtControllerCPUUsage,avgVirtHandlerMemoryUsageInMB,minVirtHandlerMemoryUsageInMB,maxVirtHandlerMemoryUsageInMB,avgVirtHandlerCPUUsage,virtControllerWorkqueueAddRate,virtControllerWorkqueueDepth,virtControllerWorkqueueP99Latency

# aggregate sig-performance-kwok ${KUBEVIRT_PROVIDER} results in weekly directory
perf-report-creator weekly-report --output-dir=${OUTPUT_DIR}/weekly/periodic-kubevirt-e2e-${KUBEVIRT_PROVIDER}-sig-performance-kwok \
  --results-dir=${OUTPUT_DIR}/results/periodic-kubevirt-e2e-${KUBEVIRT_PROVIDER}-sig-performance-kwok \
  --vmi-metrics-list=vmiCreationToRunningSecondsP50,vmiCreationToRunningSecondsP95,LIST-virtualmachineinstances-count,LIST-pods-count,LIST-nodes-count,LIST-virtualmachineinstancemigrations-count,LIST-endpoints-count,GET-virtualmachineinstances-count,GET-pods-count,GET-nodes-count,GET-virtualmachineinstancemigrations-count,GET-endpoints-count,CREATE-virtualmachineinstances-count,CREATE-pods-count,CREATE-nodes-count,CREATE-virtualmachineinstancemigrations-count,CREATE-endpoints-count,PATCH-virtualmachineinstances-count,PATCH-pods-count,PATCH-nodes-count,PATCH-virtualmachineinstancemigrations-count,PATCH-endpoints-count,UPDATE-virtualmachineinstances-count,UPDATE-pods-count,UPDATE-nodes-count,UPDATE-virtualmachineinstancemigrations-count,UPDATE-endpoints-count,avgVirtAPIMemoryUsageInMB,minVirtAPIMemoryUsageInMB,maxVirtAPIMemoryUsageInMB,avgVirtAPICPUUsage,avgVirtControllerMemoryUsageInMB,minVirtControllerMemoryUsageInMB,maxVirtControllerMemoryUsageInMB,avgVirtControllerCPUUsage,avgVirtHandlerMemoryUsageInMB,minVirtHandlerMemoryUsageInMB,maxVirtHandlerMemoryUsageInMB,avgVirtHandlerCPUUsage \
  --vm-metrics-list vmiCreationToRunningSecondsP50,vmiCreationToRunningSecondsP95,LIST-virtualmachineinstances-count,WATCH-virtualmachineinstances-count,LIST-pods-count,WATCH-pods-count,LIST-nodes-count,LIST-virtualmachineinstancemigrations-count,WATCH-virtualmachineinstancemigrations-count,LIST-endpoints-count,GET-virtualmachineinstances-count,GET-pods-count,GET-nodes-count,GET-virtualmachineinstancemigrations-count,GET-endpoints-count,CREATE-virtualmachineinstances-count,CREATE-pods-count,CREATE-nodes-count,CREATE-virtualmachineinstancemigrations-count,CREATE-endpoints-count,PATCH-virtualmachineinstances-count,PATCH-pods-count,PATCH-nodes-count,PATCH-virtualmachineinstancemigrations-count,PATCH-endpoints-count,UPDATE-virtualmachineinstances-count,UPDATE-pods-count,UPDATE-nodes-count,UPDATE-virtualmachineinstancemigrations-count,UPDATE-endpoints-count,avgVirtAPIMemoryUsageInMB,minVirtAPIMemoryUsageInMB,maxVirtAPIMemoryUsageInMB,avgVirtAPICPUUsage,avgVirtControllerMemoryUsageInMB,minVirtControllerMemoryUsageInMB,maxVirtControllerMemoryUsageInMB,avgVirtControllerCPUUsage,avgVirtHandlerMemoryUsageInMB,minVirtHandlerMemoryUsageInMB,maxVirtHandlerMemoryUsageInMB,avgVirtHandlerCPUUsage,virtControllerWorkqueueAddRate,virtControllerWorkqueueDepth,virtControllerWorkqueueP99Latency

# aggregate 100-density-test results
perf-report-creator weekly-report --output-dir=${OUTPUT_DIR}/weekly/periodic-kubevirt-performance-cluster-100-density-test \
  --results-dir=${OUTPUT_DIR}/results/periodic-kubevirt-performance-cluster-100-density-test \
  --vmi-metrics-list=vmiCreationToRunningSecondsP50,vmiCreationToRunningSecondsP95,LIST-virtualmachineinstances-count,WATCH-virtualmachineinstances-count,LIST-pods-count,WATCH-pods-count,LIST-nodes-count,LIST-virtualmachineinstancemigrations-count,WATCH-virtualmachineinstancemigrations-count,LIST-endpoints-count,GET-virtualmachineinstances-count,GET-pods-count,GET-nodes-count,GET-virtualmachineinstancemigrations-count,GET-endpoints-count,CREATE-virtualmachineinstances-count,CREATE-pods-count,CREATE-nodes-count,CREATE-virtualmachineinstancemigrations-count,CREATE-endpoints-count,PATCH-virtualmachineinstances-count,PATCH-pods-count,PATCH-nodes-count,PATCH-virtualmachineinstancemigrations-count,PATCH-endpoints-count,UPDATE-virtualmachineinstances-count,UPDATE-pods-count,UPDATE-nodes-count,UPDATE-virtualmachineinstancemigrations-count,UPDATE-endpoints-count,avgVirtAPIMemoryUsageInMB,minVirtAPIMemoryUsageInMB,maxVirtAPIMemoryUsageInMB,avgVirtAPICPUUsage,avgVirtControllerMemoryUsageInMB,minVirtControllerMemoryUsageInMB,maxVirtControllerMemoryUsageInMB,avgVirtControllerCPUUsage,avgVirtHandlerMemoryUsageInMB,minVirtHandlerMemoryUsageInMB,maxVirtHandlerMemoryUsageInMB,avgVirtHandlerCPUUsage,virtControllerWorkqueueAddRate,virtControllerWorkqueueDepth,virtControllerWorkqueueP99Latency \

cp -ru ${OUTPUT_DIR}/* ${BENCHMARKS_DIR}
