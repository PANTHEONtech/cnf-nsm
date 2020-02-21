FROM ubuntu:18.04

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
	iproute2 \
	ca-certificates && \
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

# Build agent
RUN mkdir -p $GOPATH/src/go.cdnf.io/cnf-nsm
WORKDIR $GOPATH/src/go.cdnf.io/cnf-nsm
COPY . ./
RUN make dep-install
RUN make nsm-agent-linux
RUN cp cmd/nsm-agent-linux/nsm-agent-linux /usr/local/bin/nsm-agent-linux

# Copy configs
COPY ./docker/nsm-agent/etcd.conf /opt/nsm-agent/
ENV CONFIG_DIR /opt/nsm-agent/

ENTRYPOINT ["/bin/bash"]
CMD ["-c", "/usr/local/bin/nsm-agent-linux"]