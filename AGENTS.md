# SceneMint Agent Notes

## Project

SceneMint 是轻量图片生成站点：Go HTTP 后端 + Astro/React 前端。生产环境由 Go 服务同源提供 `/api` 和 `web/dist`。

额度数据使用本地 bbolt，默认路径为 `data/quota.db`；不要误引入 SQLite。

## Development

- 后端路由基于 Echo，API 错误响应优先沿用 `gox/server` envelope。
- 前端 UI 沿用现有 Astro React island、Tailwind、shadcn/Base UI 组件风格。
- 变更保持小而直接；不要顺手重构无关代码或引入新依赖。

## Commit

- 提交信息优先使用简短的 Conventional Commits 风格：`type(scope): summary`，例如 `fix(api): handle quota errors`。
- 常用 `type` 包括 `feat`、`fix`、`docs`、`refactor`、`test`、`chore`；每个 commit 尽量只包含一件事。

## Verification

优先跑最小相关检查：

```sh
go test ./...
cd web && bun run build
```
