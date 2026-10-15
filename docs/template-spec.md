# 模板规范

面向模板作者的完整参考。一个模板 = 一个目录：`template.yaml`（描述文件）+ `files/`（全部源文件）。

```txt
mytpl/
├── template.yaml
└── files/            # 源文件，组织结构自由（建议镜像产物结构）
    ├── main.go.tpl   # 约定：要渲染的文件加 .tpl 后缀（仅约定，清单才是事实来源）
    └── ...
```

## template.yaml schema

```yaml
name: mytpl                  # 模板名，缺省取目录名
description: 一句话描述      # 向导与 list 中展示
schema: 1                    # 模板 schema 版本，可省略（缺省 1）；引擎拒绝高于当前支持版本的模板
delims: ecb                  # 内容渲染默认分隔符，可省略（默认 ecb）

options:                     # 模板选项，可空
  - key: sqlite              # 必填，^[a-z][a-z0-9_]*$
    type: boolean            # 必填：boolean / string / choice / multichoice
    prompt: 集成 SQLite      # 必填，向导中的提问
    help: 详细说明           # 可选，向导中的辅助说明
    group: 构建              # 可选，向导分页：同名 group 的选项同页展示（缺省自动分页）
    default: false           # 可选，缺省为该类型零值
    choices:                 # choice/multichoice 必填
      - {value: chi, label: chi}
    pattern: "^[a-z]+$"      # 可选，string 类型的正则校验

files:                       # 产物清单，逐条映射
  - src: .gitattributes      # 必填，相对 files/ 的路径
    dest: .gitattributes     # 可选，缺省 = src 去掉 .tpl 后缀；固定 {{ }} 变量模板
    render: true             # 可选，缺省 false（字节级复制）
    delims: brackets         # 可选，单文件覆盖模板级分隔符
    when: docker_image       # 可选，expr-lang 条件表达式

tests:                       # 自测矩阵：go-create 自身测试逐组生成并编译验证
  - name: minimal            # 无 set = 全默认值
  - name: full
    set: {sqlite: true}      # 值类型须与选项 type 匹配（boolean 用布尔、multichoice 用列表）
```

## 分隔符预设

| 预设名 | 左 | 右 | 说明 |
|---|---|---|---|
| `ecb`（默认） | `<%` | `%>` | ASP.NET embedded code block 术语 |
| `go` | `{{` | `}}` | Go text/template 默认 |
| `brackets` | `[[` | `]]` | 双方括号 |
| `[left, right]` | 自定义 | 自定义 | 两元素列表 |

- 内容渲染用分隔符可配置；**dest 路径模板固定 `{{ }}`**，只放变量、禁逻辑
- 产物中的 `{{.Version}}`（goreleaser）、`${{ }}`（GitHub Actions）在 ecb 分隔符下原样穿过，无需转义
- 渲染内容中需要字面分隔符时：`<% "<%" %>`

## 渲染上下文

render: true 的文件内容中可用：

| 字段 | 含义 |
|---|---|
| `.Name` | 项目名/二进制名 |
| `.Module` | 模块路径 |
| `.Repo` | github.com 模块的 owner/repo 段，非 GitHub 模块为空串 |
| `.PkgName` | Name 清理后的合法 Go 标识符（去连字符等） |
| `.GoVersion` | 当前 Go 工具链版本，如 1.26.4 |
| `.License` | `mit` / `apache-2.0` / `gpl-3.0` / `""`（无） |
| `.CopyrightHolder` | 版权人 |
| `.Year` | 当前年份 |
| `.Options.<key>` | 模板选项值（boolean→bool，string/choice→string，multichoice→[]string） |

模板函数：`lower`（转小写）、`has`（multichoice 成员判断：`<%if has .Options.ci_platforms "windows-latest"%>`）。

## when 表达式

expr-lang 语法，直接使用选项 key 作为变量：

```yaml
when: sqlite                                # boolean 真值
when: http_framework == "gin"               # 比较
when: goreleaser || docker_image            # 逻辑或
when: embed_frontend && !sqlite             # 组合
when: '"windows-latest" in ci_platforms'    # multichoice 成员（注意 yaml 引号嵌套）
```

表达式在模板加载期编译校验：引用未声明的选项 key、结果非布尔都会报错。

## 校验规则速查

加载期报错：清单 src 不存在、dest 含 `..` 或绝对路径、选项 schema 不合法、when 编译失败、delims 预设未知、YAML 未知字段。

生成期警告：`files/` 下未列入清单的文件被跳过；静态复制文件内容含分隔符（疑似漏标 render）。

## tests 约定

- 每个选项的每个非默认值至少被一组用例覆盖
- 外加一个全开组合
- 测试会真实执行 `go mod tidy` + `go build ./...`（需网络拉依赖），embed 类场景按规范自动创建 web/dist 占位文件

## 内置 web 模板：镜像与发布选项

| 选项 | 默认值 | 产物与行为 |
|---|---|---|
| `docker_dev` | `true` | 生成 `docker-dev.yml` 和 Dockerfile，分支推送或手动触发后发布 GHCR `:dev` 镜像 |
| `docker_image` | `false` | 生成 Dockerfile，并通过 GoReleaser 发布正式版本镜像 |
| `goreleaser` | `true` | 生成 `.goreleaser.yml` 和 `release.yml`；`docker_image=true` 时也会生成这两项 |

`docker_dev` 可单独开启（`docker_image=false, goreleaser=false`），也可单独关闭，不影响正式发布。开发镜像流程在 Actions 内构建前后端，再以 `context: .` 将生成的二进制交给现有 Dockerfile 打包。启用 `embed_frontend` 时复用 `scripts/build-web.sh`，支持未创建前端时的占位文件。

`master/main/dev/develop` 共用 `ghcr.io/<小写 github.repository>:dev`，以固定并发组、`cancel-in-progress: false` 避免同时发布；GitHub 默认只保留一个待运行任务。开发镜像固定为 Linux amd64，保留独立的 lint/test 检查，`test.yml` 仍照常生成和运行。

## 自定义模板

放置于 `~/.config/go-create/templates/<name>/`（全平台统一），结构同上。同名模板覆盖内置模板。`go-create list` 可查看全部模板及来源（builtin/user）。
