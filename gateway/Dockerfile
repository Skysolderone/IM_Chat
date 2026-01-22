FROM golang:1.25-alpine AS builder

WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建网关服务
WORKDIR /app/gateway
RUN CGO_ENABLED=0 GOOS=linux go build -o gateway .

# 最终镜像
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app

# 创建日志目录
RUN mkdir -p /app/logs

# 从构建阶段复制二进制文件
COPY --from=builder /app/gateway/gateway .

# 复制配置文件
COPY --from=builder /app/config ./config

EXPOSE 8085

CMD ["./gateway"]
