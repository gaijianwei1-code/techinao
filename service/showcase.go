package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
)

// PublishShowcaseInput 发布精选作品入参；data 为画布项目完整 JSON。
type PublishShowcaseInput struct {
	ProjectID   string          `json:"projectId"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Cover       string          `json:"cover"`
	Preview     string          `json:"preview"`
	Data        json.RawMessage `json:"data"`
}

func ListShowcaseWorks(q model.Query, userID string) (model.ShowcaseWorkList, error) {
	q.Normalize()
	items, total, err := repository.ListShowcaseWorks(q, model.ShowcaseStatusPublished)
	if err != nil {
		return model.ShowcaseWorkList{}, err
	}
	summaries := make([]model.ShowcaseWorkSummary, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, showcaseSummary(item, userID))
	}
	return model.ShowcaseWorkList{
		Items:      summaries,
		Total:      int(total),
		Categories: model.ShowcaseCategories,
	}, nil
}

// ListAllShowcaseWorks 管理端列表，不过滤上架状态。
func ListAllShowcaseWorks(q model.Query, userID string) (model.ShowcaseWorkList, error) {
	q.Normalize()
	items, total, err := repository.ListShowcaseWorks(q, "")
	if err != nil {
		return model.ShowcaseWorkList{}, err
	}
	summaries := make([]model.ShowcaseWorkSummary, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, showcaseSummary(item, userID))
	}
	return model.ShowcaseWorkList{
		Items:      summaries,
		Total:      int(total),
		Categories: model.ShowcaseCategories,
	}, nil
}

func GetShowcaseWorkDetail(id string, userID string) (model.ShowcaseWork, error) {
	work, err := repository.GetShowcaseWork(id)
	if err != nil {
		return model.ShowcaseWork{}, err
	}
	if work.Status != model.ShowcaseStatusPublished && work.UserID != userID {
		return model.ShowcaseWork{}, errors.New("作品不存在或已下架")
	}
	return work, nil
}

func PublishShowcaseWork(ctx context.Context, input PublishShowcaseInput) (model.ShowcaseWork, error) {
	user, ok := UserFromContext(ctx)
	if !ok || user.ID == "" {
		return model.ShowcaseWork{}, errors.New("请先登录")
	}
	if len(input.Data) == 0 {
		return model.ShowcaseWork{}, errors.New("画布项目数据不能为空")
	}
	var project struct {
		Title string `json:"title"`
	}
	_ = json.Unmarshal(input.Data, &project)

	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = strings.TrimSpace(project.Title)
	}
	if title == "" {
		return model.ShowcaseWork{}, errors.New("请填写作品标题")
	}

	category := strings.TrimSpace(input.Category)
	if !showcaseCategoryValid(category) {
		category = model.ShowcaseCategories[0]
	}

	now := time.Now().Format(time.RFC3339)
	work := model.ShowcaseWork{
		ID:           newID("showcase"),
		UserID:       user.ID,
		ProjectID:    strings.TrimSpace(input.ProjectID),
		Title:        title,
		Description:  strings.TrimSpace(input.Description),
		Category:     category,
		Cover:        strings.TrimSpace(input.Cover),
		Preview:      strings.TrimSpace(input.Preview),
		AuthorName:   strings.TrimSpace(user.DisplayName),
		AuthorAvatar: strings.TrimSpace(user.AvatarURL),
		ProjectData:  string(input.Data),
		LikedUserIDs: []string{},
		Status:       model.ShowcaseStatusPublished,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if work.AuthorName == "" {
		work.AuthorName = strings.TrimSpace(user.Username)
	}
	if err := repository.SaveShowcaseWork(work); err != nil {
		return model.ShowcaseWork{}, err
	}
	return work, nil
}

func ToggleShowcaseLike(ctx context.Context, id string) (bool, int, error) {
	user, ok := UserFromContext(ctx)
	if !ok || user.ID == "" {
		return false, 0, errors.New("请先登录")
	}
	work, err := repository.GetShowcaseWork(id)
	if err != nil {
		return false, 0, err
	}

	liked := false
	next := make([]string, 0, len(work.LikedUserIDs)+1)
	for _, uid := range work.LikedUserIDs {
		if uid == user.ID {
			liked = true
			continue
		}
		next = append(next, uid)
	}
	if liked {
		work.LikedUserIDs = next
		work.Likes--
	} else {
		work.LikedUserIDs = append(next, user.ID)
		work.Likes++
	}
	if work.Likes < 0 {
		work.Likes = 0
	}
	if err := repository.UpdateShowcaseWorkLikes(work); err != nil {
		return false, 0, err
	}
	return !liked, work.Likes, nil
}

// ForkShowcaseWork 返回精选作品的画布数据，由前端复制为用户自己的项目。
func ForkShowcaseWork(ctx context.Context, id string) (json.RawMessage, error) {
	user, ok := UserFromContext(ctx)
	if !ok || user.ID == "" {
		return nil, errors.New("请先登录")
	}
	work, err := repository.GetShowcaseWork(id)
	if err != nil {
		return nil, err
	}
	if work.Status != model.ShowcaseStatusPublished && work.UserID != user.ID {
		return nil, errors.New("作品不存在或已下架")
	}
	if strings.TrimSpace(work.ProjectData) == "" {
		return nil, errors.New("作品画布数据为空")
	}
	return json.RawMessage(work.ProjectData), nil
}

func SetShowcaseStatus(ids []string, status string) error {
	if status != model.ShowcaseStatusPublished && status != model.ShowcaseStatusHidden {
		return errors.New("状态参数无效")
	}
	return repository.UpdateShowcaseStatus(ids, status)
}

// DeleteShowcaseWork 删除精选作品。
func DeleteShowcaseWork(id string) error {
	return repository.DeleteShowcaseWork(id)
}

func showcaseSummary(work model.ShowcaseWork, userID string) model.ShowcaseWorkSummary {
	liked := false
	for _, uid := range work.LikedUserIDs {
		if uid == userID {
			liked = true
			break
		}
	}
	return model.ShowcaseWorkSummary{
		ID:           work.ID,
		UserID:       work.UserID,
		Title:        work.Title,
		Category:     work.Category,
		Cover:        work.Cover,
		AuthorName:   work.AuthorName,
		AuthorAvatar: work.AuthorAvatar,
		Likes:        work.Likes,
		Liked:        liked,
		CreatedAt:    work.CreatedAt,
	}
}

func showcaseCategoryValid(category string) bool {
	for _, item := range model.ShowcaseCategories {
		if item == category {
			return true
		}
	}
	return false
}
