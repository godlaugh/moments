package handler

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kingwrcy/moments/db"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

type TagHandler struct {
	base BaseHandler
}

func NewTagHandler(injector do.Injector) *TagHandler {
	return &TagHandler{do.MustInvoke[BaseHandler](injector)}
}

// ---------- VO ----------

type tagListResp struct {
	Tags []string  `json:"tags"` // 全路径扁平列表（按路径排序），兼容老字段用法
	Tree []TagNode `json:"tree"` // 层级树
}

type TagNode struct {
	Id       int32     `json:"id"`
	Name     string    `json:"name"`
	ParentId int32     `json:"parentId"`
	Path     string    `json:"path"` // 如 "技术/Go"
	Count    int64     `json:"count"` // 直接挂在该标签上的 memo 数
	Total    int64     `json:"total"` // 含全部子孙的 memo 数
	Children []TagNode `json:"children"`
}

// ---------- 校验 ----------

const (
	maxTagNameLen = 50  // 单段名字最大长度（rune）
	maxTagDepth   = 10  // 最大层级
	maxTagPathLen = 500 // 整条路径最大长度（字节）
)

var ErrInvalidTagPath = errors.New("invalid tag path")

// ValidateTagPath 校验 "a/b/c" 形式路径：段非空、不含逗号、无超限
func ValidateTagPath(path string) error {
	if path == "" {
		return ErrInvalidTagPath
	}
	if len(path) > maxTagPathLen {
		return ErrInvalidTagPath
	}
	segs := strings.Split(path, "/")
	if len(segs) > maxTagDepth {
		return ErrInvalidTagPath
	}
	for _, seg := range segs {
		seg = strings.TrimSpace(seg)
		if seg == "" || seg == "." || seg == ".." {
			return ErrInvalidTagPath
		}
		if len([]rune(seg)) > maxTagNameLen {
			return ErrInvalidTagPath
		}
		if strings.Contains(seg, ",") {
			return ErrInvalidTagPath
		}
	}
	return nil
}

// ---------- 写入 ----------

// upsertTagPath 逐级查找或创建标签，返回叶子标签 id。须在事务内调用。
func upsertTagPath(tx *gorm.DB, userId int32, path string) (int32, error) {
	if err := ValidateTagPath(path); err != nil {
		return 0, err
	}
	segs := strings.Split(path, "/")
	parent := int32(0)
	for _, seg := range segs {
		name := strings.TrimSpace(seg)
		var found db.Tag
		err := tx.Where("userId = ? AND parentId = ? AND name = ?", userId, parent, name).First(&found).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			now := time.Now()
			created := db.Tag{UserId: userId, ParentId: parent, Name: name, CreatedAt: &now, UpdatedAt: &now}
			if err := tx.Create(&created).Error; err != nil {
				// 并发下唯一索引冲突则重查
				var retry db.Tag
				if e2 := tx.Where("userId = ? AND parentId = ? AND name = ?", userId, parent, name).First(&retry).Error; e2 == nil {
					created = retry
				} else {
					return 0, err
				}
			}
			found = created
		} else if err != nil {
			return 0, err
		}
		parent = found.Id
	}
	return parent, nil
}

// syncMemoTags 重建 memo 的标签关联，并同步老 tags 列（全路径+尾逗号，回滚兼容）。须在事务内调用。
func syncMemoTags(tx *gorm.DB, memo *db.Memo, userId int32, paths []string) error {
	if err := tx.Where("memoId = ?", memo.Id).Delete(&db.MemoTag{}).Error; err != nil {
		return err
	}
	seen := map[string]bool{}
	var tagIds []int32
	var validPaths []string
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if err := ValidateTagPath(p); err != nil {
			return err
		}
		if seen[p] {
			continue
		}
		seen[p] = true
		id, err := upsertTagPath(tx, userId, p)
		if err != nil {
			return err
		}
		tagIds = append(tagIds, id)
		validPaths = append(validPaths, p)
	}
	for _, tagId := range tagIds {
		if err := tx.Create(&db.MemoTag{MemoId: memo.Id, TagId: tagId}).Error; err != nil {
			return err
		}
	}
	var oldCol *string
	if len(validPaths) > 0 {
		s := strings.Join(validPaths, ",") + ","
		oldCol = &s
	}
	memo.Tags = oldCol
	return tx.Model(&db.Memo{}).Where("id = ?", memo.Id).Update("tags", oldCol).Error
}

// UpsertTagPath 导出版本，供存量数据迁移使用
func UpsertTagPath(tx *gorm.DB, userId int32, path string) (int32, error) {
	return upsertTagPath(tx, userId, path)
}

// ---------- 读取 ----------

func loadUserTags(q *gorm.DB, userId int32) ([]db.Tag, error) {
	var tags []db.Tag
	err := q.Where("userId = ?", userId).Order("name asc").Find(&tags).Error
	return tags, err
}

// memoTagCounts 批量统计各标签直接引用的 memo 数
func memoTagCounts(q *gorm.DB, tags []db.Tag) map[int32]int64 {
	counts := map[int32]int64{}
	if len(tags) == 0 {
		return counts
	}
	ids := make([]int32, len(tags))
	for i := range tags {
		ids[i] = tags[i].Id
	}
	type cntRow struct {
		TagId int32
		Cnt   int64
	}
	var rows []cntRow
	q.Model(&db.MemoTag{}).Select("tagId as tag_id, count(*) as cnt").Where("tagId IN ?", ids).Group("tagId").Scan(&rows)
	for _, r := range rows {
		counts[r.TagId] = r.Cnt
	}
	return counts
}

// buildTagTree 把扁平 Tag 列表组装为树；count/total 依据 counts 计算；
// path 为全路径；parentId 悬空（父不存在）按根处理，路径递归带 visited 防环。
func buildTagTree(tags []db.Tag, counts map[int32]int64) []TagNode {
	if len(tags) == 0 {
		return []TagNode{}
	}
	byId := map[int32]*db.Tag{}
	for i := range tags {
		byId[tags[i].Id] = &tags[i]
	}
	pathOf := func(t *db.Tag) string {
		var segs []string
		visited := map[int32]bool{}
		cur := t
		for cur != nil && !visited[cur.Id] {
			visited[cur.Id] = true
			segs = append([]string{cur.Name}, segs...)
			parent, ok := byId[cur.ParentId]
			if !ok || parent.Id == cur.Id {
				break
			}
			cur = parent
		}
		return strings.Join(segs, "/")
	}
	childrenOf := map[int32][]*db.Tag{}
	for i := range tags {
		t := &tags[i]
		if _, ok := byId[t.ParentId]; !ok || t.ParentId == t.Id {
			childrenOf[0] = append(childrenOf[0], t)
		} else {
			childrenOf[t.ParentId] = append(childrenOf[t.ParentId], t)
		}
	}
	var build func(parentId int32) []TagNode
	build = func(parentId int32) []TagNode {
		children := childrenOf[parentId]
		if len(children) == 0 {
			return []TagNode{}
		}
		nodes := make([]TagNode, 0, len(children))
		for _, t := range children {
			node := TagNode{
				Id:       t.Id,
				Name:     t.Name,
				ParentId: t.ParentId,
				Path:     pathOf(t),
				Count:    counts[t.Id],
			}
			node.Children = build(t.Id)
			node.Total = node.Count
			for _, c := range node.Children {
				node.Total += c.Total
			}
			nodes = append(nodes, node)
		}
		sort.Slice(nodes, func(i, j int) bool { return nodes[i].Name < nodes[j].Name })
		return nodes
	}
	return build(0)
}

// pruneTagTree 仅保留 total>0 的节点（对齐老版"只显示被引用标签"的行为）
func pruneTagTree(nodes []TagNode) []TagNode {
	result := []TagNode{}
	for _, n := range nodes {
		n.Children = pruneTagTree(n.Children)
		if n.Total > 0 {
			result = append(result, n)
		}
	}
	return result
}

// treePaths 树转全路径扁平列表
func treePaths(nodes []TagNode) []string {
	paths := []string{}
	var walk func(ns []TagNode)
	walk = func(ns []TagNode) {
		for _, n := range ns {
			paths = append(paths, n.Path)
			walk(n.Children)
		}
	}
	walk(nodes)
	sort.Strings(paths)
	return paths
}

// attachTagPaths 为 memo 列表批量填充 TagPaths（两次批量查询，无 N+1）
func attachTagPaths(q *gorm.DB, memos []db.Memo) error {
	if len(memos) == 0 {
		return nil
	}
	memoIds := make([]int32, len(memos))
	for i := range memos {
		memoIds[i] = memos[i].Id
	}
	var relations []db.MemoTag
	if err := q.Where("memoId IN ?", memoIds).Find(&relations).Error; err != nil {
		return err
	}
	if len(relations) == 0 {
		return nil
	}
	tagIdSet := map[int32]bool{}
	for _, r := range relations {
		tagIdSet[r.TagId] = true
	}
	tagIds := make([]int32, 0, len(tagIdSet))
	for id := range tagIdSet {
		tagIds = append(tagIds, id)
	}
	var tags []db.Tag
	if err := q.Where("id IN ?", tagIds).Find(&tags).Error; err != nil {
		return err
	}
	// 迭代补齐祖先节点（叶子查询不含父标签，逐轮上溯直到全部闭合）
	for round := 0; round < maxTagDepth; round++ {
		missing := map[int32]bool{}
		for i := range tags {
			if tags[i].ParentId > 0 {
				found := false
				for j := range tags {
					if tags[j].Id == tags[i].ParentId {
						found = true
						break
					}
				}
				if !found {
					missing[tags[i].ParentId] = true
				}
			}
		}
		if len(missing) == 0 {
			break
		}
		parentIds := make([]int32, 0, len(missing))
		for id := range missing {
			parentIds = append(parentIds, id)
		}
		var parents []db.Tag
		if err := q.Where("id IN ?", parentIds).Find(&parents).Error; err != nil {
			return err
		}
		if len(parents) == 0 {
			break
		}
		tags = append(tags, parents...)
	}
	byId := map[int32]*db.Tag{}
	for i := range tags {
		byId[tags[i].Id] = &tags[i]
	}
	pathCache := map[int32]string{}
	var pathOf func(t *db.Tag) string
	pathOf = func(t *db.Tag) string {
		if p, ok := pathCache[t.Id]; ok {
			return p
		}
		// 用负值标记计算中，防环
		pathCache[t.Id] = ""
		var segs []string
		visited := map[int32]bool{}
		cur := t
		for cur != nil && !visited[cur.Id] {
			visited[cur.Id] = true
			segs = append([]string{cur.Name}, segs...)
			parent, ok := byId[cur.ParentId]
			if !ok || parent.Id == cur.Id {
				break
			}
			cur = parent
		}
		p := strings.Join(segs, "/")
		pathCache[t.Id] = p
		return p
	}
	memoPaths := map[int32][]string{}
	for _, r := range relations {
		if t, ok := byId[r.TagId]; ok {
			memoPaths[r.MemoId] = append(memoPaths[r.MemoId], pathOf(t))
		}
	}
	for i := range memos {
		if ps, ok := memoPaths[memos[i].Id]; ok {
			sort.Strings(ps)
			memos[i].TagPaths = ps
		}
	}
	return nil
}

// ---------- 筛选 ----------

// tagIdsForPath 在 scope 用户（nil=全部用户）的标签森林中按路径查找标签（名字忽略大小写），
// 返回"自身+全部子孙"的 id 集合
func tagIdsForPath(tags []db.Tag, scopeUserId *int32, path string) map[int32]bool {
	result := map[int32]bool{}
	if path == "" {
		return result
	}
	segs := strings.Split(strings.ToLower(strings.TrimSpace(path)), "/")
	byLowerName := map[string][]*db.Tag{} // key: parentId|lower(name)
	for i := range tags {
		if scopeUserId != nil && tags[i].UserId != *scopeUserId {
			continue
		}
		key := fmt.Sprintf("%d|%s", tags[i].ParentId, strings.ToLower(tags[i].Name))
		byLowerName[key] = append(byLowerName[key], &tags[i])
	}
	childrenOf := map[int32][]*db.Tag{}
	for i := range tags {
		if scopeUserId != nil && tags[i].UserId != *scopeUserId {
			continue
		}
		childrenOf[tags[i].ParentId] = append(childrenOf[tags[i].ParentId], &tags[i])
	}
	var collect func(t *db.Tag)
	collect = func(t *db.Tag) {
		if result[t.Id] {
			return
		}
		result[t.Id] = true
		for _, c := range childrenOf[t.Id] {
			collect(c)
		}
	}
	var descend func(parentId int32, segIdx int)
	descend = func(parentId int32, segIdx int) {
		if segIdx >= len(segs) {
			return
		}
		key := fmt.Sprintf("%d|%s", parentId, segs[segIdx])
		for _, t := range byLowerName[key] {
			if segIdx == len(segs)-1 {
				collect(t)
			} else {
				descend(t.Id, segIdx+1)
			}
		}
	}
	descend(0, 0)
	return result
}

// ---------- API ----------

// List godoc
//
//	@Tags		Tag
//	@Summary	标签列表(层级树)
//	@Accept		json
//	@Produce	json
//	@Param		username	query		string	false	"按用户名查询该用户的标签树(游客可用的浏览场景);不传则取当前登录用户"
//	@Param		x-api-token	header		string	false	"登录TOKEN"
//	@Success	200			{object}	tagListResp
//	@Router		/api/tag/list [post]
func (t TagHandler) List(c echo.Context) error {
	ctx := c.(CustomContext)
	currentUser := ctx.CurrentUser()

	var userId int32
	if username := c.QueryParam("username"); username != "" {
		var target db.User
		err := t.base.db.Where("username = ?", username).First(&target).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return FailRespWithMsg(c, Fail, "不存在的用户")
		} else if err != nil {
			return FailResp(c, Fail)
		}
		userId = target.Id
	} else {
		if currentUser == nil {
			return SuccessResp(c, tagListResp{Tags: []string{}, Tree: []TagNode{}})
		}
		userId = currentUser.Id
	}

	tags, err := loadUserTags(t.base.db, userId)
	if err != nil {
		return FailResp(c, Fail)
	}
	counts := memoTagCounts(t.base.db, tags)
	tree := pruneTagTree(buildTagTree(tags, counts))
	return SuccessResp(c, tagListResp{
		Tags: treePaths(tree),
		Tree: tree,
	})
}
