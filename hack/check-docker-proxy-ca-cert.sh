#!/bin/bash

CA_CERT_FILE=${CA_CERT_FILE:-"/etc/docker-mirror-proxy/ca.crt"}

if ! command -v openssl &> /dev/null
then
    echo "This script requires openssl to be installed"
    exit 1
fi

# 90 days in seconds = 7,776,000
EXPIRE_IN_90_DAYS="7776000"

openssl x509 -enddate -noout -in "$CA_CERT_FILE"  -checkend "$EXPIRE_IN_90_DAYS" | grep -q 'Certificate will expire'

if [ $? == 0 ]
then
    echo "The docker-mirror-proxy CA certificate will expire within the next 90 days"
    echo "Please plan a maintenance window to renew this certificate"
    exit 1
else
    echo "Certificate check passed. The certificate will not expire in the next 90 days"
fi
