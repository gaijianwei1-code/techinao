package service

import (
	"strings"

	"github.com/tigerowo/infinite-canvas/model"
)

const ModelChannelProtocolLKMedia = "lkmedia"

func IsLKMediaChannel(channel model.ModelChannel) bool {
	return strings.EqualFold(strings.TrimSpace(channel.Protocol), ModelChannelProtocolLKMedia)
}

// LK888 媒体渠道的图片/视频模型都是异步媒体接口，不能用 chat/completions 测试；
// 文本模型仍走 OpenAI 兼容聊天测试。
func testLKMediaChannelModel(channel model.ModelChannel, modelName string) (string, error) {
	if isImageModelName(modelName) {
		return "图片模型已跳过聊天测试，请在生图工作台验证。", nil
	}
	if isVideoModelName(modelName) {
		return "LK888 媒体模型为异步生成，请在视频创作台测试。", nil
	}
	return testOpenAIChannelModel(channel, modelName)
}
