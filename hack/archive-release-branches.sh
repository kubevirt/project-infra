#!/bin/bash
# Lock archived release branches in kubevirt/kubevirt (VEP #233 Phase 2)
#
# Run AFTER the project-infra PR merges and the branchprotector
# completes at least one cycle (hourly at :54).
#
# Usage: hack/archive-release-branches.sh [BRANCH...]
#   Without arguments: locks all 50 legacy branches (release-0.4 through release-0.57)
#   With arguments:    locks only the specified branches
#
# Prerequisites:
#   - gh CLI authenticated with admin access to kubevirt/kubevirt
#   - PR https://github.com/kubevirt/project-infra/pull/5041 merged

set -euo pipefail

REPO="kubevirt/kubevirt"

DEFAULT_BRANCHES=(
  release-0.4 release-0.6 release-0.8 release-0.9
  release-0.10 release-0.11 release-0.12 release-0.13
  release-0.14 release-0.15 release-0.16 release-0.17
  release-0.18 release-0.19 release-0.20 release-0.21
  release-0.22 release-0.23 release-0.24 release-0.26
  release-0.27 release-0.29 release-0.30 release-0.31
  release-0.32 release-0.33 release-0.34 release-0.35
  release-0.36 release-0.37 release-0.38 release-0.39
  release-0.40 release-0.41 release-0.42 release-0.43
  release-0.44 release-0.45 release-0.46 release-0.47
  release-0.48 release-0.49 release-0.50 release-0.51
  release-0.52 release-0.53 release-0.54 release-0.55
  release-0.56 release-0.57
)

if [[ $# -gt 0 ]]; then
  BRANCHES=("$@")
else
  BRANCHES=("${DEFAULT_BRANCHES[@]}")
fi

failed=()

for branch in "${BRANCHES[@]}"; do
  echo -n "Locking ${branch}... "
  if gh api "repos/${REPO}/branches/${branch}/protection" \
    --method PUT \
    --input - <<'BODY' > /dev/null 2>&1
{
  "lock_branch": true,
  "enforce_admins": true,
  "required_status_checks": null,
  "required_pull_request_reviews": null,
  "restrictions": null
}
BODY
  then
    echo "ok"
  else
    echo "FAILED"
    failed+=("${branch}")
  fi
done

echo ""
echo "Locked $((${#BRANCHES[@]} - ${#failed[@]}))/${#BRANCHES[@]} branches."

if [[ ${#failed[@]} -gt 0 ]]; then
  echo "Failed branches: ${failed[*]}"
  exit 1
fi
