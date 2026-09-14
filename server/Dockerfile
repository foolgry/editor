# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装依赖（git 和 gcc 是必需的，gcc 用于 CGO）
RUN apk add --no-cache git gcc musl-dev

# 复制依赖文件
COPY go.mod go.sum ./

# Go 依赖下载代理（优先国内镜像，失败时回退官方）
ENV GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.google.cn

RUN go mod download

# 复制源码（包内含多个 go 文件，整目录复制）
COPY *.go .

# 构建
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o server .

# 运行阶段
FROM alpine:latest

WORKDIR /app

# 安装 SQLite 依赖
RUN apk add --no-cache ca-certificates sqlite-libs

# 复制可执行文件
COPY --from=builder /app/server .

# 创建数据目录
RUN mkdir -p data
RUN mkdir -p frontend

# 暴露端口
EXPOSE 8080

# 运行（环境变量通过 docker compose env_file 或 -e 传入）
CMD ["./server"]
