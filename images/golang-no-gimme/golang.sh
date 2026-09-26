#!/usr/bin/env bash
# Copyright 2018 The Kubernetes Authors.
# Copyright 2021 The KubeVirt Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ORIG_GO_VERSION="$(GOTOOLCHAIN=local go version | awk '/^go version go[0-9]+(.[0-9]+){1,2} .+/ { sub("go", "", $3); print($3) }')"
if [[ -v GIMME_GO_VERSION ]]; then
  if [[ "${ORIG_GO_VERSION}" == "${GIMME_GO_VERSION}" ]]; then
    echo "golang ${GIMME_GO_VERSION} is already installed"
  else
    # based on https://go.dev/doc/manage-install
    # go is installed in /usr/local/origgo, and we are using the
    # /usr/local/go symbolic link to make it available in the PATH
    # Now, we'll use the standard go command to download a specific
    # version of go, and then replace the link to point to this
    # version. Eventually, the go command line is the desired one.
    echo "Installing golang ${GIMME_GO_VERSION}"
    set -x
    GO_BIN="go${GIMME_GO_VERSION}"
    go install "golang.org/dl/${GO_BIN}@latest"
    "${HOME}/go/bin/${GO_BIN}" download
    unlink /usr/local/go
    ln -s "${HOME}/go" /usr/local/go
    ln -s "/usr/local/go/bin/${GO_BIN}" /usr/local/go/bin/go
    set +x
  fi
else
  echo "using the default golang version ${ORIG_GO_VERSION}"
fi
