# 架构文档

> 本文档从代码提炼，修改代码后需同步更新。最后核对版本：2026-08（基于 `multi-level-tag` 分支）。

## 总览

```
┌────────────────────────────────────────────────┐
│ 浏览器                                         │
│  Nuxt3 SPA (ssr: false)                        │
│  front/  →  pnpm generate  →  静态文件         │
└──────────────┬─────────────────────────────────┘
               │ 全部 POST /api/*，JWT 放 header x-api-token
┌──────────────▼─────────────────────────────────┐
│ Go 后端 (Echo + GORM + SQLite)                 │
│  backend/                                      │
│  ├─ main.go        启动 + DI 容器装配          │
│  ├─ router.go      全部路由（唯一定义处）      │
│  ├─ handler/       API 实现（业务逻辑）        │
│  ├─ db/            GORM 模型 + 初始化          │
│  ├─ vo/            请求/响应结构体 + AppConfig │
│  ├─ middleware/    auth.go（JWT 验证）         │
│  ├─ pkg/           mail 等独立工具             │
│  └─ init_static_files_prod.go  prod 构建内嵌前端│
└──────────────┬─────────────────────────────────┘
               │
        SQLite 单文件 ($DB) + 上传目录 ($UPLOAD_DIR)
```

- 单二进制部署：`-tags prod` 构建时把 `backend/public/`（前端产物）内嵌进可执行文件
- 开发时前后端分离：前端 dev server 把 `/api` 代理到 `localhost:37892`（见 `front/nuxt.config.ts` 的 `vite.server.proxy`）

## 后端关键机制

### 依赖注入（samber/do）

`main.go` 里用 `do.Provide` 注册单例（Echo 实例、`*vo.AppConfig`、`*gorm.DB`、`zerolog.Logger`），各 Handler 通过 `NewXxxHandler(injector)` 取依赖。新增 Handler 的套路：

1. `handler/xxx.go` 写 `NewXxxHandler`，内嵌 `BaseHandler`（提供 `db`/`cfg`/`log`/`injector`）
2. `router.go` 的 `setupRouter` 里实例化并挂路由
3. `main.go` 无需改动（Handler 不走 do.Provide，直接构造）

### 鉴权

- 登录成功后签发 JWT，前端存 localStorage（`front/store.ts` 的 `useGlobalState`），每次请求带 header **`x-api-token`**
- `middleware/auth.go` 校验 token，通过后把 `*db.User` 塞进 `CustomContext`，Handler 里 `ctx.CurrentUser()` 取当前用户
- 大部分接口需要登录（游客可看的接口内部自行判断，如 memo/list 游客返回非定时 memo）

### 统一响应与错误码（handler/handler.go）

所有 `/api/*` 返回 HTTP 200，业务结果看 body：

```json
{ "code": 0, "message": "", "data": ... }
```

| code | 含义 | 前端行为 |
|------|------|---------|
| 0 | 成功 | 取 `data` |
| 1 | 失败 | toast message |
| 2 | 参数错误 | toast message |
| 3 | Token 无效 | 清空本地登录态并跳首页 |
| 4 | Token 缺失 | 同上 |

前端封装见 `front/utils/index.ts` 的 `useMyFetch`，**全站 API 都走它**，没有别的请求层。

### 配置

- 环境变量 → `vo.AppConfig`（`vo/config.go`），`.env` 由 godotenv 按**当前工作目录**加载
- 用户级/系统级动态配置（是否开评论、S3 参数等）存 `SysConfig` 表的 JSON blob（见 DATA_MODEL.md）

### 版本备份与迁移

- 启动检测版本变化 → 自动备份 SQLite 到 `$DB` 同目录（`backup.go`）
- `migrate.go` / `migrate_friend_link.go` 存放历史数据迁移；schema 变更靠 GORM `AutoMigrate`（`db/db.go`）

## 前端关键机制

- **Nuxt3 SPA**（`ssr: false`），页面在 `front/pages/`，路由按文件名约定
- **状态**：没有 Pinia。全局登录态 = `createGlobalState(useStorage)` 持久化到 localStorage（`store.ts`）；系统配置用 Nuxt 内置 `useState("sysConfig")`（内存态，`app.vue` 启动时拉取）
- **请求**：统一 `useMyFetch`（见上），文件上传走 `useUpload`（自动区分 S3 / 服务器）
- **UI**：Nuxt UI (v2) + Tailwind + `@iconify-json/*` 图标；toast 用 `vue-sonner`；暗色模式 `@nuxtjs/color-mode`
- **Markdown**：`markdown-it` + shiki 代码高亮（`utils/index.ts` 底部全局初始化）

## 已知坑（改代码前必读）

1. README 的 PORT 默认值写 3000 是**错的**，代码实际默认 **37892**
2. 裸机构建必须 `-tags prod` 才内嵌前端；不带只出 API
3. `handler/tag.go` 的 swagger 注释里 `@Router` 写错成 file 的路径了，看路由以 `router.go` 为准
4. 无测试、无 Go/TS lint 硬约束：验证方式 = `go build ./...` + `pnpm generate`/`pnpm run dev` + 起服务 curl
5. `front/package.json` 的 `packageManager: pnpm@10.10.0`，pnpm 9 会因 workspace 配置报错，必须 pnpm 10.x
