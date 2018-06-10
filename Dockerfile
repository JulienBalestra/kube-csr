FROM golang:1.10 as builder

COPY . /go/src/github.com/JulienBalestra/kube-csr

RUN make -C /go/src/github.com/JulienBalestra/kube-csr re

FROM busybox:latest

COPY --from=builder /go/src/github.com/JulienBalestra/kube-csr/kube-csr /usr/local/bin/kube-csr
