package models

type Shop struct {
	ShopID    int    `json:"shop_id" gorm:"primaryKey"`
	UserID    int    `json:"user_id" binding:"required" example:"1"`
	Name      string `json:"name" binding:"required" example:"จันทร์เพ็ญ"`
	Phone     string `json:"phone" binding:"required" example:"0985490445"`
	Address   string `json:"address" binding:"required" example:"123 ถ.สุขุมวิท แขวงคลองเตย เขตคลองเตย กทม 10110"`
	ImgQrcode string `json:"img_qrcode" example:"https://image.url/qrcode.jpg"`
	ImgShop   string `json:"img_shop" example:"https://image.url/homeshop.jpg"`
	Pincode   string `json:"pin_code" binding:"required,len=6" example:"191047"`
}
