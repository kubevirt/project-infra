# BASE
FROM registry.fedoraproject.org/fedora:36

# COPY app configs
WORKDIR /src
ENV LC_ALL=en_US.UTF-8
COPY ./_config/src/ ./

# UPDATE BASE and land app frameworks and dependencies
RUN dnf update -y && \
    dnf install -y langpacks-en glibc-all-langpacks @development-tools \
      redhat-rpm-config openssl-devel gcc-c++ tar jq bzip2 fontconfig \
      nodejs npm python3-pip ruby ruby-devel rubygems rubygems-devel \
      rubygem-bundler rubygem-json rubygem-nenv rubygem-rake && \
    alternatives --install /usr/bin/python python /usr/bin/python3 1 && \
    npm config set user 0 && \
    npm config set unsafe-perm true && \
    npm install -g markdownlint-cli casperjs phantomjs-prebuilt yaspeller && \
    cd /src && bundle install && bundle update && cd && \
    pip install --upgrade pip && \
    pip install mkdocs mkdocs-awesome-pages-plugin mkdocs-htmlproofer-plugin && \
    gem list && \
    rpm -e --nodeps libX11 libX11-common libXrender libXft && \
    dnf erase -y @development-tools gcc qt5-srpm-macros \
      xkeyboard-config qrencode-libs memstrack \
      openssl-devel ruby-devel rubygems-devel \
      nodejs-docs rubygem-rdoc glibc-all-langpacks vim-minimal tar \
      diffutils npm pigz bzip2 xz python3-pip jq -x git -x make && \
    dnf clean all && \
    rm -rf /root/{.bundle,.config,.npm,anaconda*,original-ks.cfg} /tmp/phantomjs /var/cache/dnf

EXPOSE 4000/tcp
EXPOSE 8000/tcp
