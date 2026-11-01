# ---------- 构建阶段 ----------
FROM golang:1.27-alpine AS builder

# 国内网络可改为 https://goproxy.cn,direct
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY} \
    CGO_ENABLED=0 \
    GOOS=linux

WORKDIR /src

# 先拷贝依赖清单, 利用镜像层缓存
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags "-s -w" -o /out/main .

# ---------- 运行阶段 ----------
FROM alpine:3.20

ENV TZ=Asia/Shanghai \
    WORKDIR=/app

RUN apk add --no-cache tzdata ca-certificates && \
    mkdir -p ${WORKDIR}/manifest/config

WORKDIR ${WORKDIR}

COPY --from=builder /out/main ./main
COPY resource ./resource
# 默认配置(Docker 场景: mysql/redis 使用容器服务名), 生产请挂载自己的配置覆盖
COPY manifest/docker/config.compose.yaml ./manifest/config/config.yaml

EXPOSE 8000

CMD ["./main"]
