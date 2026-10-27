package handler

import (
	"encoding/json"
	"net/http"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/service"
)

// showcaseUserID 从请求上下文取当前用户 ID（未登录返回空）。
func showcaseUserID(r *http.Request) string {
	user, ok := service.UserFromContext(r.Context())
	if !ok {
		return ""
	}
	return user.ID
}

// ShowcaseWorks 精选列表（公开，登录后附带点赞状态）。
func ShowcaseWorks(w http.ResponseWriter, r *http.Request) {
	result, err := service.ListShowcaseWorks(parseQuery(r), showcaseUserID(r))
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, result)
}

// ShowcaseWorkDetail 精选作品详情（含完整画布数据，用于预览与创作过程回放）。
func ShowcaseWorkDetail(w http.ResponseWriter, r *http.Request, id string) {
	result, err := service.GetShowcaseWorkDetail(id, showcaseUserID(r))
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, result)
}

// PublishShowcaseWork 发布画布项目到创作精选。
func PublishShowcaseWork(w http.ResponseWriter, r *http.Request) {
	var input service.PublishShowcaseInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Fail(w, "发布参数无效")
		return
	}
	result, err := service.PublishShowcaseWork(r.Context(), input)
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, result)
}

// LikeShowcaseWork 点赞 / 取消点赞。
func LikeShowcaseWork(w http.ResponseWriter, r *http.Request, id string) {
	liked, likes, err := service.ToggleShowcaseLike(r.Context(), id)
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, model.ShowcaseWorkSummary{ID: id, Liked: liked, Likes: likes})
}

// ForkShowcaseWork 复制精选作品画布到自己的项目。
func ForkShowcaseWork(w http.ResponseWriter, r *http.Request, id string) {
	data, err := service.ForkShowcaseWork(r.Context(), id)
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, map[string]any{"data": data})
}

// AdminShowcaseWorks 管理端精选列表（含下架作品）。
func AdminShowcaseWorks(w http.ResponseWriter, r *http.Request) {
	result, err := service.ListAllShowcaseWorks(parseQuery(r), showcaseUserID(r))
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, result)
}

// AdminSetShowcaseStatus 批量上下架精选作品。
func AdminSetShowcaseStatus(w http.ResponseWriter, r *http.Request) {
	var request struct {
		IDs    []string `json:"ids"`
		Status string   `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || len(request.IDs) == 0 {
		Fail(w, "参数无效")
		return
	}
	if err := service.SetShowcaseStatus(request.IDs, request.Status); err != nil {
		FailError(w, err)
		return
	}
	OK(w, true)
}

// AdminDeleteShowcaseWork 删除精选作品。
func AdminDeleteShowcaseWork(w http.ResponseWriter, r *http.Request, id string) {
	if err := service.DeleteShowcaseWork(id); err != nil {
		FailError(w, err)
		return
	}
	OK(w, true)
}
