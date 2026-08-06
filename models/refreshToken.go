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
	RevokedReason *string `gorm:"column:revoked_reason;type:varchar(100)" json:"revoked_reason"` // เหตุผลที่ยกเลิก เช่น "logout"
	DeviceID  *string   `gorm:"column:device_id;type:varchar(255);index" json:"device_id"` // id เครื่องที่ login ไว้ ใช้ logout เฉพาะเครื่อง
	DeviceName *string  `gorm:"column:device_name;type:varchar(255)" json:"device_name"` // ชื่อรุ่นเครื่องแบบอ่านง่าย ใช้โชว์ผลอย่างเดียว ไม่ใช้แยก session
	User      User      `gorm:"foreignKey:UserID;references:UserID" json:"-"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
