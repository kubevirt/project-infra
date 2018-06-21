#!/bin/bash

set -e

PUB_KEY=$1
RECIPIENT=$2
TMP_DIR=secrets

rm ${TMP_DIR} -rf
mkdir ${TMP_DIR}
cp -a id_rsa* ${TMP_DIR}/
cp -a group_vars ${TMP_DIR}/
cp -a inventory ${TMP_DIR}/

echo "Adding the following files:"
echo

tar -cvf ${RECIPIENT}.tar ${TMP_DIR} | sed 's/^/  /'

openssl rand -out secret.key 32
openssl rsautl -encrypt -oaep -pubin -inkey <(ssh-keygen -e -f ${PUB_KEY} -m PKCS8) -in secret.key -out ${RECIPIENT}.key.enc
openssl aes-256-cbc -in ${RECIPIENT}.tar -out ${RECIPIENT}.tar.enc -pass file:secret.key
rm secret.key
rm -rf ${TMP_DIR}
rm ${RECIPIENT}.tar

echo 
echo "Created ${RECIPIENT}.tar.enc and ${RECIPIENT}.key.enc."
