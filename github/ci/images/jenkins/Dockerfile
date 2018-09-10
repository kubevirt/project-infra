# This image is based off the latest Jenkins LTS
FROM jenkins/jenkins:lts-alpine

USER root

# Add the docker binary so running Docker commands from the master works nicely
RUN apk -U add docker \
    python \
    python-dev \
    py-pip \
    build-base \
  && pip install jinja2-cli \
  && rm -rf /var/cache/apk/*

RUN install-plugins.sh antisamy-markup-formatter \
  matrix-auth \
  blueocean:1.7.0 \
  job-dsl \
  github-oauth \
  swarm \
  throttle-concurrents \
  publish-over-ssh \
  build-timeout \
  xunit \
  test-results-analyzer \
  ws-cleanup \
  cloudbees-folder \
  credentials-binding \
  timestamper \
  workflow-aggregator \
  github-branch-source \
  pipeline-github-lib \
  pipeline-stage-view \
  git \
  subversion \
  pam-auth \
  mailer

USER jenkins

RUN mkdir -p /tmp/init.groovy.j2/
COPY init.groovy/01-init-jenkins.groovy /tmp/init.groovy.j2/
COPY init.groovy/02-security.groovy /tmp/init.groovy.j2/
COPY init.groovy/03-credentials.groovy /tmp/init.groovy.j2/
COPY init.groovy/90-seed.groovy /tmp/init.groovy.j2/
COPY init.groovy/99-remove-files.groovy /tmp/init.groovy.j2/
COPY jobs.groovy /tmp/

COPY entrypoint.sh /
ENTRYPOINT ["/entrypoint.sh"]
