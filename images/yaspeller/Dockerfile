FROM registry.fedoraproject.org/fedora:latest

RUN yum -y update && \
  yum install -y npm && \
  yum clean all && \
  rm -rf /var/cache/yum/*
RUN npm install yaspeller -g
