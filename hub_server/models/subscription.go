package models

import (
	"time"
)

// Subscription 订阅表 - 记录用户订阅的视频号作者
type Subscription struct {
	ID     uint `json:"id" gorm:"primaryKey"`
	UserID uint `json:"user_id" gorm:"index"` // Hub 用户ID

	// 微信视频号用户信息
	WxUsername  string `json:"wx_username" gorm:"uniqueIndex:idx_user_wx;not null"` // finderUsername
	WxNickname  string `json:"wx_nickname"`
	WxHeadUrl   string `json:"wx_head_url"`
	WxSignature string `json:"wx_signature"`

	// 订阅元数据
	VideoCount    int       `json:"video_count" gorm:"default:0"`
	LastFetchedAt time.Time `json:"last_fetched_at"`
	Status        string    `json:"status" gorm:"default:'active'"` // active, paused

	// Relations
	Videos []SubscribedVideo `json:"videos,omitempty" gorm:"foreignKey:SubscriptionID"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SubscribedVideo 订阅视频表 - 存储订阅用户的视频详情
type SubscribedVideo struct {
	ID             uint `json:"id" gorm:"primaryKey"`
	SubscriptionID uint `json:"subscription_id" gorm:"index"`

	// 视频基本信息（来自微信）
	ObjectID      string `json:"object_id" gorm:"uniqueIndex:idx_sub_video;not null"` // 视频ID
	ObjectNonceID string `json:"object_nonce_id"`                                     // Nonce ID
	Title         string `json:"title"`
	CoverURL      string `json:"cover_url"`
	Description   string `json:"description" gorm:"type:text"`

	// 媒体信息
	Duration int `json:"duration"` // 时长（秒）
	Width    int `json:"width"`
	Height   int `json:"height"`

	// 统计数据
	LikeCount    int `json:"like_count"`
	CommentCount int `json:"comment_count"`

	// 播放信息（用于解密播放）
	VideoURL   string `json:"video_url"`   // 加密视频URL
	DecryptKey string `json:"decrypt_key"` // 解密密钥

	PublishedAt time.Time `json:"published_at"` // 微信发布时间（createTime）
	CreatedAt   time.Time `json:"created_at"`   // 添加到数据库时间
}
