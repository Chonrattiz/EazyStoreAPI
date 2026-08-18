package models

// ChartItem แทนข้อมูลแต่ละจุดในกราฟยอดขาย
type ChartItem struct {
	Date       string  `json:"date"`
	TotalSales float64 `json:"total_sales"`
}

// PaymentStats สรุปสัดส่วนวิธีการชำระเงิน
type PaymentStats struct {
	PaidCash     float64 `json:"paid_cash"`
	PaidTransfer float64 `json:"paid_transfer"`
	DebtAmount   float64 `json:"debt_amount"`
}

// TopProduct สินค้าขายดี
type TopProduct struct {
	ProductName string  `json:"product_name"`
	TotalQty    int     `json:"total_qty"`
	TotalSales  float64 `json:"total_sales"`
}

// DebtSummary ภาพรวมหนี้สิน
type DebtSummary struct {
	TotalOutstanding   float64 `json:"total_outstanding"`
	CollectedThisMonth float64 `json:"collected_this_month"`
}

// AgingStats สรุปรายงานอายุหนี้ตามกลุ่มความเสี่ยง
type AgingStats struct {
	Safe    float64 `json:"safe"`    // 1-15 days
	Warning float64 `json:"warning"` // 16-30 days
	Danger  float64 `json:"danger"`  // >30 days
}

// TopDebtor ลูกหนี้ยอดสูงสุด
type TopDebtor struct {
	DebtorID    int     `json:"debtor_id"`
	Name        string  `json:"name"`
	CurrentDebt float64 `json:"current_debt"`
}

// DebtCollection สรุปความเคลื่อนไหวหนี้
type DebtCollection struct {
	NewDebt       float64 `json:"new_debt"`
	CollectedDebt float64 `json:"collected_debt"`
}

// AgingDebtor รายละเอียดลูกหนี้ในรายงานอายุหนี้
type AgingDebtor struct {
	DebtorID    int     `json:"debtor_id"`
	Name        string  `json:"name"`
	Phone       string  `json:"phone"`
	ImgDebtor   string  `json:"img_debtor"`
	DebtSince   string  `json:"debt_since"`
	AmountOwed  float64 `json:"amount_owed"`
	DaysOverdue int     `json:"days_overdue"`
	Bucket      string  `json:"-"`
}
