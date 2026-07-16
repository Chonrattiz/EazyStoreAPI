package models

// Category โครงสร้างข้อมูลสำหรับหมวดหมู่สินค้าที่ตรงกับฐานข้อมูล
type Category struct {
	CategoryID int    `json:"category_id" gorm:"primaryKey;column:category_id"`
	ShopID     int    `json:"shop_id" gorm:"column:shop_id;not null;index"`
	Name       string `json:"name" gorm:"column:name"`
	Status     bool   `json:"status" gorm:"column:status;not null;default:true"`
}

type CreateCategoryInput struct {
	ShopID int    `json:"shop_id" binding:"required"`
	Name   string `json:"name" binding:"required"`
}

type UpdateCategoryInput struct {
	ShopID int    `json:"shop_id" binding:"required"`
	Name   string `json:"name" binding:"required"`
}

// TableName กำหนดชื่อตารางให้ตรงกับในฐานข้อมูล (ตามรูป DBeaver)
func (Category) TableName() string {
	return "category"
}
