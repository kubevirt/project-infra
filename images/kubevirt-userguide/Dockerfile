FROM registry.fedoraproject.org/fedora:latest

RUN yum -y update && \
  yum install -y python39 python3-pip && \
  yum clean all && \
  rm -rf /var/cache/yum/*
RUN alternatives --install /usr/bin/python python /usr/bin/python3 1 && \
  pip install --upgrade pip && \
  pip install mkdocs && \
  mkdocs --version && \
  mkdocs new user-guide

EXPOSE 8000/tcp
