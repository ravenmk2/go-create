# AI Agents 工作规范

<%if .Options.display_name%><% .Options.display_name %><%else%><% .Name %><%end%>：<%if .Options.description%><% .Options.description %><%else%>一句话项目说明占位（这个项目是什么、做什么）。<%end%>

## 目录结构

```txt
<% .Name %>/
├── <% .Name %>.go          # 对外 API 主体（根目录包）
└── .github/workflows/      # GitHub Actions
```

## 文档索引

文档变化时同步更新

- README.md：项目说明与徽章
