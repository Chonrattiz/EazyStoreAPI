package models

// Debtor แทนโครงสร้างข้อมูลลูกหนี้
type Debtor struct {
	DebtorID    int     `json:"debtor_id" gorm:"primaryKey;autoIncrement"`
	ShopID      int     `json:"shop_id"  example:"1"`
	Name        string  `json:"name" example:"ป้าเพ็ญ"`
	Phone       string  `json:"phone" example:"0654891234"`
	Address     string  `json:"address" example:"123 ถ.สุขุมวิท แขวงคลองเตย เขตคลองเตย กทม 10110"`
	ImgDebtor   string  `json:"img_debtor" example:"https://image.url/debtor.jpg"`
	CreditLimit float64 `json:"credit_limit" example:"2000"`
	CurrentDebt float64 `json:"current_debt" example:"0"`
}
