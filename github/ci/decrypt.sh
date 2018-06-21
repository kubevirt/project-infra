#!/bin/bash

set -e

PRIVATE_KEY=$1
RECIPIENT=$2

openssl rsautl -decrypt -oaep -inkey ${PRIVATE_KEY} -in ${RECIPIENT}.key.enc -out secret.key
openssl aes-256-cbc -d -in ${RECIPIENT}.tar.enc -out ${RECIPIENT}.tar -pass file:secret.key

echo "File ${RECIPIENT}.tar decrypted."
