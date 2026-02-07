package models

// Category โครงสร้างข้อมูลสำหรับหมวดหมู่สินค้าที่ตรงกับฐานข้อมูล
type Category struct {
    CategoryID int    `json:"category_id" gorm:"primaryKey;column:category_id"`
    Name       string `json:"name" gorm:"column:name"`
}

// TableName กำหนดชื่อตารางให้ตรงกับในฐานข้อมูล (ตามรูป DBeaver)
func (Category) TableName() string {
    return "category"
}