#!/usr/bin/env bash

IMAGE_REPO="$1"
VPP_VERSION="$2"

source $(dirname $0)/labels.env
source $(dirname $0)/get-ligato-version.sh
VERSION="$(get_ligato_version)"

docker build --tag="${IMAGE_REPO}/${VPP_DEV_IMAGE_LABEL}:${VERSION}" \
	--build-arg VPP_VERSION="${VPP_VERSION}" \
	-f docker/nsm-agent/vpp-dev.Dockerfile .

docker build --tag="${IMAGE_REPO}/${VPP_PROD_IMAGE_LABEL}:${VERSION}" \
	--build-arg VPP_VERSION="${VPP_VERSION}" \
	--build-arg DEV_IMAGE="${IMAGE_REPO}/${VPP_DEV_IMAGE_LABEL}:${VERSION}" \
	-f docker/nsm-agent/vpp-prod.Dockerfile .

