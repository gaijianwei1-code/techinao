package model

const (
	ShowcaseStatusPublished = "published"
	ShowcaseStatusHidden    = "hidden"
)

// ShowcaseCategories 创作精选的展示分类。
var ShowcaseCategories = []string{"精选画布", "专业影视", "短剧漫剧", "动漫游戏", "教育生活"}

// ShowcaseWork 创作精选作品，发布时复制画布项目完整数据。
type ShowcaseWork struct {
	ID           string   `json:"id" gorm:"primaryKey"`
	UserID       string   `json:"userId" gorm:"index"`
	ProjectID    string   `json:"projectId"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Category     string   `json:"category"`
	Cover        string   `json:"cover"`
	Preview      string   `json:"preview"`
	AuthorName   string   `json:"authorName"`
	AuthorAvatar string   `json:"authorAvatar"`
	ProjectData  string   `json:"projectData" gorm:"type:text"`
	Likes        int      `json:"likes"`
	LikedUserIDs []string `json:"-" gorm:"serializer:json"`
	Status       string   `json:"status" gorm:"index;default:published"`
	CreatedAt    string   `json:"createdAt" gorm:"index"`
	UpdatedAt    string   `json:"updatedAt"`
}

// ShowcaseWorkSummary 精选列表摘要，不包含 ProjectData 大字段。
type ShowcaseWorkSummary struct {
	ID           string `json:"id"`
	UserID       string `json:"userId"`
	Title        string `json:"title"`
	Category     string `json:"category"`
	Cover        string `json:"cover"`
	AuthorName   string `json:"authorName"`
	AuthorAvatar string `json:"authorAvatar"`
	Likes        int    `json:"likes"`
	Liked        bool   `json:"liked"`
	CreatedAt    string `json:"createdAt"`
}

// ShowcaseWorkList 精选分页结果。
type ShowcaseWorkList struct {
	Items       []ShowcaseWorkSummary `json:"items"`
	Total       int                   `json:"total"`
	Categories  []string              `json:"categories"`
}
