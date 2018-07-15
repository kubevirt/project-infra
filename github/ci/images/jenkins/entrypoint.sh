#!/bin/bash

# Populate groovy scripts from the jenkins secret
for filepath in /tmp/init.groovy.j2/*; do
    fname=$(basename $filepath)
    jinja2 $filepath /etc/jenkins/data --format=json > "/var/jenkins_home/init.groovy.d/$fname"
done

/sbin/tini -- /usr/local/bin/jenkins.sh
