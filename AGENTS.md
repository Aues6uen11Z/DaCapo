# 仓库指南

DaCapo 是 Wails v2 桌面应用：Go 后端（Gin，端口 `:48596`）+ Vue 3 / Quasar 前端。

## 项目结构

- `main.go` - 入口；内嵌 `frontend/dist`。
- `backend/` - Go：`model/`、`controller/`、`service/`、`router/`、`app/`、`utils/`。
- `frontend/src/` - Vue：`pages/`、`components/`、`stores/`（Pinia）、`services/`、`i18n/`、`router/`；`wailsjs/` 为生成的绑定。
- `docs/` - 用户/开发者指南（zh-CN + en-US）；内部架构见 `docs/dev/`。
- `config/`、`instances/`、`envs/`、`repos/`、`logs/` - 运行时数据，已 gitignore。

## 常用命令

- `wails dev` / `wails build` - 开发 / 生产构建。
- `go build ./...`、`go vet ./...`、`go test ./...` - 后端。
- `cd frontend && npm run dev|build|lint|format` - 前端。

## 代码风格

- Go：`gofmt -s`；文件命名 `<topic>.go`。
- 前端：2 空格缩进、单引号、分号（`.editorconfig`、Prettier、ESLint `vue3-essential`）。
- i18n：`zh-CN` 与 `en-US` 必须同时更新。

## 文档维护

- 改了某模块的代码，**顺手更新** `docs/dev/` 下对应文档；找不到对应文档时按 `docs/dev/README.md` 的结构新增。
- 发现文档与代码不符，**以代码为准**并修正文档，不要反过来。
- 稳定信息（架构、技术栈）与易变信息（模块内部逻辑）分文件存放；每篇文档开头列「相关文件」锚定真实代码路径。

## 提交与发布

遵循 Conventional Commits：`feat:`/`fix:`/`refactor:`/`build:`/`chore:`/`docs:`，小写、祈使语气。发布由推送 `v*.*.*` 标签触发（版本号由 CI 写入，勿手动改）。
