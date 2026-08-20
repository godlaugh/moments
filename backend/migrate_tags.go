package main

import (
	"strings"

	"github.com/kingwrcy/moments/db"
	"github.com/kingwrcy/moments/handler"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// migrateFlatTagsToTree 把存量 Memo.tags 扁平标签（"a,b,c,"）导入 Tag/MemoTag 表
// 幂等：已有 MemoTag 记录的 memo 跳过，重复启动无副作用
func migrateFlatTagsToTree(tx *gorm.DB, log zerolog.Logger) {
	var memos []db.Memo
	tx.Where("tags is not null and tags != ''").Find(&memos)
	if len(memos) == 0 {
		return
	}

	var migrated, skipped int
	for _, memo := range memos {
		var cnt int64
		tx.Model(&db.MemoTag{}).Where("memoId = ?", memo.Id).Count(&cnt)
		if cnt > 0 {
			skipped++
			continue
		}

		raw := *memo.Tags
		if strings.HasSuffix(raw, ",") {
			raw = raw[:len(raw)-1]
		}
		var paths []string
		for _, seg := range strings.Split(raw, ",") {
			seg = strings.TrimSpace(seg)
			// 兼容历史数据：路径形式（旧版本多级回滚场景）原样保留，扁平名成为一级标签
			if seg == "" {
				continue
			}
			if err := handler.ValidateTagPath(seg); err != nil {
				log.Warn().Msgf("迁移标签跳过非法段,memoId:%d,段:%q", memo.Id, seg)
				continue
			}
			paths = append(paths, seg)
		}
		if len(paths) == 0 {
			continue
		}

		err := tx.Transaction(func(txb *gorm.DB) error {
			for _, p := range paths {
				tagId, err := handler.UpsertTagPath(txb, memo.UserId, p)
				if err != nil {
					return err
				}
				if err := txb.Create(&db.MemoTag{MemoId: memo.Id, TagId: tagId}).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Error().Msgf("迁移memo标签失败,memoId:%d,原因:%s", memo.Id, err)
			continue
		}
		migrated++
	}
	if migrated > 0 || skipped > 0 {
		log.Info().Msgf("扁平标签迁移完成:本次迁移%d条,已迁移跳过%d条", migrated, skipped)
	}
}
