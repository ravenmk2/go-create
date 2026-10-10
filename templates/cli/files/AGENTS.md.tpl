# AI Agents 工作规范

<%if .Options.display_name%><% .Options.display_name %><%else%><% .Name %><%end%>：<%if .Options.description%><% .Options.description %><%else%>命令行工具（TODO：一句话说明这个项目是什么、做什么）。<%end%>

## 目录结构

```txt
<% .Name %>/
├── cmd/<% .Name %>/   # 程序入口，只做组装与启动
├── internal/          # 应用私有代码，禁止外部模块导入
└── scripts/           # 辅助脚本（build.sh 本地多平台构建）
```

## 文档索引

文档变化时同步更新

- <其他文档路径>：<一句话说明>
