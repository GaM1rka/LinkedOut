FROM golang:1.24-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates build-base

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

ARG SERVICE=bot-service

RUN test -d "./cmd/${SERVICE}" || (echo "Service ./cmd/${SERVICE} not found" && ls -la ./cmd && exit 1)

RUN CGO_ENABLED=1 GOOS=linux go build -tags netcgo -o /out/app ./cmd/${SERVICE}

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/app /app/app
COPY migrations /app/migrations

EXPOSE 8080

ENTRYPOINT ["/app/app"]