ARG VPP_VERSION=19.08
ARG VPP_IMAGE=ligato/vpp-base:$VPP_VERSION
ARG AGENT_IMAGE=ligato/vpp-agent:v2.3.0

FROM $AGENT_IMAGE as agent
FROM $VPP_IMAGE

RUN apt-get update && apt-get install -y --no-install-recommends \
	git \
	build-essential \
	make \
	uuid-dev \
	curl \
	wget \
	iproute2 \
	graphviz \
	iputils-ping \
	iproute2 && \
	rm -rf /var/lib/apt/lists/*

# Install Go
ENV GOLANG_VERSION 1.13.1
ENV GO111MODULE=on
RUN set -eux; \
	dpkgArch="$(dpkg --print-architecture)"; \
		case "${dpkgArch##*-}" in \
			amd64) goRelArch='linux-amd64'; ;; \
			armhf) goRelArch='linux-armv6l'; ;; \
			arm64) goRelArch='linux-arm64'; ;; \
	esac; \
 	wget -nv -O go.tgz "https://golang.org/dl/go${GOLANG_VERSION}.${goRelArch}.tar.gz"; \
 	tar -C /usr/local -xzf go.tgz; \
 	rm go.tgz;

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

# Install debugger
RUN go get github.com/go-delve/delve/cmd/dlv && dlv version;

# Install vpp-agent-init (basically supervisor)
COPY --from=agent /bin/vpp-agent-init /usr/bin/
COPY ./docker/nsm-agent/init_hook.sh /usr/bin/

# Build agent
RUN mkdir -p $GOPATH/src/go.cdnf.io/cnf-nsm
WORKDIR $GOPATH/src/go.cdnf.io/cnf-nsm
COPY . ./
RUN make dep-install
RUN make nsm-agent-vpp
RUN cp cmd/nsm-agent-vpp/nsm-agent-vpp /usr/local/bin/nsm-agent-vpp

# Copy configs
COPY \
    ./docker/nsm-agent/etcd.conf \
    ./docker/nsm-agent/supervisor.conf \
 /opt/nsm-agent/
COPY ./docker/nsm-agent/vpp.conf /etc/vpp/vpp.conf
ENV CONFIG_DIR /opt/nsm-agent/

# Fix for: https://github.com/ligato/vpp-agent/issues/1543
RUN mkdir /run/vpp

CMD rm -f /dev/shm/db /dev/shm/global_vm /dev/shm/vpe-api && \
    exec vpp-agent-init
