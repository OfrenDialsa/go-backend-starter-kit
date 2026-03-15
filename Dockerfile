FROM golang:1.26.1-alpine AS builder

ENV GIN_MODE=release

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main
RUN go build -o migrate ./cmd/migrate

FROM alpine:edge AS runner

ENV GIN_MODE=release

WORKDIR /app

RUN mkdir -p /app/data /app/env

COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

COPY --from=builder /app/main .
COPY --from=builder /app/migrate .
COPY --from=builder /app/migrations ./migrations

RUN addgroup 1000 && \
    adduser  -G 1000 -D -s /bin/sh 1000 && \
    chown -R 1000:1000 /app &&\
    chmod +x entrypoint.sh

RUN mkdir -p /app/license && chown -R 1000:1000 /app/license

USER 1000

ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["/app/main"]