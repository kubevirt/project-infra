#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

TEST_PACKAGES="${*}"
THRESHOLD="${COVERAGE_THRESHOLD:-70}"
GITHUB_TOKEN_FILE="/etc/github-commenter/oauth"
GITHUB_API="https://api.github.com"
STATUS_CONTEXT="coverage-auto"
SPYGLASS_BASE="https://prow.ci.kubevirt.io/view/gs/kubevirt-prow"

spyglass_url() {
    echo "${SPYGLASS_BASE}/pr-logs/pull/${REPO_OWNER}_${REPO_NAME}/${PULL_NUMBER}/${JOB_NAME}/${BUILD_ID}/"
}

post_github_status() {
    local state="$1"
    local description="$2"
    local target_url="${3:-}"

    if [[ ! -f "${GITHUB_TOKEN_FILE}" ]]; then
        echo "WARNING: GitHub token not found, skipping status update"
        return
    fi

    local payload
    payload=$(jq -n \
        --arg state "${state}" \
        --arg description "${description}" \
        --arg context "${STATUS_CONTEXT}" \
        --arg target_url "${target_url}" \
        '{state: $state, description: $description, context: $context, target_url: $target_url}')

    curl -s -X POST \
        -H "Authorization: token $(cat "${GITHUB_TOKEN_FILE}")" \
        -H "Accept: application/vnd.github.v3+json" \
        "${GITHUB_API}/repos/${REPO_OWNER}/${REPO_NAME}/statuses/${PULL_PULL_SHA}" \
        -d "${payload}" > /dev/null || echo "WARNING: Failed to post GitHub commit status"
}

extract_total_coverage() {
    local covfile="$1"
    if [[ ! -f "${covfile}" ]]; then
        echo "0"
        return
    fi
    go tool cover -func="${covfile}" | awk '/^total:/ { gsub(/%/, "", $NF); print $NF }'
}

post_github_status "pending" "Running coverage..." "$(spyglass_url)"
trap 'post_github_status "error" "Coverage script failed" "$(spyglass_url)"' ERR

# Run coverage on PR code
TEST_EXIT=0
go test ${TEST_PACKAGES} -coverprofile="${ARTIFACTS}/pr.cov" || TEST_EXIT=$?

if [[ ${TEST_EXIT} -ne 0 ]]; then
    echo "WARNING: go test exited with ${TEST_EXIT}, coverage profile may be partial"
fi

if [[ -f "${ARTIFACTS}/pr.cov" ]]; then
    covreport -i "${ARTIFACTS}/pr.cov" -o "${ARTIFACTS}/filtered.html"
fi

PR_COVERAGE=$(extract_total_coverage "${ARTIFACTS}/pr.cov")
echo "PR coverage: ${PR_COVERAGE}%"

# Run coverage on base branch
BASE_COVERAGE="0"
CURRENT_HEAD=$(git rev-parse HEAD)

git fetch origin "${PULL_BASE_SHA}" --depth=1 2>/dev/null || true
if git checkout "${PULL_BASE_SHA}" 2>/dev/null; then
    if go test ${TEST_PACKAGES} -coverprofile="${ARTIFACTS}/base.cov" 2>/dev/null; then
        BASE_COVERAGE=$(extract_total_coverage "${ARTIFACTS}/base.cov")
    else
        echo "WARNING: Tests failed on base branch, using 0% as base coverage"
    fi
    git checkout "${CURRENT_HEAD}" 2>/dev/null || true
else
    echo "WARNING: Could not checkout base SHA ${PULL_BASE_SHA}, using 0% as base coverage"
fi
echo "Base coverage: ${BASE_COVERAGE}%"

# Compute delta
DELTA=$(awk "BEGIN { printf \"%.1f\", ${PR_COVERAGE} - ${BASE_COVERAGE} }")
echo "Coverage delta: ${DELTA}%"

cat > "${ARTIFACTS}/coverage-summary.json" <<EOJSON
{
  "pr_coverage": ${PR_COVERAGE},
  "base_coverage": ${BASE_COVERAGE},
  "delta": ${DELTA},
  "threshold": ${THRESHOLD}
}
EOJSON

# Post final status
trap - ERR

DELTA_DISPLAY="${DELTA}"
if awk "BEGIN { exit !(${DELTA} > 0) }"; then
    DELTA_DISPLAY="+${DELTA}"
fi

BELOW_THRESHOLD=false
if awk "BEGIN { exit !(${PR_COVERAGE} < ${THRESHOLD}) }"; then
    BELOW_THRESHOLD=true
fi

TARGET_URL="$(spyglass_url)"

if [[ ${TEST_EXIT} -ne 0 ]]; then
    post_github_status "error" "${PR_COVERAGE}% (${DELTA_DISPLAY}%) -- tests failed" "${TARGET_URL}"
elif [[ "${BELOW_THRESHOLD}" == "true" ]]; then
    post_github_status "failure" "${PR_COVERAGE}% (${DELTA_DISPLAY}%) -- below threshold ${THRESHOLD}%" "${TARGET_URL}"
else
    post_github_status "success" "${PR_COVERAGE}% (${DELTA_DISPLAY}%) | threshold: ${THRESHOLD}%" "${TARGET_URL}"
fi

if [[ -f "${ARTIFACTS}/pr.cov" ]]; then
    cp "${ARTIFACTS}/pr.cov" "${ARTIFACTS}/filtered.cov"
fi

if [[ ${TEST_EXIT} -ne 0 ]]; then
    echo "FAIL: go test failed with exit code ${TEST_EXIT}"
    exit ${TEST_EXIT}
fi

if [[ "${BELOW_THRESHOLD}" == "true" ]]; then
    echo "FAIL: Coverage ${PR_COVERAGE}% is below threshold ${THRESHOLD}%"
    exit 1
fi

echo "PASS: Coverage ${PR_COVERAGE}% meets threshold ${THRESHOLD}%"
