#build stage
FROM golang:1.26-alpine3.23 AS builder
RUN apk add --no-cache git upx
WORKDIR /app
COPY ["go.mod", "go.sum", "./"]
RUN go mod download
COPY . .
# -tags timetzdata embebe la base de datos IANA de zonas horarias en el
# binario: la imagen final es "scratch" (sin sistema operativo), así que
# time.LoadLocation("America/Caracas") no encuentra /usr/share/zoneinfo en
# runtime y bizclock cae a UTC en silencio (desfasando 4h los horarios de
# los locales). Con este tag el binario trae su propio zoneinfo embebido.
RUN CGO_ENABLED=0 go build -tags timetzdata -o app ./cmd/server
RUN upx app

#final stage
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/app ./app
EXPOSE 3000
USER 65534
ENTRYPOINT ["./app"]
