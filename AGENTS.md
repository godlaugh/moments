# Moments 极简朋友圈

Go 后端（Echo + GORM/SQLite）+ Nuxt3 前端的开源朋友圈项目。
许可证 GPL-3.0：自建自用/对外提供 API 服务无需开源；修改后**分发**必须开源。

## 目录结构

- `backend/`：Go 后端（API 全在 `/api/*` 路由下）
  - `main.go` 入口；`router.go` 全部路由定义；`handler/` 各 API 实现
  - `vo/` 请求/响应结构体与环境变量配置；`db/` GORM 模型与初始化
  - `init_static_files_prod.go`：`-tags prod` 构建时内嵌 `public/` 静态文件
- `front/`：Nuxt3 SPA（`ssr: false`）
  - `pages/`、`components/`、`layouts/`；API 请求封装在 `utils/`
- `data/`（本机运行时数据，勿提交）：db.sqlite、upload/、server.log

## 后端（Go）

要求 Go 1.23.3+。常用命令（在 `backend/` 目录执行）：

```bash
go build -tags prod -ldflags="-s -w -X main.version=local -X main.commitId=local" -o ./dist/moments .
./dist/moments                                  # 运行（自动加载当前目录 .env）
```

环境变量（`backend/.env` 会被 godotenv 按**当前工作目录**自动加载，已 gitignore）：

| 变量 | 默认值 | 注意 |
|------|--------|------|
| PORT | **37892** | README 写 3000 是错的，以代码为准 |
| DB | /app/data/db.sqlite | 裸机必须显式设置，否则写 /app 失败 |
| UPLOAD_DIR | /app/data/upload | 同上 |
| JWT_KEY | 空（随机） | 不设则重启后所有 token 失效 |
| ENABLE_SWAGGER | false | true 后访问 `/swagger/index.html` |
| CORS_ORIGIN | 空 | 多个 Origin 逗号分隔 |

关键点：

- **必须 `-tags prod`** 构建才会内嵌前端静态文件；不带则只提供 API
- 首次启动自动初始化管理员 `admin`（密码见项目 README 默认值）与默认配置
- 启动时若检测到版本变化会自动备份数据库到 `$DB` 目录
- 无测试文件（`*_test.go` 不存在）；改动后用 `go build ./...` + 实际起服务 curl 验证
- 修改 handler 的 swagger 注释后需 `swag init` 重新生成 `docs/`

## 前端（Nuxt3）

要求 Node 18+，**pnpm 必须为 10.x**（`package.json` 的 `packageManager: pnpm@10.10.0`）。
`pnpm-workspace.yaml` 没有 `packages` 字段，pnpm 9 会报
`packages field missing or empty`，pnpm 10 才支持。

```bash
cd front
pnpm i
pnpm run dev       # 开发：nuxt dev --host 0.0.0.0，API 代理到 localhost:37892（见 nuxt.config.ts vite.server.proxy）
pnpm generate      # 生产：静态产物到 .output/public，之后拷贝到 backend/public 再构建后端
```

注意：dev 代理写死 `http://localhost:37892`，所以本地开发要同时起后端（默认端口即可）。

## 本机开发环境（非标准 PATH，需手动 export）

```bash
export PATH=/home/admin/aistudio/go/bin:$PATH        # Go 1.23.12（用户态安装）
export PATH=~/.npm-global/bin:$PATH                  # pnpm 10.10.0（npm 全局 prefix 改到用户目录）
```

## 当前部署（生产实例）

- 运行目录 `/home/admin/aistudio/moments/backend`，配置见 `backend/.env`（PORT=3000）
- 数据目录 `~/aistudio/moments/data/`（db.sqlite + upload/ + server.log）
- 重启：`cd ~/aistudio/moments/backend && nohup ./dist/moments > ~/aistudio/moments/data/server.log 2>&1 &`
- 公网：`https://im.huby.cc`，Caddy（系统服务）反向代理到 127.0.0.1:3000
  - Caddyfile：`/etc/caddy/Caddyfile`，改后 `sudo caddy validate --config /etc/caddy/Caddyfile && sudo systemctl reload caddy`
  - 本机已有免密 sudo（`/etc/sudoers.d/90-admin-nopasswd`）
- 验证：`curl -s -o /dev/null -w "%{http_code}\n" https://im.huby.cc/` 期望 200
- 停服坑：不要 `pkill -f "dist/moments"`（会误杀包含该字符串的 shell 命令本身），用 `pgrep -x moments` 后按 PID kill
