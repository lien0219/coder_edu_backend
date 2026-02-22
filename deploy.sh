#!/bin/bash
# ============================================
# 一键部署脚本 - Coder Edu Backend (Linux/Mac)
# 普通发布: bash deploy.sh
# 含数据库迁移: bash deploy.sh --migrate
# 配置: 复制 deploy.env.example 为 deploy.env 并填入真实值
# ============================================

set -e

# 解析参数
MIGRATE=false
for arg in "$@"; do
    case $arg in
        --migrate) MIGRATE=true ;;
    esac
done

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="$SCRIPT_DIR/deploy.env"

# 读取配置
if [ ! -f "$ENV_FILE" ]; then
    echo "未找到 deploy.env 配置文件！"
    echo "请先复制 deploy.env.example 为 deploy.env 并填入真实值"
    exit 1
fi

source "$ENV_FILE"
SERVER="$DEPLOY_SERVER"
REMOTE_PATH="$DEPLOY_PATH"
SERVICE_NAME="$DEPLOY_SERVICE"

echo ""
echo "========================================"
echo "  Coder Edu Backend 一键部署"
echo "  服务器: $SERVER"
if [ "$MIGRATE" = true ]; then
    echo "  [含数据库迁移]"
fi
echo "========================================"
echo ""

# 1. 本地编译
echo "[1/4] 编译 Linux 版本..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o coder_edu_backend .
echo "编译成功!"

# 2. 上传
echo ""
echo "[2/4] 上传到服务器..."
scp -o ConnectTimeout=10 ./coder_edu_backend "${SERVER}:${REMOTE_PATH}/coder_edu_backend.new"
echo "上传成功!"

# 3. 替换文件
echo ""
echo "[3/4] 替换文件..."
ssh -o ConnectTimeout=10 $SERVER "cd $REMOTE_PATH && cp coder_edu_backend coder_edu_backend.bak 2>/dev/null; systemctl stop $SERVICE_NAME; mv coder_edu_backend.new coder_edu_backend && chmod +x coder_edu_backend"

# 如果需要数据库迁移
if [ "$MIGRATE" = true ]; then
    echo ""
    echo "[*] 执行数据库迁移..."
    ssh -o ConnectTimeout=30 $SERVER "cd $REMOTE_PATH && ./coder_edu_backend --migrate-only 2>&1"
    if [ $? -ne 0 ]; then
        echo "数据库迁移失败！正在回滚..."
        ssh $SERVER "cd $REMOTE_PATH && mv coder_edu_backend.bak coder_edu_backend && systemctl start $SERVICE_NAME"
        echo "已回滚到上一个版本"
        exit 1
    fi
    echo "数据库迁移完成!"
fi

# 启动服务
echo ""
echo "[*] 启动服务..."
ssh $SERVER "systemctl start $SERVICE_NAME && sleep 2 && systemctl is-active $SERVICE_NAME"
echo "服务启动成功!"

# 4. 健康检查
echo ""
echo "[4/4] 健康检查..."
sleep 2
HEALTH=$(ssh $SERVER "curl -s http://localhost/api/health")

if echo "$HEALTH" | grep -q '"status":"ok"'; then
    echo ""
    echo "========================================"
    echo "  部署成功!"
    echo "========================================"
    ssh $SERVER "rm -f $REMOTE_PATH/coder_edu_backend.bak"
else
    echo "健康检查异常，正在回滚..."
    ssh $SERVER "cd $REMOTE_PATH && mv coder_edu_backend.bak coder_edu_backend && systemctl restart $SERVICE_NAME"
    echo "已回滚到上一个版本"
    exit 1
fi

# 清理
rm -f ./coder_edu_backend
echo ""
