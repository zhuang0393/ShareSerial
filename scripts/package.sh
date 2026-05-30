#!/bin/bash
# ShareSerial 项目打包脚本

VERSION=1.0.0
PACKAGE_NAME=shareserial-v${VERSION}

set -e

echo "=== ShareSerial 打包脚本 ==="

# 创建临时目录
TMP_DIR=/tmp/${PACKAGE_NAME}
rm -rf ${TMP_DIR}
mkdir -p ${TMP_DIR}

# 复制必要文件
echo "复制文件..."

# 可执行文件
cp -r bin/ ${TMP_DIR}/

# 配置文件
cp -r configs/ ${TMP_DIR}/

# 启动脚本
cp -r scripts/ ${TMP_DIR}/

# 文档
cp README.md ${TMP_DIR}/
cp DEPLOY.md ${TMP_DIR}/
cp DEPLOY-SIMPLE.md ${TMP_DIR}/

# Skill 文件（可选，用于 AI 辅助）
mkdir -p ${TMP_DIR}/.claude/skills
cp .claude/skills/shareserial-*.md ${TMP_DIR}/.claude/skills/

# Memory 文件（可选）
mkdir -p ${TMP_DIR}/memory
cp /home/hengzhuang.jin/.claude/projects/-workspace-hengzhuang-jin-ss/memory/*.md ${TMP_DIR}/memory/

# Makefile（可选，用于重新构建）
cp Makefile ${TMP_DIR}/

# 设置权限
chmod +x ${TMP_DIR}/scripts/*.sh
chmod +x ${TMP_DIR}/bin/*

# 打包
echo "打包..."
cd /tmp
tar -czvf ${PACKAGE_NAME}.tar.gz ${PACKAGE_NAME}

# 清理临时目录
rm -rf ${TMP_DIR}

# 显示结果
echo ""
echo "=== 打包完成 ==="
echo "文件: /tmp/${PACKAGE_NAME}.tar.gz"
echo "大小: $(du -h /tmp/${PACKAGE_NAME}.tar.gz | cut -f1)"
echo ""
echo "传输到 Server 机器:"
echo "  scp /tmp/${PACKAGE_NAME}.tar.gz user@server-ip:/tmp/"
echo ""
echo "Server 端部署:"
echo "  tar -xzvf ${PACKAGE_NAME}.tar.gz"
echo "  cd ${PACKAGE_NAME}"
echo "  ./scripts/deploy.sh server"
echo ""
echo "Client 端部署:"
echo "  ./scripts/deploy.sh client <SERVER_IP>"