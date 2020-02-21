ARG VPP_VERSION=19.08
ARG AGENT_IMAGE=ligato/vpp-agent:v2.3.0
ARG DEV_IMAGE=nsm-agent-vpp-dev:$VPP_VERSION

FROM $AGENT_IMAGE as agent
FROM $DEV_IMAGE as dev
FROM ubuntu:18.04 as base

RUN apt-get update && apt-get install -y --no-install-recommends \
		# general tools
		inetutils-traceroute \
		iproute2 \
		iputils-ping \
		# vpp requirements
		ca-certificates \
		libapr1 \
		libc6 \
		libmbedcrypto1 \
		libmbedtls10 \
		libmbedx509-0 \
		libnuma1 \
		openssl \
 	&& rm -rf /var/lib/apt/lists/*

# install vpp
COPY --from=dev /vpp/*.deb /opt/vpp/

RUN cd /opt/vpp/ \
 && apt-get update \
 && apt-get install -y ./*.deb \
 && rm *.deb \
 && rm -rf /var/lib/apt/lists/*

# Install vpp-agent-init (basically supervisor)
COPY --from=agent /bin/vpp-agent-init /usr/bin/
COPY ./docker/nsm-agent/init_hook.sh /usr/bin/

# Install agent
COPY --from=dev /usr/local/bin/nsm-agent-vpp /usr/local/bin/

# Copy configs
COPY \
    ./docker/nsm-agent/etcd.conf \
    ./docker/nsm-agent/supervisor.conf \
 /opt/nsm-agent/
COPY ./docker/nsm-agent/vpp.conf /etc/vpp/vpp.conf

# Final image
FROM scratch
COPY --from=base / /

ENV CONFIG_DIR /opt/nsm-agent/

# Fix for: https://github.com/ligato/vpp-agent/issues/1543
RUN mkdir /run/vpp

CMD rm -f /dev/shm/db /dev/shm/global_vm /dev/shm/vpe-api && \
    exec vpp-agent-init
