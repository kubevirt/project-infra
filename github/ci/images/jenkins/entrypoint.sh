#!/bin/bash

mkdir -p /var/jenkins_home/init.groovy.d

# Populate groovy scripts from the jenkins secret
for filepath in /tmp/init.groovy.j2/*; do
    fname=$(basename $filepath)
    jinja2 $filepath /etc/jenkins/data --format=json > "/var/jenkins_home/init.groovy.d/$fname"
done

# Populate DSL jobs
jinja2 /tmp/jobs.groovy /etc/jenkins/data --format=json > /var/jenkins_home/jobs.groovy

/sbin/tini -- /usr/local/bin/jenkins.sh
