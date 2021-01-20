FROM registry.fedoraproject.org/fedora:latest

RUN yum -y update && \
  yum install -y gcc python37 python3-pip ruby ruby-devel rubygems rubygems-devel rubygem-rake && \
  yum install -y @development-tools && \
  dnf install -y @rpm-development-tools && \
  yum clean all && \
  rm -rf /var/cache/yum/*
RUN alternatives --install /usr/bin/python python /usr/bin/python3 1 && \
  pip install --upgrade pip && \
  pip install mkdocs mkdocs-awesome-pages-plugin mkdocs-htmlproofer-plugin && \
  mkdocs --version

EXPOSE 8000/tcp
