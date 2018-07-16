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

USER jenkins

RUN mkdir -p /tmp/init.groovy.j2/
COPY init.groovy/01-init-jenkins.groovy /tmp/init.groovy.j2/
COPY init.groovy/02-security.groovy /tmp/init.groovy.j2/
COPY init.groovy/03-pipeline.groovy /tmp/init.groovy.j2/

COPY entrypoint.sh /
ENTRYPOINT ["/entrypoint.sh"]

RUN install-plugins.sh antisamy-markup-formatter matrix-auth blueocean:1.7.0
