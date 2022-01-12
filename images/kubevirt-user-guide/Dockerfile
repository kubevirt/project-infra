# BASE
FROM registry.fedoraproject.org/fedora:34

ENV LC_ALL=en_US.UTF-8

# UPDATE BASE and land app frameworks and dependencies
RUN mkdir -p /src && cd /src && \
  curl https://raw.githubusercontent.com/kubevirt/user-guide/main/_config/src/Gemfile -o Gemfile && \
  yum update -y && \
  yum install -y @development-tools \
    langpacks-en glibc-all-langpacks redhat-rpm-config openssl-devel gcc-c++ \
    tar jq bzip2 \
    nodejs npm python37 python3-pip ruby ruby-devel rubygems rubygems-devel \
    rubygem-bundler rubygem-json rubygem-nenv rubygem-rake && \
  alternatives --install /usr/bin/python python /usr/bin/python3 1 && \
  npm config set user 0 && \
  npm config set unsafe-perm true && \
  npm install -g markdownlint-cli casperjs phantomjs-prebuilt yaspeller && \
  pip install --upgrade pip && \
  pip install mkdocs mkdocs-awesome-pages-plugin mkdocs-htmlproofer-plugin && \
  mkdocs --version && \
  cd /src && bundle install && bundle update && cd && \
  gem list && \
  rpm -e --nodeps libX11 libX11-common libXrender libXft libusbx xml-common && \
  yum erase -y @development-tools gcc gtk2 subversion qt5-srpm-macros git \
    xkeyboard-config shared-mime-info qrencode-libs memstrack mod_lua \
    redhat-rpm-config openssl-devel ruby-devel rubygems-devel glibc-doc \
    nodejs-docs rubygem-rdoc dracu glibc-all-langpacks vim-minimal tar setup \
    diffutils acl npm pigz ncurses mkpasswd libXau bzip2 xz python3-pip jq && \
  dnf clean all && \
  rm -rf /root/{.bundle,.config,.npm,anaconda*,original-ks.cfg} /tmp/phantomjs /var/cache/yum

EXPOSE 8000/tcp
