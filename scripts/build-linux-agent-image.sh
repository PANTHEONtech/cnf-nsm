#!/usr/bin/env bash

IMAGE_REPO="$1"

source $(dirname $0)/labels.env
source $(dirname $0)/get-ligato-version.sh
VERSION="$(get_ligato_version)"

docker build --tag="${IMAGE_REPO}/${LINUX_DEV_IMAGE_LABEL}:${VERSION}" \
	-f docker/nsm-agent/linux-dev.Dockerfile .

docker build --tag="${IMAGE_REPO}/${LINUX_PROD_IMAGE_LABEL}:${VERSION}" \
	--build-arg DEV_IMAGE="${IMAGE_REPO}/${LINUX_DEV_IMAGE_LABEL}:${VERSION}" \
	-f docker/nsm-agent/linux-prod.Dockerfile .

