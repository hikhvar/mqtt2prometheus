FROM golang:1.24 AS builder

# enable cross-platform builds with CGO_ENABLED
# I had to first compile without buildx for buildx to then work
ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64

COPY . /build/mqtt2prometheus
WORKDIR /build/mqtt2prometheus
RUN make static_build TARGET_FILE=/bin/mqtt2prometheus

FROM gcr.io/distroless/static-debian10:nonroot
WORKDIR /
COPY --from=builder /bin/mqtt2prometheus /mqtt2prometheus
ENTRYPOINT ["/mqtt2prometheus"]
