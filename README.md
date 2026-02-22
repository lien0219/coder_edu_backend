# Coder Edu Backend

[![Go Version](https://img.shields.io/github/go-mod/go-version/coder-edu/coder_edu_backend?label=Go)](https://golang.org/)
[![Framework](https://img.shields.io/badge/Framework-Gin-blue)](https://gin-gonic.com/)
[![ORM](https://img.shields.io/badge/ORM-GORM-blue)](https://gorm.io/)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

## 项目简介

Coder Edu Backend 是一个基于 Go 语言的高性能教育平台后端服务。它为学习者提供全方位的支持，包括用户认证、学习资源管理、个性化学习路径、**AI 驱动的智能助教**、实时反馈以及完善的成就系统。

### 核心功能模块

1. **仪表盘 (Dashboard)**：实时展示今日任务、学习进度、成就预览及学习数据快照。
2. **课前准备 (Pre-class)**：包含学习诊断、目标设定、生成个性化学习路径及资源包准备。
3. **课中学习 (In-class)**：支持任务链驱动、实时学习反馈、在线协作及集成式代码编辑器（基于 Judge0）。
4. **课后回顾 (Post-class)**：记录学习日志、阶段性测验、知识迁移任务及反思指南。
5. **成就系统 (Achievement)**：丰富的徽章墙、积分等级、全球排行榜及目标管理。
6. **协作中心 (Collaboration)**：支持基于 WebSocket 的实时聊天室、私聊、群聊及资源分享。
7. **AI 智能助教 (AI Assistant)**：基于大语言模型的流式问答、RAG 知识库检索、多轮对话、代码自动诊断及 AI 学习周报生成。
8. **深度分析 (Analytics)**：多维度技能评估、学习曲线分析及智能个性化建议。
9. **编程资源管理**：专门针对 C 语言等编程资源的存储（支持 OSS/MinIO/本地）与分发。

### 特色技术实现

- **AI 智能助教系统**：
  - **SSE 流式问答**：基于 Server-Sent Events 实现逐字输出的实时对话体验，支持多轮上下文记忆（自动截取最近 5 轮历史）。
  - **RAG 知识库检索**：结合 MySQL 全文索引与意图识别引擎（知识/练习/进度/社区四大意图），在调用大模型前自动检索知识点、练习题、学习进度、社区帖子等本地数据作为上下文，实现"知识库优先、大模型兜底"的混合回答策略。
  - **AI 代码自动诊断**：学生提交代码失败后，系统自动将题目背景、用户代码和编译器报错整合为 Prompt，以启发式引导（不直接给答案）帮助学生定位 Bug。
  - **AI 学习周报生成**：自动聚合用户一周内的学习进度、练习提交正确率、社区活跃度等数据，调用大模型生成包含学习概况、技术亮点、薄弱环节及下周计划的个性化周报。
  - **AI 自动标签生成**：离线脚本批量调用大模型为知识点与练习题自动提取关键词标签，降低运营人工成本。
  - **安全与性能防护**：内置敏感词过滤、Redis 滑动窗口限流（每分钟 10 次）、高频问题 Redis 缓存（5 分钟 TTL）及 Markdown 格式规范注入。
- **智能验证码策略**：支持"15天内免验证"信任设备机制，平衡安全与体验。
- **大文件分片上传**：针对教学视频支持分片上传及进度查询，确保大文件传输稳定性。
- **在线代码执行**：集成 Judge0 API，支持多种编程语言的在线运行与自动化评测。
- **实时通信架构**：基于 Gorilla WebSocket 构建高性能聊天系统，支持消息撤回、已读确认。
- **多端存储支持**：统一存储接口，支持本地文件系统、MinIO 及阿里云 OSS。

## 技术栈

- **核心框架**: [Go](https://golang.org/) 1.24+ / [Gin](https://gin-gonic.com/)
- **数据持久化**: [MySQL](https://www.mysql.com/) / [GORM](https://gorm.io/) / [Redis](https://redis.io/)
- **对象存储**: [MinIO](https://min.io/) / [Aliyun OSS](https://www.aliyun.com/product/oss)
- **实时通信**: [Gorilla WebSocket](https://github.com/gorilla/websocket)
- **代码执行**: [Judge0](https://judge0.com/)
- **AI 能力**: LLM 大语言模型 (OpenAI 兼容接口) + RAG 检索增强 + SSE 流式输出
- **配置管理**: [Viper](https://github.com/spf13/viper)
- **日志系统**: [Zap](https://github.com/uber-go/zap) + [Lumberjack](https://github.com/natefinch/lumberjack)
- **身份认证**: [JWT](https://github.com/dgrijalva/jwt-go) (带设备信任机制)
- **API 文档**: [Swagger](https://github.com/swaggo/swag)
- **多媒体处理**: [FFmpeg](https://ffmpeg.org/)
- **监控与追踪**: [Prometheus](https://prometheus.io/) (集成 `/metrics`)

## 权限角色说明

- **学生 (Student)**: 参与课程、提交作业、练习代码、查看进度、参与社区讨论与协作。
- **教师 (Teacher)**: 管理关卡、发布任务、批改作业、查看学生进度并提供个性化建议。
- **管理员 (Admin)**: 拥有最高权限，包括用户管理、系统配置、全局资源审核及激励机制设置。

## 项目结构

```text
├── api/                      # Swagger 接口定义
│   └── swagger/
├── configs/
│   ├── config.yaml           # 配置文件（.gitignore，不提交）
│   └── config.yaml.example   # 配置模板
├── docs/                     # Swagger 生成文档
├── internal/                 # 核心业务逻辑
│   ├── app/                  # 应用启动与路由注册
│   ├── config/               # 配置加载逻辑
│   ├── controller/           # 接口控制器 (25 个 Handlers)
│   ├── middleware/            # 鉴权、日志等中间件
│   ├── model/                # 数据库模型与定义 (39 个模型)
│   ├── repository/           # 数据访问层 (26 个 DAO)
│   ├── service/              # 业务逻辑层 (28 个 Service)
│   └── util/                 # 工具函数 (JWT, Response, FFmpeg 等)
├── pkg/                      # 公共包
│   ├── database/             # MySQL / Redis 连接
│   ├── logger/               # Zap 日志初始化
│   ├── monitoring/           # Prometheus 监控
│   ├── security/             # 安全工具
│   └── tracing/              # 分布式追踪
├── scripts/                  # 工具脚本
│   ├── auto_tagging.go       # AI 自动标签生成
│   ├── secrets_handler.py    # 敏感信息加密/解密
│   ├── generate_swagger.bat  # Swagger 生成 (Windows)
│   └── generate_swagger.sh   # Swagger 生成 (Linux/macOS)
├── main.go                   # 项目启动文件
├── Dockerfile                # Docker 镜像构建（支持本地编译部署）
├── docker-compose.yml        # 容器编排（.gitignore，不提交）
├── .env.example              # Docker Compose 环境变量模板
├── nginx.conf                # Nginx 反向代理配置
├── deploy.ps1                # 一键部署脚本 (Windows PowerShell)
├── deploy.sh                 # 一键部署脚本 (Linux/macOS)
├── rollback.ps1              # 一键回滚脚本 (Windows PowerShell)
├── deploy.env                # 部署配置（.gitignore，不提交）
└── deploy.env.example        # 部署配置模板
```

## 快速开始

### 环境准备

- 安装 Go 1.24+
- 安装 MySQL 8.0+
- 安装 Redis 6.0+
- (可选) 安装 MinIO 或开通阿里云 OSS
- (可选) 获取 [Judge0 API Key](https://rapidapi.com/judge0-official/api/judge0-ce)

### 配置应用

复制 `configs/config.yaml.example` 为 `configs/config.yaml`，填入真实值：

```bash
cp configs/config.yaml.example configs/config.yaml
```

主要配置项说明：

```yaml
server:
  port: 8080        # 服务端口
  mode: "debug"     # 运行模式: debug/release

database:           # MySQL 配置
  host: "localhost"
  dbname: "coder_edu_backend"

jwt:
  secret: "your-secret-key"   # ⚠️ release 模式下必须 ≥32 字符
  expire_hours: 12

storage:
  type: "oss"       # 存储类型: local/minio/oss
  oss_endpoint: "..."
  oss_access_key: "..."

redis:              # Redis 配置 (用于限流和缓存)
  host: "localhost"
  port: 6379

judge0:             # 代码评测配置
  api_key: "..."
  url: "..."

ai:                 # AI 大模型配置 (智能助教、代码诊断、周报、自动标签)
  base_url: "https://your-ai-api-endpoint"
  api_key: "your-api-key"
  model: "your-model-name"
```

> **⚠️ JWT Secret 安全要求**：当 `server.mode` 为 `release` 时，系统会在启动时校验 `jwt.secret` 长度不少于 32 个字符，不满足条件将拒绝启动。建议使用 `openssl rand -base64 48` 生成强密钥。

### 运行应用

1. **克隆并安装依赖**
   ```bash
   git clone https://github.com/lien0219/coder_edu_backend.git
   cd coder_edu_backend
   go mod tidy
   ```

2. **直接运行**
   ```bash
   go run main.go
   ```

3. **使用热重载 (推荐)**
   ```bash
   air
   ```

### Docker 运行

1. 复制环境变量模板并填入真实值：
   ```bash
   cp .env.example .env
   ```

2. 启动服务：
   ```bash
   docker-compose up -d
   ```

> **注意**：`docker-compose.yml` 和 `.env` 均不会提交到 Git，敏感信息安全。

## 开发者指南

### API 文档查看

应用启动后，可以通过以下地址访问交互式 Swagger 文档：
- Swagger UI: `http://localhost:<port>/swagger/index.html`
- 指标监控: `http://localhost:<port>/metrics` (Prometheus 格式)
- 健康检查: `http://localhost:<port>/api/health`

## 工具脚本

项目 `scripts/` 目录下提供了多种开发辅助脚本：

### 自动标签生成 (`scripts/auto_tagging.go`)

该脚本利用 AI 大模型为数据库中的 **知识点 (KnowledgePoint)** 和 **练习题 (ExerciseQuestion)** 自动生成关键词标签，免去人工逐条打标签的繁琐操作。

#### 工作原理

1. 读取 `configs/config.yaml` 中的数据库和 AI 服务配置。
2. 连接数据库，查询所有知识点和练习题记录。
3. 对每条记录，将标题和内容/描述拼接为 Prompt，调用 AI 服务提取 3-5 个核心关键词标签。
4. 将生成的标签输出到控制台（可根据需求扩展为写回数据库）。

#### 前置条件

- 已正确配置 `configs/config.yaml` 中的 `database` 和 `ai` 部分：
  ```yaml
  ai:
    base_url: "https://your-ai-api-endpoint"
    api_key: "your-api-key"
  ```
- 数据库中已存在知识点或练习题数据。

#### 使用方式

在项目根目录下执行：

```bash
go run scripts/auto_tagging.go
```

运行后终端将输出类似：

```text
开始为 12 个知识点和 8 个练习题自动生成标签...
知识点 [C语言指针基础] -> 标签: 指针,内存地址,解引用,C语言,变量
练习题 [链表反转] -> 标签: 链表,反转,指针操作,数据结构
...
自动打标签任务完成！
```

### 敏感信息管理

项目采用多层安全策略保护敏感信息：

**1. 文件级隔离**：`configs/config.yaml`、`docker-compose.yml`、`.env`、`deploy.env` 均在 `.gitignore` 中，不会提交到仓库。仓库中只保留 `.example` 模板文件。

**2. Mask/Unmask 脚本** (`scripts/secrets_handler.py`)：

```bash
# 提交前加密敏感信息（将密码替换为 ******）
python scripts/secrets_handler.py mask

# 拉取代码后恢复敏感信息
python scripts/secrets_handler.py unmask
```

> 敏感值存储在 `.secrets.json` 中（已加入 `.gitignore`），仅保留在本地。

### Swagger 文档生成

```bash
# Windows
./scripts/generate_swagger.bat

# Linux/macOS
./scripts/generate_swagger.sh
```

---

## 生产环境部署

### 一键部署（推荐）

项目提供了自动化部署脚本，在本地交叉编译后上传到服务器，无需在服务器上安装 Go 环境。

#### 首次配置

```bash
cp deploy.env.example deploy.env
```

编辑 `deploy.env` 填入服务器信息：

```env
DEPLOY_SERVER=root@your-server-ip
DEPLOY_PATH=/opt/coder_edu_backend
DEPLOY_SERVICE=coder_edu
HEALTH_CHECK_URL=http://your-server-ip/api/health
```

#### 部署

```powershell
# Windows PowerShell
.\deploy.ps1
```

```bash
# Linux / macOS
bash deploy.sh
```

脚本自动执行以下步骤：

1. 本地交叉编译 Linux amd64 二进制文件
2. SCP 上传到服务器
3. 备份旧版本、停止服务、替换文件、启动服务
4. 健康检查确认部署成功

#### 回滚

```powershell
# 回滚到上一个版本
.\rollback.ps1
```

> **服务器要求**：阿里云 ECS 2 核 2G 即可运行（MySQL + Redis 使用 Docker，后端直接运行）。

### 环境变量参考

所有配置项均可通过环境变量覆盖 `config.yaml` 中的值（优先级更高），推荐在 Docker / K8s 环境中使用：

| 环境变量 | 对应配置项 | 说明 |
|---------|-----------|------|
| `DATABASE_HOST` | `database.host` | MySQL 主机 |
| `DATABASE_PORT` | `database.port` | MySQL 端口 |
| `DATABASE_USER` | `database.user` | MySQL 用户名 |
| `DATABASE_PASSWORD` | `database.password` | MySQL 密码 |
| `DATABASE_NAME` | `database.dbname` | 数据库名 |
| `JWT_SECRET` | `jwt.secret` | JWT 签名密钥 (release ≥32 字符) |
| `REDIS_HOST` | `redis.host` | Redis 主机 |
| `REDIS_PORT` | `redis.port` | Redis 端口 |
| `REDIS_PASSWORD` | `redis.password` | Redis 密码 |
| `SERVER_MODE` | `server.mode` | 运行模式 (`debug` / `release`) |
| `STORAGE_TYPE` | `storage.type` | 存储类型 (`local` / `minio` / `oss`) |
| `OSS_ENDPOINT` | `storage.oss_endpoint` | 阿里云 OSS Endpoint |
| `OSS_ACCESS_KEY` | `storage.oss_access_key` | OSS AccessKey |
| `OSS_SECRET_KEY` | `storage.oss_secret_key` | OSS SecretKey |
| `OSS_BUCKET` | `storage.oss_bucket` | OSS Bucket |
| `JUDGE0_API_KEY` | `judge0.api_key` | Judge0 API Key |
| `AI_BASE_URL` | `ai.base_url` | AI 服务地址 |
| `AI_API_KEY` | `ai.api_key` | AI API Key |
| `AI_MODEL` | `ai.model` | AI 模型名称 |

### 优雅关机

应用接收到 `SIGINT` / `SIGTERM` 信号后，会按以下顺序执行清理：

1. 停止后台定时任务
2. 清理 WebSocket 连接和 Redis 在线状态
3. 关闭 HTTP 服务（等待进行中的请求完成，超时 5 秒）
4. 关闭分布式追踪 Provider
5. 关闭 MySQL 数据库连接
6. 关闭 Redis 连接
7. 刷写日志缓冲区

### 注册安全策略

注册接口 (`POST /api/register`) 仅允许选择 `student` 或 `teacher` 角色。管理员账号需通过后台或数据库直接创建，无法通过公开注册获取 admin 权限。

### CORS 与 Cookie 部署方案

为了确保"15天内免验证"等功能在生产环境下安全、稳定地运行，建议在部署时参考以下方案配置 CORS 和 Cookie。

### 方案一：相同主域共享 Cookie（最推荐）

将前后端部署在同一个二级域名下。
- **前端**: `https://www.example.com`
- **后端**: `https://api.example.com`

#### 配置要点
1. **后端设置 Cookie 域名**: 在 `Set-Cookie` 时，将 `Domain` 设置为 `.example.com`。
2. **Cookie 属性**:
   - `HttpOnly: true` (防止脚本读取)
   - `Secure: true` (HTTPS 强制要求)
   - `SameSite: Lax` (同主域下兼容性最好)

---

### 方案二：完全跨域部署

如果前后端域名完全不同（如 `frontend.com` 和 `backend.com`）。

#### 配置要点
1. **全站 HTTPS**: 跨站 Cookie 必须配合 `Secure: true`，这要求必须使用 HTTPS。
2. **后端 Cookie 属性**:
   - `SameSite: None` (允许跨站发送)
   - `Secure: true` (必须开启，否则 None 不生效)
3. **CORS 严格校验**:
   - `Access-Control-Allow-Origin` 必须指定精确域名，不能用 `*`。
   - `Access-Control-Allow-Credentials` 必须为 `true`。

---

### 方案三：Nginx 反向代理（最简单）

使用 Nginx 将前后端统一到一个域名下，避开所有跨域问题：
```nginx
server {
    listen 443 ssl;
    server_name example.com;

    location / {
        proxy_pass http://frontend_server; # 前端
    }

    location /api/ {
        proxy_pass http://backend_server;  # 后端
    }
}
```
**优点**：浏览器视为同源，Cookie 传递最顺滑。

### 后端 Cookie 安全策略

项目已实现 **Cookie `Secure` 标志自动适配**：当 `server.mode` 设为 `release` 时，`trust_device_token` Cookie 会自动启用 `Secure: true`；开发模式下则为 `false`，方便本地调试。

如需自定义 Cookie `Domain`（例如共享主域名场景），可在 `auth_controller.go` 的 `SetCookie` 调用中将 domain 参数从 `""` 改为 `.yourdomain.com`。

> **注意**：在任何通过非 127.0.0.1 访问的生产环境下，HTTPS 是 Cookie 安全传输的基础。

## AI 智能检索演进路线

当前 AI 问答模块的知识检索基于 **MySQL 全文索引 + 关键词规则意图识别**，架构简洁、零额外依赖。后续计划分阶段向语义检索演进，持续提升回答质量。

### 当前架构 (v1 — 已实现)

```text
用户问题 → 关键词提取 + 意图识别(规则) → MySQL MATCH/LIKE 检索 → 拼接 Context → LLM 流式回答
```

| 能力 | 实现方式 |
|------|---------|
| 意图识别 | 关键词规则匹配（知识/练习/进度/社区/通用） |
| 知识检索 | MySQL `MATCH...AGAINST` 全文索引 + `LIKE` 模糊匹配 |
| 缓存加速 | Redis 缓存检索上下文（5 分钟 TTL） |
| 多轮对话 | 自动截取最近 5 轮历史，总长度限制 4000 字符 |

**优点**：零额外依赖、部署简单、响应速度快。

**局限**：全文索引是词频匹配，语义理解弱（如"指针怎么用"搜不到标题为"内存地址与引用"的知识点）；意图规则易误判；无跨表语义关联。

---

### 阶段一：向量语义检索 (v2 — 规划中)

引入向量数据库，将关键词匹配升级为 **Embedding 语义检索**：

```text
用户问题 → Embedding 向量化 → 向量数据库 Top-K 检索 → 拼接 Context → LLM 流式回答
```

- **向量数据库**: [Milvus](https://milvus.io/)（Go SDK 成熟）或 [Qdrant](https://qdrant.tech/)
- **Embedding 模型**: OpenAI `text-embedding-3-small` 或本地部署 `bge-m3`（免费无限调用）
- **改造点**: 新增 `VectorSearcher` 接口，注入 `QAService` 替换 `buildSearchQuery`，业务层无需大改

**预期收益**："指针怎么用" 可以语义匹配到"内存地址与引用"等相关知识点，大幅提升检索召回率。

---

### 阶段二：智能意图识别 (v3 — 规划中)

将硬编码的关键词规则替换为 **LLM Function Calling** 或轻量分类模型：

```text
用户问题 → LLM Function Calling / 轻量分类模型 → 精准意图 + 实体提取 → 定向检索
```

- **方案 A**: 利用大模型 Function Calling 能力，一次调用同时完成意图分类和关键实体提取
- **方案 B**: 本地部署轻量分类模型（如 BERT-tiny），毫秒级推理，零 API 成本
- **改造点**: 替换 `classifyIntent()` 和 `extractKeywords()` 方法

**预期收益**：消除规则误判，精准识别复合意图（如"这道题的指针哪里错了"同时触发练习 + 知识检索）。

---

### 阶段三：混合检索 + 重排序 (v4 — 远期目标)

结合向量检索与关键词检索，取长补短，并引入重排序模型：

```text
用户问题 ──┬── Embedding → 向量库 Top-K（语义召回）
           └── 关键词  → MySQL 全文索引（精确召回）
                         ↓
                  RRF / Cross-Encoder 重排序
                         ↓
                  Top-N 高质量上下文 → LLM 流式回答
```

- **融合策略**: Reciprocal Rank Fusion (RRF) 或 Cross-Encoder 重排序模型
- **推荐模型**: `bge-reranker-v2-m3`（本地部署）或 Cohere Rerank API

**预期收益**：兼具语义理解和精确匹配能力，检索准确率预期达到最优水平。

---

### 技术选型参考

| 组件 | 推荐方案 | 备注 |
|------|---------|------|
| 向量数据库 | Milvus / Qdrant | 均支持 Docker 一键部署，Go SDK 可用 |
| Embedding 模型 | `text-embedding-3-small` / `bge-m3` | 前者接口简单，后者可本地免费部署 |
| 重排序模型 | `bge-reranker-v2-m3` / Cohere Rerank | 阶段三使用，显著提升精排质量 |
| 意图分类 | LLM Function Calling / BERT-tiny | 阶段二使用，替代关键词规则 |

> **架构设计说明**：当前检索逻辑高度集中在 `qa_service.go` 的 `buildSearchQuery` 和 `AskStream` 方法中，后续只需新增 `VectorSearcher` 接口并注入 `QAService`，即可平滑升级，无需重构业务层代码。

## 许可证

本项目采用 [MIT 许可证](LICENSE)。
