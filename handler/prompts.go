package handler

import (
	"net/http"

	"github.com/tigerowo/infinite-canvas/service"
)

func Prompts(w http.ResponseWriter, r *http.Request) {
	result, err := service.ListPrompts(parseQuery(r))
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, result)
}

// PromptCover 返回提示词封面的本地缓存图片。
func PromptCover(w http.ResponseWriter, r *http.Request, name string) {
	data, contentType, err := service.ReadPromptCover(name)
	if err != nil {
		FailWithStatus(w, http.StatusNotFound, err.Error())
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=604800")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
