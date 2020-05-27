FROM fedora:32

RUN dnf install -y git && \
    dnf clean all

COPY release-blocker /bin/release-blocker
ENTRYPOINT ["/bin/release-blocker"]

