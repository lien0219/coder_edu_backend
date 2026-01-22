# Coder Edu Backend

[![Go Version](https://img.shields.io/github/go-mod/go-version/coder-edu/coder_edu_backend?label=Go)](https://golang.org/)
[![Framework](https://img.shields.io/badge/Framework-Gin-blue)](https://gin-gonic.com/)
[![ORM](https://img.shields.io/badge/ORM-GORM-blue)](https://gorm.io/)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

## 项目简介

Coder Edu Backend 是一个基于 Go 语言的高性能教育平台后端服务。它为学习者提供全方位的支持，包括用户认证、学习资源管理、个性化学习路径、实时反馈以及完善的成就系统。

### 核心功能模块

1. **仪表盘 (Dashboard)**：实时展示今日任务、学习进度、成就预览及学习数据快照。
2. **课前准备 (Pre-class)**：包含学习诊断、目标设定、生成个性化学习路径及资源包准备。
3. **课中学习 (In-class)**：支持任务链驱动、实时学习反馈、在线协作及集成式代码编辑器。
4. **课后回顾 (Post-class)**：记录学习日志、阶段性测验、知识迁移任务及反思指南。
5. **成就系统 (Achievement)**：丰富的徽章墙、积分等级、全球排行榜及目标管理。
6. **深度分析 (Analytics)**：多维度技能评估、学习曲线分析及智能个性化建议。
7. **社区互动 (Community)**：集成讨论区、问答区及资源分享平台。
8. **编程资源管理**：专门针对 C 语言等编程资源的存储与分发。

## 技术栈

- **核心框架**: [Go](https://golang.org/) 1.24+ / [Gin](https://gin-gonic.com/)
- **数据持久化**: [MySQL](https://www.mysql.com/) / [GORM](https://gorm.io/)
- **对象存储**: [MinIO](https://min.io/)
- **配置管理**: [Viper](https://github.com/spf13/viper)
- **日志系统**: [Zap](https://github.com/uber-go/zap) + [Lumberjack](https://github.com/natefinch/lumberjack)
- **身份认证**: [JWT](https://github.com/dgrijalva/jwt-go)
- **API 文档**: [Swagger](https://github.com/swaggo/swag)
- **多媒体处理**: [FFmpeg](https://ffmpeg.org/)
- **监控与追踪**: [Prometheus](https://prometheus.io/) / [OpenTelemetry](https://opentelemetry.io/) (Jaeger)

## 项目结构

```text
├── api/                  # Swagger 接口定义
├── configs/              # 配置文件目录
├── docs/                 # Swagger 生成文档
├── internal/             # 核心业务逻辑
│   ├── app/              # 应用启动入口
│   ├── config/           # 配置加载逻辑
│   ├── controller/       # 接口控制器 (API Handlers)
│   ├── middleware/       # 鉴权、日志等中间件
│   ├── model/            # 数据库模型与定义
│   ├── repository/       # 数据访问层 (DAO)
│   ├── service/          # 业务逻辑层
│   └── util/             # 常用工具函数 (JWT, Response等)
├── pkg/                  # 公共包 (数据库连接、日志初始化、监控等)
├── scripts/              # 工具脚本 (如 Swagger 生成脚本)
├── main.go               # 项目启动文件
├── Dockerfile            # Docker 镜像构建
└── docker-compose.yml    # 容器编排
```

## 快速开始

### 本地开发环境

1. **环境准备**
   - 安装 Go 1.24+
   - 安装 MySQL 8.0+
   - (可选) 安装 MinIO

2. **克隆并安装依赖**
   ```bash
   git clone <项目仓库地址>
   cd coder_edu_backend
   go mod tidy
   ```

3. **配置应用**
   编辑 `configs/config.yaml`，填入正确的数据库和中间件连接信息。

4. **运行应用**
   ```bash
   # 直接运行
   go run main.go
   
   # 或者使用热重载 (推荐)
   air
   ```

### Docker 运行

```bash
docker-compose up -d
```

## 开发者指南

### API 文档查看

应用启动后，可以通过以下地址访问交互式 Swagger 文档：
`http://localhost:<port>/swagger/index.html`

### 重新生成 Swagger 文档

如果你修改了 API 注释，请运行：
```bash
# Windows
./scripts/generate_swagger.bat

# Linux/macOS
./scripts/generate_swagger.sh
```

## 许可证

本项目采用 [MIT 许可证](LICENSE)。
