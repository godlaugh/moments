package db

import (
	"time"
)

// Tag 多级标签，任意层级，通过 parentId 构成树，parentId=0 为根
// 同一用户同一父节点下 name 唯一（唯一索引 idx_tag_unique）
type Tag struct {
	Id        int32      `gorm:"column:id;primary_key;NOT NULL" json:"id,omitempty"`
	UserId    int32      `gorm:"column:userId;NOT NULL;index:idx_tag_user" json:"userId,omitempty"`
	ParentId  int32      `gorm:"column:parentId;default:0;NOT NULL" json:"parentId,omitempty"`
	Name      string     `gorm:"column:name;NOT NULL;uniqueIndex:idx_tag_unique,priority:3" json:"name,omitempty"`
	CreatedAt *time.Time `gorm:"column:createdAt;default:CURRENT_TIMESTAMP;NOT NULL" json:"createdAt,omitempty"`
	UpdatedAt *time.Time `gorm:"column:updatedAt;NOT NULL" json:"updatedAt,omitempty"`
}

func (t *Tag) TableName() string {
	return "Tag"
}

// MemoTag memo 与 tag 的关联表
// (memoId, tagId) 联合唯一，tagId 单独索引供按标签筛选 memo
type MemoTag struct {
	Id     int32 `gorm:"column:id;primary_key;NOT NULL" json:"id,omitempty"`
	MemoId int32 `gorm:"column:memoId;NOT NULL;uniqueIndex:idx_memotag_memo_tag,priority:1" json:"memoId,omitempty"`
	TagId  int32 `gorm:"column:tagId;NOT NULL;uniqueIndex:idx_memotag_memo_tag,priority:2;index:idx_memotag_tag" json:"tagId,omitempty"`
}

func (t *MemoTag) TableName() string {
	return "MemoTag"
}
