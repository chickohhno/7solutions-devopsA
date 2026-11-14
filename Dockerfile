FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /hello-api .

FROM alpine:3.22

COPY --from=builder /hello-api /hello-api

RUN addgroup -S app && adduser -S app -G app
USER app

EXPOSE 8080

CMD ["/hello-api"]