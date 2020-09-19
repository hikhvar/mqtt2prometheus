FROM golang:1.15.2 as builder

COPY . /build/mqtt2prometheus
WORKDIR /build/mqtt2prometheus
RUN make static_build TARGET_FILE=/bin/mqtt2prometheus

FROM scratch
COPY --from=builder /bin/mqtt2prometheus /mqtt2prometheus
CMD ["/mqtt2prometheus"]
