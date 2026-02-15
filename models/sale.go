package models

import (
	"time"
)

// Sale เป็น Model หลักสำหรับตาราง sales
type Sale struct {
	SaleID        int       `json:"sale_id" gorm:"primaryKey;autoIncrement"`
	ShopID        int       `json:"shop_id" gorm:"not null"`
	DebtorID      *int      `json:"debtor_id"` // ใช้ Pointer เพื่อให้เป็นค่า Null ได้
	NetPrice      float64   `json:"net_price" gorm:"type:decimal(10,2);not null"`
	Pay           float64   `json:"pay" gorm:"type:decimal(10,2);not null"`
	PaymentMethod string    `json:"payment_method" gorm:"type:varchar(20);not null"` // 'cash', 'transfer', 'credit'
	Note          string    `json:"note" gorm:"type:text"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	CreatedBuy    string    `json:"created_buy" gorm:"type:varchar(100);not null"`

	// สำหรับดึงข้อมูล SaleItems ออกมาพร้อมกับ Sale (ถ้าใช้ GORM)
	SaleItems []SaleItem `json:"sale_items" gorm:"foreignKey:SaleID"`
}

// SaleItem เป็น Model สำหรับรายละเอียดสินค้าในบิลนั้นๆ
type SaleItem struct {
	SaleItemsID  int     `json:"sale_items_id" gorm:"primaryKey;autoIncrement"`
	SaleID       int     `json:"sale_id" gorm:"not null"`
	ProductID    int     `json:"product_id" gorm:"not null"`
	Amount       int     `json:"amount" gorm:"not null"`
	PricePerUnit float64 `json:"price_per_unit" gorm:"type:decimal(10,2);not null"`
	TotalPrice   float64 `json:"total_price" gorm:"type:decimal(10,2);not null"`
}
