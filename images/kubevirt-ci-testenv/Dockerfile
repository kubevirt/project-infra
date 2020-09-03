FROM gcr.io/k8s-testimages/bootstrap

RUN mkdir -p /opt/source/
RUN apt-get update
RUN apt-get install -y python3.7 python3-virtualenv python3-selinux git make
RUN python3 -m virtualenv -p python3 /opt/virtualenv

RUN /bin/bash -c 'source /opt/virtualenv/bin/activate; pip install -U pip setuptools molecule kubernetes kubernetes-validate openshift'
