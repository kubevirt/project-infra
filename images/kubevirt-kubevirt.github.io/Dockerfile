FROM registry.fedoraproject.org/fedora

RUN  dnf install jq git nodejs npm bzip2 glibc-langpack-en -y && \
     dnf install ruby ruby-devel openssl-devel redhat-rpm-config @development-tools -y && \
     dnf clean all && \
     rm -rf /var/cache/yum

WORKDIR /src
ENV LC_ALL=en_US.UTF-8
COPY ./resources/. /src/.

# Install Markdown linter
RUN  git clone https://github.com/igorshubovych/markdownlint-cli.git  /src/markdownlint-cli  && \
     cd /src/markdownlint-cli && \
     npm install -g  markdownlint-cli

# Install casperjs & yaspeller
RUN  bundle install && \
     npm config set user 0 && \
     npm config set unsafe-perm true && \
     npm install -g phantomjs casperjs yaspeller && \
     alternatives --install /usr/bin/python python /usr/bin/python3 1 
     