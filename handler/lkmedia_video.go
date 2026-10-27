package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/tigerowo/infinite-canvas/service"
)

const lkMediaUploadMaxBytes = 10 << 20

func prepareLKMediaVideoRequest(input aiProtocolRequest) (aiProtocolRequest, bool, error) {
	if !service.IsLKMediaChannel(input.channel) || input.endpoint != "/videos" {
		return input, false, nil
	}
	input.failureLabel = "LK888 媒体"
	body, contentType, err := normalizeLKMediaVideoBody(input.body, input.contentType, input.modelName)
	if err != nil {
		return input, true, err
	}
	input.body = body
	input.contentType = contentType
	return input, true, nil
}

// normalizeLKMediaVideoBody 把前端通用视频表单（JSON 或 multipart）转成 LK888
// 媒体协议的 {model, prompt, params} 请求体；已是该结构时只补必填参数。
func normalizeLKMediaVideoBody(body []byte, contentType string, modelName string) ([]byte, string, error) {
	var root map[string]any
	if strings.HasPrefix(strings.ToLower(contentType), "multipart/form-data") {
		payload, err := lkMediaFormPayload(body, contentType)
		if err != nil {
			return body, contentType, err
		}
		root = payload
	} else if err := json.Unmarshal(body, &root); err != nil {
		return body, contentType, err
	}
	if root == nil {
		root = map[string]any{}
	}
	params, _ := root["params"].(map[string]any)
	if params == nil {
		params = map[string]any{}
	}
	for key, value := range lkMediaFlatParams(root) {
		if _, exists := params[key]; !exists {
			params[key] = value
		}
	}
	params["duration"] = strconv.Itoa(lkMediaDuration(readStringPath(params, "duration")))
	params["resolution"] = lkMediaResolution(modelName, readStringPath(params, "resolution"))
	root["params"] = params
	if strings.TrimSpace(readStringPath(root, "model")) == "" && strings.TrimSpace(modelName) != "" {
		root["model"] = modelName
	}
	encoded, err := json.Marshal(root)
	if err != nil {
		return body, contentType, err
	}
	return encoded, "application/json", nil
}

// lkMediaFlatParams 从平铺的标准字段（seconds/size/首尾帧/参考素材）组装 params。
func lkMediaFlatParams(root map[string]any) map[string]any {
	params := map[string]any{}
	if seconds := readStringPath(root, "seconds"); seconds != "" {
		params["duration"] = seconds
	}
	if resolution := firstNonEmpty(readStringPath(root, "resolution_name"), readStringPath(root, "resolution")); resolution != "" {
		params["resolution"] = resolution
	}
	if ratio := lkMediaAspectRatio(readStringPath(root, "size")); ratio != "" {
		params["aspect_ratio"] = ratio
	}
	var frames []string
	for _, key := range []string{"first_frame_url", "last_frame_url"} {
		if url := readStringPath(root, key); url != "" {
			frames = append(frames, url)
		}
	}
	if len(frames) > 0 {
		params["mode"] = "shouweizhen"
		params["images"] = frames
	} else if imageRefs := readStringSlice(root["input_reference[]"]); len(imageRefs) > 0 {
		params["mode"] = "cankaosheng"
		params["image_url"] = imageRefs
	}
	if videos := readStringSlice(root["video_reference[]"]); len(videos) > 0 {
		if _, ok := params["mode"]; !ok {
			params["mode"] = "cankaosheng"
		}
		params["video_url"] = videos
	}
	if audios := readStringSlice(root["audio_reference[]"]); len(audios) > 0 {
		if _, ok := params["mode"]; !ok {
			params["mode"] = "cankaosheng"
		}
		params["audio_url"] = audios
	}
	return params
}

func lkMediaFormPayload(body []byte, contentType string) (map[string]any, error) {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, err
	}
	form, err := multipart.NewReader(bytes.NewReader(body), params["boundary"]).ReadForm(32 << 20)
	if err != nil {
		return nil, err
	}
	defer form.RemoveAll()
	payload := map[string]any{}
	for key, values := range form.Value {
		if len(values) == 0 {
			continue
		}
		if strings.HasSuffix(key, "[]") {
			items := make([]any, 0, len(values))
			for _, value := range values {
				if text := strings.TrimSpace(value); text != "" {
					items = append(items, text)
				}
			}
			if len(items) > 0 {
				payload[key] = items
			}
			continue
		}
		payload[key] = strings.TrimSpace(values[len(values)-1])
	}
	for key, files := range form.File {
		items := make([]any, 0, len(files))
		for _, file := range files {
			dataURL, err := lkMediaFileDataURL(file)
			if err != nil {
				return nil, err
			}
			items = append(items, dataURL)
		}
		if len(items) == 0 {
			continue
		}
		if existing, exists := payload[key]; exists {
			payload[key] = append(lkMediaAnyList(existing), items...)
		} else if len(items) == 1 && !strings.HasSuffix(key, "[]") {
			payload[key] = items[0]
		} else {
			payload[key] = items
		}
	}
	return payload, nil
}

func lkMediaAnyList(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case string:
		if strings.TrimSpace(typed) != "" {
			return []any{typed}
		}
	}
	return nil
}

func lkMediaFileDataURL(file *multipart.FileHeader) (string, error) {
	opened, err := file.Open()
	if err != nil {
		return "", err
	}
	defer opened.Close()
	data, err := io.ReadAll(io.LimitReader(opened, lkMediaUploadMaxBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > lkMediaUploadMaxBytes {
		return "", errors.New(file.Filename + " 超过 10MB，无法直接提交给媒体接口")
	}
	mimeType := strings.TrimSpace(file.Header.Get("Content-Type"))
	if mimeType == "" || strings.HasPrefix(mimeType, "application/octet-stream") {
		mimeType = http.DetectContentType(data)
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func lkMediaDuration(value string) int {
	seconds, _ := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(value), "s")))
	if seconds < 4 {
		return 4
	}
	if seconds > 15 {
		return 15
	}
	return seconds
}

func lkMediaResolution(modelName string, value string) string {
	token := strings.ToLower(strings.TrimSpace(value))
	token = strings.TrimSuffix(token, "p")
	minimax := strings.Contains(strings.ToLower(strings.TrimSpace(modelName)), "minimax")
	switch token {
	case "1080":
		if minimax {
			return "1080P"
		}
		return "1080p"
	case "2k":
		if minimax {
			return "2K"
		}
	case "4k":
		return "4K"
	case "480":
		if !minimax {
			return "480p"
		}
	case "720":
		if !minimax {
			return "720p"
		}
	}
	if minimax {
		return "768P"
	}
	return "720p"
}

func lkMediaAspectRatio(value string) string {
	switch strings.TrimSpace(value) {
	case "adaptive", "16:9", "9:16", "1:1", "4:3", "3:4", "21:9":
		return strings.TrimSpace(value)
	}
	return ""
}

// transformLKMediaVideoCreateResponse 转换任务创建响应 {code, data:{task_id}}。
func transformLKMediaVideoCreateResponse(payload []byte) ([]byte, bool) {
	var root any
	if len(payload) == 0 || json.Unmarshal(payload, &root) != nil {
		return nil, false
	}
	data := normalizeVideoPayloadMap(root)
	taskID := firstNonEmpty(readStringPath(data, "task_id"), readStringPath(data, "taskId"))
	if taskID == "" {
		return nil, false
	}
	transformed, err := json.Marshal(map[string]any{"id": taskID, "task_id": taskID, "status": "queued"})
	return transformed, err == nil
}

// transformLKMediaVideoStatusResponse 转换状态响应 {state, progress, result_url, error}。
func transformLKMediaVideoStatusResponse(payload []byte) ([]byte, bool) {
	var root map[string]any
	if len(payload) == 0 || json.Unmarshal(payload, &root) != nil {
		return nil, false
	}
	state := strings.ToLower(readStringPath(root, "state"))
	if state == "" {
		return nil, false
	}
	status := "processing"
	switch state {
	case "success", "completed":
		status = "completed"
	case "failed", "fail", "error", "cancelled", "canceled":
		status = "failed"
	case "pending", "queued":
		status = "queued"
	}
	videoURL := firstNonEmpty(readStringPath(root, "result_url"), findFirstHTTPURL(root["result_urls"]))
	errorMessage := readStringPath(root, "error")
	if errorMessage != "" && status != "completed" {
		status = "failed"
	}
	if status == "completed" && videoURL == "" {
		status = "processing"
	}
	result := map[string]any{
		"task_id":   readStringPath(root, "task_id"),
		"status":    status,
		"progress":  readIntPath(root, "progress"),
		"video_url": videoURL,
	}
	// error 为空时不能写入空对象，否则会被兜底解析成非空错误字符串
	if errorMessage != "" {
		result["error"] = map[string]any{"message": errorMessage}
	}
	transformed, err := json.Marshal(result)
	return transformed, err == nil
}
