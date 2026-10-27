package repository

import (
	"strings"

	"github.com/tigerowo/infinite-canvas/model"
	"gorm.io/gorm"
)

// ListShowcaseWorks 按条件返回精选作品分页列表；status 为空时不过滤（管理端）。
func ListShowcaseWorks(q model.Query, status string) ([]model.ShowcaseWork, int64, error) {
	db, err := DB()
	if err != nil {
		return nil, 0, err
	}
	q.Normalize()
	tx := applyShowcaseFilters(db.Model(&model.ShowcaseWork{}), q)
	if status != "" {
		tx = tx.Where("status = ?", status)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []model.ShowcaseWork
	// 列表不取 project_data 大字段。
	err = tx.Select(
		"id", "user_id", "project_id", "title", "description", "category",
		"cover", "preview", "author_name", "author_avatar",
		"likes", "liked_user_ids", "status", "created_at", "updated_at",
	).Order("created_at DESC").Offset(q.Offset()).Limit(q.PageSize).Find(&items).Error
	return items, total, err
}

// GetShowcaseWork 按 ID 查询精选作品（含完整画布数据）。
func GetShowcaseWork(id string) (model.ShowcaseWork, error) {
	db, err := DB()
	if err != nil {
		return model.ShowcaseWork{}, err
	}
	var work model.ShowcaseWork
	err = db.Where("id = ?", strings.TrimSpace(id)).First(&work).Error
	return work, err
}

// SaveShowcaseWork 新建或更新精选作品。
func SaveShowcaseWork(work model.ShowcaseWork) error {
	db, err := DB()
	if err != nil {
		return err
	}
	return db.Save(&work).Error
}

// UpdateShowcaseWorkLikes 更新点赞数与点赞用户列表。
// 注意：必须整行 Save，map Updates 不会走 liked_user_ids 的 JSON serializer，会写出非法 JSON。
func UpdateShowcaseWorkLikes(work model.ShowcaseWork) error {
	db, err := DB()
	if err != nil {
		return err
	}
	return db.Save(&work).Error
}

// UpdateShowcaseStatus 批量上下架精选作品。
func UpdateShowcaseStatus(ids []string, status string) error {
	ids = uniqueTrimmedValues(ids...)
	if len(ids) == 0 {
		return nil
	}
	db, err := DB()
	if err != nil {
		return err
	}
	return db.Model(&model.ShowcaseWork{}).
		Where("id IN ?", ids).
		Update("status", status).Error
}

// DeleteShowcaseWork 删除精选作品。
func DeleteShowcaseWork(id string) error {
	db, err := DB()
	if err != nil {
		return err
	}
	return db.Delete(&model.ShowcaseWork{}, "id = ?", strings.TrimSpace(id)).Error
}

// applyShowcaseFilters 应用精选列表的搜索条件。
func applyShowcaseFilters(tx *gorm.DB, q model.Query) *gorm.DB {
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		tx = tx.Where("title LIKE ? OR author_name LIKE ? OR description LIKE ?", like, like, like)
	}
	if q.Category != "" {
		tx = tx.Where("category = ?", q.Category)
	}
	return tx
}
