FROM registry.fedoraproject.org/fedora:33

RUN yum -y update && \
  yum install -y python37 python3-pip && \
  yum clean all && \
  rm -rf /var/cache/yum/*
RUN alternatives --install /usr/bin/python python /usr/bin/python3 1 && \
  pip install --upgrade pip && \
  pip install mkdocs mkdocs-awesome-pages-plugin && \
  mkdocs --version && \
  mkdocs new user-guide

EXPOSE 8000/tcp
