FROM registry.fedoraproject.org/fedora:latest

RUN yum -y update && \
  yum install -y npm bzip2 fontconfig python3 && \
  yum clean all && \
  rm -rf /var/cache/yum/*
RUN alternatives --install /usr/bin/python python /usr/bin/python3 1 && \
  npm config set user 0 && \
  npm config set unsafe-perm true && \
  npm install -g phantomjs casperjs
