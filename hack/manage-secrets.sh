# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright the KubeVirt Authors.

secrets_repo_dir=$(mktemp -d)

decrypt_secrets(){
    git clone --depth 1 https://kubevirt-bot@github.com/kubevirt/secrets "${secrets_repo_dir}"
    gpg --allow-secret-key-import --import /etc/pgp/token

    cd "${secrets_repo_dir}"

    git-crypt unlock

    cd - >/dev/null
}

cleanup_secrets(){
    if [ -d "${secrets_repo_dir}" ]; then
        rm -rf "${secrets_repo_dir}"
    fi
}
