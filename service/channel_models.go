package service

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/tigerowo/infinite-canvas/model"
)

// FetchUserChannelModels 由服务端代用户拉取上游 /models 模型列表。
// 本地补丁：部分上游（如 aiaaa.cc）不返回 CORS 头，浏览器直连必失败，因此改由服务端请求。
// 走 SafeProxyHTTPClient（禁止访问内网地址），避免被当成 SSRF 跳板。
func FetchUserChannelModels(protocol string, name string, baseURL string, apiKey string) ([]string, error) {
	channel := model.ModelChannel{
		Protocol: strings.ToLower(strings.TrimSpace(protocol)),
		Name:     strings.TrimSpace(name),
		BaseURL:  strings.TrimSpace(baseURL),
		APIKey:   strings.TrimSpace(apiKey),
		Weight:   1,
		Timeout:  60,
		Enabled:  true,
	}
	if channel.Protocol == "" {
		channel.Protocol = "openai"
	}
	if channel.Name == "" {
		channel.Name = "本地直连"
	}
	if channel.BaseURL == "" {
		return nil, safeMessageError{message: "缺少接口地址"}
	}
	if channel.APIKey == "" {
		return nil, safeMessageError{message: "缺少 API Key"}
	}

	request, err := http.NewRequest(http.MethodGet, BuildModelChannelURL(channel, "/models"), nil)
	if err != nil {
		return nil, safeMessageError{message: "接口地址无效"}
	}
	SetModelChannelAuthHeader(request, channel)
	response, err := SafeProxyHTTPClient().Do(request)
	if err != nil {
		return nil, safeMessageError{message: "读取模型失败：上游接口无响应或网络不可达"}
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode >= http.StatusBadRequest {
		return nil, readAdminChannelError(body, response.StatusCode, "读取模型失败")
	}

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, safeMessageError{message: "读取模型失败：上游返回内容无法解析"}
	}
	seen := map[string]bool{}
	result := make([]string, 0, len(payload.Data)+len(payload.Models))
	for _, item := range payload.Data {
		name := strings.TrimSpace(item.ID)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	for _, item := range payload.Models {
		name := strings.TrimSpace(item.Name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}
