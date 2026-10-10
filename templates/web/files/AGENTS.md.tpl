# AI Agents 工作规范

<%if .Options.display_name%><% .Options.display_name %><%else%><% .Name %><%end%>：<%if .Options.description%><% .Options.description %><%else%><一句话项目说明：这个项目是什么、做什么><%end%>

## 目录结构

```txt
<% .Name %>/
├── cmd/<% .Name %>/   # 程序入口
├── internal/          # 应用私有代码
<%if .Options.embed_frontend%>├── web/               # 前端源码与构建产物
<%end%>└── scripts/           # 辅助脚本
```

## 文档索引

文档变化时同步更新

- <其他文档路径>：<一句话说明>
