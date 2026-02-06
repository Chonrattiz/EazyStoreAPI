package models

type Product struct {
	ProductID   int    `json:"product_id" gorm:"primaryKey"`
	ShopID      int    `json:"shop_id" binding:"required" example:"1"`
	CategoryID  int    `json:"category_id" binding:"required" example:"2"`
	ProductCode string `json:"product_code" gorm:"unique"`
	Name        string `json:"name" binding:"required" example:"น้ำอัดลม โคล่า"`
	//[เทคนิค] Barcode ควรเป็น Pointer (*string) เพื่อรองรับค่า NULL
	Barcode    *string `json:"barcode" example:"885123456789"`
	ImgProduct string  `json:"img_product" binding:"required" example:"https://image.url/cola.jpg"`
	SellPrice  float64 `json:"sell_price" binding:"required,gt=0" example:"15.00"`
	CostPrice  float64 `json:"cost_price" binding:"required,gt=0" example:"10.50"`
	Stock      int     `json:"stock" binding:"gte=0" example:"100"` // gte=0 คือ มากกว่าหรือเท่ากับ 0
	Unit       string  `json:"unit" binding:"required" example:"ขวด"`
	Status     bool    `json:"status" example:"true"`
}
