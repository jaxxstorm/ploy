FROM golang:buster as builder
WORKDIR /app
COPY . .
ENV CGO_ENABLED=0
RUN go mod download \
    && go build -o /app/webserver

# Runtime container
FROM scratch
WORKDIR /app
ENV LISTEN_PORT=80
COPY --from=builder /app/webserver /app/webserver

ENTRYPOINT [ "/app/webserver" ]
