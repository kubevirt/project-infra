#!/bin/bash
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

CACHE_HOST="${CACHE_HOST:-bazel-cache.kubevirt-prow.svc.cluster.local}"
CACHE_PORT="${CACHE_PORT:-8080}"

# get the installed version of a rpm package
package_to_version () {
    rpm -q --qf "%{VERSION}" "$1"
}

# look up a binary with `command -v $1` and return the rpm package it belongs to
command_to_package () {
    # NOTE: First resolve symlinks so that we are sure that we really get the package
    # which provides the binary.
    local binary_path
    binary_path=$(readlink -f "$(command -v "$1")")
    # Get the name of the package from where the binary comes from
    rpm -q --queryformat "%{NAME}" --whatprovides "${binary_path}"
}

# get the installed package version relating to a binary
command_to_version () {
    local package
    package=$(command_to_package "$1")
    package_to_version "${package}"
}

hash_toolchains () {
    # if $CC is set bazel will use this to detect c/c++ toolchains, otherwise gcc
    # https://blog.bazel.build/2016/03/31/autoconfiguration.html
    local cc="${CC:-gcc}"
    local cc_version
    cc_version=$(command_to_version "$cc")
    # NOTE: IIRC some rules call python internally, this can't hurt
    local python_version
    python_version=$(command_to_version python)
    # the rpm packaging rules use rpmbuild
    local rpmbuild_version
    rpmbuild_version=$(command_to_version rpm)
    # combine all tool versions into a hash
    # NOTE: if we change the set of tools considered we should
    # consider prepending the hash with a """schema version""" for completeness
    local tool_versions
    tool_versions="CC:${cc_version},PY:${python_version},RPM:${rpmbuild_version}"
    echo "${tool_versions}" 1>&2;
    echo "${tool_versions}" | md5sum | cut -d" " -f1
}

get_workspace () {
    # get org/repo from prow, otherwise use $PWD
    if [[ -n "${REPO_NAME}" ]] && [[ -n "${REPO_OWNER}" ]]; then
        echo "${REPO_OWNER}/${REPO_NAME}"
    else
        echo "$(basename "$(dirname "$PWD")")/$(basename "$PWD")"
    fi
}

make_bazel_rc () {
    # this is the default for recent releases but we set it explicitly
    # since this is the only hash our cache supports
    echo "startup --host_jvm_args=-Dbazel.DigestFunction=sha256"
    # don't fail if the cache is unavailable
    echo "build --remote_local_fallback"
    # point bazel at our http cache ...
    # NOTE our caches are versioned by all path segments up until the last two
    # IE PUT /foo/bar/baz/cas/asdf -> is in cache "/foo/bar/baz"
    local cache_id
    cache_id="$(get_workspace),$(hash_toolchains)"
    local cache_url
    cache_url="http://${CACHE_HOST}:${CACHE_PORT}/${cache_id}"
    echo "build --remote_cache=${cache_url}"
}

# https://docs.bazel.build/versions/master/user-manual.html#bazelrc
# bazel will look for two RC files, taking the first option in each set of paths
# firstly:
# - The path specified by the --bazelrc=file startup option. If specified, this option must appear before the command name (e.g. build)
# - A file named .bazelrc in your base workspace directory
# - A file named .bazelrc in your home directory
bazel_rc_contents=$(make_bazel_rc)
echo "create_bazel_cache_rcs.sh: Configuring '${HOME}/.bazelrc' and '/etc/bazel.bazelrc' with"
echo "# ------------------------------------------------------------------------------"
echo "${bazel_rc_contents}"
echo "# ------------------------------------------------------------------------------"
echo "${bazel_rc_contents}" >> "${HOME}/.bazelrc"
# Aside from the optional configuration file described above, Bazel also looks for a master rc file next to the binary, in the workspace at tools/bazel.rc or system-wide at /etc/bazel.bazelrc.
# These files are here to support installation-wide options or options shared between users. Reading of this file can be disabled using the --nomaster_bazelrc option.
echo "${bazel_rc_contents}" >> "/etc/bazel.bazelrc"
# hopefully no repos create *both* of these ...
