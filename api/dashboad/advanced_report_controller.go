package controller

import (
	"EazyStoreAPI/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetAdvancedReport(c *gin.Context) {
	shopID := c.Query("shop_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if shopID == "" || startDate == "" || endDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาส่ง shop_id, start_date และ end_date ให้ครบถ้วน"})
		return
	}

	// 1. Sales Chart (Hourly if 1 day, Daily otherwise)
	type ChartItem struct {
		Date       string  `json:"date"`
		TotalSales float64 `json:"total_sales"`
	}
	var salesChart []ChartItem
	if startDate == endDate {
		// Group by Hour
		database.DB.Table("sales").
			Select("HOUR(created_time) as date, COALESCE(SUM(net_price), 0) as total_sales").
			Where("shop_id = ? AND created_at = ?", shopID, startDate).
			Group("HOUR(created_time)").
			Order("date ASC").
			Scan(&salesChart)
	} else {
		database.DB.Table("sales").
			Select("DATE(created_at) as date, COALESCE(SUM(net_price), 0) as total_sales").
			Where("shop_id = ? AND created_at >= ? AND created_at <= ?", shopID, startDate, endDate).
			Group("DATE(created_at)").
			Order("date ASC").
			Scan(&salesChart)
	}

	// 1.5 Summary Stats (Transactions, Net Sales, Average)
	var summaryStats struct {
		TotalTransactions int     `json:"total_transactions"`
		TotalSales        float64 `json:"total_sales"`
		AverageSales      float64 `json:"average_sales"`
	}
	database.DB.Table("sales").
		Select(`
			COUNT(sale_id) as total_transactions,
			COALESCE(SUM(net_price), 0) as total_sales,
			COALESCE(AVG(net_price), 0) as average_sales
		`).
		Where("shop_id = ? AND created_at >= ? AND created_at <= ?", shopID, startDate, endDate).
		Scan(&summaryStats)

	// 2. Payment Methods Breakdown
	var paymentStats struct {
		PaidCash     float64 `json:"paid_cash"`
		PaidTransfer float64 `json:"paid_transfer"`
		DebtAmount   float64 `json:"debt_amount"`
	}
	database.DB.Table("sales").
		Select(`
			COALESCE(SUM(CASE WHEN payment_method = 'จ่ายเงินสด' THEN (CASE WHEN pay >= net_price THEN net_price ELSE pay END) ELSE 0 END), 0) as paid_cash,
			COALESCE(SUM(CASE WHEN payment_method = 'โอนจ่าย' THEN (CASE WHEN pay >= net_price THEN net_price ELSE pay END) ELSE 0 END), 0) as paid_transfer,
			COALESCE(SUM(CASE WHEN pay < net_price OR payment_method = 'ค้างชำระ' THEN net_price - pay ELSE 0 END), 0) as debt_amount
		`).
		Where("shop_id = ? AND created_at >= ? AND created_at <= ?", shopID, startDate, endDate).
		Scan(&paymentStats)

	// 3. Top 5 Products
	type TopProduct struct {
		ProductName string  `json:"product_name"`
		TotalQty    int     `json:"total_qty"`
		TotalSales  float64 `json:"total_sales"`
	}
	var topProducts []TopProduct
	database.DB.Table("sale_items").
		Select("products.name as product_name, SUM(sale_items.amount) as total_qty, SUM(sale_items.total_price) as total_sales").
		Joins("JOIN sales ON sales.sale_id = sale_items.sale_id").
		Joins("JOIN products ON products.product_id = sale_items.product_id").
		Where("sales.shop_id = ? AND sales.created_at >= ? AND sales.created_at <= ?", shopID, startDate, endDate).
		Group("products.product_id, products.name").
		Order("total_qty DESC").
		Limit(5).
		Scan(&topProducts)

	// 4. Debt Summary
	var debtSummary struct {
		TotalOutstanding float64 `json:"total_outstanding"`
		CollectedThisMonth float64 `json:"collected_this_month"`
	}
	// Total Outstanding Debt across all debtors
	database.DB.Table("debtors").
		Select("COALESCE(SUM(current_debt), 0) as total_outstanding").
		Where("shop_id = ?", shopID).
		Scan(&debtSummary.TotalOutstanding)

	// Collected this period (uses the selected start and end dates)
	database.DB.Table("debt_payments").
		Select("COALESCE(SUM(amount_paid), 0) as collected_this_month").
		Joins("JOIN debtors ON debtors.debtor_id = debt_payments.debtor_id").
		Where("debtors.shop_id = ? AND DATE(payment_date) >= ? AND DATE(payment_date) <= ?", shopID, startDate, endDate).
		Scan(&debtSummary.CollectedThisMonth)

	// 5. Aging Report (from sales that are unpaid)
	var aging struct {
		Safe    float64 `json:"safe"`    // 1-15 days
		Warning float64 `json:"warning"` // 16-30 days
		Danger  float64 `json:"danger"`  // >30 days
	}
	database.DB.Table("sales").
		Select(`
			COALESCE(SUM(CASE WHEN DATEDIFF(CURDATE(), created_at) <= 15 THEN net_price - pay ELSE 0 END), 0) as safe,
			COALESCE(SUM(CASE WHEN DATEDIFF(CURDATE(), created_at) BETWEEN 16 AND 30 THEN net_price - pay ELSE 0 END), 0) as warning,
			COALESCE(SUM(CASE WHEN DATEDIFF(CURDATE(), created_at) > 30 THEN net_price - pay ELSE 0 END), 0) as danger
		`).
		Where("shop_id = ? AND (pay < net_price OR payment_method = 'ค้างชำระ')", shopID).
		Scan(&aging)

	// 6. Top 5 Debtors
	type TopDebtor struct {
		DebtorID    int     `json:"debtor_id"`
		Name        string  `json:"name"`
		CurrentDebt float64 `json:"current_debt"`
	}
	var topDebtors []TopDebtor
	database.DB.Table("debtors").
		Select("debtor_id, name, current_debt").
		Where("shop_id = ? AND current_debt > 0", shopID).
		Order("current_debt DESC").
		Limit(5).
		Scan(&topDebtors)

	// 7. Debt Collection Statement (based on selected range)
	var debtCollection struct {
		NewDebt       float64 `json:"new_debt"`
		CollectedDebt float64 `json:"collected_debt"`
	}
	database.DB.Table("sales").
		Select("COALESCE(SUM(net_price - pay), 0) as new_debt").
		Where("shop_id = ? AND DATE(created_at) >= ? AND DATE(created_at) <= ? AND (pay < net_price OR payment_method = 'ค้างชำระ')", shopID, startDate, endDate).
		Scan(&debtCollection.NewDebt)

	database.DB.Table("debt_payments").
		Select("COALESCE(SUM(amount_paid), 0) as collected_debt").
		Joins("JOIN debtors ON debtors.debtor_id = debt_payments.debtor_id").
		Where("debtors.shop_id = ? AND DATE(payment_date) >= ? AND DATE(payment_date) <= ?", shopID, startDate, endDate).
		Scan(&debtCollection.CollectedDebt)

	// Combine into response
	c.JSON(http.StatusOK, gin.H{
		"sales_chart":     salesChart,
		"summary_stats":   summaryStats,
		"payment_methods": paymentStats,
		"top_products":    topProducts,
		"debt_summary":    debtSummary,
		"aging_report":    aging,
		"top_debtors":     topDebtors,
		"debt_collection": debtCollection,
	})
}
