# API 概览

> 权威来源：路由定义 `backend/router.go`；鉴权白名单 `backend/middleware/auth.go:19-29`。
> 本文档是导航性质，参数细节以 `backend/handler/` 对应函数的 swagger 注释为准
>（`ENABLE_SWAGGER=true` 后访问 `/swagger/index.html`）。

## 通用约定

- **全部业务接口为 POST**，路径前缀 `/api`
- JWT 放 header `x-api-token`；响应恒为 HTTP 200，业务码见 `code`（0 成功，详见 ARCHITECTURE.md）
- 无 token 访问非白名单接口 → `code=4`；token 无效 → `code=3`
- 注意：即使访问白名单接口，**只要带了无效 token 也会被拒**（token 校验优先于白名单）

## 免登录接口（白名单）

| 路径 | Handler | 说明 |
|------|---------|------|
| POST /api/user/reg | user.Reg | 注册（后台可关闭） |
| POST /api/user/login | user.Login | 登录，返回 JWT |
| POST /api/user/profile | user.Profile | 无 token 时返回站点主人公开信息 |
| POST /api/user/profile/:username | user.ProfileForUser | 指定用户公开信息（前缀放行） |
| POST /api/memo/list | memo.ListMemos | 游客只见 `showType=1` 且非未来时间的 memo；`tag` 支持 `"父/子"` 全路径（含子孙），多标签逗号 AND |
| POST /api/memo/get | memo.GetMemo | 详情，同上限制 |
| POST /api/memo/like | memo.LikeMemo | 点赞 |
| POST /api/comment/add | comment.AddComment | 评论（后台可关闭） |
| POST /api/sysConfig/get | sysConfig.GetConfig | 公共配置（脱敏） |
| POST /api/friend/list | friend.GetFriendList | 友链 |
| GET /upload/* | 静态文件 | 上传文件直链 |
| GET /rss | rss.GetRss | RSS 订阅 |

## 需登录接口

### User

| 路径 | 说明 |
|------|------|
| POST /api/user/saveProfile | 修改个人资料/密码/站点配置 |

### Memo（author 或管理员）

| 路径 | 说明 |
|------|------|
| POST /api/memo/save | 新建/编辑（`id>0` 为编辑，校验归属） |
| POST /api/memo/remove | 删除（query 传 `id`） |
| POST /api/memo/setPinned | 置顶 |
| POST /api/memo/getFaviconAndTitle | 抓取外链卡片信息 |
| POST /api/memo/getDoubanMovieInfo / getDoubanBookInfo | 豆瓣影音信息 |

### Comment

| 路径 | 说明 |
|------|------|
| POST /api/comment/remove | 删除评论 |

### SysConfig（管理员）

| 路径 | 说明 |
|------|------|
| POST /api/sysConfig/save | 保存全局配置 |
| POST /api/sysConfig/getFull | 完整配置（含密钥） |

### Tag

| 路径 | 说明 |
|------|------|
| POST /api/tag/list | 标签列表：`tree`（层级树，count/total 计数）+ `tags`（全路径扁平表）；query 可传 `username` 查指定用户（免登录），否则取当前登录用户 |

### File

| 路径 | 说明 |
|------|------|
| POST /api/file/exist | SHA256 秒传检查 |
| POST /api/file/upload | 表单上传（multipart，字段 `files`） |
| POST /api/file/clean | 清理无用文件（移入 `$UPLOAD_DIR/removed`） |
| POST /api/file/s3PreSigned | S3 预签名上传地址 |

### Friend

| 路径 | 说明 |
|------|------|
| POST /api/friend/add / delete | 友链管理 |
