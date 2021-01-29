#!/bin/bash

set -ex

docker run \
       -v $(pwd):/app \
       -v ${GITHUB_TOKEN}:/etc/github-token \
       -v ${GOOGLE_APPLICATION_CREDENTIALS}:/etc/google-application-credentials \
       -v /tmp/molecule-docker:/docker-graph \
       -e GITHUB_TOKEN=/etc/github-token \
       -e GOOGLE_APPLICATION_CREDENTIALS=/etc/google-application-credentials \
       -e DOCKER_IN_DOCKER_ENABLED=true \
       --entrypoint /usr/local/bin/runner.sh \
       --privileged \
       -w /app/github/ci/prow-deploy \
       -t quay.io/kubevirtci/prow-deploy:v20210129-94779ee \
       /bin/bash -c "/usr/local/bin/molecule test"
