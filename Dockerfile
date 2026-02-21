# 第一阶段：构建Go应用
FROM golang:1.24.3-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制go.mod和go.sum文件并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制所有源代码
COPY . .

# 构建应用（关闭CGO以确保静态链接）
RUN CGO_ENABLED=0 GOOS=linux go build -o coder_edu_backend .

# 第二阶段：创建轻量级镜像
FROM alpine:3.19

# 安装必要的包
RUN apk --no-cache add ca-certificates

# 创建非root用户运行应用
RUN adduser -D appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/coder_edu_backend .

# 复制不含密钥的配置模板，实际敏感值通过环境变量注入
COPY configs/config.yaml.example ./configs/config.yaml

# 创建uploads目录用于存储文件
RUN mkdir -p uploads && chown -R appuser:appuser uploads

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8080

# 启动应用
CMD ["./coder_edu_backend"]