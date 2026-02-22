package controller

import (
	"EazyStoreAPI/database"

	"net/http"

	"github.com/gin-gonic/gin"
)

// GetSalesSummary ดึงข้อมูลสรุปยอดขาย (รายวัน/เดือน/ปี)
func GetSalesSummary(c *gin.Context) {
	shopID := c.Query("shop_id")
	startDate := c.Query("start_date") // รับรูปแบบ "YYYY-MM-DD"
	endDate := c.Query("end_date")     // รับรูปแบบ "YYYY-MM-DD"

	// ตรวจสอบว่าส่งพารามิเตอร์มาครบหรือไม่
	if shopID == "" || startDate == "" || endDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาส่ง shop_id, start_date และ end_date ให้ครบถ้วน"})
		return
	}

	// สร้าง Struct สำหรับรับค่าผลลัพธ์จาก Database
	var result struct {
		TotalSales        float64
		TotalTransactions int
		TotalCost         float64
	}

	// 1. ดึง "ยอดขายรวม (net_price)" และ "จำนวนบิล (sale_id)" จากตาราง sales
	database.DB.Table("sales").
		Select("COALESCE(SUM(net_price), 0) as total_sales, COUNT(sale_id) as total_transactions").
		Where("shop_id = ? AND created_at >= ? AND created_at <= ?", shopID, startDate, endDate).
		Scan(&result)

	// 2. ดึง "ต้นทุนรวม (amount * cost_price)" โดยการ JOIN ตาราง sale_items, sales และ products
	database.DB.Table("sale_items").
		Select("COALESCE(SUM(sale_items.amount * products.cost_price), 0) as total_cost").
		Joins("JOIN sales ON sales.sale_id = sale_items.sale_id").
		Joins("JOIN products ON products.product_id = sale_items.product_id").
		Where("sales.shop_id = ? AND sales.created_at >= ? AND sales.created_at <= ?", shopID, startDate, endDate).
		Scan(&result.TotalCost)

	// 3. คำนวณกำไรสุทธิ และส่งกลับเป็น JSON ให้ Flutter นำไปโชว์
	c.JSON(http.StatusOK, gin.H{
		"sales":        result.TotalSales,
		"cost":         result.TotalCost,
		"profit":       result.TotalSales - result.TotalCost, // กำไร = ยอดขายรวม - ต้นทุนรวม
		"transactions": result.TotalTransactions,
	})
}
