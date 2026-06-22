package controller

import (
	"EazyStoreAPI/database"

	"net/http"

	"github.com/gin-gonic/gin"
)

// GetSalesSummary ดึงข้อมูลสรุปยอดขาย (รายวัน/เดือน/ปี)
// GetSalesSummary ดึงข้อมูลสรุปยอดขาย (รายวัน/เดือน/ปี)
func GetSalesSummary(c *gin.Context) {
	shopID := c.Query("shop_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if shopID == "" || startDate == "" || endDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาส่ง shop_id, start_date และ end_date ให้ครบถ้วน"})
		return
	}

	// 1. สร้าง Struct เพิ่มตัวแปร PaidCash และ PaidTransfer
	var result struct {
		TotalRevenue      float64
		ActualPaid        float64
		PaidCash          float64 // รับเป็นเงินสด
		PaidTransfer      float64 // รับเป็นโอนจ่าย
		DebtAmount        float64
		TotalTransactions int
		TotalCost         float64
	}

	// 2. Query ขั้นเทพ: แยกคำนวณเงินสด โอน และหนี้ (หักเงินทอนให้เรียบร้อย)
	database.DB.Table("sales").
		Select(`
			COALESCE(SUM(net_price), 0) as total_revenue,

			-- ยอดรับจริงรวมทั้งหมด
			COALESCE(SUM(CASE WHEN pay >= net_price THEN net_price ELSE pay END), 0) as actual_paid,

			-- แยกเฉพาะ "จ่ายเงินสด" (ถ้าทอนเงิน ก็คิดแค่ราคาของ)
			COALESCE(SUM(CASE WHEN payment_method = 'จ่ายเงินสด' THEN 
				(CASE WHEN pay >= net_price THEN net_price ELSE pay END) 
			ELSE 0 END), 0) as paid_cash,

			-- แยกเฉพาะ "โอนจ่าย" (ปกติโอนจะพอดีเป๊ะอยู่แล้ว แต่ดักไว้เผื่อโอนขาด/เกิน)
			COALESCE(SUM(CASE WHEN payment_method = 'โอนจ่าย' THEN 
				(CASE WHEN pay >= net_price THEN net_price ELSE pay END) 
			ELSE 0 END), 0) as paid_transfer,

			-- ยอดค้างชำระ (บิลที่จ่ายไม่ครบ หรือระบุวิธีจ่ายเป็น ค้างชำระ)
			COALESCE(SUM(CASE WHEN pay < net_price OR payment_method = 'ค้างชำระ' THEN net_price - pay ELSE 0 END), 0) as debt_amount,

			COUNT(sale_id) as total_transactions
		`).
		Where("shop_id = ? AND created_at >= ? AND created_at <= ?", shopID, startDate, endDate).
		Scan(&result)

	// 3. ดึงต้นทุนรวม (เหมือนเดิม)
	database.DB.Table("sale_items").
		Select("COALESCE(SUM(sale_items.amount * products.cost_price), 0) as total_cost").
		Joins("JOIN sales ON sales.sale_id = sale_items.sale_id").
		Joins("JOIN products ON products.product_id = sale_items.product_id").
		Where("sales.shop_id = ? AND sales.created_at >= ? AND sales.created_at <= ?", shopID, startDate, endDate).
		Scan(&result.TotalCost)

	// 4. ส่งค่าทั้งหมดให้ Flutter
	c.JSON(http.StatusOK, gin.H{
		"total_revenue": result.TotalRevenue,
		"actual_paid":   result.ActualPaid,
		"paid_cash":     result.PaidCash,     // ✨ เงินในลิ้นชัก
		"paid_transfer": result.PaidTransfer, // ✨ เงินในธนาคาร
		"debt_amount":   result.DebtAmount,
		"cost":          result.TotalCost,
		"profit":        result.TotalRevenue - result.TotalCost,
		"transactions":  result.TotalTransactions,
	})
}
