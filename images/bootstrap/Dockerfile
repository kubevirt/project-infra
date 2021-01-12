# Copyright 2020 The KubeVirt Authors.
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

FROM gcr.io/k8s-testimages/bootstrap:v20201106-b01d6e7

RUN cp /usr/local/bin/runner.sh /usr/local/bin/runner_orig.sh

RUN  curl -Lo ./bazelisk https://github.com/bazelbuild/bazelisk/releases/download/v1.7.4/bazelisk-linux-amd64 && \
     chmod +x ./bazelisk && mv ./bazelisk /usr/local/bin/bazelisk && \
     cd /usr/local/bin && ln -s bazelisk bazel

# Cache the most commonly used bazel versions inside the image
RUN USE_BAZEL_VERSION=3.4.1 bazel version

COPY "runner.sh" \
  "/usr/local/bin/"
