#!/bin/bash

set -e
set -o pipefail

GIT_AUTHOR_EMAIL="email"
GIT_AUTHOR_NAME="name"
RELEASE_VERSION="release-version" # e.g. "v1-6"
SINCE_DATE="since-date"           # e.g. "2024-01-25"
RELEASE_DIR="release-$RELEASE_VERSION"
BRANCH="add-$RELEASE_VERSION"
TARGET_GITHUB_REPO="ci-performance-benchmarks"

function clone_repo() {
	echo "Cloning the repository..."
	git clone "https://github.com/$GIT_AUTHOR_NAME/$TARGET_GITHUB_REPO.git" || true
}

clone_repo

cd /src
echo "Building perf-report-creator..."
go build -o ./perf-report-creator ./robots/cmd/perf-report-creator/...

CI_PERF_REPO_DIR=$(cd /workspace/ci-performance-benchmarks && pwd)

echo "Running graph.sh..."
./robots/cmd/perf-report-creator/release-graph.sh "$CI_PERF_REPO_DIR" "$SINCE_DATE"

cd "$CI_PERF_REPO_DIR"

# Create release graph directory
mkdir -p "$CI_PERF_REPO_DIR"/$RELEASE_DIR/periodic-kubevirt-e2e-k8s-sig-performance "$CI_PERF_REPO_DIR"/$RELEASE_DIR/periodic-kubevirt-performance-cluster-100-density-test
mkdir -p "$CI_PERF_REPO_DIR"/$RELEASE_DIR/periodic-kubevirt-e2e-k8s-sig-performance/vmi "$CI_PERF_REPO_DIR"/$RELEASE_DIR/periodic-kubevirt-e2e-k8s-sig-performance/vm "$CI_PERF_REPO_DIR"/$RELEASE_DIR/periodic-kubevirt-performance-cluster-100-density-test/vmi
mv "$CI_PERF_REPO_DIR"/weekly/vmi/release-index.html "$CI_PERF_REPO_DIR"/$RELEASE_DIR/periodic-kubevirt-e2e-k8s-sig-performance/vmi
mv "$CI_PERF_REPO_DIR"/weekly/vm/release-index.html "$CI_PERF_REPO_DIR"/$RELEASE_DIR/periodic-kubevirt-e2e-k8s-sig-performance/vm
mv "$CI_PERF_REPO_DIR"/weekly/periodic-kubevirt-performance-cluster-100-density-test/vmi/release-index.html "$CI_PERF_REPO_DIR"/$RELEASE_DIR/periodic-kubevirt-performance-cluster-100-density-test/vmi

# commit changes, push and create PR
/src/hack/git-pr.sh -c "git clean -fd weekly" -T main -r ci-performance-benchmarks -m "$CI_PERF_REPO_DIR" -b $BRANCH -l $GIT_AUTHOR_NAME -n $GIT_AUTHOR_NAME -e $GIT_AUTHOR_EMAIL -d "echo 'Add release $RELEASE_VERSION benchmarks data'" -s "Generating release $RELEASE_VERSION benchmarks data" -B "Add release $RELEASE_VERSION benchmarks data" -D true

echo "PR created successfully"
echo "Process completed successfully."
