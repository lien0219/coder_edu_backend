# Coder Edu Backend
## 项目简介
Coder Edu Backend是一个基于Go语言开发的教育平台后端服务，提供用户认证、学习内容管理、学习进度跟踪等功能。该项目采用分层架构设计，具有良好的可扩展性和可维护性。

1.仪表盘模块：今日任务、目标进度、成就预览、学习数据快照

2.课前准备模块：学习诊断、目标设定、个性化学习路径、资源包

3.课中学习模块：任务链、实时反馈、协作区、在线代码编辑器

4.课后回顾模块：学习日志、阶段测验、迁移任务、反思指南

5.成就系统模块：徽章墙、积分等级、排行榜、目标管理

6.分析模块：学习进度、技能评估、个性化建议、学习曲线

7.社区模块：讨论区、问答区、资源分享


## 技术栈
- 编程语言 : Go 1.18+
- Web框架 : react
- 数据库 : mysql
- 配置管理 : Viper
- 日志系统 : 自定义日志包 ( pkg/logger )
- 认证 : JWT ( internal/util/jwt.go )
## 项目结构
```
├── .gitignore                 # Git忽略文件
├── README.md                  # 项目文档
├── api/                       # API相关代码
├── configs/                   # 配置文件目录
│   └── config.yaml            # 主配置文件
├── go.mod                     # Go模块文件
├── internal/                  # 内部包
│   ├── app/                   # 应用核心
│   │   └── app.go             # 应用入口
│   ├── config/                # 配置管理
│   │   └── config.go          # 配置加载
│   ├── controller/            # 控制器
│   │   ├── auth_controller.go # 认证控制器
│   │   ├── content_controller.go # 内容控制器
│   │   ├── health_controller.go # 健康检查控制器
│   │   ├── learning_controller.go # 学习控制器
│   │   └── user_controller.go # 用户控制器
│   ├── middleware/            # 中间件
│   │   └── auth.go            # 认证中间件
│   ├── model/                 # 数据模型
│   │   ├── achievement.go     # 成就模型
│   │   ├── base.go            # 基础模型
│   │   ├── resource.go        # 资源模型
│   │   ├── task.go            # 任务模型
│   │   └── user.go            # 用户模型
│   ├── repository/            # 数据访问层
│   │   ├── achievement_repository.go # 成就仓储
│   │   ├── resource_repository.go # 资源仓储
│   │   ├── task_repository.go # 任务仓储
│   │   └── user_repository.go # 用户仓储
│   ├── service/               # 业务逻辑层
│   │   ├── auth_service.go    # 认证服务
│   │   ├── content_service.go # 内容服务
│   │   ├── learning_service.go # 学习服务
│   │   └── user_service.go    # 用户服务
│   └── util/                  # 工具包
│       ├── jwt.go             # JWT工具
│       └── response.go        # 响应工具
├── main.go                    # 程序入口
└── pkg/                       # 公共包
    ├── database/              # 数据库相关
    ├── logger/                # 日志系统
    ├── monitoring/            # 监控相关
    ├── security/              # 安全相关
    └── tracing/               # 追踪相关
```
## 配置说明
配置文件位于 configs/config.yaml ，包含应用端口、数据库连接、日志设置等。可以通过修改此文件来配置应用行为。

## 快速开始
### 前提条件
- Go 1.18+ 已安装
- 数据库服务已启动并配置
### 运行步骤
1. 1.
   克隆项目
   
   ```
   git clone <项目仓库地址>
   cd coder_edu_backend
   ```
2. 2.
   安装依赖
   
   ```
   go mod tidy
   ```
3. 3.
   配置数据库连接
   编辑 configs/config.yaml 文件，设置正确的数据库连接信息
4. 4.
   运行应用
   
   ```
   go run main.go (fresh)
   ```
## 开发指南
### 项目架构
项目采用经典的分层架构：

- 控制器层 : 处理HTTP请求，返回响应
- 服务层 : 实现业务逻辑
- 仓储层 : 数据访问逻辑
- 模型层 : 数据结构定义
- 工具层 : 通用功能封装
### 代码规范
- 遵循Go语言标准规范
- 使用 go fmt 格式化代码
- 函数和方法注释使用GoDoc格式
### 热重载开发
推荐使用Air工具进行热重载开发：

1. 1.
   安装Air
   
   ```
   go install github.com/cosmtrek/air@latest
   ```
2. 2.
   创建Air配置文件 .air.toml
3. 3.
   运行Air
   
   ```
   air
   ```

## 许可证
本项目采用 MIT许可证 。
