ARG DEV_IMAGE=nsm-agent-linux-dev:$VPP_VERSION

FROM $DEV_IMAGE as dev
FROM ubuntu:18.04 as base

RUN apt-get update && apt-get install -y --no-install-recommends \
		# general tools
		inetutils-traceroute \
		iproute2 \
		iputils-ping \
		curl \
 	&& rm -rf /var/lib/apt/lists/*

# Install agent
COPY --from=dev /usr/local/bin/nsm-agent-linux /usr/local/bin/

# Copy configs
COPY ./docker/nsm-agent/etcd.conf /opt/nsm-agent/

# Final image
FROM scratch
COPY --from=base / /

ENV CONFIG_DIR /opt/nsm-agent/

CMD /usr/local/bin/nsm-agent-linux