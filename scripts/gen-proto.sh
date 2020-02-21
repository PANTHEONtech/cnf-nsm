#!/usr/bin/env bash

set -euo pipefail

PROTO_DIR=${1:-$PWD/proto}
cd proto

protos=$(find . -type f -name '*.proto')

for proto in $protos; do
	echo " - $proto";
	protoc \
		--proto_path=${PROTO_DIR} \
		--go_out=plugins=grpc,paths=source_relative:. \
		"${PROTO_DIR}/$proto";
done