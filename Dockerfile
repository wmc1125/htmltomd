# 使用更新的Go镜像作为基础镜像
FROM golang:1.20-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制go mod和sum文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# 使用轻量级的alpine镜像
FROM alpine:latest  

# 从builder阶段复制编译好的应用
COPY --from=builder /app/main /main

# 暴露8080端口
EXPOSE 8080

# 运行应用
CMD ["/main"]