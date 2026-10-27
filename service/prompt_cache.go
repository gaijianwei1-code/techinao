package service

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
)

// 首次启动自动同步的来源，避免安装完提示词库是空的。
// 只处理这一个来源，其余分类仍由后台「提示词管理」手动同步和每日定时任务刷新。
const firstRunPromptCategory = "youmind-gpt-image-2"

// 首次启动抓取失败时的重试次数。
const firstRunFetchAttempts = 3

// 提示词本地缓存目录，可用 PROMPT_DATA_DIR 覆盖，默认 data/prompts。
func promptCacheDir() string {
	if value := strings.TrimSpace(os.Getenv("PROMPT_DATA_DIR")); value != "" {
		return value
	}
	return filepath.Join("data", "prompts")
}

func promptCachePath(category string) string {
	return filepath.Join(promptCacheDir(), category+".json")
}

// 同步结果在本地留一份 JSON 缓存，没有网络时也能导入。
func writePromptCache(category string, items []model.Prompt) {
	dir := promptCacheDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("prompt cache mkdir failed dir=%s err=%v", dir, err)
		return
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		log.Printf("prompt cache marshal failed category=%s err=%v", category, err)
		return
	}
	if err := os.WriteFile(promptCachePath(category), data, 0o644); err != nil {
		log.Printf("prompt cache write failed category=%s err=%v", category, err)
		return
	}
	log.Printf("prompt cache written category=%s count=%d", category, len(items))
}

func readPromptCache(category string) ([]model.Prompt, error) {
	data, err := os.ReadFile(promptCachePath(category))
	if err != nil {
		return nil, err
	}
	var items []model.Prompt
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("提示词缓存为空")
	}
	return items, nil
}

// EnsurePromptCacheOnFirstRun 在分类还没有数据时导入一次：优先用本地缓存，没有缓存再联网抓取。
func EnsurePromptCacheOnFirstRun() {
	category, ok := repository.PromptCategoryByCode(firstRunPromptCategory)
	if !ok {
		return
	}
	count, err := repository.CountPromptsByCategory(category.Category)
	if err != nil {
		log.Printf("first run prompt check failed category=%s err=%v", category.Category, err)
		return
	}
	if count > 0 {
		return
	}

	items, err := readPromptCache(category.Category)
	if err != nil {
		log.Printf("prompt cache miss category=%s err=%v, fetching remote", category.Category, err)
		// 安装后网络抖动很常见，多试几次，避免首次启动留下空提示词库
		for attempt := 1; attempt <= firstRunFetchAttempts; attempt++ {
			items, err = buildPromptCategory(category.Category)
			if err == nil {
				break
			}
			log.Printf("first run prompt fetch failed category=%s attempt=%d err=%v", category.Category, attempt, err)
			if attempt < firstRunFetchAttempts {
				time.Sleep(time.Duration(attempt) * 10 * time.Second)
			}
		}
		if err != nil {
			log.Printf("first run prompt sync failed category=%s err=%v", category.Category, err)
			return
		}
		writePromptCache(category.Category, items)
	} else {
		log.Printf("prompt cache hit category=%s count=%d", category.Category, len(items))
	}

	if err := savePromptCategoryItems(category, items); err != nil {
		log.Printf("first run prompt import failed category=%s err=%v", category.Category, err)
		return
	}
	log.Printf("first run prompt import done category=%s count=%d", category.Category, len(items))
}
