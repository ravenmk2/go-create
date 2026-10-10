# AI Agents 工作规范

go-create：Go 项目初始化工具，按工程规范生成项目骨架，模板化可扩展。

## 目录结构

```txt
go-create/
├── cmd/go-create/   # 程序入口
├── internal/        # 应用私有代码（template / generate / cli / wizard）
├── templates/       # 内置模板（template.yaml + files/，embed 进二进制）
└── scripts/         # 辅助脚本
```

## 文档索引

文档变化时同步更新

- docs/architecture.md：架构分层与关键设计决策及理由
- docs/template-spec.md：模板规范完整参考（面向模板作者）
