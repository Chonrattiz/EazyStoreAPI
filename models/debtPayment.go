package models

import "time"

type DebtPayment struct {
	PaymentID     int       `json:"payment_id" gorm:"primaryKey;autoIncrement"`
	DebtorID      int       `json:"debtor_id"`
	AmountPaid    float64   `json:"amount_paid"`
	PaymentMethod string    `json:"payment_method"` // cash, transfer
	CurrentDebt   float64   `json:"current_debt"`   // ยอดหนี้คงเหลือหลังจ่าย
	PaymentDate   time.Time `json:"payment_date"`
	RecordedBy    string    `json:"recorded_by"` // ชื่อคนรับเงิน (pay_with)
}

type PayDebtRequest struct {
	ShopID        int     `json:"shop_id" binding:"required" example:"3"`             // ไอดีร้านค้า
	DebtorID      int     `json:"debtor_id" binding:"required" example:"1"`           // ไอดีลูกหนี้
	AmountPaid    float64 `json:"amount_paid" binding:"required" example:"45.00"`     // ยอดเงินที่จ่าย
	PaymentMethod string  `json:"payment_method" binding:"required" example:"จ่ายเงินสด"` // วิธีจ่าย (เงินสด/เงินโอน)
	PayWith       string  `json:"pay_with" binding:"required" example:"น้องปอ"`       // ชื่อพนักงานที่รับเงิน
	PinCode       string  `json:"pin_code" binding:"required,len=6" example:"191047"` // รหัส PIN 6 หลัก
}
