<div align="center">

# sql-agent-cli

[![Build](https://img.shields.io/badge/build-passing-success)](scripts/build.sh)
[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

**Agent-First SQL 网关。**

让 AI Agent 安全、省 token 地访问数据库，不泄露敏感信息。

[English](README.md) | [中文](README_CN.md)

</div>

---

## 什么是 Agent-First？

传统数据库工具为人类设计。`sql-agent-cli` 为运行在**上下文窗口中**的 AI Agent 设计。

Agent 查询数据库时，通常面临三个问题：

1. **Token 爆炸** — 一条 `SELECT *` 可能把上万行数据塞进 Agent 上下文，浪费大量 token。
2. **密码泄露** — Agent 直接处理连接串时，密码会进入模型日志、命令历史和聊天记录。
3. **误删误改** — `DELETE` 或 `UPDATE` 中的一个手误就能在生产环境造成灾难。

`sql-agent-cli` 通过在 Agent 和数据库之间建立**代理层**解决这三个问题。Agent 永远看不到凭证，查询结果自动压缩分页，危险操作必须人工确认。

## 核心设计

### Token 智能优化

每个查询结果都针对 Agent 上下文窗口自动优化：

- **SQL 层分页** — `LIMIT` 在数据库层注入，非内存截断。数据库只返回 Agent 能处理的数据量
- **智能截断** — 长文本和二进制字段自动摘要，不直接展示原始内容
- **确定性分页** — 默认每页 50 行，带 `has_more` 元数据；Agent 自行判断是否值得消耗更多 token
- **列数组格式** — `{"columns": [...], "rows": [...]}` 替代重复的对象键，宽表场景节省约 30% token
- **探索模式** — `--explore` 返回结构化摘要（列类型、样本值、行数估算）而非原始行。适合首次接触陌生表

### 安全代理模式

Agent **永远接触不到**数据库凭证：

- **凭证助手** — 通过外部 JSON helper 解析密码，密码绝不进入 Agent 上下文
- **加密存储** — 命名连接使用 AES-256-GCM 加密；未提供主密钥时自动生成
- **明确确认** — `DELETE`、`DROP`、`TRUNCATE`、`ALTER` 和无条件 `UPDATE` 无 `--confirm` 直接拦截
- **单语句限制** — 每次只执行一条 SQL，防止复杂注入

### 零依赖部署

拷贝 skill 包到 Agent 目录即可运行：

```bash
# OpenCode / Claude Code
cp -R dist/sql-agent ~/.config/opencode/skills/

# 或 Cursor、Copilot CLI 等
# 嵌入的二进制会自动匹配当前平台
```

无需 Docker、无需数据库驱动、无需任何运行时配置。

## 快速开始

### 给 Agent 用户使用

```bash
# 1. 构建 skill（一条命令）
bash scripts/build-skill.sh

# 2. 拷贝到 Agent 的 skills 目录
cp -R dist/sql-agent ~/.config/opencode/skills/

# 3. Agent 自动学会使用 sql-agent-cli
```

### 给开发者使用

```bash
# 克隆并构建
git clone https://github.com/davidfantasy/sql-agent-cli.git
cd sql-agent-cli
bash scripts/build.sh

# 连接数据库
./bin/sql-agent connect analytics --driver postgres \
  --host db.example.com --port 5432 --database app \
  --username analyst --password-env DB_PASSWORD

# 或使用凭证助手（推荐）
./bin/sql-agent connect prod --driver mysql \
  --credential-helper "vault-helper --profile prod"
```

### 探索与查询

```bash
# 先查结构
./bin/sql-agent schema list analytics
./bin/sql-agent schema describe analytics users

# 探索模式：摘要替代原始行（首次接触陌生表时节省 token）
./bin/sql-agent query analytics "SELECT * FROM users" --explore

# 自动分页查询（LIMIT 在数据库层注入）
./bin/sql-agent query analytics \
  "SELECT id, email, created_at FROM users ORDER BY id" --page 1

# 危险操作被拦截
./bin/sql-agent query prod "DELETE FROM sessions WHERE created_at < NOW() - INTERVAL '30 days'"
# 错误: destructive_query_requires_confirmation

# 只有人工明确批准后才可以执行
./bin/sql-agent query prod "DELETE FROM sessions WHERE created_at < NOW() - INTERVAL '30 days'" --confirm
```

## 输出格式

```json
{
  "ok": true,
  "data": {
    "columns": ["id", "email", "created_at"],
    "rows": [
      [1, "alice@example.com", "2024-01-15T09:30:00Z"],
      [2, "bob@example.com", "2024-02-20T14:45:00Z"]
    ],
    "page": 1,
    "page_size": 50,
    "returned_rows": 2,
    "has_more": false,
    "next_page": null,
    "truncated": false,
    "truncated_cells": 0
  },
  "warning": null,
  "error": null,
  "duration_ms": 14
}
```

## 架构

```
┌─────────────┐     ┌──────────────────┐     ┌──────────────┐
│  AI Agent   │────▶│  sql-agent-cli   │────▶│   Database   │
│  (Context)  │     │  (Proxy + Gate)  │     │ (MySQL/PSQL) │
└─────────────┘     └──────────────────┘     └──────────────┘
                           │
                    ┌──────┴──────┐
                    ▼             ▼
            ┌──────────┐  ┌────────────┐
            │  Safety  │  │  Credential│
            │   Gate   │  │   Helper   │
            └──────────┘  └────────────┘
```

Agent 与**受控接口**交互，而非直接接触原始数据库。CLI 负责：
- 连接加密和解析
- SQL 安全分析
- 结果压缩和分页
- 通过 helper 隔离密钥

## 路线图

| 状态 | 特性 |
|--------|---------|
| ✅ | MySQL 支持 |
| ✅ | PostgreSQL 支持 |
| ✅ | 凭证助手协议 |
| ✅ | 加密连接存储 |
| ✅ | 危险写操作确认 |
| ✅ | 自动 token 优化 |
| ✅ | SQL 层分页注入 |
| ✅ | 探索模式 (--explore) |
| 📋 | Microsoft SQL Server |
| 📋 | Oracle Database |

## 安全模型

| 威胁 | 防护措施 |
|--------|------------|
| Token 耗尽 | 智能截断 + 分页 + 列数组格式 |
| 凭证泄露 | Agent 永远看不到密码；外部 helper 解析 |
| 误写数据 | 危险 SQL 无 `--confirm` 直接拦截 |
| 本地存储泄露 | AES-256-GCM 加密；自动生成主密钥 |
| 注入攻击 | 单语句限制；禁止多查询执行 |

## 开发

```bash
# 格式化代码
bash scripts/format.sh

# 构建二进制
bash scripts/build.sh

# 运行测试（单元 + Docker 集成）
bash scripts/test.sh

# 构建带嵌入二进制的 skill
bash scripts/build-skill.sh

# 构建全平台发布包
bash scripts/package.sh
```

详见 [AGENTS.md](AGENTS.md) 了解架构细节和贡献指南。

## 许可证

[MIT](LICENSE)
