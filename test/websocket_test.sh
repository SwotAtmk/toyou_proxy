#!/bin/bash

# WebSocket代理测试脚本

echo "=== WebSocket代理测试开始 ==="

# 设置颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 项目根目录
PROJECT_DIR=$(cd "$(dirname "$0")/.." && pwd)
TEST_DIR=$PROJECT_DIR/test

echo "项目目录: $PROJECT_DIR"
echo "测试目录: $TEST_DIR"

# 检查依赖
echo -e "${YELLOW}检查依赖...${NC}"
cd $PROJECT_DIR
if ! go mod tidy; then
    echo -e "${RED}依赖检查失败${NC}"
    exit 1
fi
echo -e "${GREEN}依赖检查完成${NC}"

# 构建代理服务器
echo -e "${YELLOW}构建代理服务器...${NC}"
if ! go build -o bin/toyou-proxy ./cmd/main.go; then
    echo -e "${RED}代理服务器构建失败${NC}"
    exit 1
fi
echo -e "${GREEN}代理服务器构建完成${NC}"

# 构建测试服务器
echo -e "${YELLOW}构建测试服务器...${NC}"
if ! go build -o bin/websocket-test-server ./test/websocket/websocket_test_server.go; then
    echo -e "${RED}测试服务器构建失败${NC}"
    exit 1
fi
echo -e "${GREEN}测试服务器构建完成${NC}"

# 构建测试客户端
echo -e "${YELLOW}构建测试客户端...${NC}"
if ! go build -o bin/websocket-test-client ./test/websocket/websocket_test_client.go; then
    echo -e "${RED}测试客户端构建失败${NC}"
    exit 1
fi
echo -e "${GREEN}测试客户端构建完成${NC}"

# 启动测试服务器
echo -e "${YELLOW}启动WebSocket测试服务器...${NC}"
./bin/websocket-test-server &
TEST_SERVER_PID=$!
echo "测试服务器PID: $TEST_SERVER_PID"
sleep 2

# 检查测试服务器是否启动成功
if ! kill -0 $TEST_SERVER_PID 2>/dev/null; then
    echo -e "${RED}测试服务器启动失败${NC}"
    exit 1
fi
echo -e "${GREEN}测试服务器启动成功${NC}"

# 启动代理服务器
echo -e "${YELLOW}启动代理服务器...${NC}"
./bin/toyou-proxy -config $TEST_DIR/websocket/websocket_test_config.yaml &
PROXY_SERVER_PID=$!
echo "代理服务器PID: $PROXY_SERVER_PID"
sleep 3

# 检查代理服务器是否启动成功
if ! kill -0 $PROXY_SERVER_PID 2>/dev/null; then
    echo -e "${RED}代理服务器启动失败${NC}"
    kill $TEST_SERVER_PID 2>/dev/null
    exit 1
fi
echo -e "${GREEN}代理服务器启动成功${NC}"

# 运行测试
echo -e "${YELLOW}运行WebSocket代理测试...${NC}"
sleep 2

# 测试1: 直接连接测试服务器
echo -e "${YELLOW}测试1: 直接连接测试服务器${NC}"
./bin/websocket-test-client "ws://localhost:8081/ws"
echo -e "${GREEN}测试1完成${NC}"

sleep 2

# 测试2: 通过代理连接测试服务器
echo -e "${YELLOW}测试2: 通过代理连接测试服务器${NC}"
./bin/websocket-test-client "ws://localhost:8080/ws"
echo -e "${GREEN}测试2完成${NC}"

# 清理
echo -e "${YELLOW}清理进程...${NC}"
kill $TEST_SERVER_PID 2>/dev/null
kill $PROXY_SERVER_PID 2>/dev/null
echo -e "${GREEN}进程清理完成${NC}"

echo -e "${GREEN}=== WebSocket代理测试完成 ===${NC}"