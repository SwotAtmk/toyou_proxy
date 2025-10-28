#!/bin/bash

# Toyou Proxy 简化启动脚本

# 检查并构建可执行文件
if [ ! -f "toyou-proxy" ]; then
    echo "构建 Toyou Proxy..."
    go build -o toyou-proxy cmd/main.go
fi

# 启动代理服务器
echo "启动 Toyou Proxy 服务器..."
echo "监听端口: 80"
./toyou-proxy