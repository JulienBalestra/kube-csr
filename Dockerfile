FROM golang:1.10 as builder

RUN git clone --depth=1 https://github.com/JulienBalestra/kube-csr.git /go/src/github.com/JulienBalestra/kube-csr && \
    make -C /go/src/github.com/JulienBalestra/kube-csr

FROM busybox:latest

COPY --from=builder /go/src/github.com/JulienBalestra/kube-csr/kube-csr /usr/local/bin/kube-csr
