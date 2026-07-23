package models

// ProductUnit คือหน่วยขายเพิ่มเติมของสินค้า (เช่น ลัง, แพ็ค) ที่แปลงเข้าหน่วยฐาน
// ของสินค้า (products.unit) ได้ตรงๆ ด้วย ConversionQty เช่น เบียร์ 1 ลัง = 12 ขวด
type ProductUnit struct {
	ProductUnitID int     `json:"product_unit_id" gorm:"primaryKey;autoIncrement"`
	ProductID     int     `json:"product_id" gorm:"not null"`
	UnitName      string  `json:"unit_name" binding:"required" example:"ลัง"`
	ConversionQty int     `json:"conversion_qty" binding:"required,gt=1" example:"12"`
	Barcode       *string `json:"barcode" example:"885123400012"`
	SellPrice     float64 `json:"sell_price" binding:"required,gt=0" example:"340.00"`
	CostPrice     float64 `json:"cost_price" example:"320.00"`
	Status        bool    `json:"status" example:"true"`
}
