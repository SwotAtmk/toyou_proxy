#!/bin/bash

# Toyou Proxy 构建脚本

echo "Building Toyou Proxy..."

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    exit 1
fi

# 清理之前的构建
rm -f toyou-proxy

# 构建主程序
echo "Building main application..."
go build -o toyou-proxy cmd/main.go

if [ $? -eq 0 ]; then
    echo "Build successful!"
    echo "Executable: ./toyou-proxy"
    
    # 设置执行权限
    chmod +x toyou-proxy
    
    # 显示使用说明
    echo ""
    echo "Usage:"
    echo "  ./toyou-proxy                    # 使用默认配置"
    echo "  ./toyou-proxy -config config.yaml # 指定配置文件"
    echo "  ./toyou-proxy -help              # 显示帮助"
else
    echo "Build failed!"
    exit 1
fi