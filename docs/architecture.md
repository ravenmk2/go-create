# 架构与关键设计

go-create：按工程规范生成 Go 项目骨架的初始化工具。本文档记录架构与关键设计决策及其理由。

## 架构分层

```txt
cmd/go-create/      程序入口，只做组装
internal/
├── template/       模板模型：加载（embed + 用户目录）、校验、when 求值、渲染
├── generate/       生成编排：参数校验 → 渲染 → 写盘 → 后置动作（git init / go mod tidy）
├── cli/            cobra 命令：root（裸运行进向导）/ new（--set 解析与校验）/ list / upgrade
├── upgrade/        自更新：GitHub releases 查询 → 下载（重试/停滞检测）→ SHA256 校验 → 平台自替换
└── wizard/         交互向导：TTY 走 bubbletea 自绘 TUI（步骤指示器 + 实时摘要侧栏 + 前进/后退）；非 TTY 降级 huh accessible 模式
templates/          内置模板（cli / web / lib），embed 进二进制
```

数据流（两条收集途径汇入同一参数集）：

```txt
TUI 向导  ─┐
           ├─→ generate.Params ─→ template.Render ─→ 写盘 ─→ git init / go mod tidy
--set/--yes 参数 ─┘   （公共参数 + 模板选项）
```

## 关键设计决策

### 清单驱动的产物映射

每个模板 = `template.yaml` + `files/`（全部源文件）。`files` 清单逐条定义一个产物的完整映射：**源文件 → 是否渲染 → 内容分隔符 → 目标路径 → 生成条件**。不采用目录约定（如 render/ 与 static/ 分树）或后缀约定（.tmpl），因为清单是唯一事实来源，可 review、可校验，且目标路径与源路径解耦（dest 含变量，无法靠目录结构表达）。

### 双套分隔符分工

- **文件内容**：可配置分隔符，默认 `ecb`（`<% %>`，ASP.NET embedded code block 术语），备选 `go`（`{{ }}`）、`brackets`（`[[ ]]`）或自定义。理由：产物中高频出现 Go template 语法（goreleaser 的 `{{.Version}}`、GitHub Actions 的 `${{ }}`），若用 `{{ }}` 渲染则满屏转义（Helm 的教训）；换分隔符后这些内容零转义穿过
- **dest 路径**：固定 `{{ }}`，只放变量禁逻辑。dest 是 YAML 字符串、不落盘为真实文件名，不受 Windows 文件名禁用字符（`<>`）约束，与内容分隔符解耦

### 可靠性闭环

模板是可扩展点，扩展必然带来腐化风险，因此引擎内置多层校验：

1. **加载期**：YAML 严格模式（未知字段报错）；`schema` 版本校验（缺省 1，拒绝更高版本，为将来破坏性变更留门）；选项 schema 校验；when 表达式用 expr-lang **编译期校验**（引用未声明选项 key 直接报错）；清单 src 存在性检查
2. **生成期**：`files/` 下未列入清单的文件 → 警告；静态复制文件内容含分隔符 → 警告（疑似漏登记 render）；dest 路径安全校验（相对、无 `..`、渲染后非空）
3. **自测矩阵**：`template.yaml` 的 `tests` 段声明代表性选项组合，go-create 自身 `go test` 逐组生成到临时目录并 `go build ./...` 验证产物可编译——模板扩展的护栏

### 向导 UI：bubbletea 自绘 + huh 降级

向导交互层用 bubbletea/lipgloss 自绘（步骤指示器、实时摘要侧栏、窗体布局），而非继续堆叠 huh 表单。理由：选项多了之后 huh 单组堆叠渲染会溢出终端；且需要步骤级前进/后退与全局状态可见性，自绘 Model 控制粒度更细。选项按描述文件的 `group` 字段分页，未分组的自动按每页 5 个分页。非 TTY 环境（管道/重定向）没有交互界面可用，降级为 huh accessible（纯文本逐行问答），huh 依赖仅为此保留。

### 生成到非空目录

目标目录允许非空（支持"先 git init / clone 空仓库再生成"的场景）：交互向导在生成前显式确认（列出目录内容，提示同名文件将被覆盖）；`--yes` 视为已确认直接生成。生成期逐文件检测覆盖并输出 warning；目标目录已含 `.git` 时跳过 `git init`（避免无意义的 reinit）。后置动作（git init / go mod tidy）失败一律降级为 warning，不阻断生成；其中 go mod tidy 依赖网络，内置 3 次线性退避重试以吸收代理/仓库的瞬时故障。

### 自更新（upgrade）

`go-create upgrade` 从 GitHub releases 自更新，纯标准库实现（不引 go-update 类框架）：

- **Resolve/Apply 分离**：先只拉 release 元数据做决策（dev 永远可升级、版本相等幂等、不主动降级、显式 `--version` 允许降级），用户确认后再下载执行；`--check` 只查不下载
- **可靠性**：下载带指数退避重试 + 停滞 watchdog（30s 无数据流入判定停滞可重试，慢速下载不受影响）；资产经 checksums.txt SHA256 校验后才允许替换；不做签名校验（留作后续增量）
- **平台自替换**：Unix 同目录临时文件 rename 原子覆盖；Windows 运行中的 exe 不能覆盖但能 rename——旧文件改名 `.old` 再写入新文件（失败回滚），`.old` 留作备份、下次 upgrade 时清理
- 只支持 GitHub 源（主发布渠道），不做后台自动检查（只显式命令）

### 依赖版本不 pin

模板的 go.mod 只写 `module` + `go` 指令；框架依赖仅体现在源码 import 中，生成后由 `go mod tidy` 拉取最新兼容版本。模板零版本腐化维护；可复现性由生成项目自身提交的 go.sum 兜底。无网络时降级为提示手动执行。

### 选项类型系统

`type` 采用 Jenkins 风格命名：`boolean`（确认框→bool）/ `string`（文本输入→string）/ `choice`（单选→string）/ `multichoice`（多选→[]string）。类型同时决定表单形态与数据类型，两者一一对应无需分开声明。选项级条件（选项间依赖）暂不支持，schema 预留。

### 公共参数 vs 模板选项

`name`/`module`/`license`/`holder` 等所有模板共享的参数内置在工具中，不在描述文件重复声明。LICENSE 文本（MIT/Apache-2.0/GPL-3.0）内置工具，模板不感知。

### 模板优先级

内置模板 embed 进二进制；用户目录 `~/.config/go-create/templates/`（全平台统一）下的同名模板完全覆盖内置。新增模板 = 新增一个目录，不改代码。

## 边界（明确不做）

- 不生成前端脚手架（embed_frontend 仅 Go 侧骨架）
- 不做 git commit（只 git init）
- 不做模板版本管理、远程模板仓库
