#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

curl -LO https://releases.hashicorp.com/vault/0.9.1/vault_0.9.1_linux_amd64.zip
unzip vault_0.9.1_linux_amd64.zip
rm vault_0.9.1_linux_amd64.zip
sudo mv ./vault /usr/local/bin/