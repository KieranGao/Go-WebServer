
FROM docker.xuanyuan.me/library/golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o webserver ./cmd/server/

FROM docker.xuanyuan.me/library/alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /app/webserver .
COPY --from=builder /app/config-docker.conf config.conf
COPY --from=builder /app/resources ./resources

EXPOSE 9999

ENTRYPOINT ["./webserver"]
CMD ["-config", "config.conf"]
