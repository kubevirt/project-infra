FROM registry.fedoraproject.org/fedora

ENV LC_ALL=en_US.UTF-8

RUN  dnf install jq git nodejs npm bzip2 python3-pip glibc-langpack-en -y && \
     dnf install ruby ruby-devel openssl-devel redhat-rpm-config @development-tools -y && \
     dnf clean all && \
     rm -rf /var/cache/yum && \
     gem install jekyll bundler

COPY update-yaspeller.sh /usr/local/bin/update-yaspeller.sh

# Install MKdocs
RUN pip3 install mkdocs mkdocs-awesome-pages-plugin mkdocs-htmlproofer-plugin

# Install Markdown linter
RUN  git clone https://github.com/igorshubovych/markdownlint-cli.git /src/markdownlint && \
     cd /src/markdownlint && \
     npm install -g  markdownlint-cli


# Install casperjs & yaspeller
RUN  npm config set user 0 && \
     npm config set unsafe-perm true && \
     npm install -g phantomjs casperjs yaspeller





