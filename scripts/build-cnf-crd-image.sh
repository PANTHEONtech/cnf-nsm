#!/usr/bin/env bash

IMAGE_REPO="$1"

source $(dirname $0)/labels.env

docker build --tag="${IMAGE_REPO}/${CNF_CRD_IMAGE_LABEL}" \
	-f docker/cnf-crd/Dockerfile .

