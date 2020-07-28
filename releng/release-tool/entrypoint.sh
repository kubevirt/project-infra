#!/bin/bash

set -e

cd /home/releaser

# import gpg key
gpg --batch --allow-secret-key-import --import /home/releaser/gpg-private

# test gpg signing which also caches the passphrase so signing a git tag does not prompt for user input.
echo "test" > testfile.txt
gpg2 --batch --pinentry-mode loopback --passphrase-file /home/releaser/gpg-passphrase --yes --detach-sign -o sig.gpg testfile.txt
rm -f testfile.txt sig.gpg

echo "$@"

/usr/sbin/release-tool --github-token-file=/home/releaser/github-api-token "$@"
