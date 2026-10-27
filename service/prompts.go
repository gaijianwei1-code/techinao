package service

import (
	"regexp"
	"strings"
	"time"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
)

// videoPromptCategory 视频模板分类：各来源里与视频、分镜、剧照相关的提示词统一归到这里。
const videoPromptCategory = "video-templates"

var (
	videoPromptKeywords   = regexp.MustCompile(`(?i)视频|分镜|剧照|video|storyboard`)
	videoPromptImageLinks = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)|https?://\S+`)
)

func ListPrompts(q model.Query) (model.PromptList, error) {
	items, total, err := repository.ListPrompts(q)
	if err != nil {
		return model.PromptList{}, err
	}
	tags, err := repository.ListPromptTags(q)
	if err != nil {
		return model.PromptList{}, err
	}
	categories := promptCategoryOptions(ListPromptCategories())
	return model.PromptList{Items: items, Tags: tags, Categories: categories, Total: int(total)}, nil
}

func ListPromptCategories() []model.PromptCategory {
	categories, _ := repository.ListPromptCategories()
	return categories
}

func SavePrompt(item model.Prompt) (model.Prompt, error) {
	now := time.Now().Format(time.RFC3339)
	if item.Category == "" {
		item.Category = repository.PromptCategories()[0].Category
	}
	if item.ID == "" {
		item.ID = newID(item.Category)
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	category, ok := repository.PromptCategoryByCode(item.Category)
	if !ok {
		category = repository.PromptCategories()[0]
		item.Category = category.Category
	}
	item.GithubURL = ""
	return repository.SavePrompt(item)
}

func DeletePrompt(id string) error {
	return repository.DeletePrompt(id)
}

func DeletePrompts(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return repository.DeletePrompts(ids)
}

// savePromptCategoryItems 写入某个来源的提示词，并把视频类条目归入「视频模板」分类。
func savePromptCategoryItems(category model.PromptCategory, items []model.Prompt) error {
	video, rest := splitVideoPrompts(items)
	if err := repository.ReplacePromptCategory(category, rest); err != nil {
		return err
	}
	if videoCategory, ok := repository.PromptCategoryByCode(videoPromptCategory); ok {
		return repository.MergePromptCategoryItems(videoCategory, category.Category+"-", video)
	}
	return nil
}

// splitVideoPrompts 把提示词拆成视频模板与其余条目。
func splitVideoPrompts(items []model.Prompt) (video []model.Prompt, rest []model.Prompt) {
	for _, item := range items {
		if isVideoPrompt(item) {
			video = append(video, item)
			continue
		}
		rest = append(rest, item)
	}
	return video, rest
}

// isVideoPrompt 判断提示词是否与视频、分镜、剧照相关，忽略预览里的图片地址。
func isVideoPrompt(item model.Prompt) bool {
	text := strings.Join([]string{item.Title, item.Prompt, strings.Join(item.Tags, " "), videoPromptImageLinks.ReplaceAllString(item.Preview, " ")}, "\n")
	return videoPromptKeywords.MatchString(text)
}

func promptCategoryOptions(items []model.PromptCategory) []model.PromptCategoryOption {
	options := []model.PromptCategoryOption{}
	for _, item := range items {
		if item.Category == "" {
			continue
		}
		options = append(options, model.PromptCategoryOption{Category: item.Category, Name: item.Name})
	}
	return options
}
