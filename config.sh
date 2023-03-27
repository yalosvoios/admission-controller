#!/bin/bash

# Copyright 2018 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Generates the a CA cert, a server key, and a server cert signed by the CA.
# reference:
# https://github.com/kubernetes/kubernetes/blob/master/plugin/pkg/admission/webhook/gencerts.sh
set -o errexit
set -o nounset
set -o pipefail

CN_BASE="admission"
NAMESPACE="yalos"
CERT_DIR="./certificates"

if [ $# -ge 1 ] && [ ! -z $1 ]
  then
    CN_BASE=$1
fi

if [ $# -ge 2 ] && [ ! -z $2 ]
  then
    NAMESPACE=$2
fi

echo $NAMESPACE $CN_BASE

echo "Generating certs for a simple admission controller in ${CERT_DIR}."
mkdir -p ${CERT_DIR}
cat > ${CERT_DIR}/server.conf << EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth, serverAuth
subjectAltName = DNS:${CN_BASE}.${NAMESPACE}.svc
EOF

# Create a certificate authority
openssl genrsa -out ${CERT_DIR}/caKey.pem 2048
set +o errexit
openssl req -x509 -new -nodes -key ${CERT_DIR}/caKey.pem -days 100000 -out ${CERT_DIR}/caCert.pem -subj "//CN=${CN_BASE}_ca" -addext "subjectAltName = DNS:${CN_BASE}_ca"
if [[ $? -ne 0 ]]; then
  echo "ERROR: Failed to create CA certificate for self-signing. If the error is \"unknown option -addext\", update your openssl version."
  exit 1
fi
set -o errexit

# Create a server certiticate
openssl genrsa -out ${CERT_DIR}/serverKey.pem 2048
# Note the CN is the DNS name of the service of the webhook.
openssl req -new -key ${CERT_DIR}/serverKey.pem -out ${CERT_DIR}/server.csr -subj "//CN=${CN_BASE}.${NAMESPACE}.svc" -config ${CERT_DIR}/server.conf -addext "subjectAltName = DNS:${CN_BASE}.${NAMESPACE}.svc"
openssl x509 -req -in ${CERT_DIR}/server.csr -CA ${CERT_DIR}/caCert.pem -CAkey ${CERT_DIR}/caKey.pem -CAcreateserial -out ${CERT_DIR}/serverCert.pem -days 100000 -extensions SAN -extensions v3_req -extfile ${CERT_DIR}/server.conf


echo "updating yaml files..."
export NAMESPACE=$NAMESPACE
export CN_BASE=$CN_BASE
export CACERT=$(cat ${CERT_DIR}/caCert.pem | base64 | tr -d '\n')
export CAKEY=$(cat ${CERT_DIR}/caKey.pem | base64 | tr -d '\n')
export SERVERCERT=$(cat ${CERT_DIR}/serverCert.pem | base64 | tr -d '\n')
export SERVERKEY=$(cat ${CERT_DIR}/serverKey.pem | base64 | tr -d '\n')
cat ./templates/certs.yaml | envsubst > ./yaml/certs.yaml
cat ./templates/deployment.yaml | envsubst > ./yaml/deployment.yaml
cat ./templates/service.yaml | envsubst > ./yaml/service.yaml
cat ./templates/webhook.yaml | envsubst > ./yaml/webhook.yaml


