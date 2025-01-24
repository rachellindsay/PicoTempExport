FROM docker.io/golang:latest AS builder
RUN mkdir /app
WORKDIR /app
COPY . /app
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w"

FROM docker.io/alpine:latest
RUN mkdir /app && adduser -h /app -D picotempexport.go
WORKDIR /app
COPY --chown=picotempexport --from=builder /app/picotempexport .
EXPOSE 3030
CMD ["/app/picotempexport"]