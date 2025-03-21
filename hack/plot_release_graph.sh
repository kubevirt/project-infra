#!/bin/bash

set -e
set -o pipefail

export GH_CLI_VERSION="2.69.0"
export GO_VERSION="1.23.0"
export GIT_AUTHOR_EMAIL="<your-email>"
export GIT_AUTHOR_NAME="<github-username>"
export TARGET_GITHUB_REPO="$GIT_AUTHOR_NAME/ci-performance-benchmarks"
export RELEASE_VERSION="<release-version>"
export SINCE_DATE="<since-date>"

function install_gh_cli() {
    gh_cli_dir=$(mktemp -d)
    (
        cd  "$gh_cli_dir/"
        echo $GH_CLI_VERSION
        curl -sSL "https://github.com/cli/cli/releases/download/v${GH_CLI_VERSION}/gh_${GH_CLI_VERSION}_linux_amd64.tar.gz" -o "gh_${GH_CLI_VERSION}_linux_amd64.tar.gz"
        tar xvf "gh_${GH_CLI_VERSION}_linux_amd64.tar.gz"
    )

    export PATH="$gh_cli_dir/gh_${GH_CLI_VERSION}_linux_amd64/bin:$PATH"

    if ! command -v gh &> /dev/null; then
        echo "GitHub CLI not installed successfully"
        exit 1
    fi
    echo "GitHub CLI installed successfully"
}

function install_go() {
    echo "Installing Go..."
    curl -OL https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
    tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
    export PATH="/usr/local/go/bin:$PATH"
    
    if ! command -v go &> /dev/null; then
        echo "Go not installed successfully"
        exit 1
    fi
    echo "Go installed successfully"
    go version
}

function install_git_curl() {
    echo "Installing git and curl..."
    apt update && apt install -y git curl
    if ! command -v curl &> /dev/null; then
        echo "curl not installed successfully"
        exit 1
    fi
    echo "git and curl installed successfully"
}

function clone_repo() {
    echo "Cloning the repository..."
    git clone "https://github.com/$TARGET_GITHUB_REPO.git" || true
}

install_git_curl
install_go
install_gh_cli
clone_repo

cd /src
echo "Building perf-report-creator..."
go build -o ./perf-report-creator ./robots/cmd/perf-report-creator/...

CI_PERF_REPO_DIR=$(cd ../ci-performance-benchmarks && pwd)

echo "Running graph.sh..."
./robots/cmd/perf-report-creator/release-graph.sh "$CI_PERF_REPO_DIR" "$SINCE_DATE"

cd $CI_PERF_REPO_DIR
git config user.email "$GIT_AUTHOR_EMAIL"
git config user.name "$GIT_AUTHOR_NAME"

echo $RELEASE_VERSION
mkdir -p $CI_PERF_REPO_DIR/$RELEASE_VERSION/periodic-kubevirt-e2e-k8s-sig-performance $CI_PERF_REPO_DIR/$RELEASE_VERSION/periodic-kubevirt-performance-cluster-100-density-test
mkdir -p $CI_PERF_REPO_DIR/$RELEASE_VERSION/periodic-kubevirt-e2e-k8s-sig-performance/vmi $CI_PERF_REPO_DIR/$RELEASE_VERSION/periodic-kubevirt-e2e-k8s-sig-performance/vm $CI_PERF_REPO_DIR/$RELEASE_VERSION/periodic-kubevirt-performance-cluster-100-density-test/vmi 
mv $CI_PERF_REPO_DIR/weekly/vmi/release-index.html $CI_PERF_REPO_DIR/$RELEASE_VERSION/periodic-kubevirt-e2e-k8s-sig-performance/vmi
mv $CI_PERF_REPO_DIR/weekly/vm/release-index.html $CI_PERF_REPO_DIR/$RELEASE_VERSION/periodic-kubevirt-e2e-k8s-sig-performance/vm
mv $CI_PERF_REPO_DIR/weekly/periodic-kubevirt-performance-cluster-100-density-test/vmi/release-index.html $CI_PERF_REPO_DIR/$RELEASE_VERSION/periodic-kubevirt-performance-cluster-100-density-test/vmi

# Commit and push the changes
echo "Committing and pushing the changes..."
cd $CI_PERF_REPO_DIR
git checkout -b add-$RELEASE_VERSION || true
git clean -fd weekly
git add $RELEASE_VERSION
git commit --signoff -m "Add $RELEASE_VERSION benchmarks data"
git push "https://$GIT_AUTHOR_NAME@github.com/$TARGET_GITHUB_REPO.git" add-$RELEASE_VERSION || true

gh pr create --base kubevirt:main --head $GIT_AUTHOR_NAME:add-$RELEASE_VERSION --repo "https://github.com/kubevirt/ci-performance-benchmarks.git" --title "Add $RELEASE_VERSION benchmarks data" --body "This PR adds the $RELEASE_VERSION benchmarks data to the repository."  || true

echo "PR created successfully"
echo "Process completed successfully."