FROM golang:1.16 as builder

COPY . /build/mqtt2prometheus
WORKDIR /build/mqtt2prometheus
RUN make static_build TARGET_FILE=/bin/mqtt2prometheus

FROM gcr.io/distroless/static-debian10:nonroot
COPY --from=builder /bin/mqtt2prometheus /mqtt2prometheus
CMD ["/mqtt2prometheus"]
