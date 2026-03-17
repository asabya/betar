FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build deps
RUN apk add --no-cache git make

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/bin/betar /usr/local/bin/betar
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

EXPOSE 4001 8424

ENTRYPOINT ["betar"]
CMD ["start"]
