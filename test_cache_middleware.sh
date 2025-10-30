#!/bin/bash

# 缓存中间件测试脚本

# 设置测试变量
BASE_URL="http://localhost:8080"
TEST_URL="/api/test"

echo "开始测试缓存中间件功能..."

# 测试1: 首次请求（应该是缓存未命中）
echo "测试1: 首次请求（缓存未命中）"
curl -i -X GET "${BASE_URL}${TEST_URL}" -H "Cache-Control: no-cache" 2>/dev/null | head -20

# 等待一秒
sleep 1

# 测试2: 第二次请求（应该是缓存命中）
echo -e "\n\n测试2: 第二次请求（缓存命中）"
curl -i -X GET "${BASE_URL}${TEST_URL}" 2>/dev/null | head -20

# 测试3: 带有特定头的请求
echo -e "\n\n测试3: 带有特定头的请求"
curl -i -X GET "${BASE_URL}${TEST_URL}" -H "X-Custom-Header: test-value" 2>/dev/null | head -20

# 等待一秒
sleep 1

# 测试4: 相同URL但不同头的请求（应该是缓存未命中）
echo -e "\n\n测试4: 相同URL但不同头的请求（缓存未命中）"
curl -i -X GET "${BASE_URL}${TEST_URL}" -H "X-Custom-Header: different-value" 2>/dev/null | head -20

# 测试5: POST请求（不应该被缓存）
echo -e "\n\n测试5: POST请求（不应该被缓存）"
curl -i -X POST "${BASE_URL}${TEST_URL}" -d "test=data" 2>/dev/null | head -20

# 等待一秒
sleep 1

# 测试6: 再次GET请求（应该仍然是缓存命中）
echo -e "\n\n测试6: 再次GET请求（应该仍然是缓存命中）"
curl -i -X GET "${BASE_URL}${TEST_URL}" 2>/dev/null | head -20

echo -e "\n\n测试完成！"
echo "请检查缓存目录内容："
ls -la cache/responses/ 2>/dev/null || echo "缓存目录不存在"