package models

import "time"

// PromtReq structure for storing interaction data
type PromtReq struct {
	ChatID    int64     `json:"chat_id"`
	UserQuery string    `json:"user_query"`
	Quote     string    `json:"quote"`
	Author    string    `json:"author"`
	Timestamp time.Time `json:"timestamp"`
}

// FormattedResponse structure for YandexGPT response format
type FormattedResponse struct {
	Response string `json:"response"`
	Status   string `json:"status"`
}
