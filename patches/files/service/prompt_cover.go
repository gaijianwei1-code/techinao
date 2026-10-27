package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/tigerowo/infinite-canvas/model"
)

// 提示词封面本地缓存：YouMind 等来源的封面图托管在 cms-assets.youmind.com，
// 国内直连不通，同步时把图片抓到本地并把封面地址改写为 /api/prompt-covers/<hash>。
const (
	promptCoverRoute      = "/api/prompt-covers/"
	promptCoverWorkers    = 8
	promptCoverDirectWait = 8 * time.Second
	promptCoverMirrorWait = 30 * time.Second
	promptCoverMaxBytes   = 20 << 20
)

// 源站直连失败时改用图片镜像抓取，可用 PROMPT_IMAGE_MIRROR 覆盖（置空则只走直连）。
var promptImageMirror = strings.TrimSpace(promptImageMirrorEnv())

func promptImageMirrorEnv() string {
	if value, ok := os.LookupEnv("PROMPT_IMAGE_MIRROR"); ok {
		return value
	}
	return "https://wsrv.nl/?url="
}

// 直连失败的 host 记下来，避免每张封面都白等一次超时；进程重启后重新探测。
var promptCoverBlockedHosts sync.Map

func promptCoverMirrorURL(source string) string {
	if promptImageMirror == "" {
		return ""
	}
	return promptImageMirror + url.QueryEscape(source) + "&w=1000&output=jpg&q=82"
}

func promptCoverDir() string {
	return filepath.Join(promptCacheDir(), "covers")
}

var promptCoverNamePattern = regexp.MustCompile(`^[0-9a-f]{40}\.(jpg|jpeg|png|webp|gif|avif)$`)

// ReadPromptCover 读取本地封面缓存，供 /api/prompt-covers/:name 使用。
func ReadPromptCover(name string) ([]byte, string, error) {
	if !promptCoverNamePattern.MatchString(name) {
		return nil, "", errors.New("封面不存在")
	}
	data, err := os.ReadFile(filepath.Join(promptCoverDir(), name))
	if err != nil {
		return nil, "", errors.New("封面不存在")
	}
	return data, http.DetectContentType(data), nil
}

// CachePromptCovers 把远程封面抓到本地缓存，返回与入参一一对应的可访问地址；
// 抓取失败时退回图片镜像地址，没有镜像则保留原地址。
func CachePromptCovers(sources []string) []string {
	results := make([]string, len(sources))
	limit := make(chan struct{}, promptCoverWorkers)
	var wait sync.WaitGroup
	for index, source := range sources {
		if !strings.HasPrefix(source, "http://") && !strings.HasPrefix(source, "https://") {
			results[index] = source
			continue
		}
		wait.Add(1)
		go func(index int, source string) {
			defer wait.Done()
			limit <- struct{}{}
			defer func() { <-limit }()
			results[index] = cachePromptCover(source)
		}(index, source)
	}
	wait.Wait()
	return results
}

// cachePromptCovers 就地改写提示词封面地址。
func cachePromptCovers(items []model.Prompt) {
	sources := make([]string, len(items))
	for index, item := range items {
		sources[index] = item.CoverURL
	}
	for index, cover := range CachePromptCovers(sources) {
		items[index].CoverURL = cover
	}
}

func cachePromptCover(source string) string {
	name := promptCoverName(source)
	target := filepath.Join(promptCoverDir(), name)
	if _, err := os.Stat(target); err == nil {
		return promptCoverRoute + name
	}
	data, err := downloadPromptCover(source)
	if err != nil {
		log.Printf("prompt cover cache failed url=%s err=%v", source, err)
		return promptCoverFallback(source)
	}
	if err := os.MkdirAll(promptCoverDir(), 0o755); err != nil {
		log.Printf("prompt cover mkdir failed dir=%s err=%v", promptCoverDir(), err)
		return promptCoverFallback(source)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		log.Printf("prompt cover write failed name=%s err=%v", name, err)
		return promptCoverFallback(source)
	}
	return promptCoverRoute + name
}

func promptCoverName(source string) string {
	extension := ".jpg"
	if parsed, err := url.Parse(source); err == nil {
		switch value := strings.ToLower(path.Ext(parsed.Path)); value {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".avif":
			extension = value
		}
	}
	sum := sha1.Sum([]byte(source))
	return hex.EncodeToString(sum[:]) + extension
}

func downloadPromptCover(source string) ([]byte, error) {
	parsed, err := url.Parse(source)
	if err != nil {
		return nil, err
	}
	if _, blocked := promptCoverBlockedHosts.Load(parsed.Host); !blocked {
		data, err := fetchPromptCoverOnce(source, promptCoverDirectWait)
		if err == nil {
			return data, nil
		}
		promptCoverBlockedHosts.Store(parsed.Host, struct{}{})
	}
	mirror := promptCoverMirrorURL(source)
	if mirror == "" {
		return nil, errors.New("封面抓取失败")
	}
	return fetchPromptCoverOnce(mirror, promptCoverMirrorWait)
}

func fetchPromptCoverOnce(target string, wait time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	request.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	response, err := promptHTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, errors.New("封面抓取失败: " + response.Status)
	}
	if contentType := response.Header.Get("Content-Type"); contentType != "" && !strings.HasPrefix(contentType, "image/") {
		return nil, errors.New("封面返回的不是图片")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, promptCoverMaxBytes+1))
	if err != nil || len(data) == 0 || len(data) > promptCoverMaxBytes {
		return nil, errors.New("封面读取失败")
	}
	return data, nil
}

func promptCoverFallback(source string) string {
	if mirror := promptCoverMirrorURL(source); mirror != "" {
		return mirror
	}
	return source
}
