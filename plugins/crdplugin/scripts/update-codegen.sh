#!/bin/bash

# Copyright (c) 2020 PANTHEON.tech s.r.o. All rights reserved.
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

set -x
set -o errexit
set -o nounset
set -o pipefail

# Set directory for dependency symlinks
CRDGEN_DEPS_DIR=${1:-}
mkdir -p $CRDGEN_DEPS_DIR

# Remove any existing
find "${CRDGEN_DEPS_DIR}" -type l -delete
find "${CRDGEN_DEPS_DIR}" -type d -empty -delete

# Create symlink to generator
# TODO get version from the go.mod file
module="k8s.io/code-generator@v0.0.0-20191004115455-8e001e5d1894"
echo "setting up symplink for $module"
go list -f "${CRDGEN_DEPS_DIR}/{{ .Path }}" -m $module | xargs -L1 dirname | sort | uniq | xargs mkdir -p
go list -f "{{ .Dir }} ${CRDGEN_DEPS_DIR}/{{ .Path }}" -m $module | xargs -L1 -- ln -s

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/../../..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd ${SCRIPT_ROOT}; ls -d -1 ./${CRDGEN_DEPS_DIR}/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

# generate the code with:
chmod +x ${CODEGEN_PKG}/generate-groups.sh
${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  go.cdnf.io/cnf-nsm/plugins/crdplugin/pkg/client \
  go.cdnf.io/cnf-nsm/plugins/crdplugin/pkg/apis \
  "pantheontech:v1" \
  --go-header-file ${SCRIPT_ROOT}/plugins/crdplugin/scripts/header.go.txt