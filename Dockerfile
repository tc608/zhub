# 阶段一：构建阶段
FROM golang:alpine AS builder

WORKDIR /opt/zhub

# 复制 go.mod 文件
COPY go.mod .

# 下载并缓存依赖包
RUN go mod download

# 将源代码复制到容器中
COPY . .

# 运行 go get 命令以确保所有需要的包都被获取和更新
#RUN go get zhub/internal/monitor
#RUN go get zhub/cmd
#RUN go get zhub/internal/zsub
#RUN go get zhub/internal/config

# 构建可执行文件
RUN go build -o zhub.sh -ldflags "-s -w"


# 阶段二：运行阶段
FROM alpine:latest

# 设置工作目录
WORKDIR /opt/zhub

# 从构建阶段复制可执行文件到当前阶段
COPY --from=builder /opt/zhub .

# 复制 app.ini 配置文件到容器中
COPY app.ini .

EXPOSE 711 1216

# 设置容器启动时要执行的命令
CMD ["./zhub.sh"]
