#build stage
FROM golang:1.26-alpine3.23 AS builder
RUN apk add --no-cache git upx
WORKDIR /app
COPY ["go.mod", "go.sum", "./"]
RUN go mod download
COPY . .
RUN go build -o app ./cmd/server
RUN upx app

#final stage
FROM alpine:3.23.3
RUN apk update && apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app .
EXPOSE 3000
ENTRYPOINT ["./app"]
