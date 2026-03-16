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

    # git-crypt workflow
    if command -v git-crypt >/dev/null; then
        git-crypt unlock
    else
        echo "[WARNING] git-crypt is missing, only legacy secrets are available" >&2
    fi

    # Legacy workflow
    gpg --decrypt secrets.tar.asc | tar -xvf -
    if [ ! -f main.yml ]; then
        echo "[ERROR] Secrets file not present after unencrypting and unpacking" >&2
        exit 1
    fi

    cd - >/dev/null
}

cleanup_secrets(){
    if [ -d "${secrets_repo_dir}" ]; then
        rm -rf "${secrets_repo_dir}"
    fi
}

extract_secret(){
    local key="${1}"
    local path="${2}"

    if ! command -v yq >/dev/null; then
        curl -fsSLo ./yq https://github.com/mikefarah/yq/releases/download/v4.47.1/yq_linux_amd64
        chmod +x ./yq && mv ./yq /usr/local/bin
    fi

    mkdir -p $(dirname "${path}")
    # only remove new line at the end
    yq -r ".${key}" "${secrets_repo_dir}"/main.yml | awk 'NR>1{print PREV} {PREV=$0} END{printf("%s",$0)}' > "${path}"
}
