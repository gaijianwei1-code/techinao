package handler

import (
	"encoding/json"
	"net/http"

	"github.com/tigerowo/infinite-canvas/service"
)

// userChannelModelsRequest 用户侧「拉取模型列表」请求。
// 本地补丁：由服务端代拉上游 /models，绕开上游不支持浏览器跨域（CORS）导致的前端直连失败。
type userChannelModelsRequest struct {
	Protocol string `json:"protocol"`
	Name     string `json:"name"`
	BaseURL  string `json:"baseUrl"`
	APIKey   string `json:"apiKey"`
}

func UserChannelModels(w http.ResponseWriter, r *http.Request) {
	if _, ok := service.UserFromContext(r.Context()); !ok {
		Fail(w, "未登录或权限不足")
		return
	}
	var request userChannelModelsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		Fail(w, "请求参数错误")
		return
	}
	models, err := service.FetchUserChannelModels(request.Protocol, request.Name, request.BaseURL, request.APIKey)
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, models)
}
