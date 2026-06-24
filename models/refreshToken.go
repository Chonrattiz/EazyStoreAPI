package models

import "time"

// RefreshToken เก็บ refresh token ที่ออกให้ผู้ใช้
type RefreshToken struct {
	TokenID   uint      `gorm:"primaryKey;autoIncrement;column:token_id" json:"token_id"`
	UserID    uint      `gorm:"not null;index;column:user_id" json:"user_id"`
	Token     string    `gorm:"not null;unique;type:varchar(500);column:token" json:"-"` // ไม่ส่ง token ออกไป
	ExpiresAt time.Time `gorm:"not null;column:expires_at" json:"expires_at"`
	CreatedAt time.Time `gorm:"autoCreateTime;column:created_at" json:"created_at"`
	RevokedAt *time.Time `gorm:"column:revoked_at" json:"revoked_at"` // ถ้า logout จะเก็บเวลายกเลิก
	User      User      `gorm:"foreignKey:UserID;references:UserID" json:"-"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
