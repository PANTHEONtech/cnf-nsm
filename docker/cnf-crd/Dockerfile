FROM golang:1.13-alpine3.10 as builder

# we want a static binary
ENV CGO_ENABLED=0

RUN apk add --update git make

COPY . /go/src/go.cdnf.io/cnf-nsm

WORKDIR /go/src/go.cdnf.io/cnf-nsm

ENV GO111MODULE=on

RUN make cnf-crd

FROM alpine:3.10

COPY --from=builder /go/src/go.cdnf.io/cnf-nsm/cmd/cnf-crd/cnf-crd /cnf-crd
COPY --from=builder /go/src/go.cdnf.io/cnf-nsm/k8s/cnf-crd.yaml /k8s/cnf-crd.yaml

CMD ["/cnf-crd"]
