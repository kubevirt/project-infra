FROM registry.fedoraproject.org/fedora AS BUILD

RUN  dnf install git nodejs npm bzip2 python3-pip -y && \
     dnf clean all && \
     rm -rf /var/cache/yum 

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




