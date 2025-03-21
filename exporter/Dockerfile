# two-part dockerfile that first builds the Go app which is the Prometheus exporter that collects the metrics for Prometheus and secondly creates the image
FROM docker.io/golang:latest AS builder
RUN mkdir /app
WORKDIR /app
COPY . /app
# creates an executable called exporter (because we told it to with -o flag)
RUN CGO_ENABLED=0 GOOS=linux go build -o exporter -ldflags="-s -w"

FROM docker.io/alpine:latest
# creating a user picotempexport with home directory /app (-h) and no password (-D)
RUN mkdir /app && adduser -h /app -D picotempexport
WORKDIR /app
# copying from builder app/exporter with new owner picotempexport
COPY --chown=picotempexport --from=builder /app/exporter .
EXPOSE 3030
CMD ["/app/exporter"]