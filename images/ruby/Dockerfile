FROM quay.io/kubevirtci/bootstrap:v20210622-0a6ad2d

RUN dnf install -y \
        curl \
        expect \
        git \
        make \
        rsync \
        zlib \
        zlib-devel \
        gcc-c++ \
        patch \
        readline \
        readline-devel \
        libyaml-devel \
        libffi-devel \
        openssl-devel \
        bzip2 \
        autoconf \
        automake \
        libtool \
        bison \
        sqlite-devel \
        glibc-langpack-en \
    && dnf -y clean all

ENV LANG=en_US.UTF-8
ENV RUBY_VERSION=2.7.3

RUN cd && \
    git clone https://github.com/rbenv/rbenv.git ~/.rbenv && \
    git clone https://github.com/rbenv/ruby-build.git ~/.rbenv/plugins/ruby-build && \
    echo 'export PATH="/root/.rbenv/bin:/root/.rbenv/plugins/ruby-build/bin:$PATH"' > /etc/setup.mixin.d/ruby.sh && \
    echo 'eval "$(rbenv init -)"' >> /etc/setup.mixin.d/ruby.sh && \
    echo 'ruby -v' >> /etc/setup.mixin.d/ruby.sh

RUN export PATH="/root/.rbenv/bin:/root/.rbenv/plugins/ruby-build/bin:$PATH" && \
    eval "$(rbenv init -)" && \
    rbenv install $RUBY_VERSION && \
    rbenv global $RUBY_VERSION && \
    ruby -v && \
    echo "gem: --no-ri --no-rdoc" > ~/.gemrc && \
    gem install bundler
