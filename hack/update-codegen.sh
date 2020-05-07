#!/usr/bin/env bash

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
PROJECT_PATH="github.com/aftouh/k8s-sample-controller"

echo $CODEGEN_PKG

bash vendor/k8s.io/code-generator/generate-groups.sh "deepcopy,client,informer,lister" \
  ${PROJECT_PATH}/pkg/client ${PROJECT_PATH}/pkg/apis \
  team:v1 \
  --go-header-file "${SCRIPT_ROOT}"/hack/boilerplate.go.txt \
  --output-base "$(dirname "${BASH_SOURCE[0]}")/../../../.."
