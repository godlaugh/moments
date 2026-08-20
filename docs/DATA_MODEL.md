# 数据模型文档

> 定义在 `backend/db/*.go`，GORM + SQLite，启动时 `AutoMigrate`（`db/db.go`）。表名即结构体名。
> **标签已升级为多级标签**（2026-08 multi-level-tag 分支），新表结构见下文 [标签存储](#标签存储重点)。

## 表一览

| 表 | 模型文件 | 说明 |
|----|---------|------|
| User | `db/user.go` | 用户，含站点个性化配置（favicon/title/备案号/自定义 CSS/JS） |
| Memo | `db/memo.go` | 动态（朋友圈帖子），核心表 |
| Comment | `db/comment.go` | 评论，`memoId` 关联（无外键约束，逻辑关联） |
| Friend | `db/friend.go` | 友情链接 |
| SysConfig | `db/sysConfig.go` | 全局配置，单行 JSON blob |
| **Tag** | `db/tag.go` | **多级标签**，`parentId` 构成任意层级树 |
| **MemoTag** | `db/tag.go` | **memo↔tag 关联表**，联合唯一 |

## Memo（核心表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int32 PK | |
| content | string | Markdown 正文 |
| imgs | string | 图片路径，**JSON 数组字符串**（配合 `imgConfigs`） |
| favCount / commentCount | int32 | 冗余计数字段 |
| userId | int32 | 作者 |
| createdAt / updatedAt | *time.Time | `createdAt` 可由用户指定（未来时间 → 游客不可见） |
| music163Url / bilibiliUrl | string | 外部音乐/视频 |
| location | string | 定位 |
| externalUrl / externalTitle / externalFavicon | string | 外链卡片三件套 |
| pinned | *bool | 置顶 |
| ext | string | **JSON blob**，存 `vo.MemoExt`：音乐、豆瓣书影音等扩展 |
| showType | *int32 | 可见性（1=公开等） |
| tags | string | **逗号分隔的标签字符串**，如 `"a,b,c"` |

非数据库字段（`gorm:"-"` 或查询时拼装）：

- `User *User`、`Comments []Comment`：关联数据，查询后手工拼
- `Tags *string`：模型上无 gorm 列标签但映射 `tags` 列（逗号串原样出入）
- `ImgConfigs *[]*vo.ImgConfig`：`gorm:"-"`，前端图片配置，来源 `vo.MemoExt` / imgs 解析

## 标签存储（重点）

### 当前实现（multi-level-tag 分支，2026-08）

**独立 Tag 表 + MemoTag 关联表，任意层级**：

- `Tag(id, userId, parentId, name, createdAt, updatedAt)`：`parentId=0` 为根；同一用户同父下 name 唯一（唯一索引 `idx_tag_unique`）；`userId` 有索引
- `MemoTag(memoId, tagId)`：联合唯一 `idx_memotag_memo_tag`；`tagId` 单独索引供筛选
- `Memo.tags` 老列**保留双写**（回滚兼容）：存全路径逗号串+尾逗号，如 `"技术/Go,生活,"`
- `Memo.TagPaths`（`gorm:"-"`）：查询时由 MemoTag 批量生成，如 `["技术/Go","生活"]`

**关键机制**：

1. **写入**（`handler/tag.go` `syncMemoTags`，事务内）：删旧关联 → 校验路径（禁逗号、≤10 级、单段 ≤50 字符）→ 逐级 upsert Tag → 重建 MemoTag → 回写老 tags 列
2. **路径即标签**：前端提交 `"技术/前端/Vue"` 这类全路径，中间层级自动创建
3. **读取**（`/api/tag/list`）：返回树（`tree`，含 `count` 直接计数 / `total` 含子孙计数）+ 扁平全路径列表（`tags`，兼容老用法）；仅展示被引用的标签（`total>0` 剪枝）
4. **筛选**（`handler/memo.go` ListMemos）：按路径匹配（名字忽略大小写）→ 展开为"自身+全部子孙"id 集 → `id IN (SELECT memoId FROM MemoTag WHERE tagId IN ?)`；多标签 AND；Tag 表无匹配时**兜底回退老 `LIKE '%tag,%'`**（保护未迁移数据）
5. **迁移**（`backend/migrate_tags.go`）：启动时把存量扁平 `Memo.tags` 导入 Tag/MemoTag；已有 MemoTag 的 memo 跳过，**幂等**
6. **删除 memo**：同步清理 MemoTag

**用户隔离**：tag/list 按 username/当前用户过滤；筛选时同用户标签才参与路径匹配。

### 历史格式（老版本，仅兜底兼容）

- 存储：`Memo.tags = "a,b,c,"`（带尾逗号逗号分隔串）
- 筛选：`tags LIKE '%tag,%'`（尾逗号是命中前提）
- 逗号不可转义，标签名不能含逗号

## SysConfig（JSON blob）

`Content` 列存一个 JSON 字符串，结构见 `vo/sysConfig_vo.go`（是否允许注册、是否开评论、邮件配置、grecaptcha 等）。读写均整存整取，`/api/sysConfig/getFull`（管理员）返回全部，`/get` 返回脱敏后的公共部分。

## User 上的站点配置

每个 User 行内嵌"站点级"配置（Title、Favicon、BeianNo、Css、Js、S3 参数等）——本质是单用户站点的配置寄生在用户行上。S3 密钥明文存库，`/api/user/profile` 对非本人脱敏。

## 约定与坑

- 所有表无外键、无索引（除主键/unique），关联全靠应用层维护
- 计数字段（favCount/commentCount）在事务里增减，直接改表不会自动校正
- 时间字段 `*time.Time`，SQLite 存 UTC，前端 `date-fns` 本地化展示
- 加新列：直接在模型加字段 + 重启（AutoMigrate 加列不删列；删列/改类型不会自动迁移，需手写迁移脚本，参考 `migrate.go`）
