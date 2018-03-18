FROM golang:1.10 as builder

COPY . /go/src/github.com/hikhvar/mqtt2prometheus
WORKDIR /go/src/github.com/hikhvar/mqtt2prometheus
RUN make static_build TARGET_FILE=/bin/mqtt2prometheus

FROM scratch
COPY --from=builder /bin/mqtt2prometheus /mqtt2prometheus
CMD ["/mqtt2prometheus"]