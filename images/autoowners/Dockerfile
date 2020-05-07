FROM centos:7

ADD autoowners /usr/bin/autoowners

RUN yum install -y git

# change ownership to allow access
# when selinux is enabled
RUN mkdir -p /etc/github && \
    chgrp -R 0 /etc/github && \
    chmod -R g+rwX /etc/github

ENTRYPOINT ["/usr/bin/autoowners"]
