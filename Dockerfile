FROM golang:1.17 as builder

COPY . /build/mqtt2prometheus
WORKDIR /build/mqtt2prometheus
RUN make static_build TARGET_FILE=/bin/mqtt2prometheus

FROM gcr.io/distroless/static-debian10:nonroot
WORKDIR /
COPY --from=builder /bin/mqtt2prometheus /mqtt2prometheus
ENTRYPOINT ["/mqtt2prometheus"]
