#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

export VAULT_ADDR='http://127.0.0.1:8200'

vault write secret/default/envprinter mysecret=Password123

